package gitlab

import (
	"fmt"

	"gitlab.com/gitlab-org/api/client-go"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	IsAdmin  bool   `json:"is_admin"`
}


func (c* Client) GetCurrentUser() (*gitlab.User, error) {
	user, _, err := c.git.Users.CurrentUser()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	return user, nil
}
