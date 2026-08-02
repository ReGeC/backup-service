package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteBackup(t *testing.T) {
	dbPath := "/tmp/test.db"
	backup := NewSQLiteBackup(dbPath)
	
	assert.Equal(t, dbPath, backup.DBPath)
	assert.Equal(t, SQLite, backup.GetBackupType())
}

func TestSQLiteBackup_GetBackupType(t *testing.T) {
	backup := NewSQLiteBackup("/tmp/test.db")
	assert.Equal(t, SQLite, backup.GetBackupType())
}

func TestSQLiteBackup_CreateBackup_Success(t *testing.T) {
	// Создаем временную директорию для теста
	tempDir := t.TempDir()
	
	// Создаем временный файл БД
	dbPath := filepath.Join(tempDir, "test.db")
	dbContent := []byte("test database content")
	err := os.WriteFile(dbPath, dbContent, 0644)
	require.NoError(t, err)
	
	// Создаем бэкапер
	backup := NewSQLiteBackup(dbPath)
	
	// Создаем бэкап
	outputDir := t.TempDir()
	ctx := context.Background()
	
	fullPath, err := backup.CreateBackup(ctx, outputDir)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, fullPath)
	assert.Contains(t, fullPath, outputDir)
	
	// Проверяем, что файл создан
	_, err = os.Stat(fullPath)
	assert.NoError(t, err)
	
	// Проверяем содержимое
	content, err := os.ReadFile(fullPath)
	assert.NoError(t, err)
	assert.Equal(t, dbContent, content)
}

func TestSQLiteBackup_CreateBackup_DBFileNotFound(t *testing.T) {
	backup := NewSQLiteBackup("/nonexistent/path.db")
	outputDir := t.TempDir()
	ctx := context.Background()
	
	fullPath, err := backup.CreateBackup(ctx, outputDir)
	
	assert.Error(t, err)
	assert.Empty(t, fullPath)
	assert.Contains(t, err.Error(), "файл БД не найден")
}

func TestSQLiteBackup_CreateBackup_CanceledContext(t *testing.T) {
	// Создаем временную директорию с БД
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	err := os.WriteFile(dbPath, []byte("test"), 0644)
	require.NoError(t, err)
	
	backup := NewSQLiteBackup(dbPath)
	outputDir := t.TempDir()
	
	// Создаем отмененный контекст
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	fullPath, err := backup.CreateBackup(ctx, outputDir)
	
	assert.Error(t, err)
	assert.Empty(t, fullPath)
	assert.Equal(t, context.Canceled, err)
}

func TestSQLiteBackup_CreateBackup_OutputDirectoryNotExists(t *testing.T) {
	// Создаем БД
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	err := os.WriteFile(dbPath, []byte("test"), 0644)
	require.NoError(t, err)
	
	backup := NewSQLiteBackup(dbPath)
	
	// Используем несуществующую директорию
	outputDir := filepath.Join(tempDir, "nonexistent", "dir")
	ctx := context.Background()
	
	fullPath, err := backup.CreateBackup(ctx, outputDir)
	
	assert.Error(t, err)
	assert.Empty(t, fullPath)
	assert.Contains(t, err.Error(), "ошибка создания бэкапа")
}

func TestSQLiteBackup_buildBackupPath(t *testing.T) {
	backup := NewSQLiteBackup("/tmp/test.db")
	outputDir := t.TempDir()
	
	// Проверяем, что buildBackupPath вызывается через CreateBackup
	ctx := context.Background()
	
	// Создаем БД
	dbPath := filepath.Join(outputDir, "test.db")
	err := os.WriteFile(dbPath, []byte("test"), 0644)
	require.NoError(t, err)
	backup.DBPath = dbPath
	
	fullPath, err := backup.CreateBackup(ctx, outputDir)
	assert.NoError(t, err)
	
	// Проверяем, что путь содержит ожидаемую структуру
	assert.Contains(t, fullPath, outputDir)
}

func TestNewSQLiteBackupper(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        t.Setenv("SQLITE_ENABLE", "true")
        
        backupper, err := newSQLiteBackupper()
        require.NoError(t, err)
        assert.NotNil(t, backupper)
    })

    t.Run("disabled", func(t *testing.T) {
        t.Setenv("SQLITE_ENABLE", "false")
        
        backupper, err := newSQLiteBackupper()
        assert.ErrorIs(t, err, ErrDisabled)
        assert.Nil(t, backupper)
    })

    t.Run("config error", func(t *testing.T) {
        t.Setenv("SQLITE_ENABLE", "true")
        t.Setenv("SQLITE_PATH", "") // вызываем ошибку валидации
        
        backupper, err := newSQLiteBackupper()
        assert.Error(t, err)
        assert.Nil(t, backupper)
    })
}