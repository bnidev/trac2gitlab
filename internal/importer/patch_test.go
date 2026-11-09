package importer

import (
	"strings"
	"testing"
)

func TestRenderAuthorPatch(t *testing.T) {
	rows := []AuthorPatchRow{
		{Kind: "issue", IID: 1, AuthorID: 7},
		{Kind: "note", NoteID: 42, AuthorID: 7},
	}
	script := RenderAuthorPatch(rows, 34, false)
	for _, want := range []string{`DRY_RUN = true`, `PROJECT_ID = 34`, `"iid":1`, `"note_id":42`, `"author_id":7`, `update_column(:author_id`} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q\n%s", want, script)
		}
	}

	applied := RenderAuthorPatch(rows, 34, true)
	if !strings.Contains(applied, `DRY_RUN = false`) {
		t.Errorf("apply script should set DRY_RUN = false")
	}
}

func TestRenderShell(t *testing.T) {
	docker := RenderShell("puts 1", "gitlab", false)
	for _, want := range []string{"docker exec -i gitlab bash -s", "cat > /tmp/patch_authors.rb", "gitlab-rails runner /tmp/patch_authors.rb"} {
		if !strings.Contains(docker, want) {
			t.Errorf("docker shell missing %q\n%s", want, docker)
		}
	}

	omnibus := RenderShell("puts 1", "", true)
	if strings.Contains(omnibus, "docker exec") {
		t.Errorf("omnibus shell should not use docker exec\n%s", omnibus)
	}
	if !strings.Contains(omnibus, "gitlab-rails runner") {
		t.Errorf("omnibus shell missing runner\n%s", omnibus)
	}
}
