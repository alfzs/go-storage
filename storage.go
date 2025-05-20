package storage

import (
	"context"
	"time"
)

// Методы хранилища
type Storage[T any] interface {
	Set(ctx context.Context, key string, value T, ttl time.Duration) error
	Get(ctx context.Context, key string) (T, bool, error)
	Delete(ctx context.Context, key string) error
	Close() error
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Потокобезопасное хранилище в памяти
func NewMemory[T any](cleanupInterval time.Duration) (Storage[T], error) {
	return newMemoryStorage[T](cleanupInterval), nil
}

// Redis
func NewRedis[T any](config RedisConfig) (Storage[T], error) {
	return newRedisStorage[T](config)
}
