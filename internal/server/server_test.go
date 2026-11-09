package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
