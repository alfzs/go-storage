package storage

import (
	"fmt"
	"time"
)

type Storage interface {
	Set(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool, error)
	Delete(key string) error
	Close() error
}

type Type int

const (
	TypeMemory Type = iota
	TypeRedis
)

func New(storageType Type, config any) (Storage, error) {
	switch storageType {
	case TypeMemory:
		interval, ok := config.(time.Duration)
		if !ok {
			interval = 5 * time.Minute
		}
		return NewMemoryStorage(interval), nil

	case TypeRedis:
		redisConfig, ok := config.(RedisConfig)
		if !ok {
			return nil, fmt.Errorf("invalid Redis config")
		}
		return NewRedisStorage(redisConfig)

	default:
		return nil, fmt.Errorf("unknown storage type")
	}
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}
