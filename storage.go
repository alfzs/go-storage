package storage

import (
	"time"
)

type Storage[T any] interface {
	Set(key string, value T, ttl time.Duration) error
	Get(key string) (T, bool, error)
	Delete(key string) error
	Close() error
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
