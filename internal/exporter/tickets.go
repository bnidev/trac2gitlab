package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"trac2gitlab/pkg/trac"
)

// ExportTickets exports tickets from Trac and saves them as JSON files
func ExportTickets(client *trac.Client, outDir string, includeClosedTickets bool, includeAttachments bool) error {
	slog.Info("Starting ticket export...")

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

	slog.Debug("Tickets found", "count", len(ids))

	const maxWorkers = 10
	var wg sync.WaitGroup
	ticketChan := make(chan int, len(ids))

	for range make([]struct{}, maxWorkers) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range ticketChan {
				if err := exportSingleTicket(client, ticketsDir, id, includeAttachments); err != nil {
					slog.Error("Failed to export ticket", "ticketID", id, "error", err)
				}
			}
		}()
	}

	for _, id := range ids {
		ticketChan <- id
	}
	close(ticketChan)

	wg.Wait()

	slog.Info("Ticket export completed", "count", len(ids))
	return nil
}

// exportSingleTicket exports a single ticket and its attachments
func exportSingleTicket(client *trac.Client, ticketsDir string, id int, includeAttachments bool) error {
	slog.Debug("Exporting ticket", "ticketID", id)
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
		slog.Warn("Failed to encode ticket", "ticketID", id, "error", err)
	}
	if cerr := file.Close(); cerr != nil {
		slog.Warn("Failed to close ticket file", "ticketID", id, "error", cerr)
	}

	// Download attachments
	if includeAttachments && len(ticket.Attachments) > 0 {
		attachmentsDir := filepath.Join(ticketsDir, "attachments", fmt.Sprintf("%d", id))
		if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
			return fmt.Errorf("failed to create attachments directory for ticket #%d: %w", id, err)
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
					attErrs <- fmt.Errorf("failed to download attachment %q for ticket #%d: %w", att.Filename, id, err)
					return
				}

				safeFilename := filepath.Base(att.Filename)
				attPath := filepath.Join(attachmentsDir, safeFilename)

				if err := os.WriteFile(attPath, content, 0644); err != nil {
					attErrs <- fmt.Errorf("failed to write attachment %q for ticket #%d: %w", att.Filename, id, err)
					return
				}
			}()
		}

		attWg.Wait()
		close(attErrs)

		var errCount int
		for err := range attErrs {
			slog.Warn("Attachment error", "error", err)
			errCount++
		}

		if errCount > 0 {
			return fmt.Errorf("%d attachment(s) failed for ticket #%d", errCount, id)
		}
	}

	return nil
}
