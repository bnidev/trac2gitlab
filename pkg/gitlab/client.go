package gitlab

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bnidev/trac2gitlab/internal/config"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// basicAuthTransport adds an HTTP site gate (e.g. htaccess) in front of the
// API. It composes with token auth, which uses its own header.
type basicAuthTransport struct {
	username string
	password string
	base     http.RoundTripper
}

func (t *basicAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.SetBasicAuth(t.username, t.password)
	return t.base.RoundTrip(clone)
}

// NewGitLabClient creates a new GitLab API client.
func NewGitLabClient(config *config.Config) (*Client, error) {
	url := fmt.Sprintf("%s%s", config.GitLab.BaseURL, config.GitLab.APIPath)

	slog.Debug("Creating GitLab API client", "url", url)

	opts := []gitlab.ClientOptionFunc{gitlab.WithBaseURL(url)}
	if config.GitLab.HtaccessUser != "" {
		opts = append(opts, gitlab.WithHTTPClient(&http.Client{
			Transport: &basicAuthTransport{
				username: config.GitLab.HtaccessUser,
				password: config.GitLab.HtaccessPass,
				base:     http.DefaultTransport,
			},
		}))
	}

	git, err := gitlab.NewClient(config.GitLab.Token, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{git: git}, nil
}

// Client represents a GitLab API client
type Client struct {
	git *gitlab.Client
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
