package storage

import (
	"context"
	"sync"
	"time"
)

type memoryStorage[T any] struct {
	items   map[string]item[T]
	queues  map[string][]T
	itemMu  sync.RWMutex
	queueMu sync.RWMutex
	stop    chan struct{}
}

func newMemoryStorage[T any](cleanupInterval time.Duration) Storage[T] {
	s := &memoryStorage[T]{
		items:  make(map[string]item[T]),
		queues: make(map[string][]T),
		stop:   make(chan struct{}),
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

	s.itemMu.Lock()
	defer s.itemMu.Unlock()

	s.items[key] = item[T]{
		value:      value,
		expiration: expiration,
	}
	return nil
}

func (s *memoryStorage[T]) Get(ctx context.Context, key string) (T, bool, error) {
	s.itemMu.RLock()
	defer s.itemMu.RUnlock()

	var zero T
	item, found := s.items[key]
	if !found || item.isExpired() {
		return zero, false, nil
	}
	return item.value, true, nil
}

func (s *memoryStorage[T]) Delete(ctx context.Context, key string) error {
	s.itemMu.Lock()
	defer s.itemMu.Unlock()
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

// Enqueue добавляет элемент в конец очереди
func (s *memoryStorage[T]) Enqueue(ctx context.Context, queueName string, value T) error {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	s.queues[queueName] = append(s.queues[queueName], value)
	return nil
}

// Dequeue извлекает и удаляет элемент из начала очереди
func (s *memoryStorage[T]) Dequeue(ctx context.Context, queueName string) (T, bool, error) {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	var zero T
	queue, exists := s.queues[queueName]
	if !exists || len(queue) == 0 {
		return zero, false, nil
	}

	value := queue[0]
	s.queues[queueName] = queue[1:] // Удаляем первый элемент

	// Если очередь пуста, удаляем её из мапы
	if len(s.queues[queueName]) == 0 {
		delete(s.queues, queueName)
	}

	return value, true, nil
}

// QueueLength возвращает текущую длину очереди
func (s *memoryStorage[T]) QueueLen(ctx context.Context, queueName string) (int64, error) {
	s.queueMu.RLock()
	defer s.queueMu.RUnlock()

	queue, exists := s.queues[queueName]
	if !exists {
		return 0, nil
	}
	return int64(len(queue)), nil
}

func (s *memoryStorage[T]) deleteExpired() {
	s.itemMu.Lock()
	defer s.itemMu.Unlock()

	for key, item := range s.items {
		if item.isExpired() {
			delete(s.items, key)
		}
	}
}
