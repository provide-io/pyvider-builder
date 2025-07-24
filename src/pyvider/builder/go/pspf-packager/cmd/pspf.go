package cmd

import (
	"bytes"
	"encoding/binary"
	"os"

	"pspf-tools/go/pkg/logbowl"
	"pspf-tools/go/pkg/pspf" // Import the shared package
)

func BuildAppendedPSPF(log logbowl.Logger, outPath string, launcherBytes, uvBinBytes, pythonTgzBytes, metadataTgzBytes, payloadTgzBytes, signatureBytes, publicKeyPEMBytes []byte) error {
	log.Debug("builder", "assemble", "progress", "Assembling final PSPF package...")
	outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var currentOffset int64 = 0
	var n int

	n, err = outFile.Write(launcherBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	uvBinaryOffset := currentOffset
	n, err = outFile.Write(uvBinBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	pythonInstallTgzOffset := currentOffset
	n, err = outFile.Write(pythonTgzBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	metadataTgzOffset := currentOffset
	n, err = outFile.Write(metadataTgzBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	payloadTgzOffset := currentOffset
	n, err = outFile.Write(payloadTgzBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	packageSignatureOffset := currentOffset
	n, err = outFile.Write(signatureBytes)
	if err != nil {
		return err
	}
	currentOffset += int64(n)
	publicKeyPEMOffset := currentOffset
	n, err = outFile.Write(publicKeyPEMBytes)
	if err != nil {
		return err
	}

	footer := pspf.Footer{
		UvBinaryOffset:         uint64(uvBinaryOffset),
		UvBinarySize:           uint64(len(uvBinBytes)),
		PythonInstallTgzOffset: uint64(pythonInstallTgzOffset),
		PythonInstallTgzSize:   uint64(len(pythonTgzBytes)),
		MetadataTgzOffset:      uint64(metadataTgzOffset),
		MetadataTgzSize:        uint64(len(metadataTgzBytes)),
		PayloadTgzOffset:       uint64(payloadTgzOffset),
		PayloadTgzSize:         uint64(len(payloadTgzBytes)),
		PackageSignatureOffset: uint64(packageSignatureOffset),
		PackageSignatureSize:   uint64(len(signatureBytes)),
		PublicKeyPEMOffset:     uint64(publicKeyPEMOffset),
		PublicKeyPEMSize:       uint64(len(publicKeyPEMBytes)),
		PspfVersion:            pspf.Version,
		Reserved:               0,
		InternalFooterMagic:    pspf.InternalFooterMagic,
	}

	if err := footer.CalculateChecksum(); err != nil {
		return err
	}

	finalFooterBytesBuffer := new(bytes.Buffer)
	if err := binary.Write(finalFooterBytesBuffer, binary.LittleEndian, footer); err != nil {
		return err
	}
	outFile.Write(finalFooterBytesBuffer.Bytes())
	outFile.WriteString(pspf.MagicEOFString)

	log.Info("builder", "assemble", "success", "Final PSPF binary assembled successfully", "path", outPath)
	return nil
}
