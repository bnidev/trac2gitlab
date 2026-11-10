package state

import (
	"path/filepath"
	"testing"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "migration.yaml")
	m := New(true)
	m.StartPhase("issues")
	m.FinishPhase("issues", 11, 10, 1, 0, []string{"issue #4 failed"})
	m.Finish()
	if err := m.Save(path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !loaded.DryRun || loaded.Status != StatusCompleted {
		t.Fatalf("loaded = %+v", loaded)
	}
	phase := loaded.Phases["issues"]
	if phase == nil || phase.Total != 11 || phase.Completed != 10 || phase.Failed != 1 || phase.Status != StatusFailed {
		t.Fatalf("phase = %+v", phase)
	}
	if len(phase.Errors) != 1 {
		t.Fatalf("errors = %+v", phase.Errors)
	}
}

func TestLoadMissingState(t *testing.T) {
	m, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if m.Status != StatusPending || m.Phases == nil {
		t.Fatalf("missing state = %+v", m)
	}
}

func TestFinishPhaseSuccess(t *testing.T) {
	m := New(false)
	m.StartPhase("wiki")
	m.FinishPhase("wiki", 3, 3, 0, 0, nil)
	if m.Phases["wiki"].Status != StatusCompleted {
		t.Fatalf("phase = %+v", m.Phases["wiki"])
	}
}

func TestFinishPhaseSkipped(t *testing.T) {
	m := New(false)
	m.StartPhase("notes")
	m.FinishPhase("notes", 25, 25, 0, 16, nil)
	if got := m.Phases["notes"].Skipped; got != 16 {
		t.Fatalf("skipped = %d, want 16", got)
	}
}
