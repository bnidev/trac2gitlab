package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
	"gopkg.in/yaml.v3"
)

var projectNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

// handleCreateProject runs init for a new project directory: writes
// migration.yaml with defaults. Secrets are never part of this.
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON body, want {\"name\": ...}", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(body.Name)
	if !projectNameRe.MatchString(name) {
		http.Error(w, "invalid project name: use letters, digits, dash, underscore", http.StatusBadRequest)
		return
	}
	dir := filepath.Join(s.Root, name)
	if config.CheckConfigExists(dir) {
		http.Error(w, "project already exists", http.StatusConflict)
		return
	}
	if err := config.CreateDefaultConfig(dir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.log(r, "project created", "projectDir", dir)
	writeJSON(w, map[string]any{"name": name})
}

// handleTestTrac checks the stored Trac connection: expected XML-RPC
// methods plus plugin version. A failed connection is a normal answer,
// so this returns 200 with ok:false instead of an error status.
func (s *Server) handleTestTrac(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client, err := trac.NewTracClient(&cfg)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	if err := client.ValidateExpectedMethods(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "message": withAuthHint(err.Error(), "check TRAC_USER/TRAC_PASSWORD in .env")})
		return
	}
	msg := "expected methods ok"
	if err := client.ValidatePluginVersion(); err != nil {
		writeJSON(w, map[string]any{"ok": true, "message": msg + "; plugin: " + err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": msg + " + plugin version ok"})
}

// handleTestGitLab checks the stored GitLab connection: client, version,
// project resolution.
func (s *Server) handleTestGitLab(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	client, err := gitlab.NewGitLabClient(&cfg)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	if err := client.ValidateGitLab(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "message": withAuthHint(err.Error(), "check GITLAB_TOKEN in .env")})
		return
	}
	ref := any(cfg.GitLab.ProjectID)
	if cfg.GitLab.ProjectID == 0 {
		ref = cfg.GitLab.Project
	}
	project, err := client.GetProject(ref)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "message": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "message": "project " + project.PathWithNamespace})
}
// Secrets (passwords, tokens) are write-only: accepted on PUT, never returned.
// withAuthHint appends a credential hint to authentication failures.
func withAuthHint(msg, hint string) string {
	if strings.Contains(msg, "401") || strings.Contains(msg, "Unauthorized") || strings.Contains(msg, "unauthorized") {
		return msg + " — " + hint
	}
	return msg
}

// connectionForm is the editable, non-secret part of a project's connection.
// Secrets (passwords, tokens) are write-only: accepted on PUT, never returned.
type connectionForm struct {
	TracBaseURL  string `json:"tracBaseUrl"`
	TracRPCPath  string `json:"tracRpcPath"`
	TracUser     string `json:"tracUser,omitempty"`
	TracPassword string `json:"tracPassword,omitempty"`
	TracGateUser string `json:"tracGateUser,omitempty"`
	TracGatePass string `json:"tracGatePass,omitempty"`
	GitlabGateUser string `json:"gitlabGateUser,omitempty"`
	GitlabGatePass string `json:"gitlabGatePass,omitempty"`
	GitLabURL    string `json:"gitlabUrl"`
	GitLabToken  string `json:"gitlabToken,omitempty"`
	GitLabProj   string `json:"gitlabProject"`
	HasEnv       bool   `json:"hasEnv"`
	HasTracPass  bool   `json:"hasTracPassword"`
	HasTracGate  bool   `json:"hasTracGatePassword"`
	HasGitlabGate bool  `json:"hasGitlabGatePassword"`
	HasToken     bool   `json:"hasGitlabToken"`
}

// handleGetConnection returns the connection form without secrets.
func (s *Server) handleGetConnection(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	project := cfg.GitLab.Project
	if cfg.GitLab.ProjectID > 0 && project == "" {
		project = strconv.Itoa(cfg.GitLab.ProjectID)
	}
	writeJSON(w, connectionForm{
		TracBaseURL: cfg.Trac.BaseURL,
		TracRPCPath: cfg.Trac.RPCPath,
		TracUser:     cfg.Trac.Username,
		TracGateUser: cfg.Trac.HtaccessUser,
		GitlabGateUser: cfg.GitLab.HtaccessUser,
		GitLabURL:   cfg.GitLab.BaseURL,
		GitLabProj:  project,
		HasEnv:      hasFile(dir, ".env"),
		HasTracPass: cfg.Trac.Password != "",
		HasTracGate: cfg.Trac.HtaccessPass != "",
		HasGitlabGate: cfg.GitLab.HtaccessPass != "",
		HasToken:    cfg.GitLab.Token != "",
	})
}

// handlePutConnection updates migration.yaml (non-secrets) and .env
// (secrets + URLs). Unknown .env lines are preserved; secrets are never
// echoed back.
func (s *Server) handlePutConnection(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	var form connectionForm
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		http.Error(w, "invalid JSON connection form", http.StatusBadRequest)
		return
	}
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(form.TracBaseURL) != "" {
		cfg.Trac.BaseURL = strings.TrimRight(strings.TrimSpace(form.TracBaseURL), "/")
	}
	if strings.TrimSpace(form.TracRPCPath) != "" {
		cfg.Trac.RPCPath = strings.TrimSpace(form.TracRPCPath)
	}
	if strings.TrimSpace(form.GitLabURL) != "" {
		cfg.GitLab.BaseURL = strings.TrimRight(strings.TrimSpace(form.GitLabURL), "/")
	}
	proj := strings.TrimSpace(form.GitLabProj)
	if proj != "" {
		if id, err := strconv.Atoi(proj); err == nil {
			cfg.GitLab.ProjectID = id
			cfg.GitLab.Project = ""
		} else {
			cfg.GitLab.Project = proj
			cfg.GitLab.ProjectID = 0
		}
	}
	if err := writeConfig(dir, &cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	env := map[string]string{}
	if v := strings.TrimSpace(form.TracBaseURL); v != "" {
		env["TRAC_URL"] = v
	}
	if v := form.TracUser; v != "" {
		env["TRAC_USER"] = v
	}
	if v := form.TracPassword; v != "" {
		env["TRAC_PASSWORD"] = v
	}
	if v := form.TracGateUser; v != "" {
		env["TRAC_HTACCESS_USER"] = v
	}
	if v := form.TracGatePass; v != "" {
		env["TRAC_HTACCESS_PASSWORD"] = v
	}
	if v := form.GitlabGateUser; v != "" {
		env["GITLAB_HTACCESS_USER"] = v
	}
	if v := form.GitlabGatePass; v != "" {
		env["GITLAB_HTACCESS_PASSWORD"] = v
	}
	if v := strings.TrimSpace(form.GitLabURL); v != "" {
		env["GITLAB_URL"] = v
	}
	if v := form.GitLabToken; v != "" {
		env["GITLAB_TOKEN"] = v
	}
	if proj != "" {
		if _, err := strconv.Atoi(proj); err == nil {
			env["GITLAB_PROJECT_ID"] = proj
			env["GITLAB_PROJECT"] = ""
		} else {
			env["GITLAB_PROJECT"] = proj
			env["GITLAB_PROJECT_ID"] = ""
		}
	}
	if err := mergeDotEnv(filepath.Join(dir, ".env"), env); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.log(r, "connection saved", "projectDir", dir)
	writeJSON(w, map[string]any{"ok": true})
}

func writeConfig(dir string, cfg *config.Config) error {
	path := filepath.Join(dir, "migration.yaml")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(dir, "config.yaml")
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// mergeDotEnv upserts keys into a .env file, preserving comments, ordering
// and unknown entries. Empty values delete the key.
func mergeDotEnv(path string, updates map[string]string) error {
	var lines []string
	if raw, err := os.ReadFile(path); err == nil {
		lines = strings.Split(string(raw), "\n")
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range lines {
		key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.HasPrefix(key, "#") {
			out = append(out, line)
			continue
		}
		if v, update := updates[key]; update {
			seen[key] = true
			if v == "" {
				continue
			}
			out = append(out, key+"="+v)
			continue
		}
		out = append(out, line)
	}
	for key, v := range updates {
		if !seen[key] && v != "" {
			out = append(out, key+"="+v)
		}
	}
	joined := strings.Join(out, "\n")
	if joined != "" && !strings.HasSuffix(joined, "\n") {
		joined += "\n"
	}
	return os.WriteFile(path, []byte(joined), 0600)
}
