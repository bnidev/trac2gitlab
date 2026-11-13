package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/bnidev/trac2gitlab/internal/config"
)

// connTests remembers Trac/GitLab test results for the life of the server,
// keyed by a fingerprint of the exact settings tested. A result counts for
// the Connect step only while the saved settings still match it, so saving
// tested values keeps "Connected", and any change (in the wizard or by hand
// in .env/migration.yaml) falls back to "Not tested".
type connTests struct {
	mu  sync.Mutex
	key []byte                     // per-process HMAC key: fingerprints of secrets stay unguessable
	m   map[string]map[string]bool // projectDir+"\x00"+kind -> fingerprint -> ok
}

func newConnTests() *connTests {
	key := make([]byte, 32)
	_, _ = rand.Read(key) // crypto/rand.Read never returns an error on supported platforms
	return &connTests{key: key, m: map[string]map[string]bool{}}
}

// maxResults bounds remembered fingerprints per service; older ones are
// dropped wholesale, which only means "Not tested" for stale settings.
const maxResults = 16

func (c *connTests) record(dir, kind, fp string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := dir + "\x00" + kind
	if c.m[k] == nil || len(c.m[k]) >= maxResults {
		c.m[k] = map[string]bool{}
	}
	c.m[k][fp] = ok
}

// lookup returns nil when these exact settings were never tested.
func (c *connTests) lookup(dir, kind, fp string) *bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok, found := c.m[dir+"\x00"+kind][fp]
	if !found {
		return nil
	}
	return &ok
}

// fingerprint hashes the settings one service's test depends on.
func (c *connTests) fingerprint(cfg *config.Config, kind string) string {
	var parts []string
	if kind == "trac" {
		t := cfg.Trac
		parts = []string{t.BaseURL, t.RPCPath, t.Username, t.Password, t.HtaccessUser, t.HtaccessPass}
	} else {
		g := cfg.GitLab
		parts = []string{g.BaseURL, g.Token, g.HtaccessUser, g.HtaccessPass, g.Project, strconv.Itoa(g.ProjectID)}
	}
	mac := hmac.New(sha256.New, c.key)
	mac.Write([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(mac.Sum(nil))
}

// testRun answers one Test request.
type testRun struct {
	s       *Server
	w       http.ResponseWriter
	dir     string
	kind    string
	fp      string // fingerprint of the settings actually tested (saved + form overlay)
	unsaved bool   // the form differed from the saved settings
}

// done records the result under the tested settings' fingerprint. A test of
// unsaved values starts counting once exactly those values are saved.
func (t testRun) done(ok bool, msg string) {
	t.s.tests.record(t.dir, t.kind, t.fp, ok)
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
