package exporter

import (
	"fmt"
	"trac2gitlab/pkg/trac"
)


func ExportWiki(client *trac.Client, outDir string) error {
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

	return nil
}
