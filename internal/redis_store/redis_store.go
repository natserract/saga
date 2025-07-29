package redis_store

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/natserract/saga/internal/util"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Url       string
	KeyPrefix string
}

type RedisStore struct {
	Client *redis.Client
	prefix string
}

func NewRedisStore(cfg *Config) (*RedisStore, error) {
	opts, err := redis.ParseURL(cfg.Url)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
		return nil, err
	}
	client := redis.NewClient(opts)

	// sends a ping message
	_, err = client.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	if cfg.KeyPrefix == "" {
		cfg.KeyPrefix = "saga:"
	}

	return &RedisStore{
		Client: client,
		prefix: cfg.KeyPrefix,
	}, nil
}

// wrapperKey is used to build the key name in Redis.
func (s *RedisStore) wrapperKey(key string) string {
	return fmt.Sprintf("%s%s", s.prefix, key)
}

// Set call the Redis client to set a key-value pair with an
// expiration time, where the key name format is <prefix>:<key>.
func (s *RedisStore) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	payload, err := util.Serialize(value)
	if err != nil {
		return err
	}

	cmd := s.Client.Set(ctx, s.wrapperKey(key), payload, expiration)
	return cmd.Err()
}

// Get returns the associated value of the session's given "key".
func (s *RedisStore) Get(ctx context.Context, key string) (interface{}, error) {
	return s.Client.Get(ctx, s.wrapperKey(key)).Bytes()
}

// Delete remove key in redis, do nothing if key doesn't exist
func (s *RedisStore) Delete(ctx context.Context, key string) (bool, error) {
	cmd := s.Client.Del(ctx, key)
	if err := cmd.Err(); err != nil {
		return false, err
	}

	return cmd.Val() > 0, nil
}

// Close is used to close the redis client.
func (s *RedisStore) Close() error {
	return s.Client.Close()
}
