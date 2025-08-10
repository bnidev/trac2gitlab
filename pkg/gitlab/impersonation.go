package gitlab

import (
	"fmt"
	"time"

	gitlab_client "gitlab.com/gitlab-org/api/client-go"
)

// CreateImpersonationToken creates an impersonation token for the specified user with an optional expiration date.
func (c *Client) CreateImpersonationToken(userID int, expireDate *time.Time) (*gitlab_client.ImpersonationToken, error) {
	opts := &gitlab_client.CreateImpersonationTokenOptions{
		Name:      gitlab_client.Ptr("Trac Migration Token"),
		Scopes:    &[]string{"api"},
		ExpiresAt: expireDate,
	}

	token, _, err := c.git.Users.CreateImpersonationToken(userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation token for user %d: %w", userID, err)
	}

	return token, nil
}

// ImpersonationTokenInfo holds information about an impersonation token.
type ImpersonationTokenInfo struct {
	ID    int
	Name  string
	Token string
}

// EnsureImpersonationToken checks if an impersonation token with the given name exists for the user.
func (c *Client) EnsureImpersonationToken(userID int, tokenName string, scopes []string) (*ImpersonationTokenInfo, error) {
	tokens, _, err := c.git.Users.GetAllImpersonationTokens(userID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing impersonation tokens: %w", err)
	}

	for _, t := range tokens {
		if t.Name == tokenName && !t.Revoked {
			_, err := c.git.Users.RevokeImpersonationToken(userID, t.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to revoke existing token %q (id: %d): %w", t.Name, t.ID, err)
			}
		}
	}

	newToken, _, err := c.git.Users.CreateImpersonationToken(userID, &gitlab_client.CreateImpersonationTokenOptions{
		Name:      &tokenName,
		Scopes:    &scopes,
		ExpiresAt: gitlab_client.Ptr(time.Now().Add(24 * time.Hour)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new impersonation token: %w", err)
	}

	return &ImpersonationTokenInfo{
		ID:    newToken.ID,
		Name:  newToken.Name,
		Token: newToken.Token,
	}, nil
}
