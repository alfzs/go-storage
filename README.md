# go-storage

Универсальный абстрактный интерфейс `Storage` с двумя реализациями:

- In-memory (в памяти, с TTL и GC)
- Redis (через `go-redis/v9`)

Подходит для временного хранения, кэширования и простых KV-хранилищ.

## Интерфейс

```go
type Storage interface {
	Set(key string, value any, ttl time.Duration) error
	Get(key string) (any, bool, error)
	Delete(key string) error
	Close() error
}
```

## Установка

```bash
go get github.com/alfzs/go-storage
```

## Использование

```go
package main

import (
	"fmt"
	"time"
	"github.com/alfzs/go-storage"
)

func main() {
	store, err := storage.New(storage.TypeMemory, 2*time.Minute)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	err = store.Set("key", "value", 10*time.Second)
	if err != nil {
		panic(err)
	}

	val, found, err := store.Get("key")
	if err != nil {
		panic(err)
	}
	if found {
		fmt.Println("Found:", val)
	}
}
```

## Redis

```go
store, err := storage.New(storage.TypeRedis, storage.RedisConfig{
    Addr: "localhost:6379",
    Password: "",
    DB:   0,
})
```

## Лицензия

MIT
