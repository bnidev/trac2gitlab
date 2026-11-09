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
