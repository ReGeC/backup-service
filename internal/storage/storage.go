package storage

import (
	"context"
	"fmt"
	"time"
	"sync"
)

type StorageType string

type Storage interface {
	Save(ctx context.Context, localPath string) (string, error)
	List(ctx context.Context) ([]FileInfo, error)
	Delete(ctx context.Context, path string) error
}

type FileInfo struct {
	Name string
	Size int64
	CreatedAt time.Time
}


var registry = map[StorageType]func() (Storage, error){}
var mu sync.RWMutex

func Register(storageType StorageType, factory func() (Storage, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[storageType] = factory
}

func NewStorage(storageType StorageType) (Storage, error) {
	mu.Lock()
	defer mu.Unlock()

	factory, exists := registry[storageType]
	if !exists {
		return nil, fmt.Errorf("неизвестный тип хранилища: %s", storageType)
	}
	return factory()
}