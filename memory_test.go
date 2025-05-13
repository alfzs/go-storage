package storage_test

import (
	"sync"
	"testing"
	"time"

	"github.com/alfzs/go-storage"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_BasicOperations(t *testing.T) {
	s := storage.NewMemory(50 * time.Millisecond)

	require.NoError(t, s.Set("foo", "bar", 0))

	val, found, err := s.Get("foo")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "bar", val)

	require.NoError(t, s.Delete("foo"))

	_, found, _ = s.Get("foo")
	require.False(t, found)

	require.NoError(t, s.Close())
}

func TestMemoryStorage_TTLExpiration(t *testing.T) {
	s := storage.NewMemory(10 * time.Millisecond)

	require.NoError(t, s.Set("temp", "value", 20*time.Millisecond))
	time.Sleep(50 * time.Millisecond)

	_, found, _ := s.Get("temp")
	require.False(t, found)
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s := storage.NewMemory(1 * time.Second)
	defer s.Close()

	wg := sync.WaitGroup{}
	key := "concurrent"

	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			require.NoError(t, s.Set(key, i, 0))
			val, found, err := s.Get(key)
			require.NoError(t, err)
			require.True(t, found)
			_ = val
		}(i)
	}

	wg.Wait()
}
