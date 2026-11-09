package cli

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/state"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/spf13/cobra"
)

func migrateCmd(ctx *app.AppContext) *cobra.Command {
	var dryRun, execute bool
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Import exported data into GitLab",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRun && execute {
				return fmt.Errorf("--dry-run and --execute are mutually exclusive")
			}
			if !dryRun && !execute {
				dryRun = true
			}

			if err := ctx.EnsureProjectID(); err != nil {
				return fmt.Errorf("resolve GitLab project: %w", err)
			}

			cfg := *ctx.Config
			cfg.DryRun = dryRun

			client, err := gitlab.NewGitLabClient(&cfg)
			if err != nil {
				return err
			}
			if err := client.ValidateGitLab(); err != nil {
				return fmt.Errorf("GitLab validation failed: %w", err)
			}

			statePath := filepath.Join(cfg.ProjectDir, "state", "migration.yaml")
			run, err := state.Load(statePath)
			if err != nil {
				return err
			}
			run.DryRun = dryRun
			run.Status = state.StatusRunning
			if err := run.Save(statePath); err != nil {
				return err
			}

			failed := false
			runPhase := func(name string, fn func() (importer.Result, error)) error {
				run.StartPhase(name)
				if err := run.Save(statePath); err != nil {
					return err
				}
				result, err := fn()
				run.FinishPhase(name, result.Total, result.Completed, result.Failed, result.Errors)
				if err != nil {
					failed = true
					run.Status = state.StatusFailed
					slog.Error("Phase failed", "phase", name, "errorMsg", err)
				} else if result.Failed > 0 {
					failed = true
					run.Status = state.StatusFailed
					slog.Warn("Phase completed with errors", "phase", name, "failed", result.Failed)
				}
				if saveErr := run.Save(statePath); saveErr != nil {
					return saveErr
				}
				return nil
			}

			if dryRun {
				slog.Info("Running in dry-run mode; pass --execute to write to GitLab")
			}

			phases := []struct {
				name    string
				enabled bool
				fn      func() (importer.Result, error)
			}{
				{"milestones", cfg.ImportOptions.ImportMilestones, func() (importer.Result, error) {
					return importer.Result{}, importer.ImportMilestones(client, &cfg)
				}},
				{"issues", cfg.ImportOptions.ImportIssues, func() (importer.Result, error) {
					return importer.ImportIssues(client, &cfg)
				}},
				{"notes", cfg.ImportOptions.ImportNotes, func() (importer.Result, error) {
					return importer.ImportNotes(client, &cfg)
				}},
				{"attachments", cfg.ImportOptions.ImportAttachments, func() (importer.Result, error) {
					return importer.ImportAttachments(client, &cfg)
				}},
				{"wiki", cfg.ImportOptions.ImportWiki, func() (importer.Result, error) {
					return importer.ImportWiki(client, &cfg)
				}},
			}
			for _, phase := range phases {
				if !phase.enabled {
					continue
				}
				if err := runPhase(phase.name, phase.fn); err != nil {
					return err
				}
			}

			run.Finish()
			if err := run.Save(statePath); err != nil {
				return err
			}
			if failed {
				return fmt.Errorf("migration completed with errors; see state/migration.yaml")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing to GitLab (default)")
	cmd.Flags().BoolVar(&execute, "execute", false, "write to GitLab")
	return cmd
}
