package notifier

import (
	"backup-service/internal/config"
	"fmt"
	"encoding/json"
	"bytes"
	"context"
	"net/http"
	"time"
)

const Telegram = "telegram"

func init() {
	cfg, enabled, err := config.NewTelegramConfig()
	if enabled {
		Register(Telegram, func() (Notifier, error) {
			if err != nil {
				return nil, fmt.Errorf("Ошибка конфигурации telegram: %w", err)
			}
			return NewTelegramNotifier(cfg), nil
		})
	}
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(cfg *config.TelegramConfig) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: cfg.TelegramBotToken,
		chatID:   cfg.TelegramChatID,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Send отправляет сообщение в Telegram
func (t *TelegramNotifier) Send(ctx context.Context, message string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.botToken)

	payload := map[string]string{
		"chat_id":    t.chatID,
		"text":       message,
		"parse_mode": "HTML",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("Ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("Ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("Ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ошибка отправки сообщения: статус %d", resp.StatusCode)
	}

	return nil
}