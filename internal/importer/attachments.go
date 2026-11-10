package importer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

var (
	// attachment:"file" — a link to an attachment.
	reAttachmentRef = regexp.MustCompile(`attachment:"([^"]+)"`)
	// [[Image(file)]] / [[Image(file, width)]] — an inline image.
	reImageMacro = regexp.MustCompile(`\[\[Image\(\s*([^,)]+?)\s*(?:,[^)]*)?\)\]\]`)
)

type ticketWithAttachments struct {
	ID          int64             `json:"ID"`
	TimeChanged string            `json:"TimeChanged"`
	Attachments []trac.Attachment `json:"Attachments"`
}

type attachmentRecord struct {
	TicketID int64  `json:"ticket_id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Markdown string `json:"markdown"`
}

type attachmentMapping struct {
	Attachments []attachmentRecord `json:"attachments"`
}

func ImportAttachments(client *gitlab.Client, cfg *config.Config) (Result, error) {
	var result Result

	project, err := client.GetProject(cfg.GitLab.ProjectID)
	if err != nil {
		return result, err
	}

	mappingPath := filepath.Join(cfg.ExportOptions.ExportDir, "trac-attachments.json")
	mapping, err := loadAttachmentMapping(mappingPath)
	if err != nil {
		return result, err
	}
	index := make(map[string]attachmentRecord, len(mapping.Attachments))
	for _, record := range mapping.Attachments {
		index[attachmentKey(record.TicketID, record.Filename)] = record
	}

	tickets, err := utils.ReadFilesFromDir(filepath.Join(cfg.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return result, fmt.Errorf("read tickets: %w", err)
	}

	parsed := make([]ticketWithAttachments, 0, len(tickets))
	for _, data := range tickets {
		var ticket ticketWithAttachments
		if err := json.Unmarshal(data, &ticket); err != nil {
			return result, fmt.Errorf("parse ticket: %w", err)
		}
		parsed = append(parsed, ticket)
		result.Total += len(ticket.Attachments)
	}

	for _, ticket := range parsed {
		for _, attachment := range ticket.Attachments {
			key := attachmentKey(ticket.ID, attachment.Filename)
			if _, ok := index[key]; ok {
				result.Completed++
				continue
			}
			if cfg.DryRun {
				slog.Info("dry-run: would upload attachment", "ticket", ticket.ID, "filename", attachment.Filename)
				result.Completed++
				continue
			}

			filename := filepath.Base(attachment.Filename)
			path := filepath.Join(cfg.ExportOptions.ExportDir, "tickets", "attachments", fmt.Sprintf("%d", ticket.ID), filename)
			file, err := os.Open(path)
			if err != nil {
				result.fail(err)
				slog.Error("Skipping attachment", "ticket", ticket.ID, "filename", attachment.Filename, "errorMsg", err)
				continue
			}
			uploaded, err := client.UploadProjectMarkdown(project.ID, file, filename)
			closeErr := file.Close()
			if err != nil {
				result.fail(err)
				slog.Error("Failed to upload attachment", "ticket", ticket.ID, "filename", attachment.Filename, "errorMsg", err)
				continue
			}
			if closeErr != nil {
				result.fail(closeErr)
				slog.Error("Failed to close attachment", "ticket", ticket.ID, "filename", attachment.Filename, "errorMsg", closeErr)
				continue
			}

			record := attachmentRecord{
				TicketID: ticket.ID,
				Filename: attachment.Filename,
				URL:      uploaded.URL,
				Markdown: uploaded.Markdown,
			}
			mapping.Attachments = append(mapping.Attachments, record)
			index[key] = record
			if err := saveAttachmentMapping(mappingPath, mapping); err != nil {
				result.fail(err)
				return result, err
			}
			result.Completed++
		}

		section := attachmentSection(ticket, index)
		if section == "" {
			continue
		}
		if cfg.DryRun {
			slog.Info("dry-run: would link attachments", "ticket", ticket.ID)
			continue
		}

		issue, err := client.GetIssue(project.ID, ticket.ID)
		if err != nil {
			result.fail(err)
			slog.Error("Skipping attachment links", "ticket", ticket.ID, "errorMsg", err)
			continue
		}
		description := resolveAttachmentRefs(issue.Description, ticket, index) + section
		if description == issue.Description {
			continue
		}

		changedAt, err := time.Parse(time.RFC3339, ticket.TimeChanged)
		if err != nil {
			result.fail(err)
			slog.Error("Skipping attachment links", "ticket", ticket.ID, "errorMsg", err)
			continue
		}
		if _, err := client.UpdateIssue(project.ID, ticket.ID, &gitlab.UpdateIssueOptions{
			Description: &description,
			UpdatedAt:   &changedAt,
		}); err != nil {
			result.fail(err)
			slog.Error("Failed to link attachments", "ticket", ticket.ID, "errorMsg", err)
			continue
		}
	}

	return result, nil
}

func attachmentKey(ticketID int64, filename string) string {
	return fmt.Sprintf("%d|%s", ticketID, filename)
}

// resolveAttachmentRefs rewrites Trac's inline attachment references
// (attachment:"file" and [[Image(file, width)]]) to markdown using the uploaded
// attachment URLs. References without an upload are left unchanged.
func resolveAttachmentRefs(description string, ticket ticketWithAttachments, index map[string]attachmentRecord) string {
	if description == "" ||
		(!strings.Contains(description, "attachment:") && !strings.Contains(description, "[[Image(")) {
		return description
	}
	url := func(name string) (string, bool) {
		record, ok := index[attachmentKey(ticket.ID, name)]
		if !ok || record.URL == "" {
			return "", false
		}
		return record.URL, true
	}
	description = reAttachmentRef.ReplaceAllStringFunc(description, func(m string) string {
		name := reAttachmentRef.FindStringSubmatch(m)[1]
		if u, ok := url(name); ok {
			return "[" + name + "](" + u + ")"
		}
		return m
	})
	description = reImageMacro.ReplaceAllStringFunc(description, func(m string) string {
		name := reImageMacro.FindStringSubmatch(m)[1]
		if u, ok := url(name); ok {
			return "![" + name + "](" + u + ")"
		}
		return m
	})
	return description
}

func attachmentSection(ticket ticketWithAttachments, index map[string]attachmentRecord) string {
	var lines []string
	for _, attachment := range ticket.Attachments {
		record, ok := index[attachmentKey(ticket.ID, attachment.Filename)]
		if !ok {
			continue
		}
		link := record.Markdown
		if link == "" {
			link = fmt.Sprintf("[%s](%s)", record.Filename, record.URL)
		}
		line := "- " + link
		var meta []string
		if attachment.Author != "" {
			meta = append(meta, "uploaded by "+attachment.Author)
		}
		if !attachment.Time.IsZero() {
			meta = append(meta, "on "+attachment.Time.Format("2006-01-02"))
		}
		if len(meta) > 0 {
			line += " — _" + strings.Join(meta, " ") + "_"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n\n## Attachments\n\n" + strings.Join(lines, "\n")
}

func loadAttachmentMapping(path string) (*attachmentMapping, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &attachmentMapping{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read attachment mapping %q: %w", path, err)
	}
	var mapping attachmentMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, fmt.Errorf("parse attachment mapping %q: %w", path, err)
	}
	return &mapping, nil
}

func saveAttachmentMapping(path string, mapping *attachmentMapping) error {
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("encode attachment mapping: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create attachment mapping directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("write attachment mapping: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("replace attachment mapping: %w", err)
	}
	return nil
}
