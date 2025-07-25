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

// ExportTickets exports tickets from Trac and saves them as JSON files
func ExportTickets(client *trac.Client, outDir string, includeClosedTickets bool, includeAttachments bool) error {
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

	const maxWorkers = 10
	var wg sync.WaitGroup
	ticketChan := make(chan int, len(ids))

	for range make([]struct{}, maxWorkers) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range ticketChan {
				if err := exportSingleTicket(client, ticketsDir, id, includeAttachments); err != nil {
					log.Printf("Warning: failed to export ticket #%d: %v", id, err)
				}
			}
		}()
	}

	for _, id := range ids {
		ticketChan <- id
	}
	close(ticketChan)

	wg.Wait()

	fmt.Println("Ticket export complete.")
	return nil
}

// exportSingleTicket exports a single ticket and its attachments
func exportSingleTicket(client *trac.Client, ticketsDir string, id int, includeAttachments bool) error {
	ticket, err := client.GetTicket(id)
	if err != nil {
		return fmt.Errorf("failed to fetch ticket: %w", err)
	}

	// Write ticket JSON
	filename := filepath.Join(ticketsDir, fmt.Sprintf("ticket-%d.json", id))
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(ticket); err != nil {
		log.Printf("Warning: failed to encode ticket #%d: %v", id, err)
	}
	if cerr := file.Close(); cerr != nil {
		log.Printf("Warning: failed to close file for ticket #%d: %v", id, cerr)
	}

	// Download attachments
	if includeAttachments && len(ticket.Attachments) > 0 {
		attachmentsDir := filepath.Join(ticketsDir, "attachments", fmt.Sprintf("%d", id))
		if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
			log.Printf("Warning: failed to create attachments directory for ticket #%d: %v", id, err)
			return nil
		}

		var attWg sync.WaitGroup
		attErrs := make(chan error, len(ticket.Attachments))

		for _, att := range ticket.Attachments {
			att := att
			attWg.Add(1)
			go func() {
				defer attWg.Done()

				content, err := trac.GetAttachment(client, trac.ResourceTicket, id, att.Filename)
				if err != nil {
					log.Printf("Warning: failed to download attachment %q for ticket #%d: %v\n", att.Filename, id, err)
					attErrs <- err
					return
				}

				safeFilename := filepath.Base(att.Filename)
				attPath := filepath.Join(attachmentsDir, safeFilename)

				if err := os.WriteFile(attPath, content, 0644); err != nil {
					log.Printf("Warning: failed to write attachment %q for ticket #%d: %v\n", att.Filename, id, err)
					attErrs <- err
				}
			}()
		}

		attWg.Wait()
		close(attErrs)

		if len(attErrs) > 0 {
			return fmt.Errorf("some attachments failed for ticket #%d", id)
		}
	}

	return nil
}
