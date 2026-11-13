package server

import (
	"net/http"
	"sync"
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

// writeTest records a test result and answers with it.
func (s *Server) writeTest(w http.ResponseWriter, dir, kind string, ok bool, msg string) {
	s.tests.set(dir, kind, ok)
	writeJSON(w, map[string]any{"ok": ok, "message": msg})
}
