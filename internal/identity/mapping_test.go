package identity

import (
	"strings"
	"testing"
)

const validMapping = `
users:
  john-doe:
    trac_username: jdoe
    trac_display_name: John Doe
    trac_email: jdoe-mail
    gitlab: jdoe
    confidence: manual
  unknown:
    trac_username: mystery
    action: unresolved
`

func TestParseAndResolve(t *testing.T) {
	m, err := Parse([]byte(validMapping))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	for _, author := range []string{"jdoe", "John Doe", " JDOE-MAIL "} {
		user, ok := m.Resolve(author)
		if !ok {
			t.Fatalf("Resolve(%q) = not found", author)
		}
		if user.Key != "john-doe" {
			t.Fatalf("Resolve(%q) = %q, want john-doe", author, user.Key)
		}
		if !user.Resolved() {
			t.Fatalf("%q should be resolved", user.Key)
		}
	}

	if _, ok := m.Resolve("nobody"); ok {
		t.Fatal("Resolve(nobody) should not match")
	}
}

func TestUnresolved(t *testing.T) {
	m, err := Parse([]byte(validMapping))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	unresolved := m.Unresolved()
	if len(unresolved) != 1 || unresolved[0].Key != "unknown" {
		t.Fatalf("Unresolved() = %+v", unresolved)
	}
}

func TestDuplicateAlias(t *testing.T) {
	_, err := Parse([]byte(`
users:
  a:
    trac_username: same
    gitlab: a
  b:
    trac_username: SAME
    gitlab: b
`))
	if err == nil || !strings.Contains(err.Error(), "mapped to both") {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestMissingAlias(t *testing.T) {
	_, err := Parse([]byte(`
users:
  a:
    gitlab: a
`))
	if err == nil || !strings.Contains(err.Error(), "no Trac/HG alias") {
		t.Fatalf("expected missing alias error, got %v", err)
	}
}

func TestMissingGitLab(t *testing.T) {
	_, err := Parse([]byte(`
users:
  a:
    trac_username: a
`))
	if err == nil || !strings.Contains(err.Error(), "no gitlab username") {
		t.Fatalf("expected missing gitlab error, got %v", err)
	}
}

func TestUsersSorted(t *testing.T) {
	m, err := Parse([]byte(validMapping))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	users := m.Users()
	if len(users) != 2 || users[0].Key != "john-doe" || users[1].Key != "unknown" {
		t.Fatalf("Users() = %+v", users)
	}
}
