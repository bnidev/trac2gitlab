package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/pipeline"
	"github.com/bnidev/trac2gitlab/internal/validate"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
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
		return pipeline.Export(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
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
		_, err := pipeline.Migrate(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
		return err
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
		_, err := pipeline.Migrate(&cfg, func(line string) { s.Jobs.Append(j.ID, line) })
		return err
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

// runValidate mirrors the validate CLI command. It only reads the export
// and the GitLab target and writes reports/validation.md.
func runValidate(cfg *config.Config, log func(string)) error {
	if err := pipeline.ResolveProjectID(cfg); err != nil {
		return fmt.Errorf("resolve GitLab project: %w", err)
	}
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
