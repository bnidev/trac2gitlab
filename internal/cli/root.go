package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trac2gitlab",
	Short: "CLI to export Trac data and migrate it to GitLab",
	Long: `trac2gitlab helps you extract tickets, attachments, wiki pages, and history
from Trac and import them into a GitLab instance.`,
}

// Execute runs the root command and handles any errors
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// AddCommand functions can be called here once other commands are defined
func init() {
	rootCmd.AddCommand(versionCmd)
}
