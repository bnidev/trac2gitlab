package server

import (
	"bytes"
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

// handleTestTrac checks the Trac connection: expected XML-RPC methods plus
// plugin version. An optional JSON connectionForm body overlays unsaved form
// values on the saved config, so Test checks what the user typed without
// writing it. A failed connection is a normal answer, so this returns 200
// with ok:false instead of an error status.
func (s *Server) handleTestTrac(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unsaved, err := overlayForm(r, &cfg, "trac")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := testRun{s: s, w: w, dir: dir, kind: "trac", fp: s.tests.fingerprint(&cfg, "trac"), unsaved: unsaved}
	client, err := trac.NewTracClient(&cfg)
	if err != nil {
		t.done(false, err.Error())
		return
	}
	if err := client.ValidateExpectedMethods(); err != nil {
		t.done(false, withAuthHint(err.Error(), "check TRAC_USER/TRAC_PASSWORD in .env"))
		return
	}
	msg := "expected methods ok"
	if err := client.ValidatePluginVersion(); err != nil {
		t.done(true, msg+"; plugin: "+err.Error())
		return
	}
	t.done(true, msg+" + plugin version ok")
}

// handleTestGitLab checks the GitLab connection: client, version, project
// resolution. Takes the same optional form overlay as handleTestTrac.
func (s *Server) handleTestGitLab(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	unsaved, err := overlayForm(r, &cfg, "gitlab")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	t := testRun{s: s, w: w, dir: dir, kind: "gitlab", fp: s.tests.fingerprint(&cfg, "gitlab"), unsaved: unsaved}
	client, err := gitlab.NewGitLabClient(&cfg)
	if err != nil {
		t.done(false, err.Error())
		return
	}
	if err := client.ValidateGitLab(); err != nil {
		t.done(false, withAuthHint(err.Error(), "check GITLAB_TOKEN in .env"))
		return
	}
	ref := any(cfg.GitLab.ProjectID)
	if cfg.GitLab.ProjectID == 0 {
		ref = cfg.GitLab.Project
	}
	project, err := client.GetProject(ref)
	if err != nil {
		t.done(false, err.Error())
		return
	}
	t.done(true, "project "+project.PathWithNamespace)
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
	TracBaseURL    string `json:"tracBaseUrl"`
	TracRPCPath    string `json:"tracRpcPath"`
	TracUser       string `json:"tracUser,omitempty"`
	TracPassword   string `json:"tracPassword,omitempty"`
	TracGateUser   string `json:"tracGateUser,omitempty"`
	TracGatePass   string `json:"tracGatePass,omitempty"`
	GitlabGateUser string `json:"gitlabGateUser,omitempty"`
	GitlabGatePass string `json:"gitlabGatePass,omitempty"`
	GitLabURL      string `json:"gitlabUrl"`
	GitLabToken    string `json:"gitlabToken,omitempty"`
	GitLabProj     string `json:"gitlabProject"`
	HasEnv         bool   `json:"hasEnv"`
	HasTracPass    bool   `json:"hasTracPassword"`
	HasTracGate    bool   `json:"hasTracGatePassword"`
	HasGitlabGate  bool   `json:"hasGitlabGatePassword"`
	HasToken       bool   `json:"hasGitlabToken"`
	TracOK         *bool  `json:"tracOk"`   // result for exactly the saved settings; null when never tested
	GitLabOK       *bool  `json:"gitlabOk"` // result for exactly the saved settings; null when never tested
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
		TracBaseURL:    cfg.Trac.BaseURL,
		TracRPCPath:    cfg.Trac.RPCPath,
		TracUser:       cfg.Trac.Username,
		TracGateUser:   cfg.Trac.HtaccessUser,
		GitlabGateUser: cfg.GitLab.HtaccessUser,
		GitLabURL:      cfg.GitLab.BaseURL,
		GitLabProj:     project,
		HasEnv:         hasFile(dir, ".env"),
		HasTracPass:    cfg.Trac.Password != "",
		HasTracGate:    cfg.Trac.HtaccessPass != "",
		HasGitlabGate:  cfg.GitLab.HtaccessPass != "",
		HasToken:       cfg.GitLab.Token != "",
		TracOK:         s.tests.lookup(dir, "trac", s.tests.fingerprint(&cfg, "trac")),
		GitLabOK:       s.tests.lookup(dir, "gitlab", s.tests.fingerprint(&cfg, "gitlab")),
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
	if _, err := config.LoadProject(dir); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Only the connection keys the form owns are written back. Re-encoding
	// the loaded config would leak secrets merged from .env (gitlab.token)
	// and persist project-dir-joined paths (export_dir, users_file).
	var updates []configUpdate
	if v := strings.TrimSpace(form.TracBaseURL); v != "" {
		updates = append(updates, configUpdate{[]string{"trac", "base_url"}, strings.TrimRight(v, "/")})
	}
	if v := strings.TrimSpace(form.TracRPCPath); v != "" {
		updates = append(updates, configUpdate{[]string{"trac", "rpc_path"}, v})
	}
	if v := strings.TrimSpace(form.GitLabURL); v != "" {
		updates = append(updates, configUpdate{[]string{"gitlab", "base_url"}, strings.TrimRight(v, "/")})
	}
	proj := strings.TrimSpace(form.GitLabProj)
	if proj != "" {
		if id, err := strconv.Atoi(proj); err == nil {
			updates = append(updates,
				configUpdate{[]string{"gitlab", "project_id"}, id},
				configUpdate{[]string{"gitlab", "project"}, ""})
		} else {
			updates = append(updates,
				configUpdate{[]string{"gitlab", "project"}, proj},
				configUpdate{[]string{"gitlab", "project_id"}, 0})
		}
	}
	if err := writeConfig(dir, updates); err != nil {
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

// configUpdate sets one scalar at a key path in migration.yaml.
type configUpdate struct {
	path  []string
	value any
}

// writeConfig applies updates to the project's config file in place,
// keeping every other key, comment and ordering untouched.
func writeConfig(dir string, updates []configUpdate) error {
	path := filepath.Join(dir, "migration.yaml")
	if _, err := os.Stat(path); err != nil {
		path = filepath.Join(dir, "config.yaml")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%s: top level is not a mapping", path)
	}
	for _, u := range updates {
		var value yaml.Node
		if err := value.Encode(u.value); err != nil {
			return fmt.Errorf("encode %s: %w", strings.Join(u.path, "."), err)
		}
		if err := setYAMLKey(doc.Content[0], u.path, &value); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// setYAMLKey sets value at keys under mapping, creating nested mappings.
func setYAMLKey(mapping *yaml.Node, keys []string, value *yaml.Node) error {
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		if mapping.Content[i].Value != keys[0] {
			continue
		}
		if len(keys) == 1 {
			mapping.Content[i+1] = value
			return nil
		}
		child := mapping.Content[i+1]
		if child.Kind == yaml.ScalarNode && child.Tag == "!!null" {
			*child = yaml.Node{Kind: yaml.MappingNode}
		}
		if child.Kind != yaml.MappingNode {
			return fmt.Errorf("%s is not a mapping", keys[0])
		}
		return setYAMLKey(child, keys[1:], value)
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Value: keys[0]}
	if len(keys) == 1 {
		mapping.Content = append(mapping.Content, key, value)
		return nil
	}
	child := &yaml.Node{Kind: yaml.MappingNode}
	mapping.Content = append(mapping.Content, key, child)
	return setYAMLKey(child, keys[1:], value)
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
