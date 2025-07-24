package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"pspf-tools/go/pkg/logbowl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAndVerifyPSPF(t *testing.T) {
	tmpDir := t.TempDir()
	log := logbowl.Create("test-pspf")

	// 1. Generate keys
	privKeyPEM, pubKeyPEM, err := generateKeyPairPEM()
	require.NoError(t, err)

	// 2. Create dummy content
	launcherBytes := []byte("I am a launcher")
	uvBinBytes := []byte("I am uv")
	pythonTgzBytes := []byte("I am python tgz")
	metadataTgzBytes := []byte("I am metadata tgz")
	payloadTgzBytes := []byte("I am payload tgz")

	// 3. Sign the content
	privKeyPath := filepath.Join(tmpDir, "test.key")
	require.NoError(t, os.WriteFile(privKeyPath, privKeyPEM, 0600))
	
	var contentToSign bytes.Buffer
	contentToSign.Write(launcherBytes)
	contentToSign.Write(uvBinBytes)
	contentToSign.Write(pythonTgzBytes)
	contentToSign.Write(metadataTgzBytes)
	contentToSign.Write(payloadTgzBytes)
	signatureBytes, err := signPayload(contentToSign.Bytes(), privKeyPath)
	require.NoError(t, err)

	// 4. Build the PSPF file
	outPath := filepath.Join(tmpDir, "test-provider")
	err = BuildAppendedPSPF(log, outPath, launcherBytes, uvBinBytes, pythonTgzBytes, metadataTgzBytes, payloadTgzBytes, signatureBytes, pubKeyPEM)
	require.NoError(t, err)
	assert.FileExists(t, outPath)

	// 5. Read it back and verify the footer
	file, err := os.Open(outPath)
	require.NoError(t, err)
	defer file.Close()

	footer, err := readAndVerifyFooter(file)
	require.NoError(t, err, "readAndVerifyFooter should succeed on the built file")

	// 6. Assert footer contents are correct
	assert.Equal(t, uint64(len(launcherBytes)), footer.UvBinaryOffset)
	assert.Equal(t, uint64(len(uvBinBytes)), footer.UvBinarySize)
	assert.Equal(t, uint64(len(launcherBytes)+len(uvBinBytes)), footer.PythonInstallTgzOffset)
	assert.Equal(t, uint64(len(pythonTgzBytes)), footer.PythonInstallTgzSize)
	assert.Equal(t, uint64(len(payloadTgzBytes)), footer.PayloadTgzSize)
	assert.Equal(t, uint64(len(signatureBytes)), footer.PackageSignatureSize)
	assert.Equal(t, uint64(len(pubKeyPEM)), footer.PublicKeyPEMSize)
	assert.Equal(t, PspfVersion, footer.PspfVersion)
	assert.Equal(t, PspfInternalFooterMagic, footer.InternalFooterMagic)
}
