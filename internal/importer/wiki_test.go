package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bnidev/trac2gitlab/pkg/gitlab"
	"github.com/bnidev/trac2gitlab/pkg/trac"
)

func TestWikiBySlug(t *testing.T) {
	pages := []*gitlab.Wiki{
		{Slug: "local-dev-setup", Title: "local dev setup"},
		{Slug: "WikiStart", Title: "WikiStart"},
	}
	m := wikiBySlug(pages)
	if m["local-dev-setup"] == nil {
		t.Fatal("dashed page must match by slug, not the humanized title")
	}
	if m["wikistart"] == nil {
		t.Fatal("slug lookup must be case-insensitive")
	}
}

func TestWikiContent(t *testing.T) {
	meta := trac.WikiPage{
		Author:       "jdoe",
		Version:      3,
		LastModified: time.Date(2022, 11, 22, 0, 0, 0, 0, time.UTC),
	}
	got := wikiContent("hello\n", meta)
	for _, want := range []string{"hello", "Originally by jdoe", "on 2022-11-22", "Trac version 3"} {
		if !strings.Contains(got, want) {
			t.Errorf("wikiContent missing %q\n%s", want, got)
		}
	}
}

func TestWikiContentNoMetadata(t *testing.T) {
	if got := wikiContent("hello", trac.WikiPage{}); got != "hello" {
		t.Fatalf("wikiContent = %q, want unchanged", got)
	}
}

func TestCollectWikiPages(t *testing.T) {
	dir := t.TempDir()
	writeWiki := func(name string, version int64, lastModified time.Time) {
		meta := trac.WikiPage{Name: name, Version: version, Author: "jdoe", LastModified: lastModified}
		data, _ := json.Marshal(meta)
		base := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(base), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(base), filepath.Base(base)+".v"+strconv.FormatInt(version, 10)+".json"), data, 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(filepath.Dir(base), filepath.Base(base)+".v"+strconv.FormatInt(version, 10)+".md"), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	writeWiki("Page", 1, time.Time{})
	writeWiki("Page", 2, time.Time{})
	writeWiki("Sub/Other", 1, time.Time{})

	pages, err := CollectWikiPages(dir)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("pages = %+v", pages)
	}
	if pages["Page"].Meta.Version != 2 {
		t.Fatalf("Page version = %d, want 2", pages["Page"].Meta.Version)
	}
	if !strings.HasSuffix(pages["Page"].ContentPath, ".v2.md") {
		t.Fatalf("content path = %q", pages["Page"].ContentPath)
	}
	if pages["Sub/Other"].Meta.Version != 1 {
		t.Fatalf("Sub/Other = %+v", pages["Sub/Other"])
	}
}

func TestCollectWikiPagesMissingDir(t *testing.T) {
	pages, err := CollectWikiPages(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(pages) != 0 {
		t.Fatalf("pages = %+v", pages)
	}
}

func TestWikiAttachmentKey(t *testing.T) {
	a := wikiAttachmentKey("Page", "a.png")
	if a != wikiAttachmentKey("Page", "a.png") {
		t.Fatal("wikiAttachmentKey is not deterministic")
	}
	if a == wikiAttachmentKey("Other", "a.png") {
		t.Fatal("wikiAttachmentKey should include the page")
	}
}

func TestWikiAttachmentSection(t *testing.T) {
	attachments := []trac.Attachment{
		{Filename: "a.png", Author: "jdoe", Time: time.Date(2022, 11, 22, 0, 0, 0, 0, time.UTC)},
		{Filename: "b.pdf"},
	}
	index := map[string]wikiAttachmentRecord{
		wikiAttachmentKey("Page", "a.png"): {Page: "Page", Filename: "a.png", Markdown: "![a](/uploads/a.png)"},
		wikiAttachmentKey("Page", "b.pdf"): {Page: "Page", Filename: "b.pdf", URL: "/uploads/b.pdf"},
	}
	section := wikiAttachmentSection("Page", attachments, index)
	for _, want := range []string{"## Attachments", "![a](/uploads/a.png)", "[b.pdf](/uploads/b.pdf)", "uploaded by jdoe on 2022-11-22"} {
		if !strings.Contains(section, want) {
			t.Errorf("section missing %q\n%s", want, section)
		}
	}
}

func TestWikiAttachmentMappingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trac-wiki-attachments.json")
	in := &wikiAttachmentMapping{Attachments: []wikiAttachmentRecord{{
		Page: "Page", Filename: "a.png", URL: "/uploads/a.png", Markdown: "![a](/uploads/a.png)",
	}}}
	if err := saveWikiAttachmentMapping(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := loadWikiAttachmentMapping(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].Page != "Page" || out.Attachments[0].Markdown == "" {
		t.Fatalf("round trip = %+v", out.Attachments)
	}
}

func TestLoadMissingWikiAttachmentMapping(t *testing.T) {
	out, err := loadWikiAttachmentMapping(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(out.Attachments) != 0 {
		t.Fatalf("expected empty mapping, got %+v", out.Attachments)
	}
}
