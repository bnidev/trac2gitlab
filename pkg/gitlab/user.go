package gitlab

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	cfg "trac2gitlab/internal/config"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// User represents a GitLab user.
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
}

var ErrUserNotFound = errors.New("user not found")

// GetCurrentUser retrieves the currently authenticated user from GitLab.
func (c *Client) GetCurrentUser() (*gitlab.User, error) {
	user, _, err := c.git.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return user, nil
}

// CreateUser creates a new user in GitLab with the specified username, name, and email.
func (c *Client) CreateUser(username, name, email string) (*gitlab.User, error) {
	opts := &gitlab.CreateUserOptions{
		Username:            &username,
		Name:                &name,
		Email:               &email,
		ForceRandomPassword: gitlab.Ptr(true),
	}

	user, _, err := c.git.Users.CreateUser(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// CreateUserFromEmail creates a new user in GitLab based on the provided email address.
func (c *Client) CreateUserFromEmail(email string) (*gitlab.User, error) {
	parts := strings.Split(email, "@")

	username := parts[0]
	name := username
	email = strings.ToLower(email)

	return c.CreateUser(username, name, email)
}

// UpdateUser updates an existing user in GitLab with the specified options.
func (c *Client) UpdateUser(userID int, opts *gitlab.ModifyUserOptions) (*gitlab.User, error) {
	user, _, err := c.git.Users.ModifyUser(userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to update user %d: %w", userID, err)
	}

	return user, nil
}

// GetUserByID retrieves a user from GitLab by their ID.
func (c *Client) GetUserByID(userID int) (*gitlab.User, error) {
	user, _, err := c.git.Users.GetUser(userID, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID %d: %w", userID, err)
	}
	return user, nil
}

// GetUserByUsername retrieves a user from GitLab by their username.
func (c *Client) GetUserByUsername(username string) (*gitlab.User, error) {
	users, _, err := c.git.Users.ListUsers(&gitlab.ListUsersOptions{Username: &username})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username %s: %w", username, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user with username %s not found", username)
	}

	for _, user := range users {
		if user.Username == username {
			return user, nil
		}
	}

	return nil, nil
}

// GetUserByEmail retrieves a user from GitLab by their email address.
func (c *Client) GetUserByEmail(email string) (*gitlab.User, error) {
	users, _, err := c.git.Users.ListUsers(&gitlab.ListUsersOptions{Search: &email})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, ErrUserNotFound
}

// CreateIssueAsUser creates an issue in a GitLab project as a specific user identified by their email address.
func (c *Client) CreateIssueAsUser(config *cfg.Config, cache *UserSessionCache, projectID any, email string, opts *gitlab.CreateIssueOptions) (*Issue, error) {
	sess, ok := cache.Get(email)
	if ok {
		slog.Debug("Using cached impersonated client", "email", email)
		return sess.Client.CreateIssue(projectID, opts)
	}

	user, err := c.GetUserByEmail(email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			if config.ImportOptions.CreateUsers {
				user, err = c.CreateUserFromEmail(email)
				if err != nil {
					return nil, fmt.Errorf("failed to auto-create user %q: %w", email, err)
				}
			} else {
				return nil, fmt.Errorf("user %q does not exist and auto-creation is disabled", email)
			}
		} else {
			return nil, fmt.Errorf("failed to look up user %q: %w", email, err)
		}
	}

	tokenInfo, err := c.EnsureImpersonationToken(user.ID, "issue-import-token", []string{"api"})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize impersonation token: %w", err)
	}

	impersonatedConfig := *config
	impersonatedConfig.GitLab.Token = tokenInfo.Token
	impersonatedClient, err := NewGitLabClient(&impersonatedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonated client for user %q: %w", email, err)
	}

	sess = &UserSession{
		TokenInfo: tokenInfo,
		Client:    impersonatedClient,
		UserID:    user.ID,
	}

	cache.Set(email, sess)

	return impersonatedClient.CreateIssue(projectID, opts)
}
