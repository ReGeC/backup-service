package config_test

import (
	"testing"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestSQLiteConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  config.SQLiteConfig
		wantErr error
	}{
		{
			name: "valid config",
			config: config.SQLiteConfig{
				SQLiteEnable: true,
				SQLitePath:   "./test.db",
			},
			wantErr: nil,
		},
		{
			name: "empty path",
			config: config.SQLiteConfig{
				SQLiteEnable: true,
				SQLitePath:   "",
			},
			wantErr: config.ErrEmptySQLitePath,
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

func TestNewSQLiteConfigWithLoader(t *testing.T) {
	tests := []struct {
		name         string
		sqliteEnable bool
		sqlitePath   string
		wantEnabled  bool
	}{
		{
			name:         "sqlite disabled",
			sqliteEnable: false,
			wantEnabled:  false,
		},
		{
			name:         "sqlite enabled",
			sqliteEnable: true,
			sqlitePath:   "./test.db",
			wantEnabled:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := mocks.NewMockConfigLoader(t)

			loader.EXPECT().
				GetEnvAsBool("SQLITE_ENABLE", false).
				Return(tt.sqliteEnable)

			if tt.sqliteEnable {
				loader.EXPECT().
					GetEnv("SQLITE_PATH", "./test.db").
					Return(tt.sqlitePath)
			}

			cfg, enabled, err := config.NewSQLiteConfigWithLoader(loader)

			require.NotNil(t, cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, enabled)

			if tt.sqliteEnable {
				assert.Equal(t, tt.sqlitePath, cfg.SQLitePath)
			}
		})
	}
}

func TestNewSQLiteConfig(t *testing.T) {
	t.Run("Создание sqlite конфига с реальным envloader", func(t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("SQLITE_ENABLE", "true")
		t.Setenv("SQLITE_PATH", "/tmp/test.db")

		cfg, enabled, err := config.NewSQLiteConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, enabled)

		// Проверяем, что поля установились корректно
		assert.Equal(t, "/tmp/test.db", cfg.SQLitePath)
		assert.True(t, cfg.SQLiteEnable)
	})
}