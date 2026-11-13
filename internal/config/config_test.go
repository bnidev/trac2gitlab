package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProject(t *testing.T) {
	dir := t.TempDir()
	yaml := `
trac:
  url: https://trac.example.com
gitlab:
  project: group/project
export_options:
  export_dir: data
import_options:
  users_file: mappings/users.yaml
`
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GITLAB_URL", "")
	t.Setenv("GITLAB_HOST", "gitlab.example.com")
	t.Setenv("GITLAB_TOKEN", "token")
	t.Setenv("GITLAB_PROJECT", "")
	t.Setenv("GITLAB_PROJECT_ID", "")
	t.Setenv("TRAC_URL", "")
	t.Setenv("TRAC_USER", "trac-user")
	t.Setenv("TRAC_PASSWORD", "trac-pass")
	t.Setenv("TRAC_HTACCESS_USER", "gate-user")
	t.Setenv("TRAC_HTACCESS_PASSWORD", "gate-pass")

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if cfg.Trac.BaseURL != "https://trac.example.com" {
		t.Errorf("Trac.BaseURL = %q", cfg.Trac.BaseURL)
	}
	if cfg.Trac.RPCPath != "/xmlrpc" {
		t.Errorf("Trac.RPCPath = %q", cfg.Trac.RPCPath)
	}
	if cfg.Trac.Username != "trac-user" || cfg.Trac.Password != "trac-pass" {
		t.Errorf("Trac credentials not loaded from env")
	}
	if cfg.Trac.HtaccessUser != "gate-user" || cfg.Trac.HtaccessPass != "gate-pass" {
		t.Errorf("Htaccess credentials not loaded from env")
	}
	if cfg.GitLab.BaseURL != "https://gitlab.example.com" {
		t.Errorf("GitLab.BaseURL = %q", cfg.GitLab.BaseURL)
	}
	if cfg.GitLab.APIPath != "/api/v4" {
		t.Errorf("GitLab.APIPath = %q", cfg.GitLab.APIPath)
	}
	if cfg.GitLab.Token != "token" {
		t.Errorf("GitLab.Token = %q", cfg.GitLab.Token)
	}
	if cfg.GitLab.Project != "group/project" {
		t.Errorf("GitLab.Project = %q", cfg.GitLab.Project)
	}
	if cfg.ExportOptions.ExportDir != filepath.Join(dir, "data") {
		t.Errorf("ExportDir = %q", cfg.ExportOptions.ExportDir)
	}
	if cfg.ImportOptions.UsersFile != filepath.Join(dir, "mappings/users.yaml") {
		t.Errorf("UsersFile = %q", cfg.ImportOptions.UsersFile)
	}
}

func TestFindConfigPrefersMigration(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("trac: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte("trac: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	path, err := findConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "migration.yaml" {
		t.Fatalf("findConfig = %q", path)
	}
}

func TestDotEnvDoesNotOverrideEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("T2G_TEST_KEY=from-file\nT2G_TEST_ONLY_FILE='quoted'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("T2G_TEST_KEY", "from-env")
	getenv := envWithDotEnv(readDotEnv(path))
	if got := getenv("T2G_TEST_KEY"); got != "from-env" {
		t.Fatalf(".env overrode existing env: %q", got)
	}
	if got := getenv("T2G_TEST_ONLY_FILE"); got != "quoted" {
		t.Fatalf("value from .env = %q, want quoted", got)
	}
	if _, set := os.LookupEnv("T2G_TEST_ONLY_FILE"); set {
		t.Fatal("readDotEnv wrote to the process environment")
	}
}

// The wizard server loads many projects in one process; one project's .env
// must not leak into another.
func TestLoadProjectIsolatesDotEnv(t *testing.T) {
	root := t.TempDir()
	withEnv, bare := filepath.Join(root, "a"), filepath.Join(root, "b")
	for _, dir := range []string{withEnv, bare} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "migration.yaml"), []byte("gitlab:\n  project: g/p\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(withEnv, ".env"), []byte("GITLAB_TOKEN=secret-a\nTRAC_USER=alice\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"GITLAB_TOKEN", "TRAC_USER"} {
		t.Setenv(key, "") // restores the original value after the test
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}

	a, err := LoadProject(withEnv)
	if err != nil {
		t.Fatal(err)
	}
	if a.GitLab.Token != "secret-a" || a.Trac.Username != "alice" {
		t.Fatalf("project a: token %q user %q", a.GitLab.Token, a.Trac.Username)
	}
	b, err := LoadProject(bare)
	if err != nil {
		t.Fatal(err)
	}
	if b.GitLab.Token != "" || b.Trac.Username != "" {
		t.Fatalf("project b inherited project a's .env: token %q user %q", b.GitLab.Token, b.Trac.Username)
	}
}

func TestCreateDefaultConfig(t *testing.T) {
	dir := t.TempDir()
	if err := CreateDefaultConfig(dir); err != nil {
		t.Fatalf("CreateDefaultConfig: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "migration.yaml")); err != nil {
		t.Fatalf("migration.yaml not created: %v", err)
	}
	if !CheckConfigExists(dir) {
		t.Fatal("CheckConfigExists = false")
	}
}
