package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time
var Version = "dev" // Default version
var Commit = "none" // Default commit hash
var Date = "unknown" // Default date

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of pspf-packager",
	Long:  `All software has versions. This is pspf-packager's.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Use the package-level `log` variable directly.
		log.Info("system", "version", "info", "pspf-packager version information", "version", Version, "commit", Commit, "date", Date)
		// Also print to stdout for easy human consumption / scripting
		fmt.Printf("pspf-packager version %s (commit: %s, built: %s)\n", Version, Commit, Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
