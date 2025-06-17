package storage

import (
	"context"
	"time"
)

type Storage[T any] interface {
	// Операции ключ-значение
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
	Get(ctx context.Context, key string) (T, bool, error)
	Delete(ctx context.Context, key string) error
	Close() error

	// Операции с очередями
	Enqueue(ctx context.Context, queueName string, value T) error
	Dequeue(ctx context.Context, queueName string) (T, bool, error)
	Peek(ctx context.Context, queueName string) (T, bool, error)
	Remove(ctx context.Context, queueName string) (bool, error)
	QueueLen(ctx context.Context, queueName string) (int64, error)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func NewMemory[T any](cleanupInterval time.Duration) (Storage[T], error) {
	return newMemoryStorage[T](cleanupInterval), nil
}

func NewRedis[T any](config RedisConfig) (Storage[T], error) {
	return newRedisStorage[T](config)
}
