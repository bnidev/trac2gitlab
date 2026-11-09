package importer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/identity"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

// AuthorPatchRow is one issue/note author reassignment.
type AuthorPatchRow struct {
	Kind     string `json:"kind"`
	IID      int64  `json:"iid,omitempty"`
	NoteID   int64  `json:"note_id,omitempty"`
	AuthorID int64  `json:"author_id"`
}

// BuildAuthorPatch resolves the author IDs recorded during import.
func BuildAuthorPatch(cfg *config.Config, client *gitlab.Client) ([]AuthorPatchRow, []string, error) {
	users, err := identity.LoadFile(cfg.ImportOptions.UsersFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load identity mapping: %w", err)
	}
	resolver := identity.NewResolver(client, cfg.GitLab.ProjectID)

	var rows []AuthorPatchRow
	var unresolved []string

	notesPath := filepath.Join(cfg.ExportOptions.ExportDir, "trac-notes.json")
	noteMap, err := loadNoteMapping(notesPath)
	if err != nil {
		return nil, nil, err
	}
	for _, record := range noteMap.Notes {
		authorID := ResolveAuthor(users, resolver, record.Author)
		if authorID == 0 {
			authorID = record.AuthorID
		}
		if authorID == 0 {
			unresolved = append(unresolved, fmt.Sprintf("note #%d (%s)", record.NoteID, record.Author))
			continue
		}
		rows = append(rows, AuthorPatchRow{Kind: "note", NoteID: record.NoteID, AuthorID: authorID})
	}

	tickets, err := utils.ReadFilesFromDir(filepath.Join(cfg.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return nil, nil, fmt.Errorf("read tickets: %w", err)
	}
	for _, data := range tickets {
		var ticket IssueRaw
		if err := json.Unmarshal(data, &ticket); err != nil {
			return nil, nil, fmt.Errorf("parse ticket: %w", err)
		}
		if strings.TrimSpace(ticket.Attributes.Reporter) == "" {
			continue
		}
		authorID := ResolveAuthor(users, resolver, ticket.Attributes.Reporter)
		if authorID == 0 {
			unresolved = append(unresolved, fmt.Sprintf("issue #%d (%s)", ticket.ID, ticket.Attributes.Reporter))
			continue
		}
		rows = append(rows, AuthorPatchRow{Kind: "issue", IID: int64(ticket.ID), AuthorID: authorID})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		if rows[i].Kind == "issue" {
			return rows[i].IID < rows[j].IID
		}
		return rows[i].NoteID < rows[j].NoteID
	})

	return rows, unresolved, nil
}

// RenderAuthorPatch returns the Ruby script that applies the rows.
func RenderAuthorPatch(rows []AuthorPatchRow, projectID int, apply bool) string {
	data, _ := json.Marshal(rows)
	dryRun := "true"
	if apply {
		dryRun = "false"
	}
	return fmt.Sprintf(`require "json"
DRY_RUN = %s
PROJECT_ID = %d
ROWS = JSON.parse(%%q(%s))
changed = 0
ROWS.each do |row|
  if row["kind"] == "issue"
    record = Issue.find_by(project_id: PROJECT_ID, iid: row["iid"])
    label = "issue ##{row["iid"]}"
  else
    record = Note.find_by(id: row["note_id"])
    label = "note ##{row["note_id"]}"
  end
  unless record
    puts "skip #{label}: not found"
    next
  end
  if record.author_id == row["author_id"]
    puts "ok   #{label}: author_id already #{row["author_id"]}"
    next
  end
  puts "set  #{label}: author_id #{record.author_id} -> #{row["author_id"]}"
  changed += 1
  record.update_column(:author_id, row["author_id"]) unless DRY_RUN
end
puts "#{DRY_RUN ? "DRY RUN" : "APPLIED"}: #{changed} record(s)"
`, dryRun, projectID, data)
}

// RenderShell wraps the Ruby script in a copy-pasteable shell block.
func RenderShell(ruby, container string, omnibus bool) string {
	inner := fmt.Sprintf("cat > /tmp/patch_authors.rb <<'RUBY'\n%s\nRUBY\ngitlab-rails runner /tmp/patch_authors.rb\nrm -f /tmp/patch_authors.rb\n", ruby)
	if omnibus {
		return "bash -s <<'SCRIPT'\n" + inner + "SCRIPT\n"
	}
	return fmt.Sprintf("docker exec -i %s bash -s <<'SCRIPT'\n%sSCRIPT\n", container, inner)
}
