package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"
)

func TestBackupConfig_ValidateConfig(t *testing.T) {
	tests := []struct {
		name string
		config config.BackupConfig
		wantErr error
	} {
		{
			name: "valid config",
			config: config.BackupConfig{
				StorageType:    "anything",
				BackupPath:     "./backups",
				RetentionDays:  7,
				CronEnable:     true,
				BackupSchedule: "0 0 3 * * *",
			},
			wantErr: nil,
		},
		{
			name: "empty backup path",
			config: config.BackupConfig{
				StorageType:   "anything",
				RetentionDays: 7,
			},
			wantErr: config.ErrEmptyBackupPath,
		},
		{
			name: "zero retention days",
			config: config.BackupConfig{
				StorageType: "anything",
				BackupPath:  "./backups",
			},
			wantErr: config.ErrInvalidRetention,
		},
		{
			name: "negative retention days",
			config: config.BackupConfig{
				StorageType:   "anything",
				BackupPath:    "./backups",
				RetentionDays: -1,
			},
			wantErr: config.ErrInvalidRetention,
		},
		{
			name: "cron enabled without schedule",
			config: config.BackupConfig{
				StorageType:   "anything",
				BackupPath:    "./backups",
				RetentionDays: 7,
				CronEnable:    true,
			},
			wantErr: config.ErrEmptySchedule,
		},
		{
			name: "cron disabled without schedule",
			config: config.BackupConfig{
				StorageType:   "anything",
				BackupPath:    "./backups",
				RetentionDays: 7,
				CronEnable:    false,
			},
			wantErr: nil,
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

func TestNewBackupConfigWithLoader(t *testing.T) {
	loader := mocks.NewMockConfigLoader(t)

	loader.EXPECT().
		GetString([]string{"backup", "storage"}, "local").
		Return("s3")

	loader.EXPECT().
		GetString([]string{"backup", "temp_path"}, "./tmp/backups").
		Return("/tmp/backups")

	loader.EXPECT().
		GetInt([]string{"backup", "retention_days"}, 7).
		Return(30)

	loader.EXPECT().
		GetBool([]string{"cron", "enable"}, false).
		Return(true)

	loader.EXPECT().
		GetString([]string{"cron", "schedule"}, "0 0 3 * * *").
		Return("0 0 5 * * *")

	cfg, err := config.NewBackupConfigWithLoader(loader)

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "s3", cfg.StorageType)
	assert.Equal(t, "/tmp/backups", cfg.BackupPath)
	assert.Equal(t, 30, cfg.RetentionDays)
	assert.True(t, cfg.CronEnable)
	assert.Equal(t, "0 0 5 * * *", cfg.BackupSchedule)
}

func TestNewBackupConfig (t *testing.T) {
	t.Run("Создание backup конфига с реальным envloader", func (t *testing.T) {
		// Устанавливаем переменные окружения для теста
		t.Setenv("BACKUP_STORAGE", "s3")
		t.Setenv("BACKUP_TEMP_PATH", "/tmp/my-backups")
		t.Setenv("BACKUP_RETENTION_DAYS", "14")
		t.Setenv("CRON_ENABLE", "true")
		t.Setenv("CRON_SCHEDULE", "0 0 2 * * *")

		cfg, err := config.NewBackupConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)

		// Проверяем, что поля установились корректно
		assert.Equal(t, "s3", cfg.StorageType)
		assert.Equal(t, "/tmp/my-backups", cfg.BackupPath)
		assert.Equal(t, 14, cfg.RetentionDays)
		assert.True(t, cfg.CronEnable)
		assert.Equal(t, "0 0 2 * * *", cfg.BackupSchedule)
	})
}