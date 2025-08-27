package cli

import (
	"log/slog"
	"os"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trac2gitlab",
	Short: "CLI to export Trac data and migrate it to GitLab",
	Long:  "trac2gitlab helps you extract tickets, attachments, wiki pages, and history from Trac and import them into a GitLab instance.",
}

// Execute runs the root command and handles any errors
func Execute(ctx *app.AppContext) {
	SetupCommands(ctx)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Command execution failed", "errorMsg", err)
		os.Exit(1)
	}
}

// AddCommand functions can be called here once other commands are defined
func SetupCommands(ctx *app.AppContext) {
	rootCmd.AddCommand(exportCmd(ctx))
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(migrateCmd(ctx))
	rootCmd.AddCommand(versionCmd)
}
