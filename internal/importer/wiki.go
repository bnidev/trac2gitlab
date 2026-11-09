package importer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

type wikiPageFile struct {
	Meta        trac.WikiPage
	ContentPath string
}

func ImportWiki(client *gitlab.Client, cfg *config.Config) error {
	project, err := client.GetProject(cfg.GitLab.ProjectID)
	if err != nil {
		return err
	}

	wikiDir := filepath.Join(cfg.ExportOptions.ExportDir, "wiki")
	pages, err := collectWikiPages(wikiDir)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return nil
	}

	existing, err := client.ListWikis(project.ID, true)
	if err != nil {
		return fmt.Errorf("list existing wiki pages: %w", err)
	}
	byTitle := make(map[string]*gitlab.Wiki, len(existing))
	for _, page := range existing {
		byTitle[strings.ToLower(page.Title)] = page
	}

	format := gitlab.WikiFormatMarkdown
	for name, page := range pages {
		content, err := os.ReadFile(page.ContentPath)
		if err != nil {
			return fmt.Errorf("read wiki page %q: %w", name, err)
		}
		body := wikiContent(string(content), page.Meta)

		if current, ok := byTitle[strings.ToLower(name)]; ok {
			if current.Content == body {
				continue
			}
			title := name
			if _, err := client.EditWikiPage(project.ID, current.Slug, &gitlab.EditWikiPageOptions{
				Title:   &title,
				Content: &body,
				Format:  &format,
			}); err != nil {
				return fmt.Errorf("update wiki page %q: %w", name, err)
			}
			continue
		}

		title := name
		if _, err := client.CreateWikiPage(project.ID, &gitlab.CreateWikiPageOptions{
			Title:   &title,
			Content: &body,
			Format:  &format,
		}); err != nil {
			return fmt.Errorf("create wiki page %q: %w", name, err)
		}
	}

	return nil
}

func collectWikiPages(wikiDir string) (map[string]wikiPageFile, error) {
	pages := make(map[string]wikiPageFile)
	err := filepath.WalkDir(wikiDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".json") || !strings.Contains(filepath.Base(path), ".v") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var meta trac.WikiPage
		if err := json.Unmarshal(data, &meta); err != nil {
			return fmt.Errorf("parse wiki metadata %q: %w", path, err)
		}
		if meta.Name == "" {
			return nil
		}

		current, ok := pages[meta.Name]
		if ok && current.Meta.Version >= meta.Version {
			return nil
		}
		pages[meta.Name] = wikiPageFile{
			Meta:        meta,
			ContentPath: strings.TrimSuffix(path, ".json") + ".md",
		}
		return nil
	})
	if os.IsNotExist(err) {
		return pages, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan wiki export: %w", err)
	}
	return pages, nil
}

func wikiContent(content string, meta trac.WikiPage) string {
	var footer []string
	if meta.Author != "" {
		footer = append(footer, "Originally by "+meta.Author)
	}
	if !meta.LastModified.IsZero() {
		footer = append(footer, "on "+meta.LastModified.Format("2006-01-02"))
	}
	if len(footer) == 0 {
		return content
	}
	return strings.TrimRight(content, "\n") + "\n\n---\n_" + strings.Join(footer, " ") + fmt.Sprintf(" (Trac version %d)_\n", meta.Version)
}
