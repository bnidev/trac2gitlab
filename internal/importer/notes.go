package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/identity"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

type ticketWithComments struct {
	ID       int64                 `json:"ID"`
	Comments []trac.ChangeLogEntry `json:"Comments"`
}

type noteRecord struct {
	Key       string    `json:"key"`
	TicketID  int64     `json:"ticket_id"`
	NoteID    int64     `json:"note_id"`
	Author    string    `json:"author"`
	AuthorID  int64     `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type noteMapping struct {
	Notes []noteRecord `json:"notes"`
}

func ImportNotes(client *gitlab.Client, cfg *config.Config) error {
	project, err := client.GetProject(cfg.GitLab.ProjectID)
	if err != nil {
		return err
	}

	usersFile := cfg.ImportOptions.UsersFile
	if usersFile == "" {
		usersFile = "mappings/users.yaml"
	}
	users, err := identity.LoadFile(usersFile)
	if err != nil {
		return fmt.Errorf("load identity mapping: %w", err)
	}
	resolver := identity.NewResolver(client, project.ID)

	notesPath := filepath.Join(cfg.ExportOptions.ExportDir, "trac-notes.json")
	noteMap, err := loadNoteMapping(notesPath)
	if err != nil {
		return err
	}
	existing := make(map[string]bool, len(noteMap.Notes))
	for _, record := range noteMap.Notes {
		existing[record.Key] = true
	}

	tickets, err := utils.ReadFilesFromDir(filepath.Join(cfg.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return fmt.Errorf("read tickets: %w", err)
	}

	for _, data := range tickets {
		var ticket ticketWithComments
		if err := json.Unmarshal(data, &ticket); err != nil {
			return fmt.Errorf("parse ticket: %w", err)
		}

		for i, comment := range ticket.Comments {
			if comment.OldValue == nil {
				continue
			}
			body := strings.TrimSpace(*comment.OldValue)
			if body == "" || isChangesetComment(body) {
				continue
			}
			if comment.Time.IsZero() {
				slog.Warn("Skipping comment with no timestamp", "ticket", ticket.ID, "index", i)
				continue
			}

			key := noteKey(ticket.ID, comment.Time, comment.Author, body)
			if existing[key] {
				continue
			}

			authorID := resolveAuthor(users, resolver, comment.Author)
			if authorID == 0 {
				body += fmt.Sprintf("\n\n---\n_Originally by %s at %s_", comment.Author, comment.Time.Format(time.RFC3339))
			}

			note, err := client.CreateIssueNote(project.ID, ticket.ID, &gitlab.CreateIssueNoteOptions{
				Body:      &body,
				CreatedAt: &comment.Time,
			})
			if err != nil {
				return fmt.Errorf("create note for ticket %d comment %d: %w", ticket.ID, i, err)
			}

			noteMap.Notes = append(noteMap.Notes, noteRecord{
				Key:       key,
				TicketID:  ticket.ID,
				NoteID:    note.ID,
				Author:    comment.Author,
				AuthorID:  authorID,
				CreatedAt: comment.Time,
			})
			existing[key] = true
			if err := saveNoteMapping(notesPath, noteMap); err != nil {
				return err
			}
		}
	}

	return nil
}

func resolveAuthor(users *identity.Mapping, resolver *identity.Resolver, author string) int64 {
	user, ok := users.Resolve(author)
	if !ok {
		slog.Warn("Unmapped note author", "author", author)
		return 0
	}
	if !user.Resolved() {
		slog.Warn("Unresolved note author", "author", author, "key", user.Key)
		return 0
	}
	id, err := resolver.Resolve(user.GitLab)
	if err != nil {
		slog.Warn("Failed to resolve GitLab user", "author", author, "gitlab", user.GitLab, "error", err)
		return 0
	}
	return id
}

func isChangesetComment(body string) bool {
	body = strings.TrimSpace(body)
	return strings.HasPrefix(body, "In [changeset:") ||
		strings.Contains(body, "#!CommitTicketReference") ||
		strings.Contains(body, `class="changeset"`)
}

func noteKey(ticketID int64, createdAt time.Time, author, body string) string {
	sum := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%d|%s|%s|%s", ticketID, createdAt.UTC().Format(time.RFC3339Nano), author, hex.EncodeToString(sum[:8]))
}

func loadNoteMapping(path string) (*noteMapping, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &noteMapping{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read note mapping %q: %w", path, err)
	}
	var mapping noteMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parse note mapping %q: %w", path, err)
	}
	return &mapping, nil
}

func saveNoteMapping(path string, mapping *noteMapping) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("encode note mapping: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create note mapping directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write note mapping: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace note mapping: %w", err)
	}
	return nil
}
