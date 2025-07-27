package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the application
type Config struct {
	Trac          TracConfig    `yaml:"trac"`
	GitLab        GitLabConfig  `yaml:"gitlab"`
	ExportOptions ExportOptions `yaml:"export_options"`
	ImportOptions ImportOptions `yaml:"import_options"`
}

// TracConfig holds the configuration for the Trac instance
type TracConfig struct {
	BaseURL string `yaml:"base_url"`
	RPCPath string `yaml:"rpc_path"`
}

// GitLabConfig holds the configuration for the GitLab instance
type GitLabConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIPath   string `yaml:"api_path"`
	Token     string `yaml:"token"`
	ProjectID int    `yaml:"project_id"`
}

// ExportOptions holds the options for exporting data from Trac
type ExportOptions struct {
	IncludeWiki          bool   `yaml:"include_wiki"`
	IncludeAttachments   bool   `yaml:"include_attachments"`
	IncludeTicketHistory bool   `yaml:"include_ticket_history"`
	IncludeClosedTickets bool   `yaml:"include_closed_tickets"`
	ExportDir            string `yaml:"export_dir"`
}

// ImportOptions holds the options for importing data into GitLab
type ImportOptions struct {
	ImportIssues     bool `yaml:"import_issues"`
	ImportMilestones bool `yaml:"import_milestones"`
}

// LoadConfig reads the configuration from config.yaml
func LoadConfig() Config {
	if !CheckConfigExists() {
		log.Fatal("Configuration file config.yaml does not exist. Please run 'trac2gitlab init' to create it.")
	}

	f, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("Failed to open config.yaml: %v", err)
	}

	defer func() {
		if cerr := f.Close(); cerr != nil {
			log.Fatalf("Failed to close config.yaml: %v", cerr)
		}
	}()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("Failed to parse config.yaml: %v", err)
	}
	return cfg
}

// CheckConfigExists checks if the configuration file exists
func CheckConfigExists() bool {
	_, err := os.Stat("config.yaml")
	return !os.IsNotExist(err)
}

// CreateDefaultConfig creates a default configuration file
func CreateDefaultConfig() error {
	defaultConfig := Config{
		Trac: TracConfig{
			BaseURL: "https://trac.example.com",
			RPCPath: "/xmlrpc",
		},
		GitLab: GitLabConfig{
			BaseURL:   "https://gitlab.com",
			APIPath:   "/api/v4",
			Token:     "your_gitlab_token",
			ProjectID: 1,
		},
		ExportOptions: ExportOptions{
			IncludeWiki:          true,
			IncludeAttachments:   true,
			IncludeTicketHistory: true,
			IncludeClosedTickets: true,
			ExportDir:            "data",
		},
		ImportOptions: ImportOptions{
			ImportIssues:     true,
			ImportMilestones: true,
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	f, err := os.Create("config.yaml")
	if err != nil {
		return err
	}

	if _, err := f.Write(data); err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
