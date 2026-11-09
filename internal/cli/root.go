package cli

import (
	"log/slog"
	"os"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "trac2gitlab",
	Short: "CLI to export Trac data and migrate it to GitLab",
	Long:  "trac2gitlab helps you extract tickets, attachments, wiki pages, and history from Trac and import them into a GitLab instance.",
}

// Execute runs the root command and handles any errors
func Execute(ctx *app.AppContext) {
	projectDir := "."
	rootCmd.PersistentFlags().StringVar(&projectDir, "project-dir", ".", "migration project directory")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		ctx.ProjectDir = projectDir
		switch cmd.Name() {
		case "init", "version":
			return nil
		}
		cfg, err := config.LoadProject(projectDir)
		if err != nil {
			return err
		}
		ctx.Config = &cfg
		return nil
	}

	SetupCommands(ctx)

	if err := rootCmd.Execute(); err != nil {
		slog.Error("Command execution failed", "errorMsg", err)
		os.Exit(1)
	}
}

// SetupCommands wires the available commands.
func SetupCommands(ctx *app.AppContext) {
	rootCmd.AddCommand(exportCmd(ctx))
	rootCmd.AddCommand(initCmd(ctx))
	rootCmd.AddCommand(migrateCmd(ctx))
	rootCmd.AddCommand(patchCmd(ctx))
	rootCmd.AddCommand(usersCmd(ctx))
	rootCmd.AddCommand(validateCmd(ctx))
	rootCmd.AddCommand(versionCmd)
}
