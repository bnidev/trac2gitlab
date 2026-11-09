package server

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/state"
)

//go:embed all:web
var webFS embed.FS

// Server is a read-only dashboard over migration project directories.
// Slice 1 exposes projects, status (state/migration.yaml) and the cached
// preflight/validation reports. Slice 4 adds background export and dry-run:
// both write only inside the project directory, never to GitLab.
type Server struct {
	Root   string
	Logger *slog.Logger
	Jobs   *Jobs
	tmpl   *template.Template
	mux    *http.ServeMux
}

// New builds the HTTP handler. Root is the directory scanned for projects
// (usually "projects" or ".").
func New(root string, logger *slog.Logger) (*Server, error) {
	tmpl, err := template.ParseFS(webFS, "web/templates/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{Root: root, Logger: logger, Jobs: NewJobs(), tmpl: tmpl, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("GET /api/projects", s.handleProjects)
	s.mux.HandleFunc("GET /api/status", s.handleStatus)
	s.mux.HandleFunc("GET /api/preflight", s.handleReport("reports/preflight.md"))
	s.mux.HandleFunc("POST /api/preflight/run", s.handlePreflightRun)
	s.mux.HandleFunc("GET /api/users", s.handleUsers)
	s.mux.HandleFunc("POST /api/users/validate", s.handleUsersValidate)
	s.mux.HandleFunc("POST /api/export/run", s.handleExportRun)
	s.mux.HandleFunc("POST /api/migrate/dry-run", s.handleDryRun)
	s.mux.HandleFunc("GET /api/job", s.handleJob)
	s.mux.HandleFunc("GET /api/validation", s.handleReport("reports/validation.md"))

	static, err := fs.Sub(webFS, "web/static")
	if err == nil {
		s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))
	}
}

func (s *Server) log(r *http.Request, msg string, args ...any) {
	if s.Logger != nil {
		s.Logger.Info(msg, append([]any{"method", r.Method, "path", r.URL.Path}, args...)...)
	}
}

type indexData struct {
	Root     string
	Projects []Project
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	s.log(r, "dashboard view")
	data := indexData{Root: s.Root, Projects: ListProjects(s.Root)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "layout.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, ListProjects(s.Root))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	run, err := state.Load(filepath.Join(dir, "state", "migration.yaml"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, run)
}

func (s *Server) handleReport(rel string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
		path := filepath.Join(dir, rel)
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			writeJSON(w, map[string]any{"exists": false, "path": rel})
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{
			"exists":  true,
			"path":    rel,
			"verdict": parseVerdict(string(raw)),
			"checks":  cachedChecks(dir),
			"raw":     string(raw),
		})
	}
}

// cachedChecks returns the structured checklist from the last live run, if any.
func cachedChecks(dir string) []Check {
	raw, err := os.ReadFile(filepath.Join(dir, "reports", "preflight.json"))
	if err != nil {
		return nil
	}
	var result PreflightResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result.Checks
}
func parseVerdict(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "STATUS:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
