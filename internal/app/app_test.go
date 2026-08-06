// internal/app/app_basic_test.go

package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"backup-service/internal/backup"
	backupMocks "backup-service/internal/backup/mocks"
	"backup-service/internal/config"
	"backup-service/internal/notifier"
	notifierMocks "backup-service/internal/notifier/mocks"
	"backup-service/internal/scheduler"
	"backup-service/internal/storage"
	storageMocks "backup-service/internal/storage/mocks"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetCronStringForCleanup(t *testing.T) {
	require.Equal(t, "* * 1 * * *", getCronStringForCleanup())
}

func TestStop(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	a := &App{
		ctx:    ctx,
		cancel: cancel,
	}

	require.NoError(t, a.Stop())

	select {
	case <-ctx.Done():
	default:
		t.Fatal("context wasn't cancelled")
	}
}

func TestRestoreUnknownType(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := &App{
		ctx:        ctx,
		backuppers: map[string]backup.Backupper{},
	}

	err := a.Restore("backup.sql", "postgres")

	require.Error(t, err)
	require.Contains(t, err.Error(), "не найден бэкаппер")
}

func TestRestoreDownloadError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := storageMocks.NewMockStorage(t)
	b := backupMocks.NewMockBackupper(t)

	st.EXPECT().
		Download(mock.Anything, "backup.sql").
		Return("", errors.New("download failed"))

	a := &App{
		ctx:     ctx,
		storage: st,
		backuppers: map[string]backup.Backupper{
			"postgres": b,
		},
	}

	err := a.Restore("backup.sql", "postgres")

	require.Error(t, err)
	require.Contains(t, err.Error(), "получение бэкапа")
}

func TestRestoreRestoreError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := storageMocks.NewMockStorage(t)
	b := backupMocks.NewMockBackupper(t)

	st.EXPECT().
		Download(mock.Anything, "backup.sql").
		Return("/tmp/backup.sql", nil)

	b.EXPECT().
		RestoreBackup(mock.Anything, "/tmp/backup.sql").
		Return("", errors.New("restore failed"))

	a := &App{
		ctx:     ctx,
		storage: st,
		backuppers: map[string]backup.Backupper{
			"postgres": b,
		},
	}

	err := a.Restore("backup.sql", "postgres")

	require.Error(t, err)
	require.Contains(t, err.Error(), "восстановление")
}

func TestRestoreSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := storageMocks.NewMockStorage(t)
	b := backupMocks.NewMockBackupper(t)

	st.EXPECT().
		Download(mock.Anything, "backup.sql").
		Return("/tmp/backup.sql", nil)

	b.EXPECT().
		RestoreBackup(mock.Anything, "/tmp/backup.sql").
		Return("/var/lib/postgres", nil)

	a := &App{
		ctx:     ctx,
		storage: st,
		backuppers: map[string]backup.Backupper{
			"postgres": b,
		},
	}

	require.NoError(t, a.Restore("backup.sql", "postgres"))
}

func TestStartCronDisabled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			CronEnable: false,
		},
	}

	err := a.Start()

	require.Error(t, err)
	require.Contains(t, err.Error(), "CRON_ENABLE=false")
}

func TestRunBackupCycle_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := &App{
		ctx:        ctx,
		config:     &config.BackupConfig{},
		backuppers: map[string]backup.Backupper{},
		notifiers:  map[string]notifier.Notifier{},
	}

	a.runBackupCycle(ctx)
}

func TestRunBackupCycle_BackupError(t *testing.T) {
	ctx := context.Background()

	backupper := backupMocks.NewMockBackupper(t)
	repo := backupMocks.NewMockBackupLogRepository(t)

	mockNotifier := notifierMocks.NewMockNotifier(t)

	backupper.EXPECT().
		GetBackupType().
		Return("postgres")

	backupper.EXPECT().
		CreateBackup(mock.Anything, "/tmp").
		Return("", errors.New("boom"))

	repo.EXPECT().
		CreateLog(mock.Anything).
		Return(nil)

	mockNotifier.EXPECT().
		Send(mock.Anything, mock.MatchedBy(func(msg string) bool {
			return len(msg) > 0
		})).
		Return(nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			BackupPath:  "/tmp",
			StorageType: "local",
		},
		backupRepo: repo,
		backuppers: map[string]backup.Backupper{
			"postgres": backupper,
		},
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	a.runBackupCycle(ctx)
}

func TestRunBackupCycle_SaveError(t *testing.T) {
	ctx := context.Background()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "backup.sql")

	require.NoError(t, os.WriteFile(file, []byte("backup"), 0644))

	backupper := backupMocks.NewMockBackupper(t)
	repo := backupMocks.NewMockBackupLogRepository(t)
	st := storageMocks.NewMockStorage(t)
	mockNotifier := notifierMocks.NewMockNotifier(t)

	backupper.EXPECT().
		GetBackupType().
		Return("postgres")

	backupper.EXPECT().
		CreateBackup(mock.Anything, tmp).
		Return(file, nil)

	repo.EXPECT().
		CreateLog(mock.Anything).
		Return(nil)

	st.EXPECT().
		Save(mock.Anything, file).
		Return("", errors.New("save failed"))

	mockNotifier.EXPECT().
		Send(mock.Anything, mock.Anything).
		Return(nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			BackupPath:  tmp,
			StorageType: "local",
		},
		storage:    st,
		backupRepo: repo,
		backuppers: map[string]backup.Backupper{
			"postgres": backupper,
		},
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	a.runBackupCycle(ctx)
}

func TestRunBackupCycle_Success(t *testing.T) {
	ctx := context.Background()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "backup.sql")

	require.NoError(t, os.WriteFile(file, []byte("backup"), 0644))

	backupper := backupMocks.NewMockBackupper(t)
	repo := backupMocks.NewMockBackupLogRepository(t)
	st := storageMocks.NewMockStorage(t)
	mockNotifier := notifierMocks.NewMockNotifier(t)

	backupper.EXPECT().
		GetBackupType().
		Return("postgres")

	backupper.EXPECT().
		CreateBackup(mock.Anything, tmp).
		Return(file, nil)

	repo.EXPECT().
		CreateLog(mock.Anything).
		Return(nil)

	st.EXPECT().
		Save(mock.Anything, file).
		Return("remote/backup.sql", nil)

	mockNotifier.EXPECT().
		Send(
			mock.Anything,
			mock.MatchedBy(func(msg string) bool {
				return len(msg) > 0
			}),
		).
		Return(nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			BackupPath:  tmp,
			StorageType: "local",
		},
		storage:    st,
		backupRepo: repo,
		backuppers: map[string]backup.Backupper{
			"postgres": backupper,
		},
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	a.runBackupCycle(ctx)
}

func TestCleanupOldBackups_ListError(t *testing.T) {
	ctx := context.Background()

	mockStorage := storageMocks.NewMockStorage(t)
	mockNotifier := notifierMocks.NewMockNotifier(t)

	mockStorage.EXPECT().
		List(mock.Anything).
		Return(nil, errors.New("list failed"))

	mockNotifier.EXPECT().
		Send(
			mock.Anything,
			mock.MatchedBy(func(msg string) bool {
				return len(msg) > 0
			}),
		).
		Return(nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			RetentionDays: 7,
		},
		storage: mockStorage,
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	a.cleanupOldBackups(ctx)
}

func TestCleanupOldBackups_DeleteOldFiles(t *testing.T) {
	ctx := context.Background()

	mockStorage := storageMocks.NewMockStorage(t)
	mockNotifier := notifierMocks.NewMockNotifier(t)

	oldDate := time.Now().AddDate(0, 0, -30)

	mockStorage.EXPECT().
		List(mock.Anything).
		Return([]storage.FileInfo{
			{
				Name:      "db_" + oldDate.Format("2006-01-02") + "_00-00_postgres.sql.gz",
				CreatedAt: oldDate,
			},
		}, nil)

	mockStorage.EXPECT().
		Delete(
			mock.Anything,
			mock.Anything,
		).
		Return(nil)

	mockNotifier.EXPECT().
		Send(
			mock.Anything,
			mock.MatchedBy(func(msg string) bool {
				return len(msg) > 0
			}),
		).
		Return(nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			RetentionDays: 7,
		},
		storage: mockStorage,
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	a.cleanupOldBackups(ctx)
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()

	mockStorage := storageMocks.NewMockStorage(t)
	mockNotifier := notifierMocks.NewMockNotifier(t)

	mockStorage.EXPECT().
		List(mock.Anything).
		Return([]storage.FileInfo{}, nil)

	a := &App{
		ctx: ctx,
		config: &config.BackupConfig{
			RetentionDays: 7,
		},
		storage: mockStorage,
		notifiers: map[string]notifier.Notifier{
			"mock": mockNotifier,
		},
	}

	require.NoError(t, a.Cleanup())
}

func TestClose_NilDB(t *testing.T) {
	a := &App{}

	require.NoError(t, a.Close())
}

func TestClose_DB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	a := &App{
		db: db,
	}

	require.NoError(t, a.Close())
}

func TestStart_CronDisabled(t *testing.T) {
	a := &App{
		ctx: context.Background(),
		config: &config.BackupConfig{
			CronEnable: false,
		},
	}

	err := a.Start()

	require.Error(t, err)
	require.Contains(t, err.Error(), "планировщик отключен")
}

func TestStart_Integration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mockStorage := storageMocks.NewMockStorage(t)

	mockStorage.EXPECT().
		List(mock.Anything).
		Return([]storage.FileInfo{}, nil)

	a := &App{
		ctx:    ctx,
		cancel: cancel,
		config: &config.BackupConfig{
			CronEnable:     true,
			BackupSchedule: "*/10 * * * * *",
			RetentionDays:  7,
		},
		scheduler: scheduler.New(),

		backuppers: map[string]backup.Backupper{},
		notifiers:  map[string]notifier.Notifier{},

		storage: mockStorage,
	}

	done := make(chan error, 1)

	go func() {
		done <- a.Start()
	}()

	time.Sleep(300 * time.Millisecond)

	err := a.Stop()
	require.NoError(t, err)

	select {
	case err := <-done:
		require.NoError(t, err)

	case <-time.After(3 * time.Second):
		t.Fatal("Start() did not exit")
	}
}
