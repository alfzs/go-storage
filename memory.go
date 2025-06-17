package storage

import (
	"context"
	"sync"
	"time"
)

// memoryStorage представляет реализацию хранилища данных в памяти.
// Это обобщенная структура, которая может работать с любым типом данных T.
// Хранит данные в map для ключ-значение и map для очередей.
// Использует sync.RWMutex для безопасного доступа из разных горутин.
type memoryStorage[T any] struct {
	items   map[string]item[T] // Хранилище ключ-значение
	queues  map[string][]T     // Хранилище очередей (имя очереди -> элементы)
	itemMu  sync.RWMutex       // Мьютекс для доступа к items
	queueMu sync.RWMutex       // Мьютекс для доступа к queues
	stop    chan struct{}      // Канал для остановки сборщика мусора
}

// newMemoryStorage создает новый экземпляр in-memory хранилища.
// Принимает интервал очистки устаревших элементов и возвращает интерфейс Storage[T].
// Запускает фоновую горутину для периодической очистки устаревших элементов.
func newMemoryStorage[T any](cleanupInterval time.Duration) Storage[T] {
	s := &memoryStorage[T]{
		items:  make(map[string]item[T]),
		queues: make(map[string][]T),
		stop:   make(chan struct{}),
	}
	go s.runGC(cleanupInterval) // Запускаем сборщик мусора
	return s
}

// item представляет элемент хранилища с значением и временем истечения срока жизни.
type item[T any] struct {
	value      T     // Значение элемента
	expiration int64 // Время истечения в наносекундах (0 - бессрочно)
}

// isExpired проверяет, истек ли срок жизни элемента.
// Возвращает true, если expiration > 0 и текущее время превышает expiration.
func (i item[T]) isExpired() bool {
	return i.expiration > 0 && time.Now().UnixNano() > i.expiration
}

// Close останавливает фоновый сборщик мусора и освобождает ресурсы.
// Должен вызываться при завершении работы с хранилищем.
func (s *memoryStorage[T]) Close() error {
	close(s.stop) // Посылаем сигнал остановки сборщику мусора
	return nil
}

// runGC запускает сборщик мусора, который периодически удаляет устаревшие элементы.
// Работает в фоновой горутине до получения сигнала остановки.
func (s *memoryStorage[T]) runGC(interval time.Duration) {
	ticker := time.NewTicker(interval) // Таймер для периодического запуска
	defer ticker.Stop()                // Освобождаем ресурсы таймера при остановке

	for {
		select {
		case <-ticker.C: // По истечении интервала
			s.deleteExpired() // Удаляем устаревшие элементы
		case <-s.stop: // При получении сигнала остановки
			return // Завершаем работу горутины
		}
	}
}

// Set сохраняет значение в хранилище по указанному ключу.
// Принимает контекст, ключ, значение и время жизни записи (TTL).
// Если TTL > 0, устанавливает время жизни записи, иначе запись хранится бессрочно.
func (s *memoryStorage[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano() // Вычисляем время истечения
	}

	s.itemMu.Lock()         // Блокируем на запись
	defer s.itemMu.Unlock() // Гарантируем разблокировку

	s.items[key] = item[T]{
		value:      value,
		expiration: expiration,
	}
	return nil
}

// Get получает значение из хранилища по ключу.
// Возвращает значение, флаг наличия значения и ошибку.
// Если ключ не найден или срок действия истек, возвращает false во втором возвращаемом значении.
func (s *memoryStorage[T]) Get(ctx context.Context, key string) (T, bool, error) {
	s.itemMu.RLock()         // Блокируем на чтение
	defer s.itemMu.RUnlock() // Гарантируем разблокировку

	var zero T // Нулевое значение типа T для возврата по умолчанию
	item, found := s.items[key]
	if !found || item.isExpired() {
		return zero, false, nil
	}
	return item.value, true, nil
}

// Delete удаляет значение из хранилища по ключу.
// Возвращает ошибку, если операция не удалась.
func (s *memoryStorage[T]) Delete(ctx context.Context, key string) error {
	s.itemMu.Lock()         // Блокируем на запись
	defer s.itemMu.Unlock() // Гарантируем разблокировку
	delete(s.items, key)
	return nil
}

// Enqueue добавляет элемент в конец очереди.
// Принимает имя очереди и значение для добавления.
// Если очередь не существует, создает новую.
func (s *memoryStorage[T]) Enqueue(ctx context.Context, queueName string, value T) error {
	s.queueMu.Lock()         // Блокируем на запись
	defer s.queueMu.Unlock() // Гарантируем разблокировку

	s.queues[queueName] = append(s.queues[queueName], value)
	return nil
}

// Dequeue извлекает и удаляет элемент из начала очереди.
// Возвращает элемент, флаг наличия элемента и ошибку.
// Если очередь пуста, возвращает false во втором возвращаемом значении.
func (s *memoryStorage[T]) Dequeue(ctx context.Context, queueName string) (T, bool, error) {
	s.queueMu.Lock()         // Блокируем на запись
	defer s.queueMu.Unlock() // Гарантируем разблокировку

	var zero T
	queue, exists := s.queues[queueName]
	if !exists || len(queue) == 0 {
		return zero, false, nil
	}

	value := queue[0]
	s.queues[queueName] = queue[1:] // Удаляем первый элемент сдвигом слайса

	// Оптимизация: если очередь пуста, удаляем её из мапы
	if len(s.queues[queueName]) == 0 {
		delete(s.queues, queueName)
	}

	return value, true, nil
}

// Peek возвращает первый элемент из очереди без его удаления.
// Возвращает элемент, флаг наличия элемента и ошибку.
// Если очередь пуста, возвращает false во втором возвращаемом значении.
func (s *memoryStorage[T]) Peek(ctx context.Context, queueName string) (T, bool, error) {
	s.queueMu.RLock()         // Блокируем на чтение
	defer s.queueMu.RUnlock() // Гарантируем разблокировку

	var zero T
	queue, exists := s.queues[queueName]
	if !exists || len(queue) == 0 {
		return zero, false, nil
	}

	return queue[0], true, nil
}

// Remove удаляет первый элемент из очереди без его возврата.
// Возвращает флаг успешности операции и ошибку.
// Если очередь пуста, возвращает false в первом возвращаемом значении.
func (s *memoryStorage[T]) Remove(ctx context.Context, queueName string) (bool, error) {
	s.queueMu.Lock()         // Блокируем на запись
	defer s.queueMu.Unlock() // Гарантируем разблокировку

	queue, exists := s.queues[queueName]
	if !exists || len(queue) == 0 {
		return false, nil
	}

	s.queues[queueName] = queue[1:] // Удаляем первый элемент сдвигом слайса

	// Оптимизация: если очередь пуста, удаляем её из мапы
	if len(s.queues[queueName]) == 0 {
		delete(s.queues, queueName)
	}

	return true, nil
}

// QueueLen возвращает текущую длину очереди.
// Возвращает количество элементов в очереди и ошибку, если операция не удалась.
// Если очередь не существует, возвращает 0.
func (s *memoryStorage[T]) QueueLen(ctx context.Context, queueName string) (int64, error) {
	s.queueMu.RLock()         // Блокируем на чтение
	defer s.queueMu.RUnlock() // Гарантируем разблокировку

	queue, exists := s.queues[queueName]
	if !exists {
		return 0, nil
	}
	return int64(len(queue)), nil
}

// deleteExpired удаляет все элементы с истекшим сроком жизни из хранилища.
// Вызывается периодически сборщиком мусора.
func (s *memoryStorage[T]) deleteExpired() {
	s.itemMu.Lock()         // Блокируем на запись
	defer s.itemMu.Unlock() // Гарантируем разблокировку

	for key, item := range s.items {
		if item.isExpired() {
			delete(s.items, key) // Удаляем устаревший элемент
		}
	}
}
