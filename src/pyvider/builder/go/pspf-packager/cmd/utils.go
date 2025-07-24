package cmd

import (
	"archive/tar"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pspf-tools/go/pkg/logbowl"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/valyala/gozstd"
)

const maxSensibleReadSize = 2 * 1024 * 1024 * 1024 // 2 GB

func generateKeyPairPEM() (privKeyPEM, pubKeyPEM []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil { log.Error("keymgmt", "generate", "error", "Failed to generate RSA private key", "error", err); return nil, nil, err }
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privKeyBytes})
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil { log.Error("keymgmt", "generate", "error", "Failed to marshal public key", "error", err); return nil, nil, err }
	pubKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyBytes})
	return
}
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil { return nil, err }
	block, _ := pem.Decode(keyData)
	if block == nil { return nil, fmt.Errorf("failed to decode PEM block from %s", path) }
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok { return nil, fmt.Errorf("key in %s is not an RSA private key", path) }
		return rsaKey, nil
	}
	pkcs1Key, errPkcs1 := x509.ParsePKCS1PrivateKey(block.Bytes)
	if errPkcs1 != nil { return nil, fmt.Errorf("failed to parse private key from %s", path) }
	return pkcs1Key, nil
}
func loadPublicKeyFromFile(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil { return nil, fmt.Errorf("failed to read public key file %s: %w", path, err) }
	block, _ := pem.Decode(keyData)
	if block == nil { return nil, fmt.Errorf("failed to decode PEM block from %s", path) }
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil { return nil, fmt.Errorf("failed to parse public key from %s: %w", path, err) }
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok { return nil, fmt.Errorf("key from %s is not an RSA public key", path) }
	return rsaPub, nil
}
func signPayload(payload []byte, keyPath string) ([]byte, error) {
	privateKey, err := loadPrivateKey(keyPath)
	if err != nil { log.Error("signing", "load", "error", "Failed to load private key", "path", keyPath, "error", err); return nil, err }
	hashed := sha256.Sum256(payload)
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: crypto.SHA256}
	return rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], opts)
}
func verifyPayload(data, signature []byte, pubKey *rsa.PublicKey) error {
	hashed := sha256.Sum256(data)
	opts := &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthAuto, Hash: crypto.SHA256}
	return rsa.VerifyPSS(pubKey, crypto.SHA256, hashed[:], signature, opts)
}

type ManifestFileEntry struct {
	PathInArchive    string `json:"path_in_archive"`
	Sha256           string `json:"sha256"`
	ArchiveContainer string `json:"archive_container"`
}
type Manifests struct {
	UvBinarySha256 string `json:"uv_binary_sha256"`
	Files []ManifestFileEntry `json:"files"`
}

func createSourceArchive(log logbowl.Logger, sourceDir string, excludePatterns []string) ([]byte, error) {
    var buf bytes.Buffer
    zw := gozstd.NewWriter(&buf)
    tw := tar.NewWriter(zw)

    err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil { return err }

        relPath, err := filepath.Rel(sourceDir, path)
        if err != nil { return err }
        if relPath == "." { return nil }
        
        for _, pattern := range excludePatterns {
            match, err := doublestar.Match(pattern, relPath)
            if err != nil { return err }
            if match {
                log.Debug("archive", "exclude", "skip", "Excluding path based on pattern", "path", relPath, "pattern", pattern)
                if info.IsDir() { return filepath.SkipDir }
                return nil
            }
        }

        realInfo, err := os.Stat(path)
        if err != nil { return nil }

        hdr, err := tar.FileInfoHeader(realInfo, "")
        if err != nil { return err }
        hdr.Name = filepath.ToSlash(relPath)

        if err := tw.WriteHeader(hdr); err != nil { return err }

        if realInfo.Mode().IsRegular() {
            file, err := os.Open(path)
            if err != nil { return err }
            defer file.Close()
            if _, err := io.Copy(tw, file); err != nil { return err }
        }
        return nil
    })

    if err != nil { return nil, err }
    if err := tw.Close(); err != nil { return nil, err }
    if err := zw.Close(); err != nil { return nil, err }
    return buf.Bytes(), nil
}

func preparePayloadArtifacts(log logbowl.Logger, payloadDir string, configJsonBytes []byte, uvBinHashHex string, excludePatterns []string) (pythonCodeTgzBytes, metadataTgzBytes []byte, err error) {
	pythonCodeTgzBytes, err = createSourceArchive(log, payloadDir, excludePatterns)
	if err != nil { return nil, nil, err }
	
	metadataAssemblyDir, err := os.MkdirTemp("", "pspf-metadata-assembly-")
	if err != nil { return nil, nil, err }
	defer os.RemoveAll(metadataAssemblyDir)

	var metadataManifestEntries []ManifestFileEntry
	if len(configJsonBytes) > 0 {
		configPath := filepath.Join(metadataAssemblyDir, "config.json")
		if err = os.WriteFile(configPath, configJsonBytes, 0644); err != nil { return nil, nil, err }
		hash := sha256.Sum256(configJsonBytes)
		metadataManifestEntries = append(metadataManifestEntries, ManifestFileEntry{
			PathInArchive: "config.json", Sha256: hex.EncodeToString(hash[:]), ArchiveContainer: "metadata.tgz",
		})
	}

	manifestData := Manifests{ UvBinarySha256: uvBinHashHex, Files: metadataManifestEntries }
	manifestJsonBytes, err := json.MarshalIndent(manifestData, "", "  ")
	if err != nil { return nil, nil, err }
	manifestPath := filepath.Join(metadataAssemblyDir, "manifests.json")
	if err = os.WriteFile(manifestPath, manifestJsonBytes, 0644); err != nil { return nil, nil, err }

	metadataTgzBytes, err = createSourceArchive(log, metadataAssemblyDir, excludePatterns)
	if err != nil { return nil, nil, err }

	return pythonCodeTgzBytes, metadataTgzBytes, nil
}

func unTar(r io.Reader, dest string) ([]string, error) {
	zr := gozstd.NewReader(r); defer zr.Release(); tr := tar.NewReader(zr)
	var files []string
	for {
		header, err := tr.Next()
		if err == io.EOF { break }; if err != nil { return nil, err }
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)) { return nil, fmt.Errorf("zip slip detected in tar archive") }
		
		switch header.Typeflag {
		case tar.TypeDir: if err := os.MkdirAll(target, 0755); err != nil { return nil, err }
		case tar.TypeReg:
			files = append(files, header.Name)
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil { return nil, err }
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil { return nil, err }; if _, err := io.Copy(f, tr); err != nil { f.Close(); return nil, err }; f.Close()
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				if lstatInfo, lstatErr := os.Lstat(filepath.Dir(target)); lstatErr == nil && lstatInfo.Mode()&os.ModeSymlink != 0 {} else { return nil, err }
			}
			if err := os.Symlink(header.Linkname, target); err != nil { return nil, err }
		}
	}
	return files, nil
}

func readSection(f *os.File, offset, size uint64) ([]byte, error) {
	if size > maxSensibleReadSize { return nil, fmt.Errorf("section size %d exceeds limit of %d bytes", size, maxSensibleReadSize) }
	if size == 0 { return []byte{}, nil }
	data := make([]byte, size)
	n, err := f.ReadAt(data, int64(offset))
	if err != nil {
		if err == io.EOF { return nil, fmt.Errorf("ReadAt failed: unexpected EOF while reading %d bytes at offset %d (read %d bytes)", size, offset, n) }
		return nil, fmt.Errorf("ReadAt failed: %w", err)
	}
	if uint64(n) != size { return nil, fmt.Errorf("ReadAt read %d bytes, expected %d", n, size) }
	return data, nil
}
