package discover

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bnidev/trac2gitlab/internal/config"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

func TestRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tickets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "wiki"), 0755); err != nil {
		t.Fatal(err)
	}
	ticket := `{"ID":1,"Attributes":{"reporter":"jdoe","owner":"jane","description":"x","status":"closed"},"Comments":[{"Time":"2022-11-22T16:37:02Z","Author":"jdoe","Field":"comment","OldValue":"hi"}],"Attachments":[]}`
	if err := os.WriteFile(filepath.Join(dir, "tickets", "ticket-1.json"), []byte(ticket), 0644); err != nil {
		t.Fatal(err)
	}
	meta, _ := json.Marshal(trac.WikiPage{Name: "Page", Author: "jdoe", Version: 1, LastModified: time.Now()})
	if err := os.WriteFile(filepath.Join(dir, "wiki", "Page.v1.json"), meta, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "wiki", "Page.v1.md"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{ExportOptions: config.ExportOptions{ExportDir: dir}}
	result, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Tickets != 1 || result.Comments != 1 || result.WikiPages != 1 {
		t.Fatalf("counts = %+v", result)
	}
	if len(result.Authors) != 2 {
		t.Fatalf("authors = %+v", result.Authors)
	}
	var jdoe Author
	for _, a := range result.Authors {
		if a.Name == "jdoe" {
			jdoe = a
		}
	}
	if jdoe.Tickets != 1 || jdoe.Comments != 1 || jdoe.Wiki != 1 {
		t.Fatalf("jdoe = %+v", jdoe)
	}
}

func TestWriteUsers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mappings", "users.yaml")
	result := &Result{Authors: []Author{{Name: "John Doe"}}}
	if err := WriteUsers(path, result); err != nil {
		t.Fatalf("WriteUsers: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "action: unresolved") || !strings.Contains(string(data), "trac_username: John Doe") {
		t.Fatalf("users.yaml = %s", data)
	}
	if err := WriteUsers(path, result); err == nil {
		t.Fatal("WriteUsers should refuse to overwrite")
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"John Doe": "john-doe",
		"john.doe": "john.doe",
		"  ":       "unknown",
		"a / b":    "a-b",
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}
