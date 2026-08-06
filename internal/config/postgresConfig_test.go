package config_test

import (
	"testing"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
)

func TestPotgresConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      config.PostgresConfig
		wantErr     error
	} {
		{
			name:        "valid config",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "localhost",
				PGPort:      5432,
				PGUser:      "postgres",
				PGPassword:  "password",
				PGDatabase:  "backup",
			},
			wantErr:     nil,
		},
		{
			name:        "empty host",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "",
				PGPort:      5432,
				PGUser:      "postgres",
				PGPassword:  "password",
				PGDatabase:  "backup",
			},
			wantErr:     config.ErrEmptyPGHost,
		},
		{
			name:        "port is zero",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "localhost",
				PGPort:      0,
				PGUser:      "postgres",
				PGPassword:  "password",
				PGDatabase:  "backup",
			},
			wantErr:     config.ErrInvalidPGPort,
		},
		{
			name:        "port is greater than maximum",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "localhost",
				PGPort:      65536,
				PGUser:      "postgres",
				PGPassword:  "password",
				PGDatabase:  "backup",
			},
			wantErr:     config.ErrInvalidPGPort,
		},
		{
			name:        "empty user",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "localhost",
				PGPort:      5432,
				PGUser:      "",
				PGPassword:  "password",
				PGDatabase:  "backup",
			},
			wantErr:     config.ErrEmptyPGUser,
		},
		{
			name:        "empty database",
			config: config.PostgresConfig{
				PGEnable:    true,
				PGHost:      "localhost",
				PGPort:      5432,
				PGUser:      "postgres",
				PGPassword:  "password",
				PGDatabase:  "",
			},
			wantErr:     config.ErrEmptyPGDatabase,
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

func TestNewPostgresConfigWithLoader(t *testing.T) {
	tests := []struct {
		name        string
		pgEnable    bool
		pgHost      string
		pgPort      int
		pgUser      string
		pgPassword  string
		pgDatabase  string
		wantEnabled bool
	}{
		{
			name:        "postgres disabled",
			pgEnable:    false,
			wantEnabled: false,
		},
		{
			name:        "postgres enabled",
			pgEnable:    true,
			pgHost:      "localhost",
			pgPort:      5432,
			pgUser:      "postgres",
			pgPassword:  "password",
			pgDatabase:  "backup",
			wantEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := mocks.NewMockConfigLoader(t)

			loader.EXPECT().
				GetBool([]string{"pg", "enable"}, false).
				Return(tt.pgEnable)

			if tt.pgEnable {
				loader.EXPECT().
					GetString([]string{"pg", "host"}, "localhost").
					Return(tt.pgHost)

				loader.EXPECT().
					GetInt([]string{"pg", "port"}, 5432).
					Return(tt.pgPort)

				loader.EXPECT().
					GetString([]string{"pg", "user"}, "postgres").
					Return(tt.pgUser)

				loader.EXPECT().
					GetString([]string{"pg", "password"}, "").
					Return(tt.pgPassword)

				loader.EXPECT().
					GetString([]string{"pg", "database"}, "postgres").
					Return(tt.pgDatabase)
			}

			cfg, enabled, err := config.NewPostgresConfigWithLoader(loader)

			require.NotNil(t, cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, enabled)

			if tt.pgEnable {
				assert.Equal(t, tt.pgHost, cfg.PGHost)
				assert.Equal(t, tt.pgPort, cfg.PGPort)
				assert.Equal(t, tt.pgUser, cfg.PGUser)
				assert.Equal(t, tt.pgPassword, cfg.PGPassword)
				assert.Equal(t, tt.pgDatabase, cfg.PGDatabase)
			}
		})
	}
}

func TestNewPostgresConfig (t *testing.T) {
	t.Run("Создание postgres конфига с реальным envloader", func (t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("PG_ENABLE", "true")
		t.Setenv("PG_HOST", "localhost")
		t.Setenv("PG_PORT", "5432")
		t.Setenv("PG_USER", "testuser")
		t.Setenv("PG_PASSWORD", "testpass")
		t.Setenv("PG_DATABASE", "testdb")

		cfg, enabled, err := config.NewPostgresConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, enabled)

		// Проверяем, что поля установились корректно
		assert.Equal(t, "localhost", cfg.PGHost)
		assert.Equal(t, 5432, cfg.PGPort)
		assert.Equal(t, "testuser", cfg.PGUser)
		assert.Equal(t, "testpass", cfg.PGPassword)
		assert.Equal(t, "testdb", cfg.PGDatabase)
		assert.True(t, cfg.PGEnable)
	})
}