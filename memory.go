package storage

import (
	"sync"
	"time"
)

type memoryStorage struct {
	items map[string]item
	mu    sync.RWMutex
	stop  chan struct{}
}

func newMemoryStorage(cleanupInterval time.Duration) Storage {
	s := &memoryStorage{
		items: make(map[string]item),
		stop:  make(chan struct{}),
	}
	go s.runGC(cleanupInterval)
	return s
}

type item struct {
	value      any
	expiration int64
}

func (i item) isExpired() bool {
	return i.expiration > 0 && time.Now().UnixNano() > i.expiration
}

func (s *memoryStorage) Set(key string, value any, ttl time.Duration) error {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.items[key] = item{
		value:      value,
		expiration: expiration,
	}
	return nil
}

func (s *memoryStorage) Get(key string) (any, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, found := s.items[key]
	if !found || item.isExpired() {
		return nil, false, nil
	}
	return item.value, true, nil
}

func (s *memoryStorage) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

func (s *memoryStorage) Close() error {
	close(s.stop)
	return nil
}

func (s *memoryStorage) runGC(interval time.Duration) {
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

func (s *memoryStorage) deleteExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, item := range s.items {
		if item.isExpired() {
			delete(s.items, key)
		}
	}
}
