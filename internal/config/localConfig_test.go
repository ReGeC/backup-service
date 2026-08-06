package config_test

import (
	"testing"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestLocalConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.LocalConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: config.LocalConfig{
				LocalStoragePath: "./backups",
			},
			wantErr: nil,
		},
		{
			name: "empty local storage path",
			config: config.LocalConfig{
				LocalStoragePath: "",
			},
			wantErr: config.ErrEmptyLocalStoragePath,
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

func TestNewLocalConfigWithLoader(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		wantPath      string
		wantEnabled   bool
		wantErr       error
	}{
		{
			name:        "valid config",
			envValue:    "/data/backups",
			wantPath:    "/data/backups",
			wantEnabled: true,
			wantErr:     nil,
		},
		{
			name:        "invalid config",
			envValue:    "",
			wantPath:    "",
			wantEnabled: false,
			wantErr:     config.ErrEmptyLocalStoragePath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := mocks.NewMockConfigLoader(t)

			loader.EXPECT().
				GetString([]string{"local_storage", "path"}, "./backups").
				Return(tt.envValue)

			cfg, enabled, err := config.NewLocalConfigWithLoader(loader)

			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantPath, cfg.LocalStoragePath)
			assert.Equal(t, tt.wantEnabled, enabled)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestNewLocalConfig (t *testing.T) {
	t.Run("Создание local конфига с реальным envloader", func (t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("LOCAL_STORAGE_PATH", "/var/backups")

		cfg, enabled, err := config.NewLocalConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, enabled)

		// Проверяем, что поля установились корректно
		assert.Equal(t, "/var/backups", cfg.LocalStoragePath)
	})
}