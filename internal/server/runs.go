package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/exporter"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/state"
	"github.com/bnidev/trac2gitlab/internal/validate"
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
		return runMigrate(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, job)
}

// handleValidateRun starts the target comparison in the background.
// Validate only reads the export and the GitLab target and writes
// reports/validation.md.
func (s *Server) handleValidateRun(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.log(r, "validate", "projectDir", dir)
	job, err := s.Jobs.Start(dir, "validate", func(j *Job) error {
		return runValidate(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	writeJSON(w, job)
}

// executeRequest is the explicit confirmation body for POST /api/migrate/execute.
type executeRequest struct {
	Confirm bool `json:"confirm"`
}

// handleExecute starts migrate with writes enabled. It requires an explicit
// confirmation in the request body and a passing cached preflight verdict
// (READY or READY WITH WARNINGS). There is no force bypass.
func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	var body executeRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body, want {\"confirm\": true}", http.StatusBadRequest)
		return
	}
	if !body.Confirm {
		http.Error(w, "explicit confirmation required: {\"confirm\": true}", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if verdict := cachedVerdict(dir); verdict != "READY" && verdict != "READY WITH WARNINGS" {
		http.Error(w, "preflight verdict is "+quoted(verdict)+"; run preflight until READY or READY WITH WARNINGS first", http.StatusConflict)
		return
	}
	cfg.DryRun = false
	s.log(r, "execute", "projectDir", dir)
	job, err := s.Jobs.Start(dir, "execute", func(j *Job) error {
		return runMigrate(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
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

// cachedVerdict returns the verdict from the last live preflight run, or ""
// when preflight never ran in the wizard.
func cachedVerdict(dir string) string {
	raw, err := os.ReadFile(filepath.Join(dir, "reports", "preflight.json"))
	if err != nil {
		return ""
	}
	var result PreflightResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return ""
	}
	return result.Verdict
}

func quoted(s string) string {
	if s == "" {
		return "missing"
	}
	return s
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

// runValidate mirrors the validate CLI command. It only reads the export
// and the GitLab target and writes reports/validation.md.
func runValidate(cfg *config.Config, log func(string)) error {
	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		return fmt.Errorf("create gitlab client: %w", err)
	}
	log("comparing export with gitlab target")
	report, err := validate.Run(client, cfg)
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}
	path := filepath.Join(cfg.ProjectDir, "reports", "validation.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(report.Markdown(filepath.Base(cfg.ProjectDir))), 0644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	log("verdict: " + report.Verdict())
	if report.Verdict() == "FAIL" {
		return fmt.Errorf("validation failed (%d errors)", len(report.Errors))
	}
	return nil
}

// runMigrate mirrors the migrate CLI command. cfg.DryRun selects preview
// (state only) vs execute (writes to GitLab).
func runMigrate(cfg *config.Config, log func(string)) error {
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
	run.DryRun = cfg.DryRun
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
