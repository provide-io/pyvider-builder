package cmd

import (
	"os"
	"path/filepath"
	"github.com/spf13/cobra"
)

var (
	keygenOutDir       string
	privateKeyFileName string
	publicKeyFileName  string
)

var keygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Generates a new PSS RSA key pair for package integrity signing.",
	Run: func(cmdCobra *cobra.Command, args []string) {
		privOutPath := filepath.Join(keygenOutDir, privateKeyFileName)
		pubOutPath := filepath.Join(keygenOutDir, publicKeyFileName)

		if info, err := os.Stat(privOutPath); err == nil {
			log.Warn("keymgmt", "generate", "skip", "Private key already exists, skipping generation.", "path", privOutPath, "created", info.ModTime().Format("2006-01-02 15:04:05"))
			os.Exit(1)
		}
		if info, err := os.Stat(pubOutPath); err == nil {
			log.Warn("keymgmt", "generate", "skip", "Public key already exists, skipping generation.", "path", pubOutPath, "created", info.ModTime().Format("2006-01-02 15:04:05"))
			os.Exit(1)
		}

		log.Info("keymgmt", "generate", "progress", "Generating new 4096-bit RSA key pair...")
		privKeyBytes, pubKeyBytes, err := generateKeyPairPEM()
		if err != nil { os.Exit(1) }

		if err := os.WriteFile(privOutPath, privKeyBytes, 0600); err != nil {
			log.Error("keymgmt", "write", "error", "Failed to write private key", "path", privOutPath, "error", err)
			os.Exit(1)
		}
		log.Info("keymgmt", "write", "success", "Private key saved", "path", privOutPath)

		if err := os.WriteFile(pubOutPath, pubKeyBytes, 0644); err != nil {
			log.Error("keymgmt", "write", "error", "Failed to write public key", "path", pubOutPath, "error", err)
			os.Exit(1)
		}
		log.Info("keymgmt", "write", "success", "Public key saved", "path", pubOutPath)
	},
}

func init() {
	rootCmd.AddCommand(keygenCmd)
	keygenCmd.Flags().StringVarP(&keygenOutDir, "out-dir", "d", ".", "Directory to save the key pair.")
	keygenCmd.Flags().StringVar(&privateKeyFileName, "private-key-file", "provider-private.key", "Filename for the private key.")
	keygenCmd.Flags().StringVar(&publicKeyFileName, "public-key-file", "provider-public.key", "Filename for the public key.")
}
