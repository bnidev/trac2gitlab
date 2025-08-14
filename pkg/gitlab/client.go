package gitlab

import (
	"fmt"
	"log/slog"

	"github.com/bnidev/trac2gitlab/internal/config"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Client represents a GitLab API client
type Client struct {
	git *gitlab.Client
}

// NewGitLabClient creates a new GitLab API client.
func NewGitLabClient(config *config.Config) (*Client, error) {
	url := fmt.Sprintf("%s%s", config.GitLab.BaseURL, config.GitLab.APIPath)

	slog.Debug("Creating GitLab API client", "url", url)

	git, err := gitlab.NewClient(config.GitLab.Token, gitlab.WithBaseURL(url))
	if err != nil {
		return nil, err
	}
	return &Client{git: git}, nil
}

// Version represents the GitLab version information.
type Version struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
}

// GetVersion retrieves the GitLab version information.
func (c *Client) GetVersion() (*Version, error) {
	var version *Version
	versionRaw, _, err := c.git.Version.GetVersion()
	if err != nil {
		return nil, err
	}
	version = &Version{
		Version:  versionRaw.Version,
		Revision: versionRaw.Revision,
	}
	return version, nil
}

// ValidateGitLab validates the GitLab connection by checking the version and current user.
func (c *Client) ValidateGitLab() error {
	slog.Debug("Validating GitLab connection...")

	version, err := c.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get GitLab version: %w", err)
	}
	slog.Debug("GitLab version retrieved", "version", version.Version, "revision", version.Revision)

	user, _, err := c.git.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	slog.Debug("Authenticated user", "id", user.ID, "name", user.Name, "username", user.Username)

	return nil
}
