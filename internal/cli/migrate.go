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

			runPhase := func(name string, fn func() (importer.Result, error)) error {
				run.StartPhase(name)
				if err := run.Save(statePath); err != nil {
					return err
				}
				result, err := fn()
				run.FinishPhase(name, result.Total, result.Completed, result.Failed, result.Errors)
				if err != nil {
					run.Status = state.StatusFailed
					run.Finish()
				}
				if saveErr := run.Save(statePath); saveErr != nil {
					return saveErr
				}
				return err
			}

			if dryRun {
				slog.Info("Running in dry-run mode; pass --execute to write to GitLab")
			}

			if cfg.ImportOptions.ImportMilestones {
				if err := runPhase("milestones", func() (importer.Result, error) {
					return importer.Result{}, importer.ImportMilestones(client, &cfg)
				}); err != nil {
					return fmt.Errorf("milestone import failed: %w", err)
				}
			}

			if cfg.ImportOptions.ImportIssues {
				if err := runPhase("issues", func() (importer.Result, error) {
					return importer.ImportIssues(client, &cfg)
				}); err != nil {
					return fmt.Errorf("issue import failed: %w", err)
				}
			}

			if cfg.ImportOptions.ImportNotes {
				if err := runPhase("notes", func() (importer.Result, error) {
					return importer.ImportNotes(client, &cfg)
				}); err != nil {
					return fmt.Errorf("note import failed: %w", err)
				}
			}

			if cfg.ImportOptions.ImportAttachments {
				if err := runPhase("attachments", func() (importer.Result, error) {
					return importer.ImportAttachments(client, &cfg)
				}); err != nil {
					return fmt.Errorf("attachment import failed: %w", err)
				}
			}

			if cfg.ImportOptions.ImportWiki {
				if err := runPhase("wiki", func() (importer.Result, error) {
					return importer.ImportWiki(client, &cfg)
				}); err != nil {
					return fmt.Errorf("wiki import failed: %w", err)
				}
			}

			run.Finish()
			if err := run.Save(statePath); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview changes without writing to GitLab (default)")
	cmd.Flags().BoolVar(&execute, "execute", false, "write to GitLab")
	return cmd
}
