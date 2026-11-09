// Package discover summarises an exported Trac project and its authors.
package discover

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/internal/importer"
	"github.com/bnidev/trac2gitlab/internal/utils"
	"gopkg.in/yaml.v3"
)

type Author struct {
	Name        string `json:"name"`
	Tickets     int    `json:"tickets"`
	Owned       int    `json:"owned"`
	Comments    int    `json:"comments"`
	Wiki        int    `json:"wiki"`
	Attachments int    `json:"attachments"`
}

type Result struct {
	Tickets     int      `json:"tickets"`
	Comments    int      `json:"comments"`
	WikiPages   int      `json:"wiki_pages"`
	Attachments int      `json:"attachments"`
	Authors     []Author `json:"authors"`
}

func Run(cfg *config.Config) (*Result, error) {
	result := &Result{}
	authors := make(map[string]*Author)
	add := func(name string, apply func(*Author)) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		a, ok := authors[name]
		if !ok {
			a = &Author{Name: name}
			authors[name] = a
		}
		apply(a)
	}

	files, err := utils.ReadFilesFromDir(filepath.Join(cfg.ExportOptions.ExportDir, "tickets"), ".json", os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read tickets: %w", err)
	}
	for _, data := range files {
		var ticket importer.IssueRaw
		if err := json.Unmarshal(data, &ticket); err != nil {
			return nil, fmt.Errorf("parse ticket: %w", err)
		}
		result.Tickets++
		add(ticket.Attributes.Reporter, func(a *Author) { a.Tickets++ })
		add(ticket.Attributes.Owner, func(a *Author) { a.Owned++ })
		result.Attachments += len(ticket.Attachments)
		for _, attachment := range ticket.Attachments {
			add(attachment.Author, func(a *Author) { a.Attachments++ })
		}
		for _, comment := range ticket.Comments {
			if comment.OldValue == nil || comment.Time.IsZero() {
				continue
			}
			body := strings.TrimSpace(*comment.OldValue)
			if body == "" || importer.IsChangesetComment(body) {
				continue
			}
			result.Comments++
			add(comment.Author, func(a *Author) { a.Comments++ })
		}
	}

	pages, err := importer.CollectWikiPages(filepath.Join(cfg.ExportOptions.ExportDir, "wiki"))
	if err != nil {
		return nil, fmt.Errorf("read wiki export: %w", err)
	}
	result.WikiPages = len(pages)
	for _, page := range pages {
		add(page.Meta.Author, func(a *Author) { a.Wiki++ })
		result.Attachments += len(page.Meta.Attachments)
		for _, attachment := range page.Meta.Attachments {
			add(attachment.Author, func(a *Author) { a.Attachments++ })
		}
	}

	for _, author := range authors {
		result.Authors = append(result.Authors, *author)
	}
	sort.Slice(result.Authors, func(i, j int) bool {
		return strings.ToLower(result.Authors[i].Name) < strings.ToLower(result.Authors[j].Name)
	})
	return result, nil
}

type userEntry struct {
	TracUsername string `yaml:"trac_username,omitempty"`
	TracEmail    string `yaml:"trac_email,omitempty"`
	Action       string `yaml:"action"`
	Confidence   string `yaml:"confidence"`
}

type usersFile struct {
	Users map[string]userEntry `yaml:"users"`
}

// WriteUsers writes a users.yaml skeleton with every author unresolved.
func WriteUsers(path string, result *Result) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists; edit it instead of regenerating", path)
	}
	users := make(map[string]userEntry, len(result.Authors))
	for _, author := range result.Authors {
		key := slug(author.Name)
		for i := 2; ; i++ {
			if _, ok := users[key]; !ok {
				break
			}
			key = fmt.Sprintf("%s-%d", slug(author.Name), i)
		}
		entry := userEntry{Action: "unresolved", Confidence: "manual"}
		if strings.Contains(author.Name, "@") {
			entry.TracEmail = author.Name
		} else {
			entry.TracUsername = author.Name
		}
		users[key] = entry
	}
	data, err := yaml.Marshal(usersFile{Users: users})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func slug(name string) string {
	value := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '@' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '-' || r == ' ' || r == '_' || r == '/' {
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}
