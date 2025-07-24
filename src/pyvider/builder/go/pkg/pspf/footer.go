package pspf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

const (
	// Canonical PSPF v0.1 Specification Constants
	Version             uint16 = 0x0003
	FooterSize          int    = 108
	MagicEOFString      string = "!PSPF\x00\x00\x00"
	InternalFooterMagic uint32 = 0x30505350 // '0PSP'
)

// Footer is the canonical struct for the 108-byte PSPF v0.1 footer.
// This is the single source of truth for all Go components.
type Footer struct {
	UvBinaryOffset, UvBinarySize, PythonInstallTgzOffset, PythonInstallTgzSize, MetadataTgzOffset, MetadataTgzSize, PayloadTgzOffset, PayloadTgzSize, PackageSignatureOffset, PackageSignatureSize, PublicKeyPEMOffset, PublicKeyPEMSize uint64
	PspfVersion, Reserved                                                                                                                                                                                          uint16
	FooterStructChecksum, InternalFooterMagic                                                                                                                                                                      uint32
}

// CalculateChecksum computes and sets the checksum for the footer.
func (f *Footer) CalculateChecksum() error {
	tempFooter := *f
	tempFooter.FooterStructChecksum = 0
	checksumBuffer := new(bytes.Buffer)
	if err := binary.Write(checksumBuffer, binary.LittleEndian, tempFooter); err != nil {
		return fmt.Errorf("failed to write temp footer to buffer for checksumming: %w", err)
	}
	f.FooterStructChecksum = crc32.ChecksumIEEE(checksumBuffer.Bytes())
	return nil
}

// VerifyChecksum checks if the stored checksum is valid.
func (f *Footer) VerifyChecksum() (bool, error) {
	storedChecksum := f.FooterStructChecksum
	tempFooter := *f
	tempFooter.FooterStructChecksum = 0
	checksumBuffer := new(bytes.Buffer)
	if err := binary.Write(checksumBuffer, binary.LittleEndian, tempFooter); err != nil {
		return false, fmt.Errorf("failed to write temp footer to buffer for checksumming: %w", err)
	}
	calculatedChecksum := crc32.ChecksumIEEE(checksumBuffer.Bytes())
	return storedChecksum == calculatedChecksum, nil
}
