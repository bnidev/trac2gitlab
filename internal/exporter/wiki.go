package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"trac2gitlab/pkg/trac"
)

// ExportWiki exports all wiki pages from a Trac instance to markdown files
func ExportWiki(client *trac.Client, outDir string, includeAttachments bool) error {
	fmt.Println("Exporting wiki...")

	pages, err := client.GetWikiPageNames()
	if err != nil {
		return fmt.Errorf("failed to get wiki page names: %w", err)
	}
	if len(pages) == 0 {
		fmt.Println("No wiki pages found.")
	}

	fmt.Printf("Found %d wiki page%s\n", len(pages), func() string {
		if len(pages) == 1 {
			return ""
		}
		return "s"
	}())

	wikiDir := filepath.Join(outDir, "wiki")
	if err := os.MkdirAll(wikiDir, 0755); err != nil {
		return fmt.Errorf("failed to create wiki directory: %w", err)
	}

	for _, pageName := range pages {
		wikiMeta, err := client.GetWikiPageInfo(pageName)
		if err != nil {
			return fmt.Errorf("failed to get wiki page info for %q: %w", pageName, err)
		}

		for version := int64(1); version <= wikiMeta.Version; version++ {
			content, err := client.GetWikiPageVersion(pageName, version)
			if err != nil {
				return fmt.Errorf("failed to get wiki page %q version %d: %w", pageName, version, err)
			}

			if content == nil {
				fmt.Printf("Warning: wiki page %q version %d is empty or not found\n", pageName, version)
				continue
			}

			// Save content
			filename := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.md", pageName, version))
			if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
				return fmt.Errorf("failed to create directories for wiki page %q: %w", pageName, err)
			}
			file, err := os.Create(filename)
			if err != nil {
				return fmt.Errorf("failed to create file for wiki page %q version %d: %w", pageName, version, err)
			}
			defer file.Close()

			if _, err := file.WriteString(*content); err != nil {
				return fmt.Errorf("failed to write content for wiki page %q version %d: %w", pageName, version, err)
			}

			// Save metadata
			meta, err := client.GetWikiPageInfoVersion(pageName, version)
			if err != nil {
				return fmt.Errorf("failed to get wiki page info for %q version %d: %w", pageName, version, err)
			}

			metaFile := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.json", pageName, version))
			metaFileHandle, err := os.Create(metaFile)
			if err != nil {
				return fmt.Errorf("failed to create metadata file for wiki page %q version %d: %w", pageName, version, err)
			}
			defer metaFileHandle.Close()

			if err := json.NewEncoder(metaFileHandle).Encode(meta); err != nil {
				return fmt.Errorf("failed to write metadata for wiki page %q version %d: %w", pageName, version, err)
			}

			// Export attachments for each ticket
			if len(wikiMeta.Attachments) > 0 && includeAttachments {
				attachmentsDir := filepath.Join(wikiDir, "attachments", fmt.Sprintf("%s", pageName))
				if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
					log.Printf("Warning: failed to create attachments directory for ticket #%s: %v\n", pageName, err)
					// continue without attachments but don't skip the whole ticket export
				} else {
					for _, att := range wikiMeta.Attachments {
						content, err := trac.GetAttachment(client, trac.ResourceWiki, pageName, att.Filename)
						if err != nil {
							log.Printf("Warning: failed to download attachment %q for ticket #%s: %v\n", att.Filename, pageName, err)
							continue
						}

						// Sanitize filename (basic example)
						safeFilename := filepath.Base(att.Filename)
						attPath := filepath.Join(attachmentsDir, safeFilename)

						if err := os.WriteFile(attPath, content, 0644); err != nil {
							log.Printf("Warning: failed to write attachment %q for wiki #%s: %v\n", att.Filename, pageName, err)
						}
					}
				}
			}

		}
	}

	fmt.Println("Wiki export complete.")
	return nil
}
