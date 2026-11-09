package server

import (
	"net/http"
	"os"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/identity"
)

// UserRow is one users.yaml entry for the dashboard table.
type UserRow struct {
	Key      string   `json:"key"`
	GitLab   string   `json:"gitlab"`
	Action   string   `json:"action"`
	Resolved bool     `json:"resolved"`
	Aliases  []string `json:"aliases"`
}

// handleUsers returns the parsed users.yaml mapping. Missing file yields
// exists:false so the UI can suggest `discover --write-users`.
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := cfg.ImportOptions.UsersFile
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		writeJSON(w, map[string]any{"exists": false, "path": "mappings/users.yaml"})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mapping, err := identity.Parse(raw)
	if err != nil {
		writeJSON(w, map[string]any{"exists": true, "path": "mappings/users.yaml", "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"exists": true, "path": "mappings/users.yaml", "users": toRows(mapping)})
}

// handleUsersValidate mirrors `users validate` without --check-gitlab:
// pure local parsing, no network, no mutation.
func (s *Server) handleUsersValidate(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	raw, err := os.ReadFile(cfg.ImportOptions.UsersFile)
	if os.IsNotExist(err) {
		writeJSON(w, map[string]any{"ok": false, "error": "mappings/users.yaml not found; run: trac2gitlab discover --project-dir " + dir + " --write-users mappings/users.yaml"})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mapping, err := identity.Parse(raw)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	unresolved := mapping.Unresolved()
	keys := make([]string, 0, len(unresolved))
	for _, u := range unresolved {
		keys = append(keys, u.Key)
	}
	writeJSON(w, map[string]any{
		"ok":         len(unresolved) == 0,
		"total":      len(mapping.Users()),
		"unresolved": keys,
		"users":      toRows(mapping),
	})
}

func toRows(m *identity.Mapping) []UserRow {
	users := m.Users()
	rows := make([]UserRow, 0, len(users))
	for _, u := range users {
		rows = append(rows, UserRow{
			Key:      u.Key,
			GitLab:   u.GitLab,
			Action:   u.Action,
			Resolved: u.Resolved(),
			Aliases:  u.Aliases(),
		})
	}
	return rows
}
