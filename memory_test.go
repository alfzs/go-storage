package storage_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alfzs/go-storage"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorage_StringOperations(t *testing.T) {
	s, _ := storage.NewMemory[string](50 * time.Millisecond)
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "foo", "bar", 0))

	val, found, err := s.Get(ctx, "foo")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "bar", val)

	require.NoError(t, s.Delete(ctx, "foo"))

	_, found, _ = s.Get(ctx, "foo")
	require.False(t, found)
}

func TestMemoryStorage_StructOperations(t *testing.T) {
	type testStruct struct {
		Name string
		Age  int
	}

	s, _ := storage.NewMemory[testStruct](50 * time.Millisecond)
	defer s.Close()
	ctx := context.Background()

	testData := testStruct{Name: "Bob", Age: 30}
	require.NoError(t, s.Set(ctx, "struct", testData, 0))

	val, found, err := s.Get(ctx, "struct")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, testData, val)
}

func TestMemoryStorage_TTLExpiration(t *testing.T) {
	s, _ := storage.NewMemory[string](10 * time.Millisecond)
	defer s.Close()
	ctx := context.Background()

	require.NoError(t, s.Set(ctx, "temp", "value", 20*time.Millisecond))
	time.Sleep(50 * time.Millisecond)

	_, found, _ := s.Get(ctx, "temp")
	require.False(t, found)
}

func TestMemoryStorage_ConcurrentAccess(t *testing.T) {
	s, _ := storage.NewMemory[int](1 * time.Second)
	defer s.Close()
	ctx := context.Background()

	wg := sync.WaitGroup{}
	key := "concurrent"

	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			require.NoError(t, s.Set(ctx, key, i, 0))
			val, found, err := s.Get(ctx, key)
			require.NoError(t, err)
			require.True(t, found)
			_ = val
		}(i)
	}

	wg.Wait()
}
