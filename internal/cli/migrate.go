package cli

import (
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"

	"github.com/spf13/cobra"
)

func migrateCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Import exported data into GitLab",
		Run: func(cmd *cobra.Command, args []string) {
			slog.Info("Starting migration to GitLab...")

			cfg := *ctx.Config

			client, err := gitlab.NewGitLabClient(&cfg)
			if err != nil {
				slog.Error("Failed to create GitLab client", "errorMsg", err)
				return
			}

			if err = client.ValidateGitLab(); err != nil {
				slog.Error("GitLab validation failed", "errorMsg", err)
				return
			}

			if cfg.ImportOptions.ImportMilestones {
				if err = importer.ImportMilestones(client, &cfg); err != nil {
					slog.Error("Milestone import failed", "errorMsg", err)
					return
				}
			}

			if cfg.ImportOptions.ImportIssues {
				if err = importer.ImportIssues(client, &cfg); err != nil {
					slog.Error("Issue import failed", "errorMsg", err)
					return
				}
			}
		},
	}
}
