package storage

import (
	"context"
	"sync"
	"time"
)

type memoryStorage[T any] struct {
	items map[string]item[T]
	mu    sync.RWMutex
	stop  chan struct{}
}

func newMemoryStorage[T any](cleanupInterval time.Duration) Storage[T] {
	s := &memoryStorage[T]{
		items: make(map[string]item[T]),
		stop:  make(chan struct{}),
	}
	go s.runGC(cleanupInterval)
	return s
}

type item[T any] struct {
	value      T
	expiration int64
}

func (i item[T]) isExpired() bool {
	return i.expiration > 0 && time.Now().UnixNano() > i.expiration
}

func (s *memoryStorage[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = item[T]{
		value:      value,
		expiration: expiration,
	}
	return nil
}

func (s *memoryStorage[T]) Get(ctx context.Context, key string) (T, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var zero T
	item, found := s.items[key]
	if !found || item.isExpired() {
		return zero, false, nil
	}
	return item.value, true, nil
}

func (s *memoryStorage[T]) Delete(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

func (s *memoryStorage[T]) Close() error {
	close(s.stop)
	return nil
}

func (s *memoryStorage[T]) runGC(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.deleteExpired()
		case <-s.stop:
			return
		}
	}
}

func (s *memoryStorage[T]) deleteExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, item := range s.items {
		if item.isExpired() {
			delete(s.items, key)
		}
	}
}
