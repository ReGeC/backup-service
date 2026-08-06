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

func TestSQLiteBackup_RestoreBackup_Success(t *testing.T) {
	tempDir := t.TempDir()

	// Создаем файл бэкапа
	backupPath := filepath.Join(tempDir, "backup.db")
	backupContent := []byte("backup database content")

	err := os.WriteFile(backupPath, backupContent, 0644)
	require.NoError(t, err)

	// Исходная БД, которую не должны изменять
	dbPath := filepath.Join(tempDir, "original.db")
	err = os.WriteFile(dbPath, []byte("original database"), 0644)
	require.NoError(t, err)

	backup := NewSQLiteBackup(dbPath)

	restoredPath, err := backup.RestoreBackup(context.Background(), backupPath)

	assert.NoError(t, err)
	assert.NotEmpty(t, restoredPath)

	// Проверяем существование файла
	_, err = os.Stat(restoredPath)
	assert.NoError(t, err)

	// Проверяем содержимое восстановленной БД
	content, err := os.ReadFile(restoredPath)
	assert.NoError(t, err)
	assert.Equal(t, backupContent, content)

	// Проверяем, что оригинальная БД не была изменена
	originalContent, err := os.ReadFile(dbPath)
	assert.NoError(t, err)
	assert.Equal(t, []byte("original database"), originalContent)
}

func TestSQLiteBackup_RestoreBackup_BackupFileNotFound(t *testing.T) {
	backup := NewSQLiteBackup("/tmp/test.db")

	restoredPath, err := backup.RestoreBackup(
		context.Background(),
		"/nonexistent/backup.db",
	)

	assert.Error(t, err)
	assert.Empty(t, restoredPath)
	assert.Contains(t, err.Error(), "файл бэкапа не найден")
}

func TestSQLiteBackup_RestoreBackup_CanceledContext(t *testing.T) {
	tempDir := t.TempDir()

	backupPath := filepath.Join(tempDir, "backup.db")
	err := os.WriteFile(backupPath, []byte("backup"), 0644)
	require.NoError(t, err)

	backup := NewSQLiteBackup(filepath.Join(tempDir, "db.sqlite"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	restoredPath, err := backup.RestoreBackup(ctx, backupPath)

	assert.Error(t, err)
	assert.Empty(t, restoredPath)
	assert.Equal(t, context.Canceled, err)
}

func TestSQLiteBackup_RestoreBackup_CannotCreateRestoredFile(t *testing.T) {
	tempDir := t.TempDir()

	backupPath := filepath.Join(tempDir, "backup.db")
	err := os.WriteFile(backupPath, []byte("backup"), 0644)
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "db.db")

	backup := NewSQLiteBackup(dbPath)

	// Получаем реальный путь, который будет использовать RestoreBackup
	restoredPath := buildRestoredPath(dbPath) + ".db"

	// Создаем директорию с таким же именем,
	// чтобы os.Create(restoredPath) упал
	err = os.Mkdir(restoredPath, 0755)
	require.NoError(t, err)

	resultPath, err := backup.RestoreBackup(
		context.Background(),
		backupPath,
	)

	assert.Error(t, err)
	assert.Empty(t, resultPath)
	assert.Contains(t, err.Error(), "ошибка создания восстановленной БД")
}
