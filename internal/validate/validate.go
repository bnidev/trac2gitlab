// Package validate compares the Trac export with the GitLab target.
package validate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/identity"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/labels"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

const tolerance = 2 * time.Second

type Section struct {
	Expected int
	Found    int
	Matched  int
	Failed   int
	Details  []string
}

type Report struct {
	Issues      Section
	Notes       Section
	Attachments Section
	Wiki        Section
	Errors      []string
	Warnings    []string
}

func (r *Report) Verdict() string {
	if len(r.Errors) > 0 {
		return "FAIL"
	}
	if len(r.Warnings) > 0 {
		return "PASS WITH WARNINGS"
	}
	return "PASS"
}

func (r *Report) fail(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	r.Errors = append(r.Errors, msg)
}

func (r *Report) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// Markdown renders the report.
func (r *Report) Markdown(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Validation — %s\n\n", name)
	b.WriteString("| Area | Expected | Found | Matched | Failed |\n")
	b.WriteString("|---|---|---|---|---|\n")
	writeRow(&b, "Issues", r.Issues)
	writeRow(&b, "Notes", r.Notes)
	writeRow(&b, "Attachments", r.Attachments)
	writeRow(&b, "Wiki", r.Wiki)
	b.WriteString("\n")
	if len(r.Errors) > 0 {
		b.WriteString("## Errors\n\n")
		for _, msg := range r.Errors {
			fmt.Fprintf(&b, "- ✗ %s\n", msg)
		}
		b.WriteString("\n")
	}
	if len(r.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, msg := range r.Warnings {
			fmt.Fprintf(&b, "- ⚠ %s\n", msg)
		}
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "STATUS: %s\n", r.Verdict())
	return b.String()
}

func writeRow(b *strings.Builder, name string, section Section) {
	fmt.Fprintf(b, "| %s | %d | %d | %d | %d |\n", name, section.Expected, section.Found, section.Matched, section.Failed)
}

type noteRecord struct {
	NoteID    int64     `json:"note_id"`
	TicketID  int64     `json:"ticket_id"`
	Author    string    `json:"author"`
	AuthorID  int64     `json:"author_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type noteMap struct {
	Notes []noteRecord `json:"notes"`
}

type attachmentRecord struct {
	TicketID int64  `json:"ticket_id"`
	Page     string `json:"page"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Markdown string `json:"markdown"`
}

type attachmentMap struct {
	Attachments []attachmentRecord `json:"attachments"`
}

func Run(client *gitlab.Client, cfg *config.Config) (*Report, error) {
	report := &Report{}

	project, err := client.GetProject(cfg.GitLab.ProjectID)
	if err != nil {
		return nil, err
	}

	users, _ := identity.LoadFile(cfg.ImportOptions.UsersFile)
	resolver := identity.NewResolver(client, project.ID)
	labelOpts := labels.Options{
		Enabled: cfg.Labels.Enabled,
		Prefix:  cfg.Labels.Prefix,
		Fields:  cfg.Labels.Fields,
	}

	tickets, err := loadTickets(cfg)
	if err != nil {
		return nil, err
	}

	issues := make(map[int64]*gitlab.Issue)
	report.Issues.Expected = len(tickets)
	for _, ticket := range tickets {
		iid := int64(ticket.ID)
		issue, err := client.GetIssue(project.ID, iid)
		if err != nil {
			report.Issues.Failed++
			report.fail("issue #%d not found", iid)
			continue
		}
		issues[iid] = issue
		report.Issues.Found++
		report.Issues.Matched++

		checkIssueTimestamps(report, issue, ticket)
		checkIssueAuthor(report, users, resolver, issue, ticket)
		checkIssueLabels(report, labelOpts, issue, ticket)
	}

	checkNotes(report, client, cfg, project.ID, tickets)
	checkAttachments(report, client, cfg, project.ID, tickets, issues)
	checkWiki(report, client, cfg, project.ID)

	return report, nil
}

func checkIssueTimestamps(report *Report, issue *gitlab.Issue, ticket importer.IssueRaw) {
	created, err := time.Parse(time.RFC3339, ticket.TimeCreated)
	if err == nil && !closeEnough(issue.CreatedAt, created) {
		report.warn("issue #%d created_at drift: GitLab %v vs Trac %s", ticket.ID, formatTime(issue.CreatedAt), ticket.TimeCreated)
	}
	if strings.EqualFold(ticket.Attributes.Status, "closed") {
		changed, err := time.Parse(time.RFC3339, ticket.TimeChanged)
		if err == nil && !closeEnough(issue.ClosedAt, changed) {
			report.warn("issue #%d closed_at drift: GitLab %v vs Trac %s", ticket.ID, formatTime(issue.ClosedAt), ticket.TimeChanged)
		}
	}
}

func checkIssueAuthor(report *Report, users *identity.Mapping, resolver *identity.Resolver, issue *gitlab.Issue, ticket importer.IssueRaw) {
	if users == nil {
		return
	}
	expected := importer.ResolveAuthor(users, resolver, ticket.Attributes.Reporter)
	if expected == 0 {
		return
	}
	if issue.Author == nil || issue.Author.ID != expected {
		report.warn("issue #%d author not patched (want user %d)", ticket.ID, expected)
	}
}

func checkIssueLabels(report *Report, opts labels.Options, issue *gitlab.Issue, ticket importer.IssueRaw) {
	expected := opts.Names(map[string]string{
		"status":     ticket.Attributes.Status,
		"type":       ticket.Attributes.Type,
		"priority":   ticket.Attributes.Priority,
		"resolution": ticket.Attributes.Resolution,
		"component":  ticket.Attributes.Component,
		"version":    ticket.Attributes.Version,
		"severity":   ticket.Attributes.Severity,
	})
	for _, label := range expected {
		if !contains(issue.Labels, label) {
			report.warn("issue #%d missing label %q", ticket.ID, label)
		}
	}
}

func checkNotes(report *Report, client *gitlab.Client, cfg *config.Config, projectID any, tickets []importer.IssueRaw) {
	noteMap, err := loadNoteMap(cfg)
	if err != nil {
		report.fail("read note mapping: %v", err)
		return
	}
	report.Notes.Expected = countComments(tickets)
	report.Notes.Found = len(noteMap.Notes)

	for _, record := range noteMap.Notes {
		notes, err := client.ListIssueNotes(projectID, record.TicketID)
		if err != nil {
			report.Notes.Failed++
			report.fail("list notes for issue #%d: %v", record.TicketID, err)
			continue
		}
		var found *gitlab.Note
		for _, note := range notes {
			if note.ID == record.NoteID {
				found = note
				break
			}
		}
		if found == nil {
			report.Notes.Failed++
			report.fail("note %d on issue #%d not found", record.NoteID, record.TicketID)
			continue
		}
		report.Notes.Matched++
		if !closeEnough(found.CreatedAt, record.CreatedAt) {
			report.warn("note %d created_at drift: GitLab %v vs Trac %v", record.NoteID, formatTime(found.CreatedAt), record.CreatedAt)
		}
		if record.AuthorID > 0 && found.Author.ID != record.AuthorID {
			report.warn("note %d author not patched (want user %d)", record.NoteID, record.AuthorID)
		}
	}
}

func checkAttachments(report *Report, client *gitlab.Client, cfg *config.Config, projectID any, tickets []importer.IssueRaw, issues map[int64]*gitlab.Issue) {
	for _, ticket := range tickets {
		report.Attachments.Expected += len(ticket.Attachments)
	}
	attachmentMap, err := loadAttachmentMap(filepath.Join(cfg.ExportOptions.ExportDir, "trac-attachments.json"))
	if err != nil {
		report.fail("read attachment mapping: %v", err)
		return
	}
	report.Attachments.Found += len(attachmentMap.Attachments)
	for _, record := range attachmentMap.Attachments {
		issue := issues[record.TicketID]
		if issue == nil {
			report.Attachments.Failed++
			report.fail("attachment %q references missing issue #%d", record.Filename, record.TicketID)
			continue
		}
		if linkPresent(issue.Description, record) {
			report.Attachments.Matched++
		} else {
			report.warn("attachment %q not linked on issue #%d", record.Filename, record.TicketID)
		}
	}

	wikiMap, err := loadAttachmentMap(filepath.Join(cfg.ExportOptions.ExportDir, "trac-wiki-attachments.json"))
	if err != nil {
		report.fail("read wiki attachment mapping: %v", err)
		return
	}
	wikiExpected, _ := importer.CollectWikiPages(filepath.Join(cfg.ExportOptions.ExportDir, "wiki"))
	wikis, err := client.ListWikis(projectID, true)
	if err != nil {
		report.fail("list wiki pages: %v", err)
		return
	}
	contentBySlug := make(map[string]string, len(wikis))
	for _, page := range wikis {
		contentBySlug[strings.ToLower(page.Slug)] = page.Content
	}
	for name, file := range wikiExpected {
		if cfg.ExportOptions.WikiExcluded(name, file.Meta.Author) {
			continue
		}
		report.Attachments.Expected += len(file.Meta.Attachments)
	}
	report.Attachments.Found += len(wikiMap.Attachments)
	for _, record := range wikiMap.Attachments {
		if linkPresent(contentBySlug[strings.ToLower(record.Page)], record) {
			report.Attachments.Matched++
		} else {
			report.warn("wiki attachment %q not linked on %q", record.Filename, record.Page)
		}
	}
}

func checkWiki(report *Report, client *gitlab.Client, cfg *config.Config, projectID any) {
	pages, err := importer.CollectWikiPages(filepath.Join(cfg.ExportOptions.ExportDir, "wiki"))
	if err != nil {
		report.fail("scan wiki export: %v", err)
		return
	}
	wikis, err := client.ListWikis(projectID, false)
	if err != nil {
		report.fail("list wiki pages: %v", err)
		return
	}
	report.Wiki.Found = len(wikis)
	// Match by slug: GitLab humanizes the title but keeps the slug equal to the
	// Trac page name.
	bySlug := make(map[string]bool, len(wikis))
	for _, page := range wikis {
		bySlug[strings.ToLower(page.Slug)] = true
	}
	for name, page := range pages {
		if cfg.ExportOptions.WikiExcluded(name, page.Meta.Author) {
			continue
		}
		report.Wiki.Expected++
		if bySlug[strings.ToLower(name)] {
			report.Wiki.Matched++
		} else {
			report.Wiki.Failed++
			report.fail("wiki page %q not found", name)
		}
	}
}

func loadTickets(cfg *config.Config) ([]importer.IssueRaw, error) {
	files, err := utils.ReadFilesFromDir(filepath.Join(cfg.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return nil, err
	}
	tickets := make([]importer.IssueRaw, 0, len(files))
	for _, data := range files {
		var ticket importer.IssueRaw
		if err := json.Unmarshal(data, &ticket); err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}
	sort.Slice(tickets, func(i, j int) bool { return tickets[i].ID < tickets[j].ID })
	return tickets, nil
}

func loadNoteMap(cfg *config.Config) (*noteMap, error) {
	var mapping noteMap
	data, err := os.ReadFile(filepath.Join(cfg.ExportOptions.ExportDir, "trac-notes.json"))
	if os.IsNotExist(err) {
		return &mapping, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

func loadAttachmentMap(path string) (*attachmentMap, error) {
	var mapping attachmentMap
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &mapping, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

func countComments(tickets []importer.IssueRaw) int {
	count := 0
	for _, ticket := range tickets {
		for _, comment := range ticket.Comments {
			if comment.Time.IsZero() {
				continue
			}
			if _, ok := importer.CommentBody(comment); ok {
				count++
			}
		}
	}
	return count
}

func closeEnough(got *time.Time, want time.Time) bool {
	if got == nil {
		return false
	}
	diff := got.Sub(want)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "nil"
	}
	return t.UTC().Format(time.RFC3339)
}

func linkPresent(content string, record attachmentRecord) bool {
	if content == "" {
		return false
	}
	if record.Markdown != "" && strings.Contains(content, record.Markdown) {
		return true
	}
	return record.URL != "" && strings.Contains(content, record.URL)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
