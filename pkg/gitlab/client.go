package gitlab

import (
	"fmt"
	"log"

	"gitlab.com/gitlab-org/api/client-go"
)

type Client struct {
	git *gitlab.Client
}

// NewGitLabClient creates a new GitLab API client.
func NewGitLabClient(baseURL, apiPath string, accessToken string) (*Client, error) {
	url := fmt.Sprintf("%s%s", baseURL, apiPath)

	fmt.Printf("Connecting to GitLab at %s\n", url)

	git, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(url))
	if err != nil {
		return nil, err

	}
	return &Client{git: git}, nil
}

type Version struct {
	Version string `json:"version"`
	Revision string `json:"revision"`
}

func (c* Client) GetVersion() (*Version, error) {
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


func (c *Client) ValidateGitLab() error {
	fmt.Println("Validating GitLab connection...")

	version, err := c.GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get GitLab version: %w", err)
	}
	fmt.Printf("GitLab version: %s (Revision %s)\n", version.Version, version.Revision)

	user, _, err := c.git.Users.CurrentUser()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}

	fmt.Printf("Authenticated as: [%d] %s (%s)\n", user.ID, user.Name, user.Username)

	return nil
}
