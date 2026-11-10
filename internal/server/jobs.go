package server

import (
	"fmt"
	"sync"
	"time"
)

// Job tracks one background run (export, dry-run) for a project directory.
type Job struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Project   string    `json:"project"`
	Status    string    `json:"status"` // running | completed | failed
	StartedAt time.Time `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	Error     string    `json:"error,omitempty"`
	Log       []string  `json:"log"`
}

func (j *Job) append(line string) {
	j.Log = append(j.Log, time.Now().UTC().Format("15:04:05")+" "+line)
	if len(j.Log) > 200 {
		j.Log = j.Log[len(j.Log)-200:]
	}
}

// Jobs is a per-project background job registry. At most one job runs per
// project directory at a time.
type Jobs struct {
	mu   sync.Mutex
	byID map[string]*Job
	seq  uint64
}

func NewJobs() *Jobs { return &Jobs{byID: make(map[string]*Job)} }

// Start launches fn in the background unless the project already has a
// running job. The returned job is safe to read via Get; Log appends are
// serialized through the registry mutex.
func (r *Jobs) Start(project, kind string, fn func(j *Job) error) (*Job, error) {
	r.mu.Lock()
	for _, j := range r.byID {
		if j.Project == project && j.Status == "running" {
			r.mu.Unlock()
			return nil, fmt.Errorf("a %s job is already running for this project", j.Kind)
		}
	}
	// Keep only the latest job per project: drop this project's finished
	// jobs so the registry does not grow for the process lifetime.
	for id, j := range r.byID {
		if j.Project == project && j.Status != "running" {
			delete(r.byID, id)
		}
	}
	r.seq++
	job := &Job{ID: fmt.Sprintf("%s-%d", kind, r.seq), Kind: kind, Project: project, Status: "running", StartedAt: time.Now().UTC()}
	job.append(kind + " started")
	r.byID[job.ID] = job
	r.mu.Unlock()

	go func() {
		if err := fn(job); err != nil {
			r.mu.Lock()
			job.Status = "failed"
			job.Error = err.Error()
			job.append("failed: " + err.Error())
			now := time.Now().UTC()
			job.EndedAt = &now
			r.mu.Unlock()
			return
		}
		r.mu.Lock()
		job.Status = "completed"
		job.append(kind + " completed")
		now := time.Now().UTC()
		job.EndedAt = &now
		r.mu.Unlock()
	}()
	return job, nil
}

// Append adds a timestamped line to the job log.
func (r *Jobs) Append(id, line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if j, ok := r.byID[id]; ok {
		j.append(line)
	}
}

// Latest returns the most recently started job for a project, or nil.
func (r *Jobs) Latest(project string) *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	var best *Job
	for _, j := range r.byID {
		if j.Project != project {
			continue
		}
		if best == nil || j.StartedAt.After(best.StartedAt) {
			best = j
		}
	}
	if best == nil {
		return nil
	}
	cp := *best
	cp.Log = append([]string(nil), best.Log...)
	return &cp
}
