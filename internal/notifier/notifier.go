package notifier

import (
	"context"
	"fmt"
	"log"
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

func InitNotifiers() map[string]Notifier {
	notifiers := make(map[string]Notifier)

	for typ := range registry {
		notifier, err := NewNotifier(typ)
		if err != nil {
			log.Printf("Ошибка инициализации %s: %v", typ, err)
		}
		notifiers[typ] = notifier
	}

	return notifiers
}

func SendAll(notifiers map[string]Notifier, ctx context.Context, message string) {
	for typ, notifier := range notifiers {
		err := notifier.Send(ctx, message)
		if err != nil {
			log.Printf("Уведомление не %s доставлено: %v", typ, err)
		}
	}
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