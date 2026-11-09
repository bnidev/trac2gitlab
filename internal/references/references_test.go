package references

import (
	"path/filepath"
	"strings"
	"testing"
)

func testRefs() *Refs {
	return &Refs{
		enabled:     true,
		tickets:     map[int64]int64{1: 3, 3: 5},
		changesets:  map[string]string{"abc123456789": "def456789abc"},
		wiki:        map[string]string{"local-dev-setup": "local-dev-setup", "my page": "my-page"},
		baseURL:     "https://gitlab.example.com",
		projectPath: "group/project",
	}
}

func TestTicketRewrite(t *testing.T) {
	r := testRefs()
	got := r.Rewrite("#1 #3 ticket:1 [ticket:3] ## 1 #99")
	want := "#3 #5 #3 #5 ## 1 #99"
	if got != want {
		t.Fatalf("Rewrite = %q, want %q", got, want)
	}
}

func TestChangesetRewrite(t *testing.T) {
	r := testRefs()
	got := r.Rewrite("changeset:abc1234 and [abc1234] and [deadbeef]")
	if !strings.Contains(got, "[abc1234](https://gitlab.example.com/group/project/-/commit/def456789abc)") {
		t.Fatalf("changeset link missing: %q", got)
	}
	if strings.Count(got, "def456789abc") != 2 {
		t.Fatalf("expected two rewritten changesets: %q", got)
	}
	if !strings.Contains(got, "[deadbeef]") {
		t.Fatalf("unknown bracket hash should stay: %q", got)
	}
}

func TestWikiRewrite(t *testing.T) {
	r := testRefs()
	base := "https://gitlab.example.com/group/project/-/wikis/"
	cases := map[string]string{
		"[wiki:local-dev-setup Setup]": "[Setup](" + base + "local-dev-setup)",
		"[wiki:my page]":               "[my page](" + base + "my-page)",
		"wiki:local-dev-setup":         "[local-dev-setup](" + base + "local-dev-setup)",
		"[Guide](local-dev-setup.md)":  "[Guide](" + base + "local-dev-setup)",
	}
	for in, want := range cases {
		if got := r.Rewrite(in); got != want {
			t.Errorf("Rewrite(%q) = %q, want %q", in, got, want)
		}
	}
	if got := r.Rewrite("wiki:missing"); got != "wiki:missing" {
		t.Errorf("unknown wiki = %q", got)
	}
}

func TestSlug(t *testing.T) {
	if got := Slug("My Page / Sub_page"); got != "my-page-/-sub-page" {
		t.Fatalf("Slug = %q", got)
	}
}

func TestFencedCodeUntouched(t *testing.T) {
	r := testRefs()
	text := "before #1\n```\n#1\n```\nafter #1"
	got := r.Rewrite(text)
	if !strings.Contains(got, "```\n#1\n```") {
		t.Fatalf("code block was rewritten: %q", got)
	}
	if !strings.Contains(got, "before #3") || !strings.Contains(got, "after #3") {
		t.Fatalf("prose not rewritten: %q", got)
	}
}

func TestDisabled(t *testing.T) {
	r := &Refs{enabled: false, tickets: map[int64]int64{1: 3}}
	if got := r.Rewrite("#1"); got != "#1" {
		t.Fatalf("disabled rewrite = %q", got)
	}
}

func TestLoadMissingFiles(t *testing.T) {
	r, err := Load(Options{
		Enabled:        true,
		TicketsFile:    filepath.Join(t.TempDir(), "missing-tickets.json"),
		ChangesetsFile: filepath.Join(t.TempDir(), "missing-changesets.json"),
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := r.Rewrite("#1"); got != "#1" {
		t.Fatalf("missing mapping should not rewrite: %q", got)
	}
}
