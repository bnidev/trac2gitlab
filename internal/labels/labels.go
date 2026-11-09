// Package labels maps Trac ticket fields to GitLab labels.
package labels

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

type Options struct {
	Enabled       bool
	Prefix        string
	Fields        []string
	CreateMissing bool
}

var defaultFields = []string{"type", "priority", "resolution", "component", "version", "severity", "status"}

func (o Options) Names(attrs map[string]string) []string {
	if !o.Enabled {
		return nil
	}
	prefix := o.Prefix
	if prefix == "" {
		prefix = "trac"
	}
	fields := o.Fields
	if len(fields) == 0 {
		fields = defaultFields
	}

	seen := make(map[string]bool)
	var names []string
	for _, field := range fields {
		value := strings.TrimSpace(attrs[field])
		if value == "" {
			continue
		}
		name := prefix + ":" + field + ":" + slug(value)
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// Ensure creates missing labels. It is idempotent.
func Ensure(client *gitlab.Client, projectID any, names []string) error {
	for _, name := range names {
		existing, err := client.GetLabelbyName(projectID, name)
		if err != nil {
			return fmt.Errorf("look up label %q: %w", name, err)
		}
		if existing != nil {
			continue
		}
		labelName := name
		if _, err := client.CreateLabel(projectID, &gitlab.CreateLabelOptions{Name: &labelName}); err != nil {
			return fmt.Errorf("create label %q: %w", name, err)
		}
	}
	return nil
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", ":", "-", "_", "-")
	value = replacer.Replace(value)
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if r == '-' && !lastDash {
			b.WriteRune(r)
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
