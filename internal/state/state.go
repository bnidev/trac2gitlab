// Package state tracks migration phase progress.
package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Phase struct {
	Status    string   `yaml:"status"`
	Total     int      `yaml:"total"`
	Completed int      `yaml:"completed"`
	Failed    int      `yaml:"failed"`
	Skipped   int      `yaml:"skipped,omitempty"`
	Errors    []string `yaml:"errors,omitempty"`
}

type Migration struct {
	Status     string            `yaml:"status"`
	DryRun     bool              `yaml:"dry_run"`
	StartedAt  time.Time         `yaml:"started_at"`
	FinishedAt *time.Time        `yaml:"finished_at,omitempty"`
	Phases     map[string]*Phase `yaml:"phases,omitempty"`
}

func New(dryRun bool) *Migration {
	return &Migration{
		Status:    StatusPending,
		DryRun:    dryRun,
		StartedAt: time.Now().UTC(),
		Phases:    make(map[string]*Phase),
	}
}

func Load(path string) (*Migration, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Migration{Status: StatusPending, Phases: make(map[string]*Phase)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state %q: %w", path, err)
	}
	var m Migration
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse state %q: %w", path, err)
	}
	if m.Phases == nil {
		m.Phases = make(map[string]*Phase)
	}
	return &m, nil
}

func (m *Migration) Save(path string) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	return nil
}

func (m *Migration) StartPhase(name string) {
	if m.StartedAt.IsZero() {
		m.StartedAt = time.Now().UTC()
	}
	m.Status = StatusRunning
	m.Phases[name] = &Phase{Status: StatusRunning}
}

func (m *Migration) FinishPhase(name string, total, completed, failed, skipped int, errs []string) {
	phase := m.Phases[name]
	if phase == nil {
		phase = &Phase{}
		m.Phases[name] = phase
	}
	phase.Total = total
	phase.Completed = completed
	phase.Failed = failed
	phase.Skipped = skipped
	phase.Errors = errs
	if failed > 0 {
		phase.Status = StatusFailed
	} else {
		phase.Status = StatusCompleted
	}
}

func (m *Migration) Finish() {
	now := time.Now().UTC()
	m.FinishedAt = &now
	if m.Status != StatusFailed {
		m.Status = StatusCompleted
	}
}
