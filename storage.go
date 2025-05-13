package storage

import (
	"time"
)

type Storage interface {
	Set(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool, error)
	Delete(key string) error
	Close() error
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func NewMemory(cleanupInterval time.Duration) (Storage, error) {
	return newMemoryStorage(cleanupInterval), nil
}

func NewRedis(config RedisConfig) (Storage, error) {
	return newRedisStorage(config)
}
