// Package pipeline runs the export and migrate orchestrations shared by the
// CLI and the web wizard, so both front-ends execute the same steps.
package pipeline

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/exporter"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/state"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// Progress receives one human-readable line per step. It may be nil.
type Progress func(line string)

func (p Progress) emit(format string, args ...any) {
	if p != nil {
		p(fmt.Sprintf(format, args...))
	}
}

// ErrPhaseFailures reports that migrate finished but at least one phase
// failed or had failed objects. Details are in state/migration.yaml.
var ErrPhaseFailures = errors.New("migration completed with errors; see state/migration.yaml")

// Export runs the read-only Trac export into the configured export dir.
// Any phase failure stops the export and is returned.
func Export(cfg *config.Config, progress Progress) error {
	client, err := trac.NewTracClient(cfg)
	if err != nil {
		return fmt.Errorf("create Trac client: %w", err)
	}
	if err := client.ValidateExpectedMethods(); err != nil {
		return fmt.Errorf("trac client validation: %w", err)
	}
	if err := client.ValidatePluginVersion(); err != nil {
		return fmt.Errorf("trac plugin version validation: %w", err)
	}

	start := time.Now()
	steps := []struct {
		name    string
		enabled bool
		fn      func() error
	}{
		{"ticket fields", cfg.ExportOptions.IncludeTicketFields, func() error { return exporter.ExportTicketFields(client, cfg) }},
		{"tickets", true, func() error { return exporter.ExportTickets(client, cfg) }},
		{"milestones", true, func() error { return exporter.ExportMilestones(client, cfg) }},
		{"wiki", cfg.ExportOptions.IncludeWiki, func() error { return exporter.ExportWiki(client, cfg) }},
		{"users", cfg.ExportOptions.IncludeUsers, func() error { return exporter.ExportUsers(client, cfg) }},
	}
	for _, s := range steps {
		if !s.enabled {
			continue
		}
		progress.emit("exporting %s", s.name)
		if err := s.fn(); err != nil {
			return fmt.Errorf("export %s: %w", s.name, err)
		}
		progress.emit("exported %s", s.name)
	}
	slog.Info("Export completed successfully", "duration", time.Since(start))
	return nil
}

// ResolveProjectID resolves gitlab.project to a numeric ID and path when
// they are not already known.
func ResolveProjectID(cfg *config.Config) error {
	if cfg.GitLab.ProjectID > 0 && cfg.GitLab.ProjectPath != "" {
		return nil
	}
	if cfg.GitLab.Project == "" && cfg.GitLab.ProjectID <= 0 {
		return fmt.Errorf("gitlab.project_id or gitlab.project is required")
	}
	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		return err
	}
	ref := any(cfg.GitLab.Project)
	if cfg.GitLab.Project == "" {
		ref = cfg.GitLab.ProjectID
	}
	project, err := client.GetProject(ref)
	if err != nil {
		return fmt.Errorf("resolve gitlab project %q: %w", cfg.GitLab.Project, err)
	}
	cfg.GitLab.ProjectID = int(project.ID)
	cfg.GitLab.ProjectPath = project.PathWithNamespace
	return nil
}

// Migrate runs the enabled import phases and records each one in
// state/migration.yaml. cfg.DryRun selects preview vs writing to GitLab.
// It returns the final state; ErrPhaseFailures means the run finished with
// failed phases or objects.
func Migrate(cfg *config.Config, progress Progress) (*state.Migration, error) {
	if err := ResolveProjectID(cfg); err != nil {
		return nil, fmt.Errorf("resolve GitLab project: %w", err)
	}
	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.ValidateGitLab(); err != nil {
		return nil, fmt.Errorf("GitLab validation failed: %w", err)
	}

	statePath := filepath.Join(cfg.ProjectDir, "state", "migration.yaml")
	run, err := state.Load(statePath)
	if err != nil {
		return nil, err
	}
	run.DryRun = cfg.DryRun
	run.Status = state.StatusRunning
	if err := run.Save(statePath); err != nil {
		return nil, err
	}
	if cfg.DryRun {
		slog.Info("Running in dry-run mode; pass --execute to write to GitLab")
	}

	phases := []struct {
		name    string
		enabled bool
		fn      func() (importer.Result, error)
	}{
		{"milestones", cfg.ImportOptions.ImportMilestones, func() (importer.Result, error) { return importer.ImportMilestones(client, cfg) }},
		{"issues", cfg.ImportOptions.ImportIssues, func() (importer.Result, error) { return importer.ImportIssues(client, cfg) }},
		{"notes", cfg.ImportOptions.ImportNotes, func() (importer.Result, error) { return importer.ImportNotes(client, cfg) }},
		{"attachments", cfg.ImportOptions.ImportAttachments, func() (importer.Result, error) { return importer.ImportAttachments(client, cfg) }},
		{"wiki", cfg.ImportOptions.ImportWiki, func() (importer.Result, error) { return importer.ImportWiki(client, cfg) }},
	}
	failed := false
	for _, p := range phases {
		if !p.enabled {
			continue
		}
		progress.emit("phase %s", p.name)
		run.StartPhase(p.name)
		if err := run.Save(statePath); err != nil {
			return run, err
		}
		result, err := p.fn()
		run.FinishPhase(p.name, result.Total, result.Completed, result.Failed, result.Skipped, result.Errors)
		switch {
		case err != nil:
			failed = true
			run.Status = state.StatusFailed
			slog.Error("Phase failed", "phase", p.name, "errorMsg", err)
			progress.emit("phase %s failed: %v", p.name, err)
		case result.Failed > 0:
			failed = true
			run.Status = state.StatusFailed
			slog.Warn("Phase completed with errors", "phase", p.name, "failed", result.Failed)
		}
		if result.Skipped > 0 {
			slog.Warn("Phase skipped content", "phase", p.name, "skipped", result.Skipped)
		}
		if err := run.Save(statePath); err != nil {
			return run, err
		}
		progress.emit("phase %s: %d/%d completed, %d failed, %d skipped", p.name, result.Completed, result.Total, result.Failed, result.Skipped)
	}

	run.Finish()
	if err := run.Save(statePath); err != nil {
		return run, err
	}
	if failed {
		return run, ErrPhaseFailures
	}
	return run, nil
}
