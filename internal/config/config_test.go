package config_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"backup-service/internal/config"
	mocks "backup-service/internal/config/mocks"
)

func TestNewBackupConfig (t *testing.T) {
	t.Run("Создание конфига с реальным envloader", func (t *testing.T) {
		cfg, err := config.NewBackupConfig()

		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.NotNil(t, cfg.GetLoader())
	})
}

func TestNewBackupConfigWithLoader(t *testing.T) {
	t.Run("Создание конфига с mock loader-ом", func(t *testing.T) {
		mockLoader := mocks.NewMockConfigLoader(t)
		
		// Настройка возврата значений для мока
		mockLoader.On("GetEnv", "BACKUP_STORAGE", "local").Return("s3")
		mockLoader.On("GetEnv", "BACKUP_TEMP_PATH", "./tmp/backups").Return("/custom/backups")
		mockLoader.On("GetEnvAsInt", "BACKUP_RETENTION_DAYS", 7).Return(30)
		mockLoader.On("GetEnvAsBool", "CRON_ENABLE", false).Return(true)
		mockLoader.On("GetEnv", "BACKUP_SCHEDULE", "0 3 * * *").Return("0 6 * * *")

		cfg, err := config.NewBackupConfigWithLoader(mockLoader)

		require.NoError(t, err)
		assert.NotNil(t, cfg)

		// Проверка правильности значений
		assert.Equal(t, "s3", cfg.StorageType)
		assert.Equal(t, "/custom/backups", cfg.BackupPath)
		assert.Equal(t, 30, cfg.RetentionDays)
		assert.True(t, cfg.CronEnable)
		assert.Equal(t, "0 6 * * *", cfg.BackupSchedule)
		
		mockLoader.AssertExpectations(t)
	})
}