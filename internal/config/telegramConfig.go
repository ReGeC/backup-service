package config

import (
	"errors"

	"backup-service/internal/config/loader"
)

var (
	ErrEmptyTelegramBotToken = errors.New("telegram bot token is empty")
	ErrEmptyTelegramChatID = errors.New("telegram chat id is empty")
)

type TelegramConfig struct {
	loader ConfigLoader

	TelegramEnable bool

	TelegramBotToken string
	TelegramChatID   string
}

func (t *TelegramConfig) LoadConfig() (bool, error) {
	t.TelegramEnable = t.loader.GetBool([]string{"telegram", "enable"}, false)
	if !t.TelegramEnable {
		return false, nil
	}

	t.TelegramBotToken = t.loader.GetString([]string{"telegram", "bot_token"}, "")
	t.TelegramChatID = t.loader.GetString([]string{"telegram", "chat_id"}, "")

	return true, t.ValidateConfig()
}

func (t *TelegramConfig) ValidateConfig() error {
	if t.TelegramBotToken == "" {
		return ErrEmptyTelegramBotToken
	}
	if t.TelegramChatID == "" {
		return ErrEmptyTelegramChatID
	}

	return nil
}

func NewTelegramConfig() (*TelegramConfig, bool, error) {
	return NewTelegramConfigWithLoader(&loader.EnvLoader{})
}

func NewTelegramConfigWithLoader(loader ConfigLoader) (*TelegramConfig, bool, error) {
	telegramCfg := &TelegramConfig{loader: loader}
	enabled, err := telegramCfg.LoadConfig()
	return telegramCfg, enabled, err
}
