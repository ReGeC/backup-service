// internal/notifier/noop.go (опционально)
package notifier

import "context"

const Noop = "noop"

func init() {
	Register(Noop, func() (Notifier, error) {
		return &NoopNotifier{}, nil
	})
}

// NoopNotifier — заглушка, которая ничего не делает
type NoopNotifier struct{}

func (n *NoopNotifier) Send(ctx context.Context, message string) error {
	// Ничего не делаем
	return nil
}
