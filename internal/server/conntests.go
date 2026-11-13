package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/bnidev/trac2gitlab/internal/config"
)

// connTests remembers the last Trac/GitLab test result per project for the
// life of the server, so a reload keeps the Connect step's state. Saving a
// service's settings clears its result.
type connTests struct {
	mu sync.Mutex
	m  map[string]bool // key: projectDir + "\x00" + kind
}

func newConnTests() *connTests { return &connTests{m: map[string]bool{}} }

func (c *connTests) set(dir, kind string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[dir+"\x00"+kind] = ok
}

func (c *connTests) clear(dir, kind string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, dir+"\x00"+kind)
}

// get returns nil when the service was not tested since its last save.
func (c *connTests) get(dir, kind string) *bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok, found := c.m[dir+"\x00"+kind]
	if !found {
		return nil
	}
	return &ok
}

// testRun answers one Test request. Results of tests against unsaved form
// values are reported but not remembered: they say nothing about the saved
// connection the Connect step is about.
type testRun struct {
	s       *Server
	w       http.ResponseWriter
	dir     string
	kind    string
	unsaved bool
}

func (t testRun) done(ok bool, msg string) {
	if !t.unsaved {
		t.s.tests.set(t.dir, t.kind, ok)
	}
	writeJSON(t.w, map[string]any{"ok": ok, "message": msg, "unsaved": t.unsaved})
}

// overlayForm applies the service's non-empty fields from an optional JSON
// connectionForm body onto cfg, the same way handlePutConnection would save
// them. It reports whether any value differs from the saved one.
func overlayForm(r *http.Request, cfg *config.Config, kind string) (bool, error) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		return false, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return false, nil
	}
	var f connectionForm
	if err := json.Unmarshal(body, &f); err != nil {
		return false, fmt.Errorf("invalid JSON connection form")
	}
	changed := false
	set := func(dst *string, v string) {
		if v != "" && v != *dst {
			*dst = v
			changed = true
		}
	}
	if kind == "trac" {
		set(&cfg.Trac.BaseURL, strings.TrimRight(strings.TrimSpace(f.TracBaseURL), "/"))
		set(&cfg.Trac.RPCPath, strings.TrimSpace(f.TracRPCPath))
		set(&cfg.Trac.Username, f.TracUser)
		set(&cfg.Trac.Password, f.TracPassword)
		set(&cfg.Trac.HtaccessUser, f.TracGateUser)
		set(&cfg.Trac.HtaccessPass, f.TracGatePass)
		return changed, nil
	}
	url := strings.TrimSpace(f.GitLabURL)
	if url != "" && !strings.Contains(url, "://") {
		url = "https://" + url
	}
	set(&cfg.GitLab.BaseURL, strings.TrimRight(url, "/"))
	set(&cfg.GitLab.Token, f.GitLabToken)
	set(&cfg.GitLab.HtaccessUser, f.GitlabGateUser)
	set(&cfg.GitLab.HtaccessPass, f.GitlabGatePass)
	if proj := strings.TrimSpace(f.GitLabProj); proj != "" {
		if id, err := strconv.Atoi(proj); err == nil {
			if id != cfg.GitLab.ProjectID {
				cfg.GitLab.ProjectID, cfg.GitLab.Project, changed = id, "", true
			}
		} else if proj != cfg.GitLab.Project || cfg.GitLab.ProjectID != 0 {
			cfg.GitLab.Project, cfg.GitLab.ProjectID, changed = proj, 0, true
		}
	}
	return changed, nil
}
