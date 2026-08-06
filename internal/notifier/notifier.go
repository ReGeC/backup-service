package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

var ErrDisabled = errors.New("Notifier is disabled: ")

//go:generate mockery
type Notifier interface {
	// Функция отправки уведомлений
	Send(ctx context.Context, message string) error
}

var registry = map[string]func() (Notifier, error){}
var mu sync.RWMutex

func Register(typ string, factory func() (Notifier, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()

	registry = make(map[string]func() (Notifier, error))
}

func InitNotifiers() map[string]Notifier {
	notifiers := make(map[string]Notifier)

	for typ := range registry {
		notifier, err := NewNotifier(typ)
		if err != nil {
			if !errors.Is(err, ErrDisabled) {
				slog.Error("Ошибка инициализации", "notifier", typ, "error", err)
			}
			continue
		}
		notifiers[typ] = notifier
	}

	return notifiers
}

func SendAll(notifiers map[string]Notifier, ctx context.Context, message string) {
	for typ, notifier := range notifiers {
		err := notifier.Send(ctx, message)
		if err != nil {
			slog.Warn("Уведомление не доставлено", "notifier", typ, "error", err)
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
