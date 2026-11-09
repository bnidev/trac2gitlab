package identity

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bnidev/trac2gitlab/pkg/gitlab"
)

// Resolver resolves GitLab usernames to numeric user IDs.
// It prefers project members so it works with a project Owner token.
type Resolver struct {
	client    *gitlab.Client
	projectID any

	mu      sync.Mutex
	members map[string]int64
	loaded  bool
}

func NewResolver(client *gitlab.Client, projectID any) *Resolver {
	return &Resolver{client: client, projectID: projectID}
}

// Resolve returns the GitLab user ID for username.
func (r *Resolver) Resolve(username string) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return 0, fmt.Errorf("empty gitlab username")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.loadMembers(); err != nil {
		return 0, err
	}
	if id, ok := r.members[strings.ToLower(username)]; ok {
		return id, nil
	}

	user, err := r.client.GetUserByUsername(username)
	if err != nil {
		return 0, err
	}
	if user == nil {
		return 0, fmt.Errorf("gitlab user %q not found", username)
	}
	r.members[strings.ToLower(username)] = user.ID
	return user.ID, nil
}

func (r *Resolver) loadMembers() error {
	if r.loaded {
		return nil
	}
	members, err := r.client.GetProjectMembers(r.projectID)
	if err != nil {
		return fmt.Errorf("list project members: %w", err)
	}
	r.members = make(map[string]int64, len(members))
	for _, member := range members {
		r.members[strings.ToLower(member.Username)] = member.ID
	}
	r.loaded = true
	return nil
}
