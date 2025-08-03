package gitlab

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
}

func (c *Client) GetCurrentUser() (*gitlab.User, error) {
	user, _, err := c.git.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return user, nil
}

func (c *Client) CreateUser(username, name, email string) (*gitlab.User, error) {
	opts := &gitlab.CreateUserOptions{
		Username: &username,
		Name:     &name,
		Email:    &email,
	}

	user, _, err := c.git.Users.CreateUser(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

func (c *Client) UpdateUser(userID int, opts *gitlab.ModifyUserOptions) (*gitlab.User, error) {
	user, _, err := c.git.Users.ModifyUser(userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to update user %d: %w", userID, err)
	}

	return user, nil
}

func (c *Client) GetUserByID(userID int) (*gitlab.User, error) {
	user, _, err := c.git.Users.GetUser(userID, gitlab.GetUsersOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID %d: %w", userID, err)
	}
	return user, nil
}

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

func (c *Client) GetUserByEmail(email string) (*gitlab.User, error) {
	users, _, err := c.git.Users.ListUsers(&gitlab.ListUsersOptions{Search: &email})
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user with email %s not found", email)
	}

	for _, user := range users {
		if user.Email == email {
			return user, nil
		}
	}

	return nil, nil
}

func (c *Client) CreateImpersonationToken(userID int) (*gitlab.ImpersonationToken, error) {
	opts := &gitlab.CreateImpersonationTokenOptions{
		Name:   gitlab.Ptr("Trac Migration Token"),
		Scopes: &[]string{"api"},
	}

	token, _, err := c.git.Users.CreateImpersonationToken(userID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create impersonation token for user %d: %w", userID, err)
	}

	return token, nil
}

func (c *Client) RevokeImpersonationToken(userID, token int) error {
	_, err := c.git.Users.RevokeImpersonationToken(userID, token)
	if err != nil {
		return fmt.Errorf("failed to revoke impersonation token for user %d: %w", userID, err)
	}
	return nil
}

func (c *Client) GetImpersonationTokens(userID int) ([]*gitlab.ImpersonationToken, error) {
	tokens, _, err := c.git.Users.GetAllImpersonationTokens(userID, &gitlab.GetAllImpersonationTokensOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list impersonation tokens for user %d: %w", userID, err)
	}
	return tokens, nil
}

func (c* Client) RevokeAllImpersonationTokens(userID int) error {
	tokens, err := c.GetImpersonationTokens(userID)
	if err != nil {
		return fmt.Errorf("failed to get impersonation tokens for user %d: %w", userID, err)
	}

	for _, token := range tokens {
		if err := c.RevokeImpersonationToken(userID, token.ID); err != nil {
			return fmt.Errorf("failed to revoke token %d for user %d: %w", token.ID, userID, err)
		}
	}

	return nil
}
