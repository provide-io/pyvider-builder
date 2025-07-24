package cmd

import (
	"os"
	"os/exec"
	"strings"

	"pspf-tools/go/pkg/logbowl"
)

// buildWheelsFromDependencies takes a list of dependency specifiers, resolves them
// using `uv`, and returns the path to a temporary directory containing the built wheels.
func buildWheelsFromDependencies(log logbowl.Logger, dependencies []string) (string, error) {
	if len(dependencies) == 0 {
		log.Info("builder", "deps", "skip", "No Python dependencies provided.")
		// Return a new empty temp dir so the caller can safely defer os.RemoveAll
		return os.MkdirTemp("", "pspf-empty-wheels-")
	}

	wheelDir, err := os.MkdirTemp("", "pspf-wheels-")
	if err != nil {
		return "", err
	}

	log.Info("builder", "deps", "progress", "Building Python wheels from dependencies...", "count", len(dependencies))

	var localPaths []string
	var remoteDeps []string

	for _, dep := range dependencies {
		// A simple heuristic to differentiate local paths from PyPI packages.
		// This could be improved, but works for `./...` and `...>=...`.
		if strings.HasPrefix(dep, ".") || strings.HasPrefix(dep, "/") || strings.HasPrefix(dep, "..") {
			localPaths = append(localPaths, dep)
		} else {
			remoteDeps = append(remoteDeps, dep)
		}
	}

	uvPath, err := exec.LookPath("uv")
	if err != nil {
		return "", err
	}

	// Install remote dependencies first
	if len(remoteDeps) > 0 {
		log.Debug("builder", "deps", "progress", "Fetching remote dependencies...", "deps", remoteDeps)
		// CORRECTED: Use `uv run pip download --dest ...`
		downloadArgs := append([]string{"run", "pip", "download", "--dest", wheelDir}, remoteDeps...)
		cmd := exec.Command(uvPath, downloadArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
	}

	// Build local paths into wheels
	for _, path := range localPaths {
		log.Debug("builder", "deps", "progress", "Building local dependency...", "path", path)
		// CORRECTED: Use the required `uv run pip wheel` command structure.
		buildArgs := []string{"run", "pip", "wheel", path, "--wheel-dir", wheelDir}
		cmd := exec.Command(uvPath, buildArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", err
		}
	}

	log.Info("builder", "deps", "success", "Successfully built all Python wheels.", "outputDir", wheelDir)
	return wheelDir, nil
}
