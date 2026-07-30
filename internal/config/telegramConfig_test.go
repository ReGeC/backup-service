package config_test

import (
	"testing"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestTelegramConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.TelegramConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: config.TelegramConfig{
				TelegramEnable:   true,
				TelegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				TelegramChatID:   "123456789",
			},
			wantErr: nil,
		},
		{
			name: "empty bot token",
			config: config.TelegramConfig{
				TelegramEnable:   true,
				TelegramBotToken: "",
				TelegramChatID:   "123456789",
			},
			wantErr: config.ErrEmptyTelegramBotToken,
		},
		{
			name: "empty chat id",
			config: config.TelegramConfig{
				TelegramEnable:   true,
				TelegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
				TelegramChatID:   "",
			},
			wantErr: config.ErrEmptyTelegramChatID,
		},
		{
			name: "both empty",
			config: config.TelegramConfig{
				TelegramEnable:   true,
				TelegramBotToken: "",
				TelegramChatID:   "",
			},
			wantErr: config.ErrEmptyTelegramBotToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNewTelegramConfigWithLoader(t *testing.T) {
	tests := []struct {
		name              string
		telegramEnable    bool
		telegramBotToken  string
		telegramChatID    string
		wantEnabled       bool
	}{
		{
			name:           "telegram disabled",
			telegramEnable: false,
			wantEnabled:    false,
		},
		{
			name:             "telegram enabled",
			telegramEnable:   true,
			telegramBotToken: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
			telegramChatID:   "123456789",
			wantEnabled:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := mocks.NewMockConfigLoader(t)

			loader.EXPECT().
				GetEnvAsBool("TELEGRAM_ENABLE", false).
				Return(tt.telegramEnable)

			if tt.telegramEnable {
				loader.EXPECT().
					GetEnv("TELEGRAM_BOT_TOKEN", "").
					Return(tt.telegramBotToken)

				loader.EXPECT().
					GetEnv("TELEGRAM_CHAT_ID", "").
					Return(tt.telegramChatID)
			}

			cfg, enabled, err := config.NewTelegramConfigWithLoader(loader)

			require.NotNil(t, cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, enabled)

			if tt.telegramEnable {
				assert.Equal(t, tt.telegramBotToken, cfg.TelegramBotToken)
				assert.Equal(t, tt.telegramChatID, cfg.TelegramChatID)
			}
		})
	}
}

func TestNewTelegramConfig(t *testing.T) {
	t.Run("Создание telegram конфига с реальным envloader", func(t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("TELEGRAM_ENABLE", "true")
		t.Setenv("TELEGRAM_BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
		t.Setenv("TELEGRAM_CHAT_ID", "123456789")

		cfg, enabled, err := config.NewTelegramConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, enabled)
		
		// Проверяем, что поля установились корректно
		assert.Equal(t, "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", cfg.TelegramBotToken)
		assert.Equal(t, "123456789", cfg.TelegramChatID)
		assert.True(t, cfg.TelegramEnable)
	})
}