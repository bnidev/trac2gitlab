package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"trac2gitlab/pkg/trac"
)

// ExportTickets exports tickets from Trac and saves them as JSON files
func ExportTickets(client *trac.Client, outDir string, includeClosedTickets bool) error {
	fmt.Println("Exporting tickets...")

	query := "max=0"
	if !includeClosedTickets {
		query += "&status!=closed"
	}

	ids, err := client.GetAllTicketIDs(query)
	if err != nil {
		return fmt.Errorf("failed to get ticket IDs: %w", err)
	}

	ticketsDir := filepath.Join(outDir, "tickets")
	if err := os.MkdirAll(ticketsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tickets directory: %w", err)
	}

	fmt.Printf("Found %d ticket%s\n", len(ids), func() string {
		if len(ids) == 1 {
			return ""
		}
		return "s"
	}())

	for _, id := range ids {
		ticket, err := client.GetTicket(id)
		if err != nil {
			log.Printf("Warning: failed to fetch ticket #%d: %v\n", id, err)
			continue
		}

		filename := filepath.Join(ticketsDir, fmt.Sprintf("ticket-%d.json", id))
		file, err := os.Create(filename)
		if err != nil {
			log.Printf("Warning: failed to write ticket #%d: %v\n", id, err)
			continue
		}
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(ticket); err != nil {
			log.Printf("Warning: failed to encode ticket #%d: %v\n", id, err)
		}

		if cerr := file.Close(); cerr != nil {
			log.Fatalf("Failed to close config.yaml: %v", cerr)
		}
	}

	fmt.Println("Ticket export complete.")
	return nil
}
