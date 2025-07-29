package saga

import (
	"sync"

	"github.com/natserract/saga/internal/contract"
)

type MemoryIdempotencyStore struct {
	mu    sync.Mutex
	store map[string]bool
}

func NewMemoryIdempotencyStore() contract.Store {
	return &MemoryIdempotencyStore{
		store: make(map[string]bool),
	}
}

func (s *MemoryIdempotencyStore) MarkCompleted(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = true
}

func (s *MemoryIdempotencyStore) IsCompleted(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.store[key]
}
