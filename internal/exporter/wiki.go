package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"trac2gitlab/pkg/trac"
)

func ExportWiki(client *trac.Client, outDir string, includeAttachments bool) error {
	fmt.Println("Exporting wiki...")

	pages, err := client.GetWikiPageNames()
	if err != nil {
		return fmt.Errorf("failed to get wiki page names: %w", err)
	}
	if len(pages) == 0 {
		fmt.Println("No wiki pages found.")
		return nil
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

			if err := exportWikiPage(client, wikiDir, pName, idx, len(pages), includeAttachments); err != nil {
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

	fmt.Println("Wiki export complete.")
	return nil
}

func exportWikiPage(client *trac.Client, wikiDir, pageName string, pageIndex, totalPages int, includeAttachments bool) error {
	wikiMeta, err := client.GetWikiPageInfo(pageName)
	if err != nil {
		return fmt.Errorf("failed to get wiki page info for %q: %w", pageName, err)
	}

	fmt.Printf("Exporting wiki page (%d/%d) %q ...\n", pageIndex+1, totalPages, pageName)
	for version := int64(1); version <= wikiMeta.Version; version++ {
		// fmt.Printf("Exporting wiki page (%d/%d) %q version %d/%d...\n", pageIndex+1, totalPages, pageName, version, wikiMeta.Version)
		content, err := client.GetWikiPageVersion(pageName, version)
		if err != nil {
			return fmt.Errorf("failed to get wiki page %q version %d: %w", pageName, version, err)
		}

		if content == nil {
			fmt.Printf("Warning: wiki page %q version %d is empty or not found\n", pageName, version)
			continue
		}

		filename := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.md", pageName, version))

		// Create directories for the wiki page
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
				return fmt.Errorf("failed to write content for wiki page %q version %d: %w (also failed to close file: %v)", pageName, version, err, cerr)
			}
			return fmt.Errorf("failed to write content for wiki page %q version %d: %w", pageName, version, err)
		}
		if cerr := file.Close(); cerr != nil {
			return fmt.Errorf("failed to close file for wiki page %q version %d: %w", pageName, version, cerr)
		}

		// Write metadata to file
		meta, err := client.GetWikiPageInfoVersion(pageName, version)
		if err != nil {
			return fmt.Errorf("failed to get wiki page info for %q version %d: %w", pageName, version, err)
		}

		metaFile := filepath.Join(wikiDir, fmt.Sprintf("%s.v%d.json", pageName, version))
		metaFileHandle, err := os.Create(metaFile)
		if err != nil {
			return fmt.Errorf("failed to create metadata file for wiki page %q version %d: %w", pageName, version, err)
		}
		if err := json.NewEncoder(metaFileHandle).Encode(meta); err != nil {
			if cerr := metaFileHandle.Close(); cerr != nil {
				return fmt.Errorf("failed to write metadata for wiki page %q version %d: %w (also failed to close file: %v)", pageName, version, err, cerr)
			}
			return fmt.Errorf("failed to write metadata for wiki page %q version %d: %w", pageName, version, err)
		}
		if cerr := metaFileHandle.Close(); cerr != nil {
			return fmt.Errorf("failed to close metadata file for wiki page %q version %d: %w", pageName, version, cerr)
		}
	}

	// Export attachments once per page
	if len(wikiMeta.Attachments) > 0 && includeAttachments {
		fmt.Printf("Exporting attachments for wiki page %s...\n", pageName)
		attachmentsDir := filepath.Join(wikiDir, "attachments", pageName)
		if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
			log.Printf("Warning: failed to create attachments directory for page %s: %v\n", pageName, err)
		} else {
			for _, att := range wikiMeta.Attachments {
				content, err := trac.GetAttachment(client, trac.ResourceWiki, pageName, att.Filename)
				if err != nil {
					log.Printf("Warning: failed to download attachment %q for page %s: %v\n", att.Filename, pageName, err)
					continue
				}
				safeFilename := filepath.Base(att.Filename)
				attPath := filepath.Join(attachmentsDir, safeFilename)

				if err := os.WriteFile(attPath, content, 0644); err != nil {
					log.Printf("Warning: failed to write attachment %q for wiki %s: %v\n", att.Filename, pageName, err)
				}
			}
		}
	}

	return nil
}
