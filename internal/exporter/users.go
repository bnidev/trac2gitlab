package exporter

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"trac2gitlab/internal/config"
	"trac2gitlab/pkg/trac"
)

func ExportUsers(client *trac.Client, config *config.Config) error {
	ids, err := client.GetAllTicketIDs("max=0")
	if err != nil {
		return fmt.Errorf("failed to get ticket IDs: %w", err)
	}

	slog.Info("Starting user export...", "count", len(ids))

	var users []string

	for _, id := range ids {

		ticket, err := client.GetTicket(id)
		if err != nil {
			return fmt.Errorf("failed to fetch ticket: %w", err)
		}

		for attribute := range ticket.Attributes {
			if attribute == "reporter" || attribute == "owner" {
				user, ok := ticket.Attributes[attribute]
				if !ok || user == nil {
					slog.Warn("Skipping empty user field", "field", attribute, "ticketID", id)
					continue
				}
				if userStr, ok := user.(string); ok && userStr != "" {
					if !slices.Contains(users, userStr) {
						users = append(users, userStr)
						slog.Debug("Found new user", "user", userStr, "ticketID", id)
					} else {
						slog.Debug("User already exists", "user", userStr, "ticketID", id)
					}
				} else {
					slog.Warn("Unexpected user type", "type", fmt.Sprintf("%T", user), "ticketID", id)
				}
			}
		}

	}

	if err := os.MkdirAll(config.ExportOptions.ExportDir, 0755); err != nil {
		return fmt.Errorf("failed to create tickets directory: %w", err)
	}
	usersFile := filepath.Join(config.ExportOptions.ExportDir, "users.txt")
	file, err := os.Create(usersFile)
	if err != nil {
		return fmt.Errorf("failed to create users file: %w", err)
	}
	for _, user := range users {
		if _, err := file.WriteString(user + "\n"); err != nil {
			return fmt.Errorf("failed to write user to file: %w", err)
		}
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("failed to close users file: %w", err)
	}

	slog.Info("User export completed", "count", len(users))
	return nil
}
