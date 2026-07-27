package notifier

import (
	"context"
	"fmt"
	"sync"
)

type NotifierType string

type Notifier interface {
	// Функция отправки уведомлений
	Send (ctx context.Context, message string) error
}

var registry = map[NotifierType]func() (Notifier, error){}
var mu sync.RWMutex

func Register(typ NotifierType, factory func() (Notifier, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func NewNotifier(typ NotifierType) (Notifier, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, ok := registry[typ]
	if !ok {
		return nil, fmt.Errorf("Неизвестный тип уведомителя: %s", typ)
	}
	return factory()
}