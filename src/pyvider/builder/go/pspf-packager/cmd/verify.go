package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"pspf-tools/go/pkg/pspf" // Import the shared package
)

var (
	verifyPublicKeyFile string
)

var verifyCmd = &cobra.Command{
	Use:   "verify <pspf_package_file> --public-key <public_key.crt>",
	Short: "Verifies a PSPF package, including its PSS signature and manifest.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmdCobra *cobra.Command, args []string) {
		filePath := args[0]
		log.Info("verify", "start", "progress", "Verifying PSPF package", "path", filePath)

		file, err := os.Open(filePath)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to open file", "path", filePath, "error", err)
			os.Exit(1)
		}
		defer file.Close()

		footer, err := readAndVerifyFooter(file)
		if err != nil {
			log.Error("verify", "validate", "error", "PSPF Footer parsing/validation failed", "error", err)
			os.Exit(1)
		}
		log.Info("verify", "validate", "success", "PSPF Footer parsed and validated.", "version", fmt.Sprintf("0x%04x", footer.PspfVersion))

		launcherSize := footer.UvBinaryOffset
		launcherBytes, err := readSection(file, 0, launcherSize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read Launcher", "error", err)
			os.Exit(1)
		}
		uvBinBytes, err := readSection(file, footer.UvBinaryOffset, footer.UvBinarySize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read UV Binary", "error", err)
			os.Exit(1)
		}
		pythonTgzBytes, err := readSection(file, footer.PythonInstallTgzOffset, footer.PythonInstallTgzSize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read Python Install Tarball", "error", err)
			os.Exit(1)
		}
		metadataTgzBytes, err := readSection(file, footer.MetadataTgzOffset, footer.MetadataTgzSize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read metadata.tgz", "error", err)
			os.Exit(1)
		}
		payloadTgzBytes, err := readSection(file, footer.PayloadTgzOffset, footer.PayloadTgzSize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read payload.tgz", "error", err)
			os.Exit(1)
		}
		signatureBytes, err := readSection(file, footer.PackageSignatureOffset, footer.PackageSignatureSize)
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read Package Signature", "error", err)
			os.Exit(1)
		}

		log.Info("verify", "signing", "progress", "Verifying Package Integrity PSS Signature...")
		if verifyPublicKeyFile == "" {
			log.Error("verify", "validate", "error", "No public key file provided (--public-key).")
			os.Exit(1)
		}
		pubKey, err := loadPublicKeyFromFile(verifyPublicKeyFile)
		if err != nil {
			log.Error("verify", "load", "error", "Could not load public key", "path", verifyPublicKeyFile, "error", err)
			os.Exit(1)
		}

		var contentToVerify bytes.Buffer
		contentToVerify.Write(launcherBytes)
		contentToVerify.Write(uvBinBytes)
		contentToVerify.Write(pythonTgzBytes)
		contentToVerify.Write(metadataTgzBytes)
		contentToVerify.Write(payloadTgzBytes)

		if err := verifyPayload(contentToVerify.Bytes(), signatureBytes, pubKey); err != nil {
			log.Error("verify", "signing", "failure", "PACKAGE PSS SIGNATURE INVALID", "error", err)
			os.Exit(1)
		}
		log.Info("verify", "signing", "success", "Package PSS Signature is valid.")

		log.Info("verify", "checksum", "progress", "Verifying component checksums from manifests.json...")
		tempExtractDir, err := os.MkdirTemp("", "pspf-verify-extract-")
		if err != nil {
			log.Error("verify", "extract", "error", "Failed to create temp dir", "error", err)
			os.Exit(1)
		}
		defer os.RemoveAll(tempExtractDir)

		if _, err := unTar(bytes.NewReader(metadataTgzBytes), tempExtractDir); err != nil {
			log.Error("verify", "extract", "error", "Failed to extract metadata archive", "error", err)
			os.Exit(1)
		}

		manifestsBytes, err := os.ReadFile(filepath.Join(tempExtractDir, "manifests.json"))
		if err != nil {
			log.Error("verify", "read", "error", "Failed to read manifests.json", "error", err)
			os.Exit(1)
		}

		var manifests Manifests
		if err = json.Unmarshal(manifestsBytes, &manifests); err != nil {
			log.Error("verify", "parse", "error", "Failed to parse manifests.json", "error", err)
			os.Exit(1)
		}

		computedUvBinHash := sha256.Sum256(uvBinBytes)
		if manifests.UvBinarySha256 != hex.EncodeToString(computedUvBinHash[:]) {
			log.Error("verify", "checksum", "failure", "Embedded UV binary CHECKSUM MISMATCH against manifest.", "expected", manifests.UvBinarySha256, "actual", hex.EncodeToString(computedUvBinHash[:]))
			os.Exit(1)
		}
		log.Info("verify", "checksum", "success", "Embedded UV binary checksum matches manifest.")

		log.Info("verify", "finish", "success", "PSPF package verification successful.")
	},
}

var infoCmd = &cobra.Command{
	Use:   "info <pspf_package_file>",
	Short: "Displays information about a PSPF package.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		file, err := os.Open(filePath)
		if err != nil {
			log.Error("info", "read", "error", "Failed to open file", "path", filePath, "error", err)
			os.Exit(1)
		}
		defer file.Close()

		footer, err := readAndVerifyFooter(file)
		if err != nil {
			log.Error("info", "validate", "error", "PSPF Footer parsing/validation failed", "error", err)
			os.Exit(1)
		}

		fmt.Printf("PSPF Package Information for: %s\n", filePath)
		fmt.Printf("  PSPF Version: 0x%04x\n", footer.PspfVersion)
		fmt.Printf("  UV Binary Size: %d bytes\n", footer.UvBinarySize)
		fmt.Printf("  Python Install Size: %d bytes\n", footer.PythonInstallTgzSize)
		fmt.Printf("  Metadata Size: %d bytes\n", footer.MetadataTgzSize)
		fmt.Printf("  Payload Size: %d bytes\n", footer.PayloadTgzSize)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	rootCmd.AddCommand(infoCmd)
	verifyCmd.Flags().StringVar(&verifyPublicKeyFile, "public-key", "", "Path to the public key file for signature verification.")
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
