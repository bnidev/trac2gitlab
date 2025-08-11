package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"trac2gitlab/internal/config"
	"trac2gitlab/pkg/trac"
)

// ExportWiki exports wiki pages from Trac and saves them as Markdown files
func ExportWiki(client *trac.Client, config *config.Config) error {
	slog.Info("Starting wiki export...")

	pages, err := client.GetWikiPageNames()
	if err != nil {
		return fmt.Errorf("failed to get wiki page names: %w", err)
	}
	if len(pages) == 0 {
		slog.Info("No wiki pages found, skipping export.")
		return nil
	}

	slog.Debug("Wiki pages found", "count", len(pages))

	wikiDir := filepath.Join(config.ExportOptions.ExportDir, "wiki")
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		return fmt.Errorf("failed to create wiki directory: %w", err)
	}

	const concurrency = 10
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for pageIndex, pageName := range pages {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, pName string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := exportWikiPage(client, wikiDir, pName, idx, len(pages), config.ExportOptions.IncludeAttachments); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(pageIndex, pageName)
	}

	wg.Wait()
	if firstErr != nil {
		return firstErr
	}

	slog.Info("Wiki export completed", "count", len(pages))
	return nil
}

func exportWikiPage(client *trac.Client, wikiDir, pageName string, pageIndex, totalPages int, includeAttachments bool) error {
	wikiMeta, err := client.GetWikiPageInfo(pageName)
	if err != nil {
		return fmt.Errorf("failed to get wiki page info for %q: %w", pageName, err)
	}

	slog.Debug("Exporting wiki page", "current", pageIndex+1, "total", totalPages, "page", pageName)

	for version := int64(1); version <= wikiMeta.Version; version++ {
		content, err := client.GetWikiPageVersion(pageName, version)
		if err != nil {
			slog.Warn("Failed to get wiki page version", "page", pageName, "version", version, "error", err)
			continue
		}

		if content == nil {
			slog.Warn("Wiki page version empty or not found", "page", pageName, "version", version)
			continue
		}

		filename := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.md", pageName, version))

		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			return fmt.Errorf("failed to create directories for wiki page %q: %w", pageName, err)
		}

		// Write content to file
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create file for wiki page %q version %d: %w", pageName, version, err)
		}

		if _, err := file.WriteString(*content); err != nil {
			if cerr := file.Close(); cerr != nil {
				slog.Warn("Failed to close file for wiki page", "page", pageName, "version", version, "error", cerr)
			}
			return fmt.Errorf("failed to write content for wiki page %q version %d: %w", pageName, version, err)
		}

		if cerr := file.Close(); cerr != nil {
			slog.Warn("Failed to close file for wiki page", "page", pageName, "version", version, "error", cerr)
		}

		// Write metadata to file
		meta, err := client.GetWikiPageInfoVersion(pageName, version)
		if err != nil {
			slog.Warn("Failed to get wiki page metadata", "page", pageName, "version", version, "error", err)
			continue
		}

		metaFile := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.json", pageName, version))
		metaFileHandle, err := os.Create(metaFile)
		if err != nil {
			return fmt.Errorf("failed to create metadata file for wiki page %q version %d: %w", pageName, version, err)
		}

		if err := json.NewEncoder(metaFileHandle).Encode(meta); err != nil {
			if cerr := metaFileHandle.Close(); cerr != nil {
				slog.Warn("Failed to close metadata file", "page", pageName, "version", version, "error", cerr)
			}
			return fmt.Errorf("failed to write metadata for wiki page %q version %d: %w", pageName, version, err)
		}

		if cerr := metaFileHandle.Close(); cerr != nil {
			slog.Warn("Failed to close metadata file", "page", pageName, "version", version, "error", cerr)
		}
	}

	// Export attachments once per page
	if len(wikiMeta.Attachments) > 0 && includeAttachments {
		slog.Debug("Exporting attachments for wiki page", "page", pageName, "count", len(wikiMeta.Attachments))

		attachmentsDir := filepath.Join(wikiDir, "attachments", pageName)
		if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
			slog.Warn("Failed to create attachments directory", "page", pageName, "error", err)
		} else {
			for _, att := range wikiMeta.Attachments {
				content, err := trac.GetAttachment(client, trac.ResourceWiki, pageName, att.Filename)
				if err != nil {
					slog.Warn("Failed to download attachment", "page", pageName, "filename", att.Filename, "error", err)
					continue
				}
				safeFilename := filepath.Base(att.Filename)
				attPath := filepath.Join(attachmentsDir, safeFilename)

				if err := os.WriteFile(attPath, content, 0644); err != nil {
					slog.Warn("Failed to write attachment", "page", pageName, "filename", att.Filename, "error", err)
					continue
				}

				slog.Debug("Attachment written", "page", pageName, "filename", att.Filename)
			}
		}
	}

	return nil
}
