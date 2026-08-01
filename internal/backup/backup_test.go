package backup_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"backup-service/internal/backup"
	mocks "backup-service/internal/backup/mocks"
	"backup-service/internal/models"
)

func resetRegistry(t *testing.T) {
	t.Helper()

	backup.ResetRegistry()

	t.Cleanup(func() {
		backup.ResetRegistry()
	})
}

func TestRunBackup_Success(t *testing.T) {
	resetRegistry(t)

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.sql")
	backupContent := []byte("backup content")

	require.NoError(t, os.WriteFile(backupPath, backupContent, 0644))

	backupper := mocks.NewMockBackupper(t)
	repository := mocks.NewMockBackupLogRepository(t)

	ctx := context.Background()

	backupper.EXPECT().
		CreateBackup(ctx, tmpDir).
		Return(backupPath, nil).
		Once()

	backupper.EXPECT().
		GetBackupType().
    	Return("postgres").
    	Once()

	repository.
		On("Create", mock.AnythingOfType("*models.BackupLog")).
		Run(func(args mock.Arguments) {
			logEntry := args.Get(0).(*models.BackupLog)

			assert.Equal(t, backupPath, logEntry.Name)
			assert.Equal(t, int64(len(backupContent)), logEntry.Size)
			assert.Equal(t, "local", logEntry.Storage)
			assert.Equal(t, "success", logEntry.Status)
			assert.Empty(t, logEntry.Error)
		}).
		Return(nil).
		Once()

	result, err := backup.RunBackup(
		ctx,
		backupper,
		repository,
		tmpDir,
		"local",
	)

	require.NoError(t, err)
	assert.Equal(t, backupPath, result)
}

func TestRunBackup_CreateBackupError(t *testing.T) {
	resetRegistry(t)

	tmpDir := t.TempDir()
	createErr := errors.New("backup command failed")

	backupper := mocks.NewMockBackupper(t)
	repository := mocks.NewMockBackupLogRepository(t)

	ctx := context.Background()

	backupper.EXPECT().
		CreateBackup(ctx, tmpDir).
		Return("", createErr).
		Once()

	backupper.EXPECT().
		GetBackupType().
    	Return("postgres").
    	Once()

	repository.
		On("Create", mock.AnythingOfType("*models.BackupLog")).
		Run(func(args mock.Arguments) {
			logEntry := args.Get(0).(*models.BackupLog)

			assert.Equal(t, "postgres_backup", logEntry.Name)
			assert.Equal(t, int64(0), logEntry.Size)
			assert.Equal(t, "s3", logEntry.Storage)
			assert.Equal(t, "failed", logEntry.Status)
			assert.Equal(t, createErr.Error(), logEntry.Error)
		}).
		Return(nil).
		Once()

	result, err := backup.RunBackup(
		ctx,
		backupper,
		repository,
		tmpDir,
		"s3",
	)

	require.Error(t, err)
	assert.Empty(t, result)

	assert.ErrorIs(t, err, backup.ErrBackupCreation)
	assert.ErrorIs(t, err, createErr)
}

func TestRunBackup_StatError(t *testing.T) {
	resetRegistry(t)

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.sql")

	backupper := mocks.NewMockBackupper(t)
	repository := mocks.NewMockBackupLogRepository(t)

	ctx := context.Background()

	// CreateBackup говорит, что backup создан,
	// но такого файла на диске нет.
	backupper.EXPECT().
		CreateBackup(ctx, tmpDir).
		Return(backupPath, nil).
		Once()

	backupper.EXPECT().
		GetBackupType().
    	Return("postgres").
    	Once()

	repository.
		On("Create", mock.AnythingOfType("*models.BackupLog")).
		Run(func(args mock.Arguments) {
			logEntry := args.Get(0).(*models.BackupLog)

			assert.Equal(t, backupPath, logEntry.Name)
			assert.Equal(t, int64(0), logEntry.Size)
			assert.Equal(t, "local", logEntry.Storage)
			assert.Equal(t, "success", logEntry.Status)
			assert.Contains(
				t,
				logEntry.Error,
				"не удалось получить размер файла:",
			)
		}).
		Return(nil).
		Once()

	result, err := backup.RunBackup(
		ctx,
		backupper,
		repository,
		tmpDir,
		"local",
	)

	require.NoError(t, err)
	assert.Equal(t, backupPath, result)
}

func TestRunBackup_RepositoryError(t *testing.T) {
	resetRegistry(t)

	tmpDir := t.TempDir()
	backupPath := filepath.Join(tmpDir, "backup.sql")

	require.NoError(t, os.WriteFile(
		backupPath,
		[]byte("backup"),
		0644,
	))

	repositoryErr := errors.New("database unavailable")

	backupper := mocks.NewMockBackupper(t)
	repository := mocks.NewMockBackupLogRepository(t)

	ctx := context.Background()

	backupper.EXPECT().
		CreateBackup(ctx, tmpDir).
		Return(backupPath, nil).
		Once()

	backupper.EXPECT().
		GetBackupType().
    	Return("postgres").
    	Once()

	repository.
		On("Create", mock.AnythingOfType("*models.BackupLog")).
		Return(repositoryErr).
		Once()

	result, err := backup.RunBackup(
		ctx,
		backupper,
		repository,
		tmpDir,
		"local",
	)

	// Ошибка БД только логируется.
	// Сам backup считается успешным.
	require.NoError(t, err)
	assert.Equal(t, backupPath, result)
}

func TestRunBackup_RepositoryErrorAfterBackupFailure(t *testing.T) {
	resetRegistry(t)

	tmpDir := t.TempDir()

	createErr := errors.New("backup command failed")
	repositoryErr := errors.New("database unavailable")

	backupper := mocks.NewMockBackupper(t)
	repository := mocks.NewMockBackupLogRepository(t)

	ctx := context.Background()

	backupper.EXPECT().
		CreateBackup(ctx, tmpDir).
		Return("", createErr).
		Once()

	backupper.EXPECT().
		GetBackupType().
    	Return("postgres").
    	Once()

	repository.
		On("Create", mock.AnythingOfType("*models.BackupLog")).
		Return(repositoryErr).
		Once()

	result, err := backup.RunBackup(
		ctx,
		backupper,
		repository,
		tmpDir,
		"local",
	)

	require.Error(t, err)
	assert.Empty(t, result)

	// Ошибка CreateBackup должна остаться основной ошибкой.
	assert.ErrorIs(t, err, backup.ErrBackupCreation)
	assert.ErrorIs(t, err, createErr)
	assert.NotErrorIs(t, err, repositoryErr)
}

func TestNewBackupper_Success(t *testing.T) {
	resetRegistry(t)

	typ := "test-new-backupper-success"
	expected := mocks.NewMockBackupper(t)

	backup.Register(typ, func() (backup.Backupper, error) {
		return expected, nil
	})

	result, err := backup.NewBackupper(typ)

	require.NoError(t, err)
	assert.Same(t, expected, result)
}

func TestNewBackupper_UnsupportedType(t *testing.T) {
	resetRegistry(t)

	result, err := backup.NewBackupper("unsupported-backup-type")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Неподдерживаемый тип бэкапа")
}

func TestNewBackupper_FactoryError(t *testing.T) {
	resetRegistry(t)

	typ := "test-new-backupper-error"
	factoryErr := errors.New("failed to initialize backupper")

	backup.Register(typ, func() (backup.Backupper, error) {
		return nil, factoryErr
	})

	result, err := backup.NewBackupper(typ)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, factoryErr)
}

func TestInitBackuppers(t *testing.T) {
	resetRegistry(t)

	successType := "test-init-success"
	errorType := "test-init-error"

	expected := mocks.NewMockBackupper(t)
	factoryErr := errors.New("failed to initialize")

	backup.Register(successType, func() (backup.Backupper, error) {
		return expected, nil
	})

	backup.Register(errorType, func() (backup.Backupper, error) {
		return nil, factoryErr
	})

	result := backup.InitBackuppers()

	require.NotNil(t, result)

	assert.Contains(t, result, successType)
	assert.Same(t, expected, result[successType])

	// Если фабрика вернула ошибку,
	// backupper не должен попасть в результат.
	assert.NotContains(t, result, errorType)
}

func TestInitBackuppers_Disabled(t *testing.T) {
    resetRegistry(t)

    backup.Register("disabled", func() (backup.Backupper, error) {
        return nil, backup.ErrDisabled
    })

    result := backup.InitBackuppers()

    assert.Empty(t, result)
}

func TestRegister_OverridesExistingFactory(t *testing.T) {
	resetRegistry(t)
	
	typ := "test-register-override"

	first := mocks.NewMockBackupper(t)
	second := mocks.NewMockBackupper(t)

	backup.Register(typ, func() (backup.Backupper, error) {
		return first, nil
	})

	result, err := backup.NewBackupper(typ)

	require.NoError(t, err)
	assert.Same(t, first, result)

	backup.Register(typ, func() (backup.Backupper, error) {
		return second, nil
	})

	result, err = backup.NewBackupper(typ)

	require.NoError(t, err)
	assert.Same(t, second, result)
}
