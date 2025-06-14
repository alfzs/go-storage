package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisStorage[T any] struct {
	client *redis.Client
}

func newRedisStorage[T any](cfg RedisConfig) (Storage[T], error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &redisStorage[T]{client: client}, nil
}

func (s *redisStorage[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
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

func (s *redisStorage[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("redis get failed: %w", err)
	}

	var out T
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

func (s *redisStorage[T]) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// Enqueue добавляет элемент в очередь
func (s *redisStorage[T]) Enqueue(ctx context.Context, queueName string, value T) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	if err := s.client.RPush(ctx, queueName, data).Err(); err != nil {
		return fmt.Errorf("redis rpush failed: %w", err)
	}

	return nil
}

// Dequeue извлекает и удаляет элемент из очереди
func (s *redisStorage[T]) Dequeue(ctx context.Context, queueName string) (T, bool, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	val, err := s.client.LPop(ctx, queueName).Result()
	if err == redis.Nil {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("redis lpop failed: %w", err)
	}

	var out T
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

// QueueLength возвращает длину очереди
func (s *redisStorage[T]) QueueLen(ctx context.Context, queueName string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	length, err := s.client.LLen(ctx, queueName).Result()
	if err != nil {
		return 0, fmt.Errorf("redis llen failed: %w", err)
	}

	return length, nil
}

func (s *redisStorage[T]) Close() error {
	return s.client.Close()
}
