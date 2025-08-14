package exporter

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

// ExportUsers exports unique users from Trac tickets and saves them to a file
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

		for _, comment := range ticket.Comments {
			if comment.Author == "" {
				slog.Debug("Skipping empty comment author", "ticketID", id)
				continue
			}
			if !slices.Contains(users, comment.Author) {
				users = append(users, comment.Author)
				slog.Debug("Found new user in comment", "user", comment.Author, "ticketID", id)
			} else {
				slog.Debug("User already exists in comments", "user", comment.Author, "ticketID", id)
			}
		}

		for _, attachment := range ticket.Attachments {
			if attachment.Author == "" {
				slog.Debug("Skipping empty attachment author", "ticketID", id)
				continue
			}
			if !slices.Contains(users, attachment.Author) {
				users = append(users, attachment.Author)
				slog.Debug("Found new user in attachment", "user", attachment.Author, "ticketID", id)
			} else {
				slog.Debug("User already exists in attachments", "user", attachment.Author, "ticketID", id)
			}
		}

		for _, history := range ticket.History {
			if history.Author == "" {
				slog.Debug("Skipping empty history author", "ticketID", id)
				continue
			}
			if !slices.Contains(users, history.Author) {
				users = append(users, history.Author)
				slog.Debug("Found new user in history", "user", history.Author, "ticketID", id)
			} else {
				slog.Debug("User already exists in history", "user", history.Author, "ticketID", id)
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
