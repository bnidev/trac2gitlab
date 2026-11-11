package exporter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// TestExportMilestonesNameWithSlash guards against milestone titles containing
// a path separator ("Project Matrix CR 11/2026") writing outside the milestones
// directory and being dropped from the import.
func TestExportMilestonesNameWithSlash(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		if strings.Contains(string(body), "milestone.getAll") {
			_, _ = w.Write([]byte(`<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><string>Project Matrix CR 11/2026</string></value>
</data></array></value></param></params></methodResponse>`))
			return
		}
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<methodResponse><params><param><value><struct>
<member><name>name</name><value><string>Project Matrix CR 11/2026</string></value></member>
<member><name>description</name><value><string>overview</string></value></member>
</struct></value></param></params></methodResponse>`))
	}))
	defer srv.Close()

	dir := filepath.Join(t.TempDir(), "data")
	cfg := &config.Config{}
	cfg.Trac.BaseURL = srv.URL
	cfg.Trac.RPCPath = "/xmlrpc"
	cfg.ExportOptions.ExportDir = dir

	client, err := trac.NewTracClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ExportMilestones(client, cfg); err != nil {
		t.Fatalf("ExportMilestones: %v", err)
	}

	path := filepath.Join(dir, "milestones", "milestone-Project Matrix CR 11-2026.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("expected sanitized milestone file: %v", err)
	}
	if !strings.Contains(string(data), `"name": "Project Matrix CR 11/2026"`) {
		t.Fatalf("milestone name not preserved in body: %s", data)
	}
}
