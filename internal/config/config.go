package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the application
type Config struct {
	General       General          `yaml:"general"`
	Trac          TracConfig       `yaml:"trac"`
	GitLab        GitLabConfig     `yaml:"gitlab"`
	ExportOptions ExportOptions    `yaml:"export_options"`
	ImportOptions ImportOptions    `yaml:"import_options"`
	References    ReferenceOptions `yaml:"references"`
	Labels        LabelOptions     `yaml:"labels"`

	ProjectDir string `yaml:"-"`
	DryRun     bool   `yaml:"-"`
}

// LabelOptions controls Trac field to GitLab label mapping.
type LabelOptions struct {
	Enabled       bool     `yaml:"enabled"`
	Prefix        string   `yaml:"prefix"`
	Fields        []string `yaml:"fields"`
	CreateMissing bool     `yaml:"create_missing"`
}

// ReferenceOptions controls Trac reference rewriting.
type ReferenceOptions struct {
	Enabled        bool   `yaml:"enabled"`
	TicketsFile    string `yaml:"tickets_file"`
	ChangesetsFile string `yaml:"changesets_file"`
	WikiFile       string `yaml:"wiki_file"`
}

// General holds general configuration options
type General struct {
	LogLevel string `yaml:"log_level"`
}

// TracConfig holds the configuration for the Trac instance.
// Two credential layers exist: an optional HTTP site gate (htaccess,
// HTACCESS_USER/HTACCESS_PASSWORD) applied at the transport level, and the
// Trac account itself (TRAC_USER/TRAC_PASSWORD). A single request carries
// one Basic Auth header, so when both are set the site gate wins on the
// wire; the Trac account still identifies the API user where the plugin
// requires one.
type TracConfig struct {
	BaseURL string `yaml:"base_url"`
	URL     string `yaml:"url"`
	RPCPath string `yaml:"rpc_path"`
	// Credentials come from the environment or .env, never from YAML.
	Username string `yaml:"-"`
	Password string `yaml:"-"`
	// Optional HTTP site gate (e.g. htaccess) in front of Trac.
	HtaccessUser string `yaml:"-"`
	HtaccessPass string `yaml:"-"`
}

// GitLabConfig holds the configuration for the GitLab instance
type GitLabConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIPath   string `yaml:"api_path"`
	Token     string `yaml:"token"`
	ProjectID int    `yaml:"project_id"`
	Project   string `yaml:"project"`

	ProjectPath string `yaml:"-"`
}

// ExportOptions holds the options for exporting data from Trac
type ExportOptions struct {
	IncludeWiki            bool     `yaml:"include_wiki"`
	IncludeAttachments     bool     `yaml:"include_attachments"`
	IncludeTicketHistory   bool     `yaml:"include_ticket_history"`
	IncludeClosedTickets   bool     `yaml:"include_closed_tickets"`
	IncludeTicketFields    bool     `yaml:"include_ticket_fields"`
	AdditionalTicketFields []string `yaml:"additional_ticket_fields"`
	IncludeUsers           bool     `yaml:"include_users"`
	ExportDir              string   `yaml:"export_dir"`
}

// ImportOptions holds the options for importing data into GitLab
type ImportOptions struct {
	ImportIssues      bool   `yaml:"import_issues"`
	ImportMilestones  bool   `yaml:"import_milestones"`
	ImportNotes       bool   `yaml:"import_notes"`
	ImportAttachments bool   `yaml:"import_attachments"`
	ImportWiki        bool   `yaml:"import_wiki"`
	UsersFile         string `yaml:"users_file"`
	CreateUsers       bool   `yaml:"create_users"`
}

// LoadConfig reads the configuration from the current directory.
func LoadConfig() (Config, error) {
	return LoadProject(".")
}

// LoadProject reads a migration project directory.
//
// Configuration is read from <dir>/migration.yaml (preferred) or
// <dir>/config.yaml. <dir>/.env is loaded first; existing environment
// variables win. Relative export/users paths are resolved against <dir>.
func LoadProject(projectDir string) (Config, error) {
	loadDotEnv(filepath.Join(projectDir, ".env"))

	path, err := findConfig(projectDir)
	if err != nil {
		return Config{}, err
	}

	cfg := DefaultConfig()
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("Error closing config file", "path", path, "errorMsg", cerr)
		}
	}()
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	applyEnv(&cfg)
	normalize(&cfg, projectDir)
	warnIncomplete(&cfg)

	return cfg, nil
}

func findConfig(projectDir string) (string, error) {
	for _, name := range []string{"migration.yaml", "config.yaml"} {
		path := filepath.Join(projectDir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no migration.yaml or config.yaml in %q", projectDir)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("TRAC_URL"); v != "" {
		cfg.Trac.BaseURL = v
	}
	if cfg.Trac.BaseURL == "" {
		cfg.Trac.BaseURL = cfg.Trac.URL
	}
	cfg.Trac.BaseURL = strings.TrimRight(cfg.Trac.BaseURL, "/")
	cfg.Trac.Username = os.Getenv("TRAC_USER")
	cfg.Trac.Password = os.Getenv("TRAC_PASSWORD")
	cfg.Trac.HtaccessUser = os.Getenv("HTACCESS_USER")
	cfg.Trac.HtaccessPass = os.Getenv("HTACCESS_PASSWORD")

	if v := firstEnv("GITLAB_URL", "GITLAB_HOST"); v != "" {
		cfg.GitLab.BaseURL = v
	}
	if cfg.GitLab.BaseURL != "" && !strings.Contains(cfg.GitLab.BaseURL, "://") {
		cfg.GitLab.BaseURL = "https://" + cfg.GitLab.BaseURL
	}
	cfg.GitLab.BaseURL = strings.TrimRight(cfg.GitLab.BaseURL, "/")
	if v := os.Getenv("GITLAB_TOKEN"); v != "" {
		cfg.GitLab.Token = v
	}
	if v := os.Getenv("GITLAB_PROJECT"); v != "" {
		cfg.GitLab.Project = v
	}
	if v := os.Getenv("GITLAB_PROJECT_ID"); v != "" {
		if id, err := strconv.Atoi(v); err == nil {
			cfg.GitLab.ProjectID = id
		}
	}
}

func normalize(cfg *Config, projectDir string) {
	cfg.ProjectDir = projectDir
	if cfg.Trac.RPCPath == "" {
		cfg.Trac.RPCPath = "/xmlrpc"
	}
	if cfg.GitLab.APIPath == "" {
		cfg.GitLab.APIPath = "/api/v4"
	}
	if cfg.ExportOptions.ExportDir == "" {
		cfg.ExportOptions.ExportDir = "data"
	}
	if !filepath.IsAbs(cfg.ExportOptions.ExportDir) {
		cfg.ExportOptions.ExportDir = filepath.Join(projectDir, cfg.ExportOptions.ExportDir)
	}
	if cfg.ImportOptions.UsersFile == "" {
		cfg.ImportOptions.UsersFile = "mappings/users.yaml"
	}
	if !filepath.IsAbs(cfg.ImportOptions.UsersFile) {
		cfg.ImportOptions.UsersFile = filepath.Join(projectDir, cfg.ImportOptions.UsersFile)
	}
	if cfg.References.TicketsFile == "" {
		cfg.References.TicketsFile = "mappings/trac-tickets.json"
	}
	if !filepath.IsAbs(cfg.References.TicketsFile) {
		cfg.References.TicketsFile = filepath.Join(projectDir, cfg.References.TicketsFile)
	}
	if cfg.References.ChangesetsFile == "" {
		cfg.References.ChangesetsFile = "mappings/hg-changesets.json"
	}
	if !filepath.IsAbs(cfg.References.ChangesetsFile) {
		cfg.References.ChangesetsFile = filepath.Join(projectDir, cfg.References.ChangesetsFile)
	}
	if cfg.References.WikiFile == "" {
		cfg.References.WikiFile = "mappings/wiki-pages.json"
	}
	if !filepath.IsAbs(cfg.References.WikiFile) {
		cfg.References.WikiFile = filepath.Join(projectDir, cfg.References.WikiFile)
	}
	if cfg.Labels.Prefix == "" {
		cfg.Labels.Prefix = "trac"
	}
	if len(cfg.Labels.Fields) == 0 {
		cfg.Labels.Fields = []string{"type", "priority", "resolution", "component", "version", "severity", "status"}
	}
	for i, field := range cfg.Labels.Fields {
		cfg.Labels.Fields[i] = strings.ToLower(strings.TrimSpace(field))
	}
}

func warnIncomplete(cfg *Config) {
	if cfg.Trac.BaseURL == "" || cfg.Trac.RPCPath == "" {
		slog.Warn("Trac configuration is incomplete. Check migration.yaml or TRAC_URL.")
	}
	if cfg.GitLab.BaseURL == "" || cfg.GitLab.APIPath == "" || cfg.GitLab.Token == "" {
		slog.Warn("GitLab configuration is incomplete. Check GITLAB_URL and GITLAB_TOKEN.")
	}
	if cfg.GitLab.ProjectID <= 0 && cfg.GitLab.Project == "" {
		slog.Warn("GitLab project is not set. Set gitlab.project_id or GITLAB_PROJECT.")
	}
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				slog.Warn("Failed to set environment variable from .env", "key", key, "error", err)
			}
		}
	}
}

// CheckConfigExists checks if a config file exists in the project directory.
func CheckConfigExists(projectDir string) bool {
	_, err := findConfig(projectDir)
	return err == nil
}

// DefaultConfig returns the default migration configuration.
func DefaultConfig() Config {
	return Config{
		General: General{
			LogLevel: "info",
		},
		Trac: TracConfig{
			BaseURL: "https://trac.example.com",
			RPCPath: "/xmlrpc",
		},
		GitLab: GitLabConfig{
			BaseURL: "https://gitlab.com",
			APIPath: "/api/v4",
			Project: "group/project",
		},
		ExportOptions: ExportOptions{
			IncludeWiki:          true,
			IncludeAttachments:   true,
			IncludeTicketHistory: true,
			IncludeClosedTickets: true,
			IncludeTicketFields:  true,
			IncludeUsers:         true,
			ExportDir:            "data",
		},
		ImportOptions: ImportOptions{
			ImportIssues:      true,
			ImportMilestones:  true,
			ImportNotes:       true,
			ImportAttachments: true,
			ImportWiki:        true,
			UsersFile:         "mappings/users.yaml",
			CreateUsers:       false,
		},
		References: ReferenceOptions{
			Enabled:        true,
			TicketsFile:    "mappings/trac-tickets.json",
			ChangesetsFile: "mappings/hg-changesets.json",
			WikiFile:       "mappings/wiki-pages.json",
		},
		Labels: LabelOptions{
			Enabled:       true,
			Prefix:        "trac",
			Fields:        []string{"type", "priority", "resolution", "component", "version", "severity", "status"},
			CreateMissing: true,
		},
	}
}

// CreateDefaultConfig creates <dir>/migration.yaml.
func CreateDefaultConfig(projectDir string) error {
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return err
	}

	path := filepath.Join(projectDir, "migration.yaml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
