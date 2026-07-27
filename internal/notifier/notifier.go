package notifier

import (
	"context"
	"fmt"
	"sync"
)

type Notifier interface {
	// Функция отправки уведомлений
	Send (ctx context.Context, message string) error
}

var registry = map[string]func() (Notifier, error){}
var mu sync.RWMutex

func Register(typ string, factory func() (Notifier, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func NewNotifier(typ string) (Notifier, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, ok := registry[typ]
	if !ok {
		return nil, fmt.Errorf("Неизвестный тип уведомителя: %s", typ)
	}
	return factory()
}