package saga

import (
	"context"
	"log"
	"time"

	"github.com/natserract/saga/internal/contract"
	"github.com/natserract/saga/internal/redis_store"
	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyStore struct {
	store *redis_store.RedisStore
}

func NewRedisIdempotencyStore(url string) contract.Store {
	store, err := redis_store.NewRedisStore(&redis_store.Config{
		Url: url,
	})
	if err != nil {
		log.Fatalf("failed to create redis store: %v", err)
		return nil
	}

	return &RedisIdempotencyStore{
		store: store,
	}
}

func (s *RedisIdempotencyStore) MarkCompleted(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.store.Set(ctx, key, true, 0) // Use a 0 expiration for persistent storage
}

func (s *RedisIdempotencyStore) IsCompleted(key string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	val, err := s.store.Get(ctx, key)
	if err == redis.Nil || val == "" {
		return false
	} else if err != nil {
		return false
	}
	return val == "true"
}
