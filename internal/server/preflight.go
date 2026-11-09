package server

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/preflight"
)

// Check is one row of the preflight checklist UI.
type Check struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	State  string `json:"state"` // pass | warn | fail
}

// PreflightResult is the JSON body for the live preflight run.
type PreflightResult struct {
	Verdict  string   `json:"verdict"`
	Checks   []Check  `json:"checks"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// handlePreflightRun executes the read-only preflight checks in-process,
// writes reports/preflight.md (same file the CLI writes) and returns the
// structured verdict. Preflight never mutates the target.
func (s *Server) handlePreflightRun(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.log(r, "preflight run", "projectDir", dir)

	report := preflight.Run(&cfg)
	path := filepath.Join(cfg.ProjectDir, "reports", "preflight.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(path, []byte(report.Markdown(filepath.Base(cfg.ProjectDir))), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, toResult(report))
}

func toResult(r *preflight.Report) PreflightResult {
	pass := func(ok bool) string {
		if ok {
			return "pass"
		}
		return "fail"
	}
	checks := []Check{
		{Name: "Trac XML-RPC", Detail: "expected methods + plugin version", State: pass(r.TracOK)},
		{Name: "GitLab project", Detail: "project resolves, role " + roleOrUnknown(r.Role), State: pass(r.GitLabOK)},
		{Name: "Historical timestamps", Detail: "Owner or Administrator required", State: pass(r.TimestampOK)},
		{Name: "Target state", Detail: itoa(r.TargetIssues) + " existing issues", State: warnIf(r.TargetIssues > 0, "pass")},
		{Name: "Identities", Detail: itoa(r.Unresolved) + " unresolved in users.yaml", State: warnIf(r.Unresolved > 0, "pass")},
	}
	// A blocked role fails the timestamp check explicitly.
	if r.Blocked {
		checks[2].State = "fail"
	}
	return PreflightResult{
		Verdict:  r.Verdict,
		Checks:   checks,
		Errors:   r.Errors,
		Warnings: r.Warnings,
	}
}

func roleOrUnknown(role string) string {
	if role == "" {
		return "unknown"
	}
	return role
}

func warnIf(cond bool, ok string) string {
	if cond {
		return "warn"
	}
	return ok
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
