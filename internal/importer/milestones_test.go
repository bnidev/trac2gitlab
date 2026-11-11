package importer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

// TestImportMilestonesDryRunWritesNothing guards the dry-run contract: the
// default `migrate` must not create or update milestones on the target.
func TestImportMilestonesDryRunWritesNothing(t *testing.T) {
	var creates int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/1":
			_, _ = w.Write([]byte(`{"id":1,"name":"proj"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v4/projects/1/milestones":
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v4/projects/1/milestones":
			atomic.AddInt32(&creates, 1)
			_, _ = w.Write([]byte(`{"id":5,"iid":1,"title":"M1","state":"active"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	mdir := filepath.Join(dir, "milestones")
	if err := os.MkdirAll(mdir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mdir, "milestone-M1.json"), []byte(`{"name":"M1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{DryRun: true}
	cfg.GitLab.BaseURL = srv.URL
	cfg.GitLab.APIPath = "/api/v4"
	cfg.GitLab.ProjectID = 1
	cfg.ExportOptions.ExportDir = dir

	client, err := gitlab.NewGitLabClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	res, err := ImportMilestones(client, cfg)
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if creates != 0 {
		t.Fatalf("dry-run created %d milestone(s), want 0", creates)
	}
	if res.Total != 1 || res.Completed != 0 {
		t.Fatalf("dry-run result = %+v, want Total=1 Completed=0", res)
	}

	cfg.DryRun = false
	res, err = ImportMilestones(client, cfg)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if creates != 1 {
		t.Fatalf("execute created %d milestone(s), want 1", creates)
	}
	if res.Completed != 1 {
		t.Fatalf("execute result = %+v, want Completed=1", res)
	}
}
