package validate

import (
	"strings"
	"testing"
	"time"
)

func TestVerdict(t *testing.T) {
	r := &Report{}
	if r.Verdict() != "PASS" {
		t.Fatalf("empty verdict = %s", r.Verdict())
	}
	r.warn("drift")
	if r.Verdict() != "PASS WITH WARNINGS" {
		t.Fatalf("warning verdict = %s", r.Verdict())
	}
	r.fail("missing")
	if r.Verdict() != "FAIL" {
		t.Fatalf("error verdict = %s", r.Verdict())
	}
}

func TestMarkdown(t *testing.T) {
	r := &Report{Issues: Section{Expected: 2, Found: 2, Matched: 2}}
	md := r.Markdown("proj")
	if !strings.Contains(md, "# Validation — proj") || !strings.Contains(md, "STATUS: PASS") {
		t.Fatalf("markdown = %s", md)
	}
}

func TestCloseEnough(t *testing.T) {
	want := time.Date(2022, 11, 22, 16, 0, 0, 0, time.UTC)
	got := want.Add(time.Second)
	if !closeEnough(&got, want) {
		t.Fatal("1s should be within tolerance")
	}
	got = want.Add(5 * time.Second)
	if closeEnough(&got, want) {
		t.Fatal("5s should be outside tolerance")
	}
	if closeEnough(nil, want) {
		t.Fatal("nil should not match")
	}
}

func TestLinkPresent(t *testing.T) {
	record := attachmentRecord{URL: "/uploads/a.png", Markdown: "![a](/uploads/a.png)"}
	if !linkPresent("see ![a](/uploads/a.png)", record) {
		t.Fatal("markdown link should match")
	}
	if !linkPresent("see /uploads/a.png", record) {
		t.Fatal("url should match")
	}
	if linkPresent("nothing", record) {
		t.Fatal("absent link should not match")
	}
}
