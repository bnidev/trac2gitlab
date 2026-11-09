package cli

import (
	"log/slog"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/spf13/cobra"
)

func initCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a default migration.yaml in the project directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.CheckConfigExists(ctx.ProjectDir) {
				slog.Info("Configuration already exists", "projectDir", ctx.ProjectDir)
				return nil
			}
			if err := config.CreateDefaultConfig(ctx.ProjectDir); err != nil {
				return err
			}
			slog.Info("Default configuration created", "path", filepath.Join(ctx.ProjectDir, "migration.yaml"))
			return nil
		},
	}
}
