package importer

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bnidev/trac2gitlab/pkg/trac"
)

func TestAttachmentKey(t *testing.T) {
	a := attachmentKey(1, "a.png")
	if a != attachmentKey(1, "a.png") {
		t.Fatal("attachmentKey is not deterministic")
	}
	if a == attachmentKey(2, "a.png") {
		t.Fatal("attachmentKey should include the ticket")
	}
}

func TestAttachmentSection(t *testing.T) {
	ticket := ticketWithAttachments{
		ID: 1,
		Attachments: []trac.Attachment{
			{Filename: "a.png", Author: "jdoe", Time: time.Date(2022, 11, 22, 0, 0, 0, 0, time.UTC)},
			{Filename: "b.pdf"},
		},
	}
	index := map[string]attachmentRecord{
		attachmentKey(1, "a.png"): {TicketID: 1, Filename: "a.png", Markdown: "![a](/uploads/a.png)"},
		attachmentKey(1, "b.pdf"): {TicketID: 1, Filename: "b.pdf", URL: "/uploads/b.pdf"},
	}
	section := attachmentSection(ticket, index, nil)
	for _, want := range []string{"## Attachments", "![a](/uploads/a.png)", "[b.pdf](/uploads/b.pdf)", "uploaded by jdoe on 2022-11-22"} {
		if !strings.Contains(section, want) {
			t.Errorf("section missing %q\n%s", want, section)
		}
	}
}

func TestAttachmentSectionEmpty(t *testing.T) {
	ticket := ticketWithAttachments{ID: 1, Attachments: []trac.Attachment{{Filename: "a.png"}}}
	if got := attachmentSection(ticket, map[string]attachmentRecord{}, nil); got != "" {
		t.Fatalf("expected empty section, got %q", got)
	}
}

func TestReferencedAttachments(t *testing.T) {
	refs := referencedAttachments(`see attachment:"a.png" and [[Image(b.png, 400)]] plus [[Image(c.png)]]`)
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		if !refs[name] {
			t.Errorf("referencedAttachments missing %q: %v", name, refs)
		}
	}
	if len(refs) != 3 {
		t.Fatalf("referencedAttachments = %v", refs)
	}
}

func TestAttachmentSectionSkipsReferenced(t *testing.T) {
	ticket := ticketWithAttachments{ID: 1, Attachments: []trac.Attachment{{Filename: "a.png"}, {Filename: "b.pdf"}}}
	index := map[string]attachmentRecord{
		attachmentKey(1, "a.png"): {TicketID: 1, Filename: "a.png", URL: "/uploads/a.png"},
		attachmentKey(1, "b.pdf"): {TicketID: 1, Filename: "b.pdf", URL: "/uploads/b.pdf"},
	}
	section := attachmentSection(ticket, index, map[string]bool{"a.png": true})
	if strings.Contains(section, "a.png") {
		t.Errorf("inline-referenced a.png should not be listed again: %s", section)
	}
	if !strings.Contains(section, "b.pdf") {
		t.Errorf("unreferenced b.pdf should still be listed: %s", section)
	}
}

func TestAttachmentMappingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trac-attachments.json")
	in := &attachmentMapping{Attachments: []attachmentRecord{{
		TicketID: 1, Filename: "a.png", URL: "/uploads/a.png", Markdown: "![a](/uploads/a.png)",
	}}}
	if err := saveAttachmentMapping(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := loadAttachmentMapping(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out.Attachments) != 1 || out.Attachments[0].TicketID != 1 || out.Attachments[0].Markdown == "" {
		t.Fatalf("round trip = %+v", out.Attachments)
	}
}

func TestLoadMissingAttachmentMapping(t *testing.T) {
	out, err := loadAttachmentMapping(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(out.Attachments) != 0 {
		t.Fatalf("expected empty mapping, got %+v", out.Attachments)
	}
}

func TestResolveAttachmentRefs(t *testing.T) {
	ticket := ticketWithAttachments{
		ID:          7,
		Attachments: []trac.Attachment{{Filename: "sketch.png"}, {Filename: "shot.jpg"}},
	}
	index := map[string]attachmentRecord{
		attachmentKey(7, "sketch.png"): {TicketID: 7, Filename: "sketch.png", URL: "https://gl/uploads/1"},
		attachmentKey(7, "shot.jpg"):   {TicketID: 7, Filename: "shot.jpg", URL: "https://gl/uploads/2"},
	}
	in := `see attachment:"sketch.png" and [[Image(shot.jpg, 400)]] but not attachment:"nope.txt"`
	want := `see [sketch.png](https://gl/uploads/1) and ![shot.jpg](https://gl/uploads/2) but not attachment:"nope.txt"`
	if got := resolveAttachmentRefs(in, ticket, index); got != want {
		t.Errorf("resolveAttachmentRefs:\n got %q\nwant %q", got, want)
	}
	if got := resolveAttachmentRefs(want, ticket, index); got != want {
		t.Errorf("resolveAttachmentRefs not idempotent: %q", got)
	}
}
