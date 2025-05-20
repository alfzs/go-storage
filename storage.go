package storage

import (
	"time"
)

// Методы хранилища
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

// Потокобезопасное хранилище в памяти
func NewMemory[T any](cleanupInterval time.Duration) (Storage[T], error) {
	return newMemoryStorage[T](cleanupInterval), nil
}

// Redis
func NewRedis[T any](config RedisConfig) (Storage[T], error) {
	return newRedisStorage[T](config)
}
