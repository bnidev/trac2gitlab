// Package identity maps Trac/Mercurial authors to GitLab users.
package identity

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// User is one entry in mappings/users.yaml.
type User struct {
	Key             string `yaml:"-"`
	TracUsername    string `yaml:"trac_username"`
	TracDisplayName string `yaml:"trac_display_name"`
	TracEmail       string `yaml:"trac_email"`
	HgAuthor        string `yaml:"hg_author"`
	GitLab          string `yaml:"gitlab"`
	Confidence      string `yaml:"confidence"`
	Action          string `yaml:"action"`
}

// Resolved reports whether this entry should be attributed in GitLab.
func (u User) Resolved() bool {
	return strings.TrimSpace(u.GitLab) != "" && u.Action != "unresolved"
}

// Aliases returns the non-empty author strings this entry matches.
func (u User) Aliases() []string {
	return nonEmpty(u.TracUsername, u.TracDisplayName, u.TracEmail, u.HgAuthor)
}

// Mapping is an alias index over mappings/users.yaml.
type Mapping struct {
	users   map[string]User
	aliases map[string]string // normalized alias -> entry key
}

type mappingFile struct {
	Users map[string]User `yaml:"users"`
}

// LoadFile reads and validates a users.yaml file.
func LoadFile(path string) (*Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read user mapping %q: %w", path, err)
	}
	return Parse(data)
}

// Parse validates and indexes a users.yaml document.
func Parse(data []byte) (*Mapping, error) {
	var f mappingFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse user mapping: %w", err)
	}
	if len(f.Users) == 0 {
		return nil, fmt.Errorf("user mapping has no users")
	}

	m := &Mapping{users: make(map[string]User, len(f.Users)), aliases: make(map[string]string)}
	for key, user := range f.Users {
		user.Key = key
		if strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("user mapping contains an entry with an empty key")
		}
		aliases := user.Aliases()
		if len(aliases) == 0 {
			return nil, fmt.Errorf("user %q has no Trac/HG alias", key)
		}
		if !user.Resolved() && user.Action != "unresolved" {
			return nil, fmt.Errorf("user %q has no gitlab username (set gitlab or action: unresolved)", key)
		}
		for _, alias := range aliases {
			norm := normalize(alias)
			if existing, ok := m.aliases[norm]; ok && existing != key {
				return nil, fmt.Errorf("alias %q is mapped to both %q and %q", alias, existing, key)
			}
			m.aliases[norm] = key
		}
		m.users[key] = user
	}
	return m, nil
}

// Resolve finds the entry whose alias matches author.
func (m *Mapping) Resolve(author string) (User, bool) {
	key, ok := m.aliases[normalize(author)]
	if !ok {
		return User{}, false
	}
	return m.users[key], true
}

// Users returns all entries sorted by key.
func (m *Mapping) Users() []User {
	out := make([]User, 0, len(m.users))
	for _, user := range m.users {
		out = append(out, user)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Unresolved returns entries that cannot be attributed in GitLab.
func (m *Mapping) Unresolved() []User {
	var out []User
	for _, user := range m.Users() {
		if !user.Resolved() {
			out = append(out, user)
		}
	}
	return out
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
