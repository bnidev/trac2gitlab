package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bnidev/trac2gitlab/internal/state"
)

func TestNextStep(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		run  *state.Migration
		want string
	}{
		{"empty", &state.Migration{Status: state.StatusPending}, "preflight"},
		{"running", &state.Migration{Status: state.StatusRunning}, "running"},
		{"failed", &state.Migration{Status: state.StatusFailed, Phases: map[string]*state.Phase{"issues": {}}}, "rerun"},
		{"dry-run", &state.Migration{Status: state.StatusCompleted, DryRun: true, Phases: map[string]*state.Phase{"issues": {}}}, "execute"},
		{"completed", &state.Migration{Status: state.StatusCompleted, Phases: map[string]*state.Phase{"issues": {}}}, "patch"},
	}
	for _, tc := range cases {
		if got := nextStep(dir, tc.run); !strings.Contains(got, tc.want) {
			t.Errorf("%s: nextStep = %q, want substring %q", tc.name, got, tc.want)
		}
	}

	if err := os.MkdirAll(filepath.Join(dir, "reports"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reports", "validation.md"), []byte("STATUS: PASS"), 0644); err != nil {
		t.Fatal(err)
	}
	completed := &state.Migration{Status: state.StatusCompleted, Phases: map[string]*state.Phase{"issues": {}}}
	if got := nextStep(dir, completed); !strings.Contains(got, "validation") {
		t.Fatalf("nextStep = %q, want validation", got)
	}
}

func TestMode(t *testing.T) {
	if mode(true) != "dry-run" || mode(false) != "execute" {
		t.Fatalf("mode = %q/%q", mode(true), mode(false))
	}
}
