package server

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/exporter"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/state"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// handleExportRun starts the Trac export in the background. Export is
// read-only against Trac; it writes data/ under the project directory.
func (s *Server) handleExportRun(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.log(r, "export run", "projectDir", dir)
	job, err := s.Jobs.Start(dir, "export", func(j *Job) error {
		return runExport(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, job)
}

// handleDryRun starts migrate in dry-run mode in the background. Dry-run
// writes only state/migration.yaml; nothing is written to GitLab.
func (s *Server) handleDryRun(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cfg.DryRun = true
	s.log(r, "dry-run", "projectDir", dir)
	job, err := s.Jobs.Start(dir, "dry-run", func(j *Job) error {
		return runDryRun(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, job)
}

// handleJob returns the latest job for a project for polling.
func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	job := s.Jobs.Latest(dir)
	if job == nil {
		writeJSON(w, map[string]any{"exists": false})
		return
	}
	writeJSON(w, job)
}

// runExport mirrors the export CLI command orchestration.
func runExport(cfg *config.Config, log func(string)) error {
	client, err := trac.NewTracClient(cfg)
	if err != nil {
		return fmt.Errorf("create Trac client: %w", err)
	}
	if err := client.ValidateExpectedMethods(); err != nil {
		return fmt.Errorf("trac client validation: %w", err)
	}
	if err := client.ValidatePluginVersion(); err != nil {
		log("plugin version warning: " + err.Error())
	}
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
		log("exporting " + s.name)
		if err := s.fn(); err != nil {
			return fmt.Errorf("export %s: %w", s.name, err)
		}
		log("exported " + s.name)
	}
	return nil
}

// runDryRun mirrors the migrate CLI command in dry-run mode.
func runDryRun(cfg *config.Config, log func(string)) error {
	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		return fmt.Errorf("create GitLab client: %w", err)
	}
	if err := client.ValidateGitLab(); err != nil {
		return fmt.Errorf("gitlab validation: %w", err)
	}
	statePath := filepath.Join(cfg.ProjectDir, "state", "migration.yaml")
	run, err := state.Load(statePath)
	if err != nil {
		return err
	}
	run.DryRun = true
	run.Status = state.StatusRunning
	if err := run.Save(statePath); err != nil {
		return err
	}
	phases := []struct {
		name    string
		enabled bool
		fn      func() (importer.Result, error)
	}{
		{"milestones", cfg.ImportOptions.ImportMilestones, func() (importer.Result, error) {
			return importer.Result{}, importer.ImportMilestones(client, cfg)
		}},
		{"issues", cfg.ImportOptions.ImportIssues, func() (importer.Result, error) {
			return importer.ImportIssues(client, cfg)
		}},
		{"notes", cfg.ImportOptions.ImportNotes, func() (importer.Result, error) {
			return importer.ImportNotes(client, cfg)
		}},
		{"attachments", cfg.ImportOptions.ImportAttachments, func() (importer.Result, error) {
			return importer.ImportAttachments(client, cfg)
		}},
		{"wiki", cfg.ImportOptions.ImportWiki, func() (importer.Result, error) {
			return importer.ImportWiki(client, cfg)
		}},
	}
	failed := false
	for _, p := range phases {
		if !p.enabled {
			continue
		}
		log("phase " + p.name)
		run.StartPhase(p.name)
		if err := run.Save(statePath); err != nil {
			return err
		}
		result, err := p.fn()
		run.FinishPhase(p.name, result.Total, result.Completed, result.Failed, result.Errors)
		if err != nil || result.Failed > 0 {
			failed = true
			run.Status = state.StatusFailed
		}
		if err := run.Save(statePath); err != nil {
			return err
		}
		log(fmt.Sprintf("phase %s: %d/%d", p.name, result.Completed, result.Total))
	}
	run.Finish()
	if err := run.Save(statePath); err != nil {
		return err
	}
	if failed {
		return fmt.Errorf("dry-run completed with errors; see state/migration.yaml")
	}
	return nil
}
