package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/validate"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/spf13/cobra"
)

func validateCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Compare the Trac export with the GitLab target",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := ctx.EnsureProjectID(); err != nil {
				return err
			}
			client, err := gitlab.NewGitLabClient(ctx.Config)
			if err != nil {
				return err
			}
			report, err := validate.Run(client, ctx.Config)
			if err != nil {
				return err
			}

			name := filepath.Base(ctx.Config.ProjectDir)
			path := filepath.Join(ctx.Config.ProjectDir, "reports", "validation.md")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(report.Markdown(name)), 0644); err != nil {
				return err
			}

			fmt.Printf("Validation: %s\n", report.Verdict())
			fmt.Printf("  issues:      %d found / %d expected\n", report.Issues.Found, report.Issues.Expected)
			fmt.Printf("  notes:       %d found / %d expected\n", report.Notes.Found, report.Notes.Expected)
			fmt.Printf("  attachments: %d found / %d expected\n", report.Attachments.Found, report.Attachments.Expected)
			fmt.Printf("  wiki:        %d found / %d expected\n", report.Wiki.Found, report.Wiki.Expected)
			fmt.Printf("  report:      %s\n", path)

			if report.Verdict() == "FAIL" {
				return fmt.Errorf("validation failed (%d errors)", len(report.Errors))
			}
			return nil
		},
	}
}
