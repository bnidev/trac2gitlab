package server

import (
	"net/http"
	"strconv"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/pipeline"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

// handlePatch emits the Rails-runner author patch as a copy-paste shell
// block. It only reads from GitLab (resolves the project and author IDs);
// the user runs the block on the GitLab host, never the wizard.
func (s *Server) handlePatch(w http.ResponseWriter, r *http.Request) {
	dir := ResolveProject(s.Root, r.URL.Query().Get("project"))
	cfg, err := config.LoadProject(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := pipeline.ResolveProjectID(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	client, err := gitlab.NewGitLabClient(&cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	q := r.URL.Query()
	apply := q.Get("apply") == "true"
	container := q.Get("container")
	if container == "" && q.Get("omnibus") != "true" {
		container = "gitlab"
	}
	rows, unresolved, err := importer.BuildAuthorPatch(&cfg, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	ruby := importer.RenderAuthorPatch(rows, cfg.GitLab.ProjectID, apply)
	writeJSON(w, map[string]any{
		"shell":      importer.RenderShell(ruby, container, q.Get("omnibus") == "true"),
		"unresolved": unresolved,
		"rows":       len(rows),
		"apply":      apply,
		"dryRun":     strconv.FormatBool(!apply),
	})
}
