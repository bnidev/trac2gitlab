package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"trac2gitlab/internal/config"
	"trac2gitlab/pkg/trac"
)

// ExportTicketField represents a ticket field with its name and options
type ExportTicketField struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

// ExportTicketFields exports ticket fields from Trac
func ExportTicketFields(client *trac.Client, config *config.Config) error {
	slog.Info("Starting ticket field export...")

	var defaultFields = []string{"priority", "component", "type"}

	fields, err := client.GetTicketFields()
	if err != nil {
		return fmt.Errorf("failed to get ticket fields: %w", err)
	}
	var exportFields []ExportTicketField
	for _, field := range fields {
		if slices.Contains(defaultFields, field.Name) {
			// add field.Name and field.Options to export struct
			slog.Debug("Default field found", "fieldName", field.Name, "options", field.Options)
			appendedField := createExportTicketField(field)
			exportFields = append(exportFields, appendedField)
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
