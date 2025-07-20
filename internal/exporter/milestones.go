package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"trac2gitlab/pkg/trac"
)

// ExportMilestones exports milestones from Trac and saves them as JSON files
func ExportMilestones(client *trac.Client, outDir string) error {
	fmt.Println("Exporting milestones...")

	milestoneNames, err := client.GetMilestoneNames()
	if err != nil {
		return fmt.Errorf("failed to get milestone names: %w", err)
	}

	milestonesDir := filepath.Join(outDir, "milestones")
	if err := os.MkdirAll(milestonesDir, 0755); err != nil {
		return fmt.Errorf("failed to create milestones directory: %w", err)
	}

	fmt.Printf("Found %d milestone%s\n", len(milestoneNames), func() string {
		if len(milestoneNames) == 1 {
			return ""
		}
		return "s"
	}())

	for _, name := range milestoneNames {
		milestone, err := client.GetMilestoneByName(name)
		if err != nil {
			log.Printf("Warning: failed to fetch milestone %q: %v\n", name, err)
			continue
		}

		filename := filepath.Join(milestonesDir, fmt.Sprintf("milestone-%s.json", name))
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Warning: failed to write milestone %q: %v\n", name, err)
			continue
		}

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(milestone); err != nil {
			log.Printf("Warning: failed to encode milestone %q: %v\n", name, err)
		}

		if cerr := file.Close(); cerr != nil {
			log.Fatalf("Failed to close milestone file %q: %v", name, cerr)
		}
	}

	fmt.Println("Milestone export complete.")
	return nil
}
