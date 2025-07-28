package saga

import "sync"

type IdempotencyStore struct {
	mu    sync.Mutex
	store map[string]bool
}

func NewIdempotencyStore() *IdempotencyStore {
	return &IdempotencyStore{
		store: make(map[string]bool),
	}
}

func (s *IdempotencyStore) MarkCompleted(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = true
}

func (s *IdempotencyStore) IsCompleted(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store[key]
}
