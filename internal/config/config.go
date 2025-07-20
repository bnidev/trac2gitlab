package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the application
type Config struct {
	Trac struct {
		BaseURL  string `yaml:"base_url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		RPCPath  string `yaml:"rpc_path"`
	} `yaml:"trac"`

	GitLab struct {
		BaseURL   string `yaml:"base_url"`
		Token     string `yaml:"token"`
		ProjectID int    `yaml:"project_id"`
	} `yaml:"gitlab"`

	Options struct {
		IncludeWiki          bool   `yaml:"include_wiki"`
		IncludeAttachments   bool   `yaml:"include_attachments"`
		IncludeTicketHistory bool   `yaml:"include_ticket_history"`
		IncludeClosedTickets bool   `yaml:"include_closed_tickets"`
		DefaultUser          string `yaml:"default_user"`
		ExportDir            string `yaml:"export_dir"`
	} `yaml:"options"`
}

// LoadConfig reads the configuration from config.yaml
func LoadConfig() Config {
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
