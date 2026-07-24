package backup

import (
	"log"
	"fmt"
	"sync"
	"time"
	"os"

	"backup-service/internal/storage"
	"backup-service/internal/models"
)

type Backupper interface {
	Create(outputDir string) (string, error)
}

type BackupType string

// Глобальная хешмапа регистра
var registry = make(map[BackupType]func() (Backupper, error))
var mu sync.RWMutex

// Register регистрирует фабрику для создания бэкаппера
func Register(backupType BackupType, factory func() (Backupper, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[backupType] = factory
}

func InitBackuppers() map[BackupType]Backupper {
	backuppers := make(map[BackupType]Backupper)

	for backupType := range registry {
		backupper, err := NewBackupper(backupType)
		if err != nil {
			log.Printf("Ошибка инициализации %s: %v", backupType, err)
			continue
		}
		backuppers[backupType] = backupper
	}

	return backuppers
}

func RunBackuppers(backuppers map[BackupType]Backupper, outputDir string, storageType string) {
	for backupType, backupper := range backuppers {
		log.Printf("Создание бэкапа %s\n", backupType)
		startTime := time.Now()

		filePath, err := backupper.Create(outputDir)
		if err != nil {
			log.Printf("Ошибка создания бэкапа %s: %v", backupType, err)

			// Логируем ошибку в БД
			logEntry := &models.BackupLog{
				Name:    string(backupType) + "_backup",
				Size:    0,
				Storage: storageType,
				Status:  "failed",
				Error:   err.Error(),
			}
			storage.GetDB().Create(logEntry)

			continue
		}

		// Получаем размер файла
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			log.Printf("Не удалось получить размер файла %s: %v", filePath, err)

			// Логируем с размером 0, но статус success (бэкап-то создан)
			logEntry := &models.BackupLog{
				Name:    filePath,
				Size:    0,
				Storage: storageType,
				Status:  "success",
				Error:   "не удалось получить размер файла: " + err.Error(),
			}
			storage.GetDB().Create(logEntry)

			elapsed := time.Since(startTime)
			log.Printf("Бэкап %s создан: %s (размер: неизвестен, время: %v)",
				backupType, filePath, elapsed)
			continue
		}

		// Сохраняем в лог успешный бэкап с размером
		logEntry := &models.BackupLog{
			Name:    filePath,
			Size:    fileInfo.Size(),
			Storage: storageType,
			Status:  "success",
		}

		if result := storage.GetDB().Create(logEntry); result.Error != nil {
			log.Printf("Ошибка сохранения лога для %s: %v", backupType, result.Error)
			// Не прерываем выполнение, бэкап уже создан
		}

		elapsed := time.Since(startTime)
		log.Printf("Бэкап %s создан: %s (размер: %.2f MB, время: %v)",
			backupType, filePath, float64(fileInfo.Size())/1024/1024, elapsed)
	}
}

func NewBackupper(backupType BackupType) (Backupper, error) {
	mu.RLock()
	defer mu.RUnlock()
	
	factory, exists := registry[backupType]
	if !exists {
		return nil, fmt.Errorf("Неподдерживаемый тип бэкапа: %s", backupType)
	}
	return factory()
}
