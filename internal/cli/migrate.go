package cli

import (
	"log/slog"
	"trac2gitlab/internal/config"
	"trac2gitlab/internal/importer"
	"trac2gitlab/pkg/gitlab"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import exported data into GitLab",
	Run: func(cmd *cobra.Command, args []string) {
		slog.Info("Starting migration to GitLab...")
		cfg := config.LoadConfig()
		client, err := gitlab.NewGitLabClient(cfg.GitLab.BaseURL, cfg.GitLab.APIPath, cfg.GitLab.Token)
		if err != nil {
			slog.Error("Failed to create GitLab client", "errorMsg", err)
			return
		}

		if err = client.ValidateGitLab(); err != nil {
			slog.Error("GitLab validation failed", "errorMsg", err)
			return
		}

		if cfg.ImportOptions.ImportMilestones {
			if err = importer.ImportMilestones(client, cfg.GitLab.ProjectID); err != nil {
				slog.Error("Milestone import failed", "errorMsg", err)
				return
			}
		}

		if cfg.ImportOptions.ImportIssues {
			if err = importer.ImportIssues(client, cfg.GitLab.ProjectID); err != nil {
				slog.Error("Issue import failed", "errorMsg", err)
				return
			}
		}

	},
}
