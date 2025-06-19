package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisStorage представляет реализацию хранилища данных на основе Redis.
// Это обобщенная структура, которая может работать с любым типом данных T.
type redisStorage[T any] struct {
	client *redis.Client // Клиент Redis для выполнения операций
}

// newRedisStorage создает новый экземпляр Redis-хранилища.
// Принимает конфигурацию RedisConfig и возвращает интерфейс Storage[T].
// Выполняет проверку соединения с Redis через команду PING.
func newRedisStorage[T any](cfg RedisConfig) (Storage[T], error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,     // Адрес Redis сервера
		Username: cfg.Username, // Имя пользователя
		Password: cfg.Password, // Пароль (если требуется)
		DB:       cfg.DB,       // Номер базы данных
	})
	ctx := context.Background()

	// Проверяем соединение с Redis
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &redisStorage[T]{client: client}, nil
}

// Set сохраняет значение в Redis по указанному ключу.
// Принимает контекст, ключ, значение и время жизни записи (TTL).
// Если TTL > 0, устанавливает время жизни записи, иначе использует redis.KeepTTL.
// Значение сериализуется в JSON перед сохранением.
func (s *redisStorage[T]) Set(ctx context.Context, key string, value T, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Сериализуем значение в JSON
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	var redisErr error
	if ttl > 0 {
		redisErr = s.client.Set(ctx, key, data, ttl).Err()
	} else {
		redisErr = s.client.Set(ctx, key, data, redis.KeepTTL).Err()
	}

	if redisErr != nil {
		return fmt.Errorf("redis set failed: %w", redisErr)
	}

	return nil
}

// Get получает значение из Redis по ключу.
// Возвращает значение, флаг наличия значения и ошибку.
// Если ключ не найден, возвращает false во втором возвращаемом значении.
// Значение десериализуется из JSON перед возвратом.
func (s *redisStorage[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var zero T // Нулевое значение типа T для возврата по умолчанию

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return zero, false, nil // Ключ не найден - это не ошибка
	}
	if err != nil {
		return zero, false, fmt.Errorf("redis get failed: %w", err)
	}

	var out T
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

// Delete удаляет значение из Redis по ключу.
// Возвращает ошибку, если операция не удалась.
func (s *redisStorage[T]) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}
	return nil
}

// Enqueue добавляет элемент в конец очереди (списка) Redis.
// Принимает имя очереди и значение для добавления.
// Значение сериализуется в JSON перед добавлением.
func (s *redisStorage[T]) Enqueue(ctx context.Context, queueName string, value T) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	// Используем RPush для добавления в конец списка
	if err := s.client.RPush(ctx, queueName, data).Err(); err != nil {
		return fmt.Errorf("redis rpush failed: %w", err)
	}

	return nil
}

// Dequeue извлекает и удаляет элемент из начала очереди (списка) Redis.
// Возвращает элемент, флаг наличия элемента и ошибку.
// Если очередь пуста, возвращает false во втором возвращаемом значении.
// Значение десериализуется из JSON перед возвратом.
func (s *redisStorage[T]) Dequeue(ctx context.Context, queueName string) (T, bool, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Используем LPop для извлечения из начала списка
	val, err := s.client.LPop(ctx, queueName).Result()
	if err == redis.Nil {
		return zero, false, nil // Очередь пуста - это не ошибка
	}
	if err != nil {
		return zero, false, fmt.Errorf("redis lpop failed: %w", err)
	}

	var out T
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

// Peek получает элемент из начала очереди без его удаления.
// Возвращает элемент, флаг наличия элемента и ошибку.
// Если очередь пуста, возвращает false во втором возвращаемом значении.
// Значение десериализуется из JSON перед возвратом.
func (s *redisStorage[T]) Peek(ctx context.Context, queueName string) (T, bool, error) {
	var zero T

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Используем LIndex с индексом 0 для получения первого элемента
	val, err := s.client.LIndex(ctx, queueName, 0).Result()
	if err == redis.Nil {
		return zero, false, nil // Очередь пуста - это не ошибка
	}
	if err != nil {
		return zero, false, fmt.Errorf("redis lindex failed: %w", err)
	}

	var out T
	if err := json.Unmarshal([]byte(val), &out); err != nil {
		return zero, false, fmt.Errorf("unmarshal failed: %w", err)
	}

	return out, true, nil
}

// Remove удаляет один элемент из начала очереди без возврата его значения.
// Возвращает флаг успешности операции и ошибку.
// Если очередь пуста, возвращает false в первом возвращаемом значении.
func (s *redisStorage[T]) Remove(ctx context.Context, queueName string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// Используем LPop, но игнорируем возвращаемое значение
	_, err := s.client.LPop(ctx, queueName).Result()
	if err == redis.Nil {
		return false, nil // Очередь пуста - считаем это успешной операцией
	}
	if err != nil {
		return false, fmt.Errorf("redis lpop failed: %w", err)
	}

	return true, nil
}

// QueueLen возвращает текущую длину очереди.
// Возвращает количество элементов в очереди и ошибку, если операция не удалась.
func (s *redisStorage[T]) QueueLen(ctx context.Context, queueName string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	length, err := s.client.LLen(ctx, queueName).Result()
	if err != nil {
		return 0, fmt.Errorf("redis llen failed: %w", err)
	}

	return length, nil
}

// Close закрывает соединение с Redis.
// Должен вызываться при завершении работы с хранилищем.
func (s *redisStorage[T]) Close() error {
	return s.client.Close()
}
