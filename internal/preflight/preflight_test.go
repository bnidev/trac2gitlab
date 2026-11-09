package preflight

import (
	"strings"
	"testing"
)

func TestRoleName(t *testing.T) {
	cases := map[int]string{50: "Owner", 40: "Maintainer", 30: "Developer", 20: "Reporter", 10: "Guest", 0: "None"}
	for level, want := range cases {
		if got := roleName(level); got != want {
			t.Errorf("roleName(%d) = %q, want %q", level, got, want)
		}
	}
}

func TestEvaluate(t *testing.T) {
	r := &Report{}
	r.evaluate()
	if r.Verdict != "READY" {
		t.Fatalf("empty verdict = %s", r.Verdict)
	}

	r.warn("w")
	r.evaluate()
	if r.Verdict != "READY WITH WARNINGS" {
		t.Fatalf("warning verdict = %s", r.Verdict)
	}

	r.fail("e")
	r.evaluate()
	if r.Verdict != "NOT READY" {
		t.Fatalf("error verdict = %s", r.Verdict)
	}

	r.Blocked = true
	r.evaluate()
	if r.Verdict != "BLOCKED" {
		t.Fatalf("blocked verdict = %s", r.Verdict)
	}
}

func TestMarkdown(t *testing.T) {
	r := &Report{Verdict: "READY", TracOK: true, GitLabOK: true, TimestampOK: true, Role: "Owner"}
	md := r.Markdown("proj")
	for _, want := range []string{"# Preflight — proj", "| Trac XML-RPC | ✓ |", "| Current role | Owner |", "STATUS: READY"} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n%s", want, md)
		}
	}
}
