package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]string // token -> username
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]string),
	}
}

func (s *SessionStore) Create(username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := generateToken()
	s.sessions[token] = username
	return token
}

func (s *SessionStore) Get(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	username, ok := s.sessions[token]
	return username, ok
}

func (s *SessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
}

func (s *SessionStore) UpdateUsername(oldUsername, newUsername string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, u := range s.sessions {
		if u == oldUsername {
			s.sessions[token] = newUsername
		}
	}
}

func generateToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
