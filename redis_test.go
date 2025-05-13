package storage_test

import (
	"testing"
	"time"

	"github.com/alfzs/go-storage"
	"github.com/stretchr/testify/require"
)

func newTestRedisStorage(t *testing.T) storage.Storage {
	s, err := storage.NewRedis(storage.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	require.NoError(t, err)
	return s
}

func TestRedisStorage_BasicOperations(t *testing.T) {
	s := newTestRedisStorage(t)
	defer s.Close()

	require.NoError(t, s.Set("foo", "bar", 0))

	val, found, err := s.Get("foo")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "bar", val)

	require.NoError(t, s.Delete("foo"))

	_, found, _ = s.Get("foo")
	require.False(t, found)
}

func TestRedisStorage_TTL(t *testing.T) {
	s := newTestRedisStorage(t)
	defer s.Close()

	require.NoError(t, s.Set("temp", "value", 1*time.Second))

	time.Sleep(2 * time.Second)

	_, found, _ := s.Get("temp")
	require.False(t, found)
}

func TestRedisStorage_ConcurrentAccess(t *testing.T) {
	s := newTestRedisStorage(t)
	defer s.Close()

	key := "concurrent"
	for i := range 100 {
		go func(i int) {
			_ = s.Set(key, i, 0)
			_, _, _ = s.Get(key)
		}(i)
	}
	time.Sleep(1 * time.Second)
}
