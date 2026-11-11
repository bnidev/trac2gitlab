package importer

import (
	"log/slog"
	"path/filepath"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/references"
)

// warnUnresolvedChangesets reports changeset references that could not be
// rewritten (no mapping loaded, or no matching entry), so they are not left
// unresolved silently.
func warnUnresolvedChangesets(refs *references.Refs) {
	if n := refs.UnresolvedChangesets(); n > 0 {
		slog.Warn("changeset references left unresolved (no mapping or no match)", "count", n)
	}
}

func loadReferences(cfg *config.Config) (*references.Refs, error) {
	pages, err := CollectWikiPages(filepath.Join(cfg.ExportOptions.ExportDir, "wiki"))
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(pages))
	for name := range pages {
		names = append(names, name)
	}
	return references.Load(references.Options{
		Enabled:        cfg.References.Enabled,
		TicketsFile:    cfg.References.TicketsFile,
		ChangesetsFile: cfg.References.ChangesetsFile,
		WikiFile:       cfg.References.WikiFile,
		WikiPages:      names,
		BaseURL:        cfg.GitLab.BaseURL,
		ProjectPath:    cfg.GitLab.ProjectPath,
	})
}
