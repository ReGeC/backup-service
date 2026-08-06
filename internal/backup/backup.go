package backup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"backup-service/internal/models"
)

var ErrDisabled = errors.New("backup is disabled")
var ErrBackupCreation = errors.New("backup creation failed")

//go:generate mockery
type Backupper interface {
	CreateBackup(ctx context.Context, outputDir string) (string, error)
	RestoreBackup(ctx context.Context, backupPath string) (string, error)
	GetBackupType() string
}

//go:generate mockery
type BackupLogRepository interface {
	CreateLog(*models.BackupLog) error
}

// Глобальная хешмапа регистра
var registry = make(map[string]func() (Backupper, error))
var mu sync.RWMutex

// Register регистрирует фабрику для создания бэкаппера
func Register(typ string, factory func() (Backupper, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()

	registry = make(map[string]func() (Backupper, error))
}

// TODO: Обработка отключенных модулей
func InitBackuppers() map[string]Backupper {
	backuppers := make(map[string]Backupper)

	for typ := range registry {
		backupper, err := NewBackupper(typ)
		if err != nil {
			if !errors.Is(err, ErrDisabled) {
				slog.Error("Ошибка инициализации", "backup_type", typ, "error", err)
			}
			continue
		}
		backuppers[typ] = backupper
	}

	return backuppers
}

func saveBackupLog(
	repository BackupLogRepository,
	logEntry *models.BackupLog,
	backupType string,
) {
	if err := repository.CreateLog(logEntry); err != nil {
		slog.Error("Ошибка сохранения лога", "backup_type", backupType, "error", err)
	}
}

func RunBackup(ctx context.Context, backupper Backupper, repository BackupLogRepository, outputDir string, storageType string) (string, error) {
	typ := backupper.GetBackupType()

	slog.Info("Создание бэкапа", "backup_type", typ)
	startTime := time.Now()

	filePath, err := backupper.CreateBackup(ctx, outputDir)
	if err != nil {
		slog.Error("Ошибка создания бэкапа", "backup_type", typ, "error", err)

		// Логируем ошибку в БД
		saveBackupLog(repository, &models.BackupLog{
			Name:    typ + "_backup",
			Size:    0,
			Storage: storageType,
			Status:  "failed",
			Error:   err.Error(),
		}, typ)

		return "", fmt.Errorf("%w: %w", ErrBackupCreation, err)
	}

	// Получаем размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		slog.Warn("Не удалось получить размер файла", "filepath", filePath, "error", err)

		// Логируем с размером 0, но статус success (бэкап создан)
		saveBackupLog(repository, &models.BackupLog{
			Name:    filePath,
			Size:    0,
			Storage: storageType,
			Status:  "success",
			Error:   "не удалось получить размер файла: " + err.Error(),
		}, typ)

		elapsed := time.Since(startTime)
		slog.Info("Бэкап создан:",
			"backup_type", typ, "filepath", filePath, "time", elapsed)

		return filePath, nil // файл создан, возвращаем путь
	}

	// Сохраняем в лог успешный бэкап с размером
	saveBackupLog(repository, &models.BackupLog{
		Name:    filePath,
		Size:    fileInfo.Size(),
		Storage: storageType,
		Status:  "success",
	}, typ)

	elapsed := time.Since(startTime)
	slog.Info("Бэкап создан:",
		"backup_type", typ, "filepath", filePath,
		"size", float64(fileInfo.Size())/1024/1024, "time", elapsed)

	return filePath, nil
}

func NewBackupper(typ string) (Backupper, error) {
	mu.RLock()
	defer mu.RUnlock()

	factory, exists := registry[typ]
	if !exists {
		return nil, fmt.Errorf("Неподдерживаемый тип бэкапа: %s", typ)
	}
	return factory()
}
