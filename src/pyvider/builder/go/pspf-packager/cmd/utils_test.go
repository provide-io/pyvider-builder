package cmd

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"pspf-tools/go/pkg/logbowl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/gozstd"
)

func TestKeyGenerationAndSigning(t *testing.T) {
	// Test Key Generation
	privKeyPEM, pubKeyPEM, err := generateKeyPairPEM()
	require.NoError(t, err, "generateKeyPairPEM should not return an error")
	assert.NotEmpty(t, privKeyPEM, "Private key PEM should not be empty")
	assert.NotEmpty(t, pubKeyPEM, "Public key PEM should not be empty")

	// Create temporary files for the keys
	tmpDir := t.TempDir()
	privKeyPath := filepath.Join(tmpDir, "test.key")
	pubKeyPath := filepath.Join(tmpDir, "test.crt")
	require.NoError(t, os.WriteFile(privKeyPath, privKeyPEM, 0600))
	require.NoError(t, os.WriteFile(pubKeyPath, pubKeyPEM, 0644))

	// Test Loading Keys
	loadedPrivKey, err := loadPrivateKey(privKeyPath)
	require.NoError(t, err, "loadPrivateKey should not return an error")
	assert.NotNil(t, loadedPrivKey, "Loaded private key should not be nil")

	loadedPubKey, err := loadPublicKeyFromFile(pubKeyPath)
	require.NoError(t, err, "loadPublicKeyFromFile should not return an error")
	assert.NotNil(t, loadedPubKey, "Loaded public key should not be nil")

	// Test Signing and Verification
	payload := []byte("This is a test payload for signing.")
	signature, err := signPayload(payload, privKeyPath)
	require.NoError(t, err, "signPayload should not return an error")
	assert.NotEmpty(t, signature, "Signature should not be empty")

	err = verifyPayload(payload, signature, loadedPubKey)
	assert.NoError(t, err, "verifyPayload should succeed with the correct key and signature")

	// Test Verification Failure (wrong payload)
	wrongPayload := []byte("This is the wrong payload.")
	err = verifyPayload(wrongPayload, signature, loadedPubKey)
	assert.Error(t, err, "verifyPayload should fail with the wrong payload")

	// Test Verification Failure (wrong key)
	_, otherPubKeyPEM, _ := generateKeyPairPEM()
	otherPubKeyPath := filepath.Join(tmpDir, "other.crt")
	require.NoError(t, os.WriteFile(otherPubKeyPath, otherPubKeyPEM, 0644))
	otherLoadedPubKey, _ := loadPublicKeyFromFile(otherPubKeyPath)
	err = verifyPayload(payload, signature, otherLoadedPubKey)
	assert.Error(t, err, "verifyPayload should fail with the wrong public key")
}

func TestCreateSourceArchive(t *testing.T) {
	tmpDir := t.TempDir()
	log := logbowl.Create("test-archive")

	// Create a dummy source directory structure
	sourceDir := filepath.Join(tmpDir, "source")
	require.NoError(t, os.Mkdir(sourceDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("hello"), 0644))

	subDir := filepath.Join(sourceDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("world"), 0644))
	
	// Create a file to be excluded
	excludeDir := filepath.Join(sourceDir, ".venv")
	require.NoError(t, os.Mkdir(excludeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(excludeDir, "ignored.txt"), []byte("ignore me"), 0644))


	// Test without excludes
	archiveBytes, err := createSourceArchive(log, sourceDir, []string{})
	require.NoError(t, err)
	assert.NotEmpty(t, archiveBytes)

	// Verify archive contents
	filesInArchive := getFilesInTarGz(t, archiveBytes)
	assert.Contains(t, filesInArchive, "file1.txt")
	assert.Contains(t, filesInArchive, "subdir/file2.txt")
	assert.Contains(t, filesInArchive, ".venv/ignored.txt")
	assert.Len(t, filesInArchive, 3, "Should contain 3 files when no excludes are used")

	// Test with excludes
	excludePatterns := []string{"**/.venv/**"}
	archiveBytesExcluded, err := createSourceArchive(log, sourceDir, excludePatterns)
	require.NoError(t, err)
	assert.NotEmpty(t, archiveBytesExcluded)

	// Verify excluded archive contents
	filesInArchiveExcluded := getFilesInTarGz(t, archiveBytesExcluded)
	assert.Contains(t, filesInArchiveExcluded, "file1.txt")
	assert.Contains(t, filesInArchiveExcluded, "subdir/file2.txt")
	assert.NotContains(t, filesInArchiveExcluded, ".venv/ignored.txt")
	assert.Len(t, filesInArchiveExcluded, 2, "Should contain 2 files after exclusion")
}

// Helper function to read a .tar.gz byte slice and return the list of files
func getFilesInTarGz(t *testing.T, data []byte) map[string]bool {
	zr := gozstd.NewReader(bytes.NewReader(data))
	defer zr.Release()
	tr := tar.NewReader(zr)
	files := make(map[string]bool)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if header.Typeflag == tar.TypeReg {
			files[header.Name] = true
		}
	}
	return files
}
