package gitlab

import (
	"log"
	"sync"
)

type UserSession struct {
	UserID    int
	TokenInfo *ImpersonationTokenInfo
	Client    *Client
}

type UserSessionCache struct {
	sessions map[string]*UserSession
	mu       sync.Mutex
}

func NewUserSessionCache() *UserSessionCache {
	return &UserSessionCache{
		sessions: make(map[string]*UserSession),
	}
}

func (usc *UserSessionCache) Get(email string) (*UserSession, bool) {
	usc.mu.Lock()
	defer usc.mu.Unlock()
	sess, ok := usc.sessions[email]
	return sess, ok
}

func (usc *UserSessionCache) Set(email string, sess *UserSession) {
	usc.mu.Lock()
	defer usc.mu.Unlock()
	usc.sessions[email] = sess
}

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
