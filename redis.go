package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStorage struct {
	client *redis.Client
	ctx    context.Context
}

func newRedisStorage(cfg RedisConfig) (Storage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &redisStorage{
		client: client,
		ctx:    ctx,
	}, nil
}

func (s *redisStorage) Set(key string, value any, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	var redisErr error
	if ttl > 0 {
		redisErr = s.client.Set(ctx, key, data, ttl).Err()
	} else {
		redisErr = s.client.Set(ctx, key, data, redis.KeepTTL).Err()
	}

	if redisErr != nil {
		return fmt.Errorf("redis set failed: %w", redisErr)
	}

	return nil
}

func (s *redisStorage) Get(key string) (any, bool, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("redis get failed: %w", err)
	}

	var out any
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return nil, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

func (s *redisStorage) Delete(key string) error {
	ctx, cancel := context.WithTimeout(s.ctx, 1*time.Second)
	defer cancel()

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

func (s *redisStorage) Close() error {
	return s.client.Close()
}
