package config

import "testing"

func TestWikiExcluded(t *testing.T) {
	opts := ExportOptions{
		WikiExclude:        []string{"Trac*", "SandBox"},
		WikiExcludeAuthors: []string{"trac"},
	}
	cases := []struct {
		name   string
		author string
		want   bool
	}{
		{"TracGuide", "alice", true},      // glob match
		{"SandBox", "alice", true},        // exact-name match
		{"WikiStart", "trac", true},       // author match
		{"WikiStart", "TRAC", true},       // author match is case-insensitive
		{"WikiStart", "alice", false},     // real author, name not excluded
		{"local-dev-setup", "trac", true}, // author match
		{"local-dev-setup", "bob", false},
		{"TitleIndex", "bob", false}, // neither rule
	}
	for _, tc := range cases {
		if got := opts.WikiExcluded(tc.name, tc.author); got != tc.want {
			t.Errorf("WikiExcluded(%q, %q) = %v, want %v", tc.name, tc.author, got, tc.want)
		}
	}
}

func TestWikiExcludedEmptyConfigExcludesNothing(t *testing.T) {
	if (ExportOptions{}).WikiExcluded("TracGuide", "trac") {
		t.Fatal("empty config must not exclude any page")
	}
}
