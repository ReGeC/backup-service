package notifier

import (
	"backup-service/internal/config"
	"backup-service/internal/config/loader"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const Telegram = "telegram"

var ErrTelegramDisabled = errors.Join(ErrDisabled, errors.New(Telegram))

var telegramConfigLoader config.ConfigLoader = &loader.EnvLoader{}

func init() {
	// Автоматическая регистрация при импорте
	Register(Telegram, newTelegramNotifier)
}

func newTelegramNotifier() (Notifier, error) {
	cfg, enabled, err := config.NewTelegramConfigWithLoader(telegramConfigLoader)
	if err != nil {
		return nil, err
	}

	if !enabled {
		return nil, ErrTelegramDisabled
	}

	client := &http.Client{Timeout: 10 * time.Second}

	return NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID, client), nil
}

type TelegramNotifier struct {
	botToken string
	chatID   string
	client   *http.Client
}

func NewTelegramNotifier(telegramBotToken, telegramChatID string, client *http.Client) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: telegramBotToken,
		chatID:   telegramChatID,
		client:   client,
	}
}

// Send отправляет сообщение в Telegram
func (t *TelegramNotifier) Send(ctx context.Context, message string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

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
