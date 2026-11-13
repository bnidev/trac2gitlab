package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
)

func TestListProjectsEmpty(t *testing.T) {
	if got := ListProjects(t.TempDir()); len(got) != 0 {
		t.Fatalf("expected no projects, got %v", got)
	}
}

func TestResolveProjectRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo", "migration.yaml"), []byte("general:\n  log_level: info\n"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"../evil", "a/b", "..", ".", "/etc"} {
		if got := ResolveProject(root, name); got != root {
			t.Fatalf("ResolveProject(%q) = %q, want root %q", name, got, root)
		}
	}
	if got := ResolveProject(root, "demo"); got != filepath.Join(root, "demo") {
		t.Fatalf("valid name resolved to %q", got)
	}
}

func TestStatusMissingStateIsPending(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo", "migration.yaml"), []byte("general:\n  log_level: info\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/status?project=demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["Status"] != "pending" {
		t.Fatalf("expected pending state, got %v", body)
	}
}

func TestPreflightRunWritesReport(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	cfg := "trac:\n  base_url: http://127.0.0.1:1\n  rpc_path: /xmlrpc\ngitlab:\n  base_url: http://127.0.0.1:1\n  api_path: /api/v4\n  project: group/proj\n"
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte(cfg), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/preflight/run?project=demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["verdict"] == "" || body["checks"] == nil {
		t.Fatalf("expected verdict+checks, got %v", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "reports", "preflight.md")); err != nil {
		t.Fatalf("expected report file: %v", err)
	}
}

func TestPatchAndValidateRejectMissingConfig(t *testing.T) {
	srv, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/patch", "/api/patch?apply=true"} {
		req := httptest.NewRequest("GET", path+"?project=nope", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("patch %s status %d, want 400", path, rec.Code)
		}
	}
	req := httptest.NewRequest("POST", "/api/validate/run?project=nope", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("validate run status %d, want 400", rec.Code)
	}
}

func TestConnectEndpoints(t *testing.T) {
	root := t.TempDir()
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	post := func(name string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/projects", strings.NewReader(`{"name":"`+name+`"}`))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := post("../evil"); rec.Code != http.StatusBadRequest {
		t.Fatalf("path traversal status %d, want 400", rec.Code)
	}
	if rec := post("demo"); rec.Code != http.StatusOK {
		t.Fatalf("create status %d: %s", rec.Code, rec.Body.String())
	}
	if rec := post("demo"); rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status %d, want 409", rec.Code)
	}
	if !config.CheckConfigExists(filepath.Join(root, "demo")) {
		t.Fatal("expected migration.yaml in new project")
	}
	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/api/project/connection?project=demo", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := get(); rec.Code != http.StatusOK {
		t.Fatalf("get connection status %d: %s", rec.Code, rec.Body.String())
	} else {
		var form map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &form); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"tracPassword", "gitlabToken", "tracGatePass", "gitlabGatePass"} {
			if _, ok := form[k]; ok {
				t.Fatalf("secret field %q leaked in GET response", k)
			}
		}
	}
	putBody := `{"tracBaseUrl":"https://trac.local","gitlabUrl":"https://gl.local","gitlabProject":"grp/proj","tracUser":"u","tracPassword":"p","tracGateUser":"gu","tracGatePass":"gp","gitlabGateUser":"hu","gitlabGatePass":"hp","gitlabToken":"t"}`
	req := httptest.NewRequest("PUT", "/api/project/connection?project=demo", strings.NewReader(putBody))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put connection status %d: %s", rec.Code, rec.Body.String())
	}
	yamlRaw, err := os.ReadFile(filepath.Join(root, "demo", "migration.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"\"p\"", "gitlabToken", "\"t\"", "TRAC_PASSWORD"} {
		if strings.Contains(string(yamlRaw), secret) {
			t.Fatalf("secret leaked into migration.yaml: %s", secret)
		}
	}
	envRaw, err := os.ReadFile(filepath.Join(root, "demo", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TRAC_URL=https://trac.local", "TRAC_USER=u", "TRAC_PASSWORD=p", "TRAC_HTACCESS_USER=gu", "TRAC_HTACCESS_PASSWORD=gp", "GITLAB_HTACCESS_USER=hu", "GITLAB_HTACCESS_PASSWORD=hp", "GITLAB_TOKEN=t", "GITLAB_PROJECT=grp/proj"} {
		if !strings.Contains(string(envRaw), want) {
			t.Fatalf("expected .env to contain %q, got:\n%s", want, envRaw)
		}
	}
	st, err := os.Stat(filepath.Join(root, "demo", ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0600 {
		t.Fatalf(".env perm %o, want 600", st.Mode().Perm())
	}
}

func TestExecuteGate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(dir, "reports"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte("general:\n  log_level: info\n"), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/migrate/execute?project=demo", strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := post(`{"confirm": false}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("no confirm status %d, want 400", rec.Code)
	}
	if rec := post(`{"confirm": true}`); rec.Code != http.StatusConflict {
		t.Fatalf("no preflight status %d, want 409", rec.Code)
	}
	if err := os.WriteFile(filepath.Join(dir, "reports", "preflight.json"), []byte(`{"verdict":"BLOCKED"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if rec := post(`{"confirm": true}`); rec.Code != http.StatusConflict {
		t.Fatalf("blocked status %d, want 409", rec.Code)
	}
	if err := os.WriteFile(filepath.Join(dir, "reports", "preflight.json"), []byte(`{"verdict":"READY"}`), 0644); err != nil {
		t.Fatal(err)
	}
	rec := post(`{"confirm": true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("ready status %d: %s", rec.Code, rec.Body.String())
	}
	var job map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if job["status"] != "running" || job["kind"] != "execute" {
		t.Fatalf("expected running execute job, got %v", job)
	}
}

func TestJobsConflictAndCompletion(t *testing.T) {
	r := NewJobs()
	release := make(chan struct{})
	first, err := r.Start("proj", "export", func(j *Job) error {
		<-release
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Start("proj", "dry-run", func(j *Job) error { return nil }); err == nil {
		t.Fatal("expected conflict for second concurrent job")
	}
	if _, err := r.Start("other", "dry-run", func(j *Job) error { return nil }); err != nil {
		t.Fatalf("different project should not conflict: %v", err)
	}
	close(release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got := r.Latest("proj"); got != nil && got.Status == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job did not complete; got %+v", r.Latest("proj"))
		}
		time.Sleep(10 * time.Millisecond)
	}
	if first.ID == "" {
		t.Fatal("expected job id")
	}
}

func TestJobsKeepsOnlyLatestPerProject(t *testing.T) {
	r := NewJobs()
	start := func() {
		if _, err := r.Start("proj", "export", func(j *Job) error { return nil }); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for {
			if j := r.Latest("proj"); j != nil && j.Status == "completed" {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("job did not complete; got %+v", r.Latest("proj"))
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	start()
	start()
	r.mu.Lock()
	n := len(r.byID)
	r.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 retained job, got %d", n)
	}
}

func TestRunEndpointsRejectMissingConfig(t *testing.T) {
	srv, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/export/run", "/api/migrate/dry-run"} {
		req := httptest.NewRequest("POST", path+"?project=nope", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s status %d, want 400", path, rec.Code)
		}
	}
	req := httptest.NewRequest("GET", "/api/job?project=nope", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("job status %d", rec.Code)
	}
}

func TestUsersEndpoints(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(dir, "mappings"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte("general:\n  log_level: info\n"), 0644); err != nil {
		t.Fatal(err)
	}
	users := "users:\n  jdoe:\n    trac_username: jdoe\n    gitlab: jdoe\n    confidence: manual\n  old-dev:\n    trac_username: old-dev\n    action: unresolved\n    confidence: manual\n"
	if err := os.WriteFile(filepath.Join(dir, "mappings", "users.yaml"), []byte(users), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/users?project=demo", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("users status %d: %s", rec.Code, rec.Body.String())
	}
	var list map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if list["exists"] != true {
		t.Fatalf("expected exists, got %v", list)
	}

	req = httptest.NewRequest("POST", "/api/users/validate?project=demo", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("validate status %d: %s", rec.Code, rec.Body.String())
	}
	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res["ok"] != false {
		t.Fatalf("expected ok=false with one unresolved, got %v", res)
	}
	if res["total"] != float64(2) {
		t.Fatalf("expected total=2, got %v", res)
	}

	missing, err := New(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req = httptest.NewRequest("GET", "/api/users", nil)
	rec = httptest.NewRecorder()
	missing.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing config status %d, want 400", rec.Code)
	}
}

func TestPreflightMissingReport(t *testing.T) {
	root := t.TempDir()
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/preflight", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

// TestPutConnectionKeepsSecretsAndPathsOut guards against re-encoding the
// loaded config: that leaked GITLAB_TOKEN from .env into migration.yaml and
// wrote project-dir-joined paths that double on the next load.
func TestPutConnectionKeepsSecretsAndPathsOut(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	original := "# keep me\ngitlab:\n  base_url: https://old.local\n  api_path: /api/v4\nexport_options:\n  export_dir: data\n  wiki_exclude_authors: [trac]\nimport_options:\n  users_file: mappings/users.yaml\n"
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("GITLAB_TOKEN=secret-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("PUT", "/api/project/connection?project=demo", strings.NewReader(`{"gitlabProject":"42","tracBaseUrl":"https://trac.local/"}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put connection status %d: %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "migration.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	yamlText := string(raw)
	for _, bad := range []string{"secret-token", "token:", dir} {
		if strings.Contains(yamlText, bad) {
			t.Fatalf("migration.yaml contains %q:\n%s", bad, yamlText)
		}
	}
	for _, want := range []string{"# keep me", "export_dir: data", "users_file: mappings/users.yaml", "project_id: 42", "base_url: https://trac.local\n", "wiki_exclude_authors: [trac]", "  api_path: /api/v4"} {
		if !strings.Contains(yamlText, want) {
			t.Fatalf("migration.yaml missing %q:\n%s", want, yamlText)
		}
	}
	cfg, err := config.LoadProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "data"); cfg.ExportOptions.ExportDir != want {
		t.Fatalf("export dir = %q, want %q", cfg.ExportOptions.ExportDir, want)
	}
}

func TestDiscoverRunWritesUsersOnce(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "demo")
	if err := os.MkdirAll(filepath.Join(dir, "data", "tickets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte("export_options:\n  export_dir: data\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ticket := `{"ID":1,"Attributes":{"reporter":"jdoe","owner":"jane","description":"x","status":"closed"},"Comments":[],"Attachments":[]}`
	if err := os.WriteFile(filepath.Join(dir, "data", "tickets", "ticket-1.json"), []byte(ticket), 0644); err != nil {
		t.Fatal(err)
	}
	srv, err := New(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	run := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/discover/run?project=demo", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	if rec := run(); rec.Code != http.StatusOK {
		t.Fatalf("discover status %d: %s", rec.Code, rec.Body.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "mappings", "users.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "jdoe") || !strings.Contains(string(raw), "unresolved") {
		t.Fatalf("users.yaml = %s", raw)
	}
	if rec := run(); rec.Code != http.StatusConflict {
		t.Fatalf("second discover status %d, want 409", rec.Code)
	}
}

func TestHasExport(t *testing.T) {
	cases := []struct {
		name   string
		config string
		files  []string // relative to the project dir
		want   bool
	}{
		{"no data dir", "general:\n  log_level: info\n", nil, false},
		{"empty data dir", "general:\n  log_level: info\n", []string{"data/"}, false},
		{"default data dir", "general:\n  log_level: info\n", []string{"data/ticket-fields.json"}, true},
		{"custom export dir", "export_options:\n  export_dir: out\n", []string{"out/ticket-fields.json"}, true},
		{"custom dir ignores data", "export_options:\n  export_dir: out\n", []string{"data/ticket-fields.json"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte(tc.config), 0644); err != nil {
				t.Fatal(err)
			}
			for _, f := range tc.files {
				p := filepath.Join(dir, f)
				if strings.HasSuffix(f, "/") {
					if err := os.MkdirAll(p, 0755); err != nil {
						t.Fatal(err)
					}
					continue
				}
				if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if got := hasExport(dir); got != tc.want {
				t.Fatalf("hasExport = %v, want %v", got, tc.want)
			}
		})
	}
}
