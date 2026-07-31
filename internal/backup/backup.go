package backup

import (
	"log"
	"fmt"
	"sync"
	"time"
	"os"


	"backup-service/internal/models"
)

//go:generate mockery
type Backupper interface {
	CreateBackup(outputDir string) (string, error)
}

//go:generate mockery
type BackupLogRepository interface {
    Create(*models.BackupLog) error
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

func InitBackuppers() map[string]Backupper {
	backuppers := make(map[string]Backupper)

	for typ := range registry {
		backupper, err := NewBackupper(typ)
		if err != nil {
			log.Printf("Ошибка инициализации %s: %v", typ, err)
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
	if err := repository.Create(logEntry); err != nil {
		log.Printf("Ошибка сохранения лога для %s: %v", backupType, err)
	}
}

func RunBackup(typ string, backupper Backupper, repository BackupLogRepository, outputDir string, storageType string) (string, error) {
	log.Printf("Создание бэкапа %s\n", typ)
	startTime := time.Now()

	filePath, err := backupper.CreateBackup(outputDir)
	if err != nil {
		log.Printf("Ошибка создания бэкапа %s: %v", typ, err)

		// Логируем ошибку в БД
		saveBackupLog(repository, &models.BackupLog{
			Name:    typ + "_backup",
			Size:    0,
			Storage: storageType,
			Status:  "failed",
			Error:   err.Error(),
		}, typ)

		return "", err
	}

	// Получаем размер файла
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Не удалось получить размер файла %s: %v", filePath, err)

		// Логируем с размером 0, но статус success (бэкап создан)
		saveBackupLog(repository, &models.BackupLog{
			Name:    filePath,
			Size:    0,
			Storage: storageType,
			Status:  "success",
			Error:   "не удалось получить размер файла: " + err.Error(),
		}, typ)

		elapsed := time.Since(startTime)
		log.Printf("Бэкап %s создан: %s (размер: неизвестен, время: %v)",
			typ, filePath, elapsed)

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
	log.Printf("Бэкап %s создан: %s (размер: %.2f MB, время: %v)",
		typ, filePath, float64(fileInfo.Size())/1024/1024, elapsed)

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
