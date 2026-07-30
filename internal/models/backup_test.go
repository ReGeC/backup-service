package models_test

import (
	"testing"

	"backup-service/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestBackupLog_TableName(t *testing.T) {
	t.Run("Возвращение имени таблицы", func(t *testing.T) {
		backupLog := models.BackupLog{}

		assert.Equal(t, "backup_logs", backupLog.TableName())
	})
}