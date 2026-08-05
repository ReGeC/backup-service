package storage

import (
	"context"
	"fmt"
	"time"
	"sync"
)

//go:generate mockery
type Storage interface {
	Save(ctx context.Context, localPath string) (string, error)
	Download(ctx context.Context, path string) (string, error)
	List(ctx context.Context) ([]FileInfo, error)
	Delete(ctx context.Context, path string) error
}

type FileInfo struct {
	Name string
	Size int64
	CreatedAt time.Time
}


var registry = map[string]func() (Storage, error){}
var mu sync.RWMutex

func Register(typ string, factory func() (Storage, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()

	registry = make(map[string]func() (Storage, error))
}

func NewStorage(typ string) (Storage, error) {
	mu.Lock()
	defer mu.Unlock()

	factory, exists := registry[typ]
	if !exists {
		return nil, fmt.Errorf("неизвестный тип хранилища: %s", typ)
	}
	return factory()
}