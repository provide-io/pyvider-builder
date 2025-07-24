package main

import (
	"archive/tar"
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"pspf-tools/go/pkg/logbowl" // Corrected import path
	"pspf-tools/go/pkg/pspf"    // Import the shared package

	"github.com/spf13/cobra"
	"github.com/valyala/gozstd"
)

const (
	maxSensibleReadSize = 2 * 1024 * 1024 * 1024 // 2 GB
)

type ConfigFromMetadata struct {
	EntryPoint string `json:"entry_point"`
}

var log logbowl.Logger

func main() {
	log = logbowl.Create("pspf-launcher")

	isInteractive := os.Getenv("PSPF_INTERACTIVE")
	if isInteractive == "true" || isInteractive == "1" {
		ExecuteInteractive()
	} else {
		runProvider()
	}
}

func runProvider() {
	exePath, err := os.Executable()
	if err != nil {
		log.Error("launcher", "init", "error", "Could not get executable path", "error", err)
		os.Exit(1)
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Error("env", "init", "error", "Could not determine user cache directory", "error", err)
		os.Exit(1)
	}

	pspfWorkDir := filepath.Join(userCacheDir, "pyvider", "providers", filepath.Base(exePath))
	if err := os.MkdirAll(pspfWorkDir, 0755); err != nil {
		log.Error("env", "init", "error", "Could not create provider cache directory", "path", pspfWorkDir, "error", err)
		os.Exit(1)
	}

	completionFilePath := filepath.Join(pspfWorkDir, ".complete")
	currentExeHash, err := calculateSelfHash(exePath)
	if err != nil {
		log.Error("launcher", "init", "error", "Failed to calculate self hash", "error", err)
		os.Exit(1)
	}

	cachedExeHashBytes, err := os.ReadFile(completionFilePath)
	if err == nil && string(cachedExeHashBytes) == currentExeHash {
		log.Info("env", "verify", "ok", "Cache is valid, reusing existing environment.")
	} else {
		log.Info("env", "init", "progress", "Cache invalid or not found. Starting one-time environment setup.")
		if err := setupEnvironment(exePath, pspfWorkDir); err != nil {
			log.Error("launcher", "init", "error", "Environment setup failed", "error", err)
			os.Exit(1)
		}
		if err := os.WriteFile(completionFilePath, []byte(currentExeHash), 0644); err != nil {
			log.Error("launcher", "finish", "error", "Failed to write completion file", "error", err)
			os.Exit(1)
		}
		log.Info("env", "finish", "ok", "One-time environment setup complete.")
	}

	executePython(pspfWorkDir)
}

func calculateSelfHash(exePath string) (string, error) {
	f, err := os.Open(exePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findExecutable(root, name1, name2 string) (string, error) {
	var foundPath string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (info.Name() == name1 || info.Name() == name2) {
			if info.Mode()&0111 != 0 {
				foundPath = path
				return io.EOF
			}
		}
		return nil
	})
	if err == io.EOF {
		return foundPath, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("executable '%s' or '%s' not found in %s", name1, name2, root)
}

func setupEnvironment(exePath, pspfWorkDir string) error {
	file, err := os.Open(exePath)
	if err != nil {
		return err
	}
	defer file.Close()
	footer, err := readAndVerifyFooter(file)
	if err != nil {
		return err
	}

	log.Debug("launcher", "read", "info", "PSPF Footer data", "footer", footer)

	launcherSize := footer.UvBinaryOffset
	launcherBytes, err := readSection(file, 0, launcherSize)
	if err != nil {
		return err
	}
	uvBinBytes, err := readSection(file, footer.UvBinaryOffset, footer.UvBinarySize)
	if err != nil {
		return err
	}
	pythonTgzBytes, err := readSection(file, footer.PythonInstallTgzOffset, footer.PythonInstallTgzSize)
	if err != nil {
		return err
	}
	metadataTgzBytes, err := readSection(file, footer.MetadataTgzOffset, footer.MetadataTgzSize)
	if err != nil {
		return err
	}
	payloadTgzBytes, err := readSection(file, footer.PayloadTgzOffset, footer.PayloadTgzSize)
	if err != nil {
		return err
	}
	signatureBytes, err := readSection(file, footer.PackageSignatureOffset, footer.PackageSignatureSize)
	if err != nil {
		return err
	}
	publicKeyPEMBytes, err := readSection(file, footer.PublicKeyPEMOffset, footer.PublicKeyPEMSize)
	if err != nil {
		return err
	}

	var contentToVerify bytes.Buffer
	contentToVerify.Write(launcherBytes)
	contentToVerify.Write(uvBinBytes)
	contentToVerify.Write(pythonTgzBytes)
	contentToVerify.Write(metadataTgzBytes)
	contentToVerify.Write(payloadTgzBytes)
	if err := verifySignaturePSS(contentToVerify.Bytes(), signatureBytes, publicKeyPEMBytes); err != nil {
		return err
	}

	os.RemoveAll(pspfWorkDir)
	os.MkdirAll(pspfWorkDir, 0755)

	if _, err := unTar(bytes.NewReader(metadataTgzBytes), filepath.Join(pspfWorkDir, "metadata_extracted")); err != nil {
		return err
	}
	payloadExtractDir := filepath.Join(pspfWorkDir, "payload_extracted")
	allExtractedFiles, err := unTar(bytes.NewReader(payloadTgzBytes), payloadExtractDir)
	if err != nil {
		return err
	}
	pythonInstallDir := filepath.Join(pspfWorkDir, "python")
	if _, err := unTar(bytes.NewReader(pythonTgzBytes), pythonInstallDir); err != nil {
		return err
	}

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	uvExePath := filepath.Join(pspfWorkDir, "uv_embedded"+suffix)
	if err := os.WriteFile(uvExePath, uvBinBytes, 0755); err != nil {
		return err
	}

	pythonExePath, err := findExecutable(pythonInstallDir, "python3", "python")
	if err != nil {
		return fmt.Errorf("could not find python executable in embedded archive: %w", err)
	}

	venvDir := filepath.Join(pspfWorkDir, ".venv")
	cmd := exec.Command(uvExePath, "venv", venvDir, "--python", pythonExePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("venv creation failed: %w\nOutput:\n%s", err, string(out))
	}

	lib64Path := filepath.Join(venvDir, "lib64")
	if _, err := os.Lstat(lib64Path); err == nil {
		if err := os.Remove(lib64Path); err != nil {
			return fmt.Errorf("failed to remove lib64 symlink: %w", err)
		}
	}

	var wheelsToInstall []string
	for _, f := range allExtractedFiles {
		if strings.HasSuffix(f, ".whl") {
			wheelsToInstall = append(wheelsToInstall, filepath.Join(payloadExtractDir, f))
		}
	}
	if len(wheelsToInstall) > 0 {
		installArgs := append([]string{"pip", "install"}, wheelsToInstall...)
		cmd = exec.Command(uvExePath, installArgs...)
		cmd.Env = append(os.Environ(), fmt.Sprintf("VIRTUAL_ENV=%s", venvDir))
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("wheel installation failed: %w\nOutput:\n%s", err, string(out))
		}
	}
	return nil
}

func executePython(pspfWorkDir string) {
	cfgBytes, err := os.ReadFile(filepath.Join(pspfWorkDir, "metadata_extracted", "config.json"))
	if err != nil {
		log.Error("launcher", "execute", "error", "Failed to read config.json", "error", err)
		os.Exit(1)
	}

	var providerConfig ConfigFromMetadata
	json.Unmarshal(cfgBytes, &providerConfig)

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	venvPython := filepath.Join(pspfWorkDir, ".venv", "bin", "python"+suffix)
	if runtime.GOOS == "windows" {
		venvPython = filepath.Join(pspfWorkDir, ".venv", "Scripts", "python.exe")
	}

	pyCmdString := fmt.Sprintf("import sys; import asyncio; import importlib; mod_name, func_name = '%s'.split(':', 1); mod = importlib.import_module(mod_name); sys.exit(asyncio.run(getattr(mod, func_name)()))", providerConfig.EntryPoint)

	pythonCmd := exec.Command(venvPython, "-c", pyCmdString)

	payloadExtractDir := filepath.Join(pspfWorkDir, "payload_extracted")
	existingPath := os.Getenv("PYTHONPATH")
	newPath := payloadExtractDir
	if existingPath != "" {
		newPath = newPath + string(os.PathListSeparator) + existingPath
	}
	pythonCmd.Env = append(os.Environ(), "PYTHONPATH="+newPath)

	pythonCmd.Stdout = os.Stdout
	pythonCmd.Stderr = os.Stderr

	if err := pythonCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	os.Exit(0)
}

func readAndVerifyFooter(f *os.File) (*pspf.Footer, error) {
	fileInfo, err := f.Stat()
	if err != nil {
		return nil, err
	}
	totalSize := fileInfo.Size()
	if totalSize < int64(pspf.FooterSize+len(pspf.MagicEOFString)) {
		return nil, fmt.Errorf("file too small")
	}

	footerBytes := make([]byte, pspf.FooterSize)
	footerOffset := totalSize - int64(pspf.FooterSize) - int64(len(pspf.MagicEOFString))
	_, err = f.ReadAt(footerBytes, footerOffset)
	if err != nil {
		return nil, err
	}

	footer := &pspf.Footer{}
	buf := bytes.NewReader(footerBytes)
	if err := binary.Read(buf, binary.LittleEndian, footer); err != nil {
		return nil, err
	}

	if footer.InternalFooterMagic != pspf.InternalFooterMagic {
		return nil, fmt.Errorf("invalid internal magic")
	}

	valid, err := footer.VerifyChecksum()
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("footer checksum mismatch")
	}
	return footer, nil
}

func readSection(f *os.File, offset, size uint64) ([]byte, error) {
	if size > maxSensibleReadSize {
		return nil, fmt.Errorf("section size %d exceeds limit of %d bytes", size, maxSensibleReadSize)
	}
	if size == 0 {
		return []byte{}, nil
	}
	data := make([]byte, size)
	n, err := f.ReadAt(data, int64(offset))
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("ReadAt failed: unexpected EOF while reading %d bytes at offset %d (read %d bytes)", size, offset, n)
		}
		return nil, fmt.Errorf("ReadAt failed: %w", err)
	}
	if uint64(n) != size {
		return nil, fmt.Errorf("ReadAt read %d bytes, expected %d", n, size)
	}
	return data, nil
}

func verifySignaturePSS(data []byte, signature []byte, publicKeyPEMBytes []byte) error {
	block, _ := pem.Decode(publicKeyPEMBytes)
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not an RSA public key")
	}

	hashed := sha256.Sum256(data)
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: crypto.SHA256}
	return rsa.VerifyPSS(rsaPub, crypto.SHA256, hashed[:], signature, opts)
}

func unTar(r io.Reader, dest string) ([]string, error) {
	zr := gozstd.NewReader(r)
	defer zr.Release()
	tr := tar.NewReader(zr)
	var files []string
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)) {
			return nil, fmt.Errorf("zip slip detected in tar archive")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			files = append(files, header.Name)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return nil, err
			}
			f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				if lstatInfo, lstatErr := os.Lstat(filepath.Dir(target)); lstatErr == nil && lstatInfo.Mode()&os.ModeSymlink != 0 {
				} else {
					return nil, err
				}
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return nil, err
			}
		}
	}
	return files, nil
}

var rootCmd = &cobra.Command{Use: "pspf-launcher", Short: "Pyvider Interactive Launcher"}

func ExecuteInteractive() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func init() {
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(runCmd)
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display information about the PSPF package",
	Run: func(cmd *cobra.Command, args []string) {
		exePath, _ := os.Executable()
		fmt.Println("PSPF Package Information for:", exePath)
		file, err := os.Open(exePath)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()
		footer, err := readAndVerifyFooter(file)
		if err != nil {
			fmt.Println("Error reading footer:", err)
			return
		}
		fmt.Printf("  PSPF Version: 0x%04x\n", footer.PspfVersion)
		fmt.Printf("  UV Binary Size: %d bytes\n", footer.UvBinarySize)
		fmt.Printf("  Python Install Size: %d bytes\n", footer.PythonInstallTgzSize)
		fmt.Printf("  Metadata Size: %d bytes\n", footer.MetadataTgzSize)
		fmt.Printf("  Payload Size: %d bytes\n", footer.PayloadTgzSize)
	},
}
var runCmd = &cobra.Command{
	Use: "run",
	Short: "Run the provider (default behavior)",
	Run: func(cmd *cobra.Command, args []string) {
		runProvider()
	},
}
