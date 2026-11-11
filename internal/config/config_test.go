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

func TestLoadDotEnvDoesNotOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("T2G_TEST_KEY=from-file\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("T2G_TEST_KEY", "from-env")
	loadDotEnv(path)
	if got := os.Getenv("T2G_TEST_KEY"); got != "from-env" {
		t.Fatalf("loadDotEnv overrode existing env: %q", got)
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
