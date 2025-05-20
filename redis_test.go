package storage_test

import (
	"sync"
	"testing"
	"time"

	"github.com/alfzs/go-storage"
	"github.com/stretchr/testify/require"
)

func newTestRedisStorage[T any](t *testing.T) storage.Storage[T] {
	s, err := storage.NewRedis[T](storage.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	require.NoError(t, err)
	return s
}

func TestRedisStorage_StringOperations(t *testing.T) {
	s := newTestRedisStorage[string](t)
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

func TestRedisStorage_StructOperations(t *testing.T) {
	type testStruct struct {
		Name string
		Age  int
	}

	s := newTestRedisStorage[testStruct](t)
	defer s.Close()

	testData := testStruct{Name: "Alice", Age: 25}
	require.NoError(t, s.Set("struct", testData, 0))

	val, found, err := s.Get("struct")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, testData, val)
}

func TestRedisStorage_TTL(t *testing.T) {
	s := newTestRedisStorage[string](t)
	defer s.Close()

	require.NoError(t, s.Set("temp", "value", 1*time.Second))

	time.Sleep(2 * time.Second)

	_, found, _ := s.Get("temp")
	require.False(t, found)
}

func TestRedisStorage_ConcurrentAccess(t *testing.T) {
	s := newTestRedisStorage[int](t)
	defer s.Close()

	key := "concurrent"
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			require.NoError(t, s.Set(key, i, 0))
			val, found, err := s.Get(key)
			require.NoError(t, err)
			require.True(t, found)
			_ = val // Проверка значения не имеет смысла в конкурентном тесте
		}(i)
	}

	wg.Wait()
}
