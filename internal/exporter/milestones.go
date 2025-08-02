package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"trac2gitlab/internal/config"
	"trac2gitlab/pkg/trac"
)

// ExportMilestones exports milestones from Trac and saves them as JSON files
func ExportMilestones(client *trac.Client, config *config.Config) error {
	slog.Info("Starting milestone export...")

	milestoneNames, err := client.GetMilestoneNames()
	if err != nil {
		return fmt.Errorf("failed to get milestone names: %w", err)
	}

	milestonesDir := filepath.Join(config.ExportOptions.ExportDir, "milestones")
	if err := os.MkdirAll(milestonesDir, 0755); err != nil {
		return fmt.Errorf("failed to create milestones directory: %w", err)
	}

	slog.Debug("Milestones found", "count", len(milestoneNames))

	for _, name := range milestoneNames {
		milestone, err := client.GetMilestoneByName(name)
		if err != nil {
			slog.Warn("Failed to fetch milestone", "name", name, "error", err)
			continue
		}

		filename := filepath.Join(milestonesDir, fmt.Sprintf("milestone-%s.json", name))
		file, err := os.Create(filename)
		if err != nil {
			slog.Warn("Failed to create milestone file", "name", name, "error", err)
			continue
		}

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(milestone); err != nil {
			slog.Warn("Failed to encode milestone", "name", name, "error", err)
		}

		if cerr := file.Close(); cerr != nil {
			slog.Warn("File closed with error", "name", name, "error", cerr)
		}
	}

	slog.Info("Milestone export completed", "count", len(milestoneNames))
	return nil
}
