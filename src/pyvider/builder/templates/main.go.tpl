package main

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

// This public key will be embedded at compile time by the pyvider-builder tool.
var embeddedPublicKey = `{{ .PublicKey }}`

// PSFP v3 Footer Structure
const (
	footerSize        = 56 // 4*8 for offsets/sizes + 32 for hash + 8 for magic
	magicNumberV3     = 0x5053465056320A21 // "!PSFPV2\n!"
)

type psfpFooter struct {
	PayloadOffset   uint64
	PayloadSize     uint64
	SignatureOffset uint64
	SignatureSize   uint64
	PayloadHash     [32]byte
	MagicNumber     uint64
}

func main() {
	log.SetFlags(0)

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("FATAL: Could not get executable path: %v", err)
	}

	footer, err := readFooter(exePath)
	if err != nil {
		log.Fatalf("FATAL: Invalid package structure: %v", err)
	}

	// --- Concurrent Verification and Environment Setup ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 2)

	// Task 1: Verify the package
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("INFO: Starting package verification...")
		err := verifyPackage(exePath, footer)
		if err != nil {
			errs <- fmt.Errorf("package verification failed: %w", err)
			cancel() // Cancel other tasks on failure
		}
		log.Println("INFO: Package verification successful.")
	}()

	// Task 2: Setup Python environment
	cacheDir := getCacheDir(footer.PayloadHash)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Println("INFO: Setting up Python environment...")
		err := setupPythonEnv(ctx, exePath, footer, cacheDir)
		if err != nil {
			errs <- fmt.Errorf("Python environment setup failed: %w", err)
			cancel()
		}
		log.Println("INFO: Python environment is ready.")
	}()

	// Wait for setup to complete, but verification can still be running
	wg.Wait()
	select {
	case err := <-errs:
		log.Fatalf("FATAL: Startup failed during verification or setup: %v", err)
	default:
		// Both setup and verification succeeded up to this point
	}

	// --- Pre-warm Python and perform secure handshake ---
	listener, secret, err := startHandshakeListener()
	if err != nil {
		log.Fatalf("FATAL: Could not start handshake listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	cmd := exec.Command(filepath.Join(cacheDir, ".venv", "bin", "python"), "-m", "pyvider")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PYVIDER_HANDSHAKE_PORT=%d", port),
		fmt.Sprintf("PYVIDER_HANDSHAKE_SECRET=%s", secret),
	)
	cmd.Stderr = os.Stderr // Redirect Python's stderr for visibility
	
	if err := cmd.Start(); err != nil {
		log.Fatalf("FATAL: Failed to start Python process: %v", err)
	}

	// Wait for Python to connect and authenticate
	conn, err := acceptAndVerifyHandshake(listener, secret)
	if err != nil {
		cmd.Process.Kill()
		log.Fatalf("FATAL: Handshake with Python process failed: %v", err)
	}
	defer conn.Close()
	
	log.Println("INFO: Python process pre-warmed and authenticated. Waiting for final verification.")

	// Wait for any remaining verification tasks
	select {
	case err := <-errs:
		log.Printf("ERROR: Verification failed after Python start: %v", err)
		conn.Write([]byte{0x00}) // Signal termination
		cmd.Process.Kill()
		os.Exit(1)
	case <-ctx.Done():
		if ctx.Err() != context.Canceled {
			log.Printf("ERROR: Context done with error: %v", ctx.Err())
			conn.Write([]byte{0x00})
			cmd.Process.Kill()
			os.Exit(1)
		}
	default:
		// All good
	}

	log.Println("INFO: All checks passed. Sending all-clear signal to Python.")
	_, err = conn.Write([]byte{0x01}) // All-clear signal
	if err != nil {
		log.Fatalf("FATAL: Failed to send all-clear signal to Python: %v", err)
	}
	conn.Close() // Close our end

	// The Python process now takes over and will print the gRPC handshake.
	// We wait for the Python process to exit.
	if err := cmd.Wait(); err != nil {
		log.Printf("WARN: Python process exited with error: %v", err)
	}
}

// ... (Other functions like readFooter, verifyPackage, setupPythonEnv, etc. would be defined here)
