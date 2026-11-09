package importer

import (
	"strings"
	"testing"
	"time"
)

func TestAuthorFooter(t *testing.T) {
	at := time.Date(2022, 11, 22, 16, 36, 0, 0, time.UTC)

	got := withAuthorFooter("body", "jdoe", &at, 0)
	if !strings.Contains(got, "_Originally by jdoe at 2022-11-22T16:36:00Z_") {
		t.Fatalf("footer missing: %q", got)
	}
	if again := withAuthorFooter(got, "jdoe", &at, 0); again != got {
		t.Fatalf("footer duplicated: %q", again)
	}
	if got := withAuthorFooter("body", "jdoe", &at, 7); got != "body" {
		t.Fatalf("resolved author should not get a footer: %q", got)
	}
	if got := withAuthorFooter("body", " ", &at, 0); got != "body" {
		t.Fatalf("empty author should not get a footer: %q", got)
	}
	if got := withAuthorFooter("body", "jdoe", nil, 0); got != "body" {
		t.Fatalf("nil time should not get a footer: %q", got)
	}
}
