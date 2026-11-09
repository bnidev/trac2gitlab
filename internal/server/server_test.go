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
)

func TestListProjectsEmpty(t *testing.T) {
	if got := ListProjects(t.TempDir()); len(got) != 0 {
		t.Fatalf("expected no projects, got %v", got)
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
