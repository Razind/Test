package store

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("link not found")
var ErrUserNotFound = errors.New("user not found")
var ErrEmailExists = errors.New("email already registered")
var ErrUnauthorized = errors.New("unauthorized")

type Link struct {
	Code   string `json:"code"`
	URL    string `json:"url"`
	UserID string `json:"user_id"`
	Clicks int    `json:"clicks"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

type MemoryStore struct {
	mu          sync.RWMutex
	links       map[string]Link
	users       map[string]User
	userByEmail map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		links:       make(map[string]Link),
		users:       make(map[string]User),
		userByEmail: make(map[string]string),
	}
}

func (s *MemoryStore) CreateUser(email, passwordHash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.userByEmail[email]; exists {
		return User{}, ErrEmailExists
	}
	user := User{
		ID:           newID(),
		Email:        email,
		PasswordHash: passwordHash,
	}
	s.users[user.ID] = user
	s.userByEmail[email] = user.ID
	return user, nil
}

func (s *MemoryStore) GetUserByEmail(email string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.userByEmail[email]
	if !ok {
		return User{}, ErrUserNotFound
	}
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) GetUser(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (s *MemoryStore) SaveLink(link Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[link.Code] = link
}

func (s *MemoryStore) GetLink(code string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	link, ok := s.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return link, nil
}

func (s *MemoryStore) ListLinksByUser(userID string) []Link {
	s.mu.RLock()
	defer s.mu.RUnlock()
	links := make([]Link, 0)
	for _, link := range s.links {
		if link.UserID == userID {
			links = append(links, link)
		}
	}
	return links
}

func (s *MemoryStore) DeleteLink(code, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[code]
	if !ok {
		return ErrNotFound
	}
	if link.UserID != userID {
		return ErrUnauthorized
	}
	delete(s.links, code)
	return nil
}

func (s *MemoryStore) IncrementClick(code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	link, ok := s.links[code]
	if !ok {
		return ErrNotFound
	}
	link.Clicks++
	s.links[code] = link
	return nil
}

func newID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
