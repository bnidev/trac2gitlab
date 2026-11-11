package importer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

type WikiPageFile struct {
	Meta        trac.WikiPage
	ContentPath string
}

func ImportWiki(client *gitlab.Client, cfg *config.Config) (Result, error) {
	var result Result

	refs, err := loadReferences(cfg)
	if err != nil {
		return result, err
	}

	project, err := client.GetProject(cfg.GitLab.ProjectID)
	if err != nil {
		return result, err
	}

	wikiDir := filepath.Join(cfg.ExportOptions.ExportDir, "wiki")
	pages, err := CollectWikiPages(wikiDir)
	if err != nil {
		return result, err
	}
	// Skip excluded pages here too, so a re-export that leaves stale files
	// behind still honours the current config.
	for name, page := range pages {
		if cfg.ExportOptions.WikiExcluded(name, page.Meta.Author) {
			slog.Info("Skipping excluded wiki page", "page", name, "author", page.Meta.Author)
			delete(pages, name)
		}
	}
	result.Total = len(pages)
	if len(pages) == 0 {
		return result, nil
	}

	existing, err := client.ListWikis(project.ID, true)
	if err != nil {
		result.fail(err)
		return result, fmt.Errorf("list existing wiki pages: %w", err)
	}
	bySlug := wikiBySlug(existing)

	attachmentsPath := filepath.Join(cfg.ExportOptions.ExportDir, "trac-wiki-attachments.json")
	attachmentMap, err := loadWikiAttachmentMapping(attachmentsPath)
	if err != nil {
		return result, err
	}
	attachmentIndex := make(map[string]wikiAttachmentRecord, len(attachmentMap.Attachments))
	for _, record := range attachmentMap.Attachments {
		attachmentIndex[wikiAttachmentKey(record.Page, record.Filename)] = record
	}

	format := gitlab.WikiFormatMarkdown
	for name, page := range pages {
		content, err := os.ReadFile(page.ContentPath)
		if err != nil {
			result.fail(err)
			slog.Error("Skipping wiki page", "page", name, "errorMsg", err)
			continue
		}
		if cfg.DryRun {
			slog.Info("dry-run: would import wiki page", "page", name)
			result.Completed++
			continue
		}

		section, err := uploadWikiAttachments(client, project.ID, cfg, name, page.Meta.Attachments, attachmentMap, attachmentIndex, attachmentsPath)
		if err != nil {
			result.fail(err)
			slog.Error("Skipping wiki page", "page", name, "errorMsg", err)
			continue
		}
		body := wikiContent(string(content)+section, page.Meta)
		body = refs.Rewrite(body)

		if current, ok := bySlug[strings.ToLower(name)]; ok {
			if current.Content == body {
				result.Completed++
				continue
			}
			title := name
			if _, err := client.EditWikiPage(project.ID, current.Slug, &gitlab.EditWikiPageOptions{
				Title:   &title,
				Content: &body,
				Format:  &format,
			}); err != nil {
				result.fail(err)
				slog.Error("Failed to update wiki page", "page", name, "errorMsg", err)
				continue
			}
			result.Completed++
			continue
		}

		title := name
		if _, err := client.CreateWikiPage(project.ID, &gitlab.CreateWikiPageOptions{
			Title:   &title,
			Content: &body,
			Format:  &format,
		}); err != nil {
			result.fail(err)
			slog.Error("Failed to create wiki page", "page", name, "errorMsg", err)
			continue
		}
		result.Completed++
	}

	warnUnresolvedChangesets(refs)
	return result, nil
}

// wikiBySlug indexes existing wiki pages by slug. GitLab humanizes the page
// title (dashes become spaces) but keeps the slug equal to the Trac page name,
// so the slug is what identifies an existing page across reruns.
func wikiBySlug(pages []*gitlab.Wiki) map[string]*gitlab.Wiki {
	bySlug := make(map[string]*gitlab.Wiki, len(pages))
	for _, page := range pages {
		bySlug[strings.ToLower(page.Slug)] = page
	}
	return bySlug
}

func CollectWikiPages(wikiDir string) (map[string]WikiPageFile, error) {
	pages := make(map[string]WikiPageFile)
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
		pages[meta.Name] = WikiPageFile{
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

type wikiAttachmentRecord struct {
	Page     string `json:"page"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Markdown string `json:"markdown"`
}

type wikiAttachmentMapping struct {
	Attachments []wikiAttachmentRecord `json:"attachments"`
}

func uploadWikiAttachments(client *gitlab.Client, projectID any, cfg *config.Config, page string, attachments []trac.Attachment, mapping *wikiAttachmentMapping, index map[string]wikiAttachmentRecord, mappingPath string) (string, error) {
	for _, attachment := range attachments {
		key := wikiAttachmentKey(page, attachment.Filename)
		if _, ok := index[key]; ok {
			continue
		}

		filename := filepath.Base(attachment.Filename)
		path := filepath.Join(cfg.ExportOptions.ExportDir, "wiki", "attachments", page, filename)
		file, err := os.Open(path)
		if err != nil {
			return "", fmt.Errorf("open wiki attachment %q for %q: %w", attachment.Filename, page, err)
		}
		uploaded, err := client.UploadWikiAttachment(projectID, file, filename, nil)
		closeErr := file.Close()
		if err != nil {
			return "", fmt.Errorf("upload wiki attachment %q for %q: %w", attachment.Filename, page, err)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close wiki attachment %q for %q: %w", attachment.Filename, page, closeErr)
		}

		record := wikiAttachmentRecord{
			Page:     page,
			Filename: attachment.Filename,
			URL:      uploaded.Link.URL,
			Markdown: uploaded.Link.Markdown,
		}
		mapping.Attachments = append(mapping.Attachments, record)
		index[key] = record
		if err := saveWikiAttachmentMapping(mappingPath, mapping); err != nil {
			return "", err
		}
	}

	return wikiAttachmentSection(page, attachments, index), nil
}

func wikiAttachmentKey(page, filename string) string {
	return page + "|" + filename
}

func wikiAttachmentSection(page string, attachments []trac.Attachment, index map[string]wikiAttachmentRecord) string {
	var lines []string
	for _, attachment := range attachments {
		record, ok := index[wikiAttachmentKey(page, attachment.Filename)]
		if !ok {
			continue
		}
		link := record.Markdown
		if link == "" {
			link = fmt.Sprintf("[%s](%s)", record.Filename, record.URL)
		}
		line := "- " + link
		var meta []string
		if attachment.Author != "" {
			meta = append(meta, "uploaded by "+attachment.Author)
		}
		if !attachment.Time.IsZero() {
			meta = append(meta, "on "+attachment.Time.Format("2006-01-02"))
		}
		if len(meta) > 0 {
			line += " — _" + strings.Join(meta, " ") + "_"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\n## Attachments\n\n" + strings.Join(lines, "\n")
}

func loadWikiAttachmentMapping(path string) (*wikiAttachmentMapping, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &wikiAttachmentMapping{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read wiki attachment mapping %q: %w", path, err)
	}
	var mapping wikiAttachmentMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parse wiki attachment mapping %q: %w", path, err)
	}
	return &mapping, nil
}

func saveWikiAttachmentMapping(path string, mapping *wikiAttachmentMapping) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("encode wiki attachment mapping: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create wiki attachment mapping directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write wiki attachment mapping: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace wiki attachment mapping: %w", err)
	}
	return nil
}
