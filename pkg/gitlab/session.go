package gitlab

import (
	"log"
	"sync"
)

// UserSession represents a user session in GitLab, including the user ID, impersonation token info, and the client used for API calls.
type UserSession struct {
	UserID    int
	TokenInfo *ImpersonationTokenInfo
	Client    *Client
}

// UserSessionCache is a thread-safe cache for storing user sessions.
type UserSessionCache struct {
	sessions map[string]*UserSession
	mu       sync.Mutex
}

// NewUserSessionCache initializes a new UserSessionCache.
func NewUserSessionCache() *UserSessionCache {
	return &UserSessionCache{
		sessions: make(map[string]*UserSession),
	}
}

// Get retrieves a user session by email from the cache.
func (usc *UserSessionCache) Get(email string) (*UserSession, bool) {
	usc.mu.Lock()
	defer usc.mu.Unlock()
	sess, ok := usc.sessions[email]
	return sess, ok
}

// Set adds or updates a user session in the cache using the user's email as the key.
func (usc *UserSessionCache) Set(email string, sess *UserSession) {
	usc.mu.Lock()
	defer usc.mu.Unlock()
	usc.sessions[email] = sess
}

// RevokeAll revokes all user sessions in the cache by iterating through each session and calling the RevokeImpersonationToken method on the provided client.
func (usc *UserSessionCache) RevokeAll(c *Client) {
	usc.mu.Lock()
	defer usc.mu.Unlock()
	for email, sess := range usc.sessions {
		if err := c.RevokeImpersonationToken(sess.UserID, sess.TokenInfo.ID); err != nil {
			log.Printf("failed to revoke token for user %s: %v", email, err)
		}
	}
	usc.sessions = make(map[string]*UserSession)
}
