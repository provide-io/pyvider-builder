package main

import (
	"archive/tar"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/gozstd"
)

// TestMainAsSubprocess is a true black-box integration test.
// It builds the launcher binary and then runs it as a subprocess with different
// environment variables to test its interactive commands and logging output.
func TestMainAsSubprocess(t *testing.T) {
	// Build the pspf-launcher binary specifically for this test.
	tmpDir := t.TempDir()
	launcherPath := filepath.Join(tmpDir, "test-launcher")

	// Assuming the test is run from the `go` module root.
	// We build the launcher from its sub-package.
	buildCmd := exec.Command("go", "build", "-o", launcherPath, "./pspf-launcher")
	// The working directory for `go build` must be the module root.
	buildCmd.Dir = ".."
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build pspf-launcher for testing")

	// --- Test Case 1: Interactive 'info' command ---
	t.Run("Interactive info command", func(t *testing.T) {
		// To test the 'info' command, we need a valid PSPF file.
		// We can't easily create one here, so we'll test that the command
		// runs and fails gracefully on a non-PSPF file (our test binary).
		cmd := exec.Command(launcherPath, "info")
		cmd.Env = append(os.Environ(), "PSPF_INTERACTIVE=true")

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		require.NoError(t, err, "Running the info command should not cause the test to fail, even if the command itself reports an error")

		output := stdout.String()
		assert.Contains(t, output, "PSPF Package Information for:")
		// It will fail to read the footer of itself, which is expected.
		// The correct error is now 'invalid internal magic'.
		assert.Contains(t, output, "Error reading footer: invalid internal magic")
	})

	// --- Test Case 2: JSON logging format ---
	t.Run("JSON logging format", func(t *testing.T) {
		// We can't test the full runProvider path easily, but we can test
		// that the logger is initialized correctly by the 'info' command.
		cmd := exec.Command(launcherPath, "info")
		cmd.Env = append(os.Environ(),
			"PSPF_INTERACTIVE=true",
			"PYVIDER_LOG_LEVEL=DEBUG",
			"PYVIDER_LOG_CONSOLE_FORMATTER=json",
		)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err)

		// The 'info' command doesn't log much, but if it did, it would be JSON.
		// This test mainly ensures the binary runs with these env vars without crashing.
		// A more advanced test would require a valid PSPF file to trigger more logs.
		assert.Contains(t, string(output), "PSPF Package Information for:")
	})

	// --- Test Case 3: Default run mode (should fail) ---
	t.Run("Default run mode fails on non-PSPF file", func(t *testing.T) {
		cmd := exec.Command(launcherPath)
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		// The command should exit with a non-zero status code.
		assert.Error(t, err, "Expected the launcher to fail when run on a non-PSPF file")

		// Check for the expected error message in the output.
		// The correct error is now 'invalid internal magic'.
		assert.Contains(t, string(output), "invalid internal magic")
	})
}

func TestUnTar_ZipSlip(t *testing.T) {
	// Create a temporary directory for the test
	dest := t.TempDir()

	// Create a tar archive with a zip slip vulnerability
	var buf bytes.Buffer
	zw := gozstd.NewWriter(&buf)
	tw := tar.NewWriter(zw)

	// Add a file with a path that attempts to escape the destination directory
	hdr := &tar.Header{
		Name: "../evil.txt",
		Mode: 0600,
		Size: int64(len("evil content")),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil content")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	zw.Close()

	// Attempt to extract the archive
	_, err := unTar(&buf, dest)
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}

	// Check that the evil file was not created
	if _, err := os.Stat(filepath.Join(dest, "../evil.txt")); !os.IsNotExist(err) {
		t.Fatal("zip slip vulnerability exploited")
	}
}

func TestUnTar_LongFileName(t *testing.T) {
	// Create a temporary directory for the test
	dest := t.TempDir()

	// Create a tar archive with a long file name
	var buf bytes.Buffer
	zw := gozstd.NewWriter(&buf)
	tw := tar.NewWriter(zw)

	// Add a file with a very long name
	longName := string(make([]byte, 1024))
	hdr := &tar.Header{
		Name: longName,
		Mode: 0600,
		Size: int64(len("some content")),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("some content")); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	zw.Close()

	// Attempt to extract the archive
	_, err := unTar(&buf, dest)
	if err != nil {
		t.Fatalf("expected no error, but got: %v", err)
	}

	// Check that the file was created
	if _, err := os.Stat(filepath.Join(dest, longName)); os.IsNotExist(err) {
		t.Fatal("long file name not created")
	}
}
