package store

import (
	"errors"
	"sync"
)

var ErrNotFound = errors.New("link not found")

type Link struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type MemoryStore struct {
	mu    sync.RWMutex
	links map[string]string
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		links: make(map[string]string),
	}
}

func (s *MemoryStore) Save(link Link) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.links[link.Code] = link.URL
}

func (s *MemoryStore) Get(code string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}
	return Link{Code: code, URL: url}, nil
}
