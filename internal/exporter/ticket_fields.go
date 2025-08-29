package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// ExportTicketField represents a ticket field with its name and options
type ExportTicketField struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

// ExportTicketFields exports ticket fields from Trac
func ExportTicketFields(client *trac.Client, config *config.Config) error {
	slog.Info("Starting ticket field export...")

	defaultFields := []string{"priority", "component", "type"}
	additionalFields := config.ExportOptions.AdditionalTicketFields

	fields, err := client.GetTicketFields()
	if err != nil {
		return fmt.Errorf("failed to get ticket fields: %w", err)
	}

	var exportFields []ExportTicketField

	processed := make(map[string]bool)

	for _, field := range fields {
		if (slices.Contains(defaultFields, field.Name) || slices.Contains(additionalFields, field.Name)) && !processed[field.Name] {
			slog.Debug("Field added", "fieldName", field.Name, "options", field.Options)
			appendedField := createExportTicketField(field)
			exportFields = append(exportFields, appendedField)
			processed[field.Name] = true
		}
	}

	if len(exportFields) == 0 {
		slog.Info("No default ticket fields found to export")
	}

	filename := filepath.Join(config.ExportOptions.ExportDir, "ticket-fields.json")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(exportFields); err != nil {
		return fmt.Errorf("failed to encode ticket fields to JSON: %w", err)
	}
	if cerr := file.Close(); cerr != nil {
		return fmt.Errorf("failed to close file: %w", cerr)
	}

	return nil
}

// createExportTicketField converts a trac.TicketField to an ExportTicketField
func createExportTicketField(field trac.TicketField) ExportTicketField {
	return ExportTicketField{
		Name:    field.Name,
		Options: field.Options,
	}
}
