package cmd

import (
	"os"
	"pspf-tools/go/pkg/logbowl"
	"github.com/spf13/cobra"
)

var (
	log logbowl.Logger
)

var rootCmd = &cobra.Command{
	Use:   "pspf-packager",
	Short: "A CLI tool for building and managing Progressive Secure Package Format (PSPF) files.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log = logbowl.Create("pspf-packager")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if log.Logger != nil { // Check if logger was initialized
			log.Error("system", "stop", "error", "Failed to execute command", "error", err)
		}
		os.Exit(1)
	}
}
