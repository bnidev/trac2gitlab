package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/app"
	"github.com/bnidev/trac2gitlab/internal/config"
)

// TestExportReturnsErrorOnTracFailure guards against export swallowing phase
// failures and reporting success: a failed run must return an error so the
// process exits non-zero.
func TestExportReturnsErrorOnTracFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := config.Config{}
	cfg.Trac.BaseURL = srv.URL
	cfg.Trac.RPCPath = "/xmlrpc"

	cmd := exportCmd(&app.AppContext{Config: &cfg})
	if cmd.RunE == nil {
		t.Fatal("export must use RunE so failures exit non-zero")
	}
	if err := cmd.RunE(cmd, nil); err == nil {
		t.Fatal("expected an error when Trac is unreachable")
	}
}
