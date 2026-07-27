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


func RunBackup(typ string, backupper Backupper, outputDir string, storageType string) (string, error) {
	log.Printf("Создание бэкапа %s\n", typ)
	startTime := time.Now()

	filePath, err := backupper.Create(outputDir)
	if err != nil {
		log.Printf("Ошибка создания бэкапа %s: %v", typ, err)

		// Логируем ошибку в БД
		logEntry := &models.BackupLog{
			Name:    typ + "_backup",
			Size:    0,
			Storage: storageType,
			Status:  "failed",
			Error:   err.Error(),
		}
		storage.GetDB().Create(logEntry)

		return "", err
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
			typ, filePath, elapsed)

		return filePath, nil // файл создан, возвращаем путь
	}

	// Сохраняем в лог успешный бэкап с размером
	logEntry := &models.BackupLog{
		Name:    filePath,
		Size:    fileInfo.Size(),
		Storage: storageType,
		Status:  "success",
	}

	if result := storage.GetDB().Create(logEntry); result.Error != nil {
		log.Printf("Ошибка сохранения лога для %s: %v", typ, result.Error)
		// Не прерываем выполнение, бэкап уже создан
	}

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
