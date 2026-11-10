package utils

import "testing"

func TestTracToMarkdownTables(t *testing.T) {
	in := "||= Stack =|| !WordPress ||\n" +
		"||= Theme           =|| BundesCI von [Twenty](https://x) ||\n" +
		"\n" +
		"||=  Name ||= Firma ||=  Adresse =||\n" +
		"|| Jan || ACME || Wien ||"
	want := "| Stack | WordPress |\n" +
		"| --- | --- |\n" +
		"| **Theme** | BundesCI von [Twenty](https://x) |\n" +
		"\n" +
		"| Name | Firma | Adresse |\n" +
		"| --- | --- | --- |\n" +
		"| Jan | ACME | Wien |"
	if got := TracToMarkdown(in); got != want {
		t.Errorf("tables:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestTracToMarkdownLinks(t *testing.T) {
	cases := map[string]string{
		"see [[#Sec|below]]":         "see [below](#Sec)",
		"see [[Page]]":               "see [Page](Page.md)",
		"see [[Page|label]]":         "see [label](Page.md)",
		"keep [[Image(a.png, 400)]]": "keep [[Image(a.png, 400)]]",
		"ref [wiki:Other]":           "ref [Other](Other.md)",
	}
	for in, want := range cases {
		if got := TracToMarkdown(in); got != want {
			t.Errorf("links: TracToMarkdown(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTracToMarkdownCodeBlocks(t *testing.T) {
	closed := "before\n{{{\ncode line\n}}}\nafter"
	wantClosed := "before\n```\ncode line\n```\nafter"
	if got := TracToMarkdown(closed); got != wantClosed {
		t.Errorf("closed code:\ngot:\n%s\nwant:\n%s", got, wantClosed)
	}

	// rollout-test: block never closed, Trac renders the rest as code.
	unclosed := "text\n{{{#!bash\nsudo su\necho hi"
	wantUnclosed := "text\n```bash\nsudo su\necho hi\n```"
	if got := TracToMarkdown(unclosed); got != wantUnclosed {
		t.Errorf("unclosed code:\ngot:\n%s\nwant:\n%s", got, wantUnclosed)
	}
}
