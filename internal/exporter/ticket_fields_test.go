package exporter

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// TestExportTicketFieldsCreatesExportDir guards against the first-run failure
// where data/ does not exist yet: ExportTicketFields must create it like the
// other exporters do.
func TestExportTicketFieldsCreatesExportDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<methodResponse><params><param><value><array><data>
<value><struct>
<member><name>name</name><value><string>priority</string></value></member>
<member><name>options</name><value><array><data><value><string>high</string></value></data></array></value></member>
</struct></value>
</data></array></value></param></params></methodResponse>`))
	}))
	defer srv.Close()

	dir := filepath.Join(t.TempDir(), "data") // does not exist yet
	cfg := &config.Config{}
	cfg.Trac.BaseURL = srv.URL
	cfg.Trac.RPCPath = "/xmlrpc"
	cfg.ExportOptions.ExportDir = dir

	client, err := trac.NewTracClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ExportTicketFields(client, cfg); err != nil {
		t.Fatalf("ExportTicketFields: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ticket-fields.json")); err != nil {
		t.Fatalf("expected ticket-fields.json to be written: %v", err)
	}
}
