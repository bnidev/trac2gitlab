package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/preflight"
	"github.com/spf13/cobra"
)

func preflightCmd(ctx *app.AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "preflight",
		Short: "Read-only migration readiness check",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := preflight.Run(ctx.Config)

			path := filepath.Join(ctx.Config.ProjectDir, "reports", "preflight.md")
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(path, []byte(report.Markdown(filepath.Base(ctx.Config.ProjectDir))), 0644); err != nil {
				return err
			}

			fmt.Printf("Preflight: %s\n", report.Verdict)
			fmt.Printf("  Trac XML-RPC:     %s\n", ok(report.TracOK))
			fmt.Printf("  GitLab project:   %s\n", ok(report.GitLabOK))
			fmt.Printf("  Role:             %s\n", report.Role)
			fmt.Printf("  Timestamps:       %s\n", ok(report.TimestampOK))
			fmt.Printf("  Target issues:    %d\n", report.TargetIssues)
			fmt.Printf("  Unresolved users: %d\n", report.Unresolved)
			fmt.Printf("  report:           %s\n", path)

			if report.Verdict == "BLOCKED" || report.Verdict == "NOT READY" {
				return fmt.Errorf("preflight %s; see reports/preflight.md", report.Verdict)
			}
			return nil
		},
	}
}

func ok(v bool) string {
	if v {
		return "✓"
	}
	return "✗"
}
