package importer

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestIsChangesetComment(t *testing.T) {
	cases := map[string]bool{
		"In [changeset:abc123] by someone":           true,
		"See #!CommitTicketReference repository=...": true,
		`<a class="changeset" title="x">abc</a>`:     true,
		"Thanks, this is fixed now":                  false,
		"":                                           false,
	}
	for body, want := range cases {
		if got := IsChangesetComment(body); got != want {
			t.Errorf("IsChangesetComment(%q) = %v, want %v", body, got, want)
		}
	}
}

func TestNoteKey(t *testing.T) {
	at := time.Date(2022, 11, 22, 16, 37, 2, 0, time.UTC)
	a := noteKey(1, at, "jdoe", "hello")
	if a != noteKey(1, at, "jdoe", "hello") {
		t.Fatal("noteKey is not deterministic")
	}
	if a == noteKey(1, at, "jdoe", "hello ") {
		t.Fatal("noteKey should change with body")
	}
	if a == noteKey(2, at, "jdoe", "hello") {
		t.Fatal("noteKey should change with ticket")
	}
}

func TestNoteMappingRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trac-notes.json")
	in := &noteMapping{Notes: []noteRecord{{
		Key:       "1|2022-11-22T16:37:02Z|jdoe|abcd",
		TicketID:  1,
		NoteID:    42,
		Author:    "jdoe",
		AuthorID:  7,
		CreatedAt: time.Date(2022, 11, 22, 16, 37, 2, 0, time.UTC),
	}}}
	if err := saveNoteMapping(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := loadNoteMapping(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(out.Notes) != 1 || out.Notes[0].NoteID != 42 || out.Notes[0].AuthorID != 7 {
		t.Fatalf("round trip = %+v", out.Notes)
	}
}

func TestLoadMissingNoteMapping(t *testing.T) {
	out, err := loadNoteMapping(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if len(out.Notes) != 0 {
		t.Fatalf("expected empty mapping, got %+v", out.Notes)
	}
}

func TestParseTicketComments(t *testing.T) {
	data := []byte(`{"ID": 1, "Comments": [{"Time": "2022-11-22T16:37:02Z", "Author": "jdoe", "Field": "comment", "OldValue": "hello", "Permanent": 1}]}`)
	var ticket ticketWithComments
	if err := json.Unmarshal(data, &ticket); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ticket.ID != 1 || len(ticket.Comments) != 1 || ticket.Comments[0].Author != "jdoe" {
		t.Fatalf("ticket = %+v", ticket)
	}
}
