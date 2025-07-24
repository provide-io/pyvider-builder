package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	buildPayloadDir       string
	buildPackageKeyPath   string
	buildOutPath          string
	buildLauncherBin      string
	buildConfigFile       string
	buildUvPath           string
	buildPublicKeyPath    string
	buildPythonInstallDir string
	buildExcludePatterns  []string
	buildDependencies     []string // New flag to accept dependencies
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Builds a self-contained PSPF package.",
	Run: func(cmdCobra *cobra.Command, args []string) {
		if buildOutPath == "" || buildLauncherBin == "" || buildPackageKeyPath == "" || buildPublicKeyPath == "" || buildPythonInstallDir == "" {
			log.Error("builder", "validate", "error", "All required flags must be provided.")
			os.Exit(1)
		}

		if _, err := os.Stat(buildPackageKeyPath); os.IsNotExist(err) {
			log.Error("builder", "validate", "failure", "Private key not found.", "path", buildPackageKeyPath)
			os.Exit(1)
		}
		if _, err := os.Stat(buildPublicKeyPath); os.IsNotExist(err) {
			log.Error("builder", "validate", "failure", "Public key not found.", "path", buildPublicKeyPath)
			os.Exit(1)
		}

		launcherData, err := os.ReadFile(buildLauncherBin)
		if err != nil {
			log.Error("builder", "read", "error", "Failed to read launcher binary", "path", buildLauncherBin, "error", err)
			os.Exit(1)
		}

		pythonInstallTgzBytes, err := createSourceArchive(log, buildPythonInstallDir, buildExcludePatterns)
		if err != nil {
			log.Error("builder", "archive", "error", "Failed to archive Python installation dir", "error", err)
			os.Exit(1)
		}

		var uvBinBytes []byte
		actualUvPath := buildUvPath
		if actualUvPath == "" {
			resolvedPath, _ := exec.LookPath("uv")
			actualUvPath = resolvedPath
		}
		uvBinBytes, err = os.ReadFile(actualUvPath)
		if err != nil {
			log.Error("builder", "read", "error", "Failed to read 'uv' binary", "path", actualUvPath, "error", err)
			os.Exit(1)
		}

		var configJsonBytes []byte
		if buildConfigFile != "" {
			configJsonBytes, err = os.ReadFile(buildConfigFile)
		} else {
			configJsonBytes = []byte(`{"entry_point": "pyvider.provider_core:setup_provider"}`)
		}

		// --- New Dependency Handling Logic ---
		wheelDir, err := buildWheelsFromDependencies(log, buildDependencies)
		if err != nil {
			log.Error("builder", "deps", "error", "Failed to build Python wheels from dependencies", "error", err)
			os.Exit(1)
		}
		defer os.RemoveAll(wheelDir)
		// --- End New Logic ---

		// Combine the built wheels with any other payload assets
		finalPayloadDir, err := os.MkdirTemp("", "pspf-final-payload-")
		if err != nil {
			log.Error("builder", "temp", "error", "Failed to create final payload directory", "error", err)
			os.Exit(1)
		}
		defer os.RemoveAll(finalPayloadDir)

		// Copy wheels
		copyDirContents(wheelDir, finalPayloadDir)
		// Copy other assets if provided
		if buildPayloadDir != "" {
			copyDirContents(buildPayloadDir, finalPayloadDir)
		}

		uvHashSum := sha256.Sum256(uvBinBytes)
		pythonCodeTgzBytes, metadataTgzBytes, err := preparePayloadArtifacts(log, finalPayloadDir, configJsonBytes, hex.EncodeToString(uvHashSum[:]), buildExcludePatterns)
		if err != nil {
			log.Error("builder", "process", "error", "Failed to prepare payload artifacts.", "error", err)
			os.Exit(1)
		}

		var contentToSign bytes.Buffer
		contentToSign.Write(launcherData)
		contentToSign.Write(uvBinBytes)
		contentToSign.Write(pythonInstallTgzBytes)
		contentToSign.Write(metadataTgzBytes)
		contentToSign.Write(pythonCodeTgzBytes)

		signature, err := signPayload(contentToSign.Bytes(), buildPackageKeyPath)
		if err != nil {
			log.Error("builder", "signing", "error", "Failed to sign package content.", "error", err)
			os.Exit(1)
		}

		pubKey, err := os.ReadFile(buildPublicKeyPath)
		if err != nil {
			log.Error("builder", "read", "error", "Failed to read public key", "path", buildPublicKeyPath, "error", err)
			os.Exit(1)
		}

		err = BuildAppendedPSPF(log, buildOutPath, launcherData, uvBinBytes, pythonInstallTgzBytes, metadataTgzBytes, pythonCodeTgzBytes, signature, pubKey)
		if err != nil {
			log.Error("builder", "assemble", "error", "Failed to assemble final PSPF package.", "error", err)
			os.Exit(1)
		}

		log.Info("builder", "finish", "success", "Provider package built successfully!", "outputPath", buildOutPath)
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)
	buildCmd.Flags().StringVar(&buildLauncherBin, "launcher-bin", "", "Path to the pre-compiled Go launcher binary.")
	buildCmd.Flags().StringVar(&buildPayloadDir, "payload-dir", "", "Directory containing additional non-Python assets.")
	buildCmd.Flags().StringVar(&buildPackageKeyPath, "package-key", "", "Path to the private key for signing.")
	buildCmd.Flags().StringVar(&buildPublicKeyPath, "public-key", "", "Path to the public key to embed in the package.")
	buildCmd.Flags().StringVarP(&buildOutPath, "out", "o", "", "Path for the final output PSPF provider binary.")
	buildCmd.Flags().StringVar(&buildConfigFile, "config", "", "Path to a custom config.json file.")
	buildCmd.Flags().StringVar(&buildUvPath, "uv-path", "", "Optional path to a specific 'uv' binary to embed.")
	buildCmd.Flags().StringVar(&buildPythonInstallDir, "python-install-dir", "", "Path to the Python installation directory to embed.")
	buildCmd.Flags().StringArrayVar(&buildExcludePatterns, "exclude", []string{}, "Glob patterns to exclude from archives.")
	buildCmd.Flags().StringArrayVar(&buildDependencies, "dependency", []string{}, "Python dependency to package (local path or PyPI specifier).")
}

func copyDirContents(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
