package cli

import (
	"fmt"
	"trac2gitlab/internal/config"
	"trac2gitlab/internal/importer"
	"trac2gitlab/pkg/gitlab"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Import exported data into GitLab",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running migration to GitLab...")
		cfg := config.LoadConfig()
		client, err := gitlab.NewGitLabClient(cfg.GitLab.BaseURL, cfg.GitLab.APIPath, cfg.GitLab.Token)
		if err != nil {
			fmt.Println("❌ Failed to create GitLab client:", err)
		}

		if err = client.ValidateGitLab(); err != nil {
			fmt.Println("❌ GitLab validation failed:", err)
			return
		}

		if cfg.ImportOptions.ImportMilestones {
			if err = importer.ImportMilestones(client, cfg.GitLab.ProjectID); err != nil {
				fmt.Println("❌ Import failed:", err)
				return
			}
		}

		if cfg.ImportOptions.ImportIssues {
			if err = importer.ImportIssues(client, cfg.GitLab.ProjectID); err != nil {
				fmt.Println("❌ Import failed:", err)
				return
			}
		}

	},
}
