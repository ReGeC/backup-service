package storage

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:generate mockery
type Storage interface {
	Save(ctx context.Context, localPath string) (string, error)
	Download(ctx context.Context, path string) (string, error)
	List(ctx context.Context) ([]FileInfo, error)
	Delete(ctx context.Context, path string) error
}

type FileInfo struct {
	Name string
	Size int64
	CreatedAt time.Time
}


var registry = map[string]func() (Storage, error){}
var mu sync.RWMutex

func Register(typ string, factory func() (Storage, error)) {
	mu.Lock()
	defer mu.Unlock()
	registry[typ] = factory
}

func ResetRegistry() {
	mu.Lock()
	defer mu.Unlock()

	registry = make(map[string]func() (Storage, error))
}

func NewStorage(typ string) (Storage, error) {
	mu.Lock()
	defer mu.Unlock()

	factory, exists := registry[typ]
	if !exists {
		return nil, fmt.Errorf("неизвестный тип хранилища: %s", typ)
	}
	return factory()
}

// CleanupOldBackups удаляет бэкапы старше retentionDays дней
func CleanupOldBackups(ctx context.Context, st Storage, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, fmt.Errorf("retention days must be greater than 0")
	}

	log.Printf("Запуск очистки бэкапов старше %d дней", retentionDays)

	// Получаем список всех файлов
	files, err := st.List(ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения списка файлов: %w", err)
	}

	if len(files) == 0 {
		log.Println("Файлов для проверки не найдено")
		return 0, nil
	}

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0
	errorsCount := 0

	for _, file := range files {
		// Проверяем контекст
		select {
		case <-ctx.Done():
			log.Printf("Очистка прервана: %v", ctx.Err())
			return deletedCount, ctx.Err()
		default:
		}

		// Парсим дату из имени файла
		fileDate, err := parseDateFromFilename(file.Name)
		if err != nil {
			// Если не удалось распарсить, пропускаем файл
			log.Printf("Не удалось распарсить дату из файла %s: %v", file.Name, err)
			continue
		}

		// Если файл старше cutoffTime, удаляем
		if fileDate.Before(cutoffTime) {
			log.Printf("Удаление старого бэкапа: %s (дата: %s)", 
				file.Name, fileDate.Format("2006-01-02"))
			
			if err := st.Delete(ctx, file.Name); err != nil {
				log.Printf("Ошибка удаления файла %s: %v", file.Name, err)
				errorsCount++
				continue
			}
			
			deletedCount++
		}
	}

	if deletedCount > 0 {
		log.Printf("Удалено %d старых бэкапов (старше %d дней)", deletedCount, retentionDays)
	} else {
		log.Printf("Старых бэкапов (старше %d дней) не найдено", retentionDays)
	}

	if errorsCount > 0 {
		log.Printf("Ошибок при удалении: %d", errorsCount)
	}

	return deletedCount, nil
}

// parseDateFromFilename парсит дату из имени файла в формате db_2026-08-05_06-56_postgres.sql.gz
func parseDateFromFilename(filename string) (time.Time, error) {
		// Проверяем, что файл начинается с "db_"
	if !strings.HasPrefix(filename, "db_") {
		return time.Time{}, fmt.Errorf("файл не является бэкапом БД: %s", filename)
	}
	
	// Ищем паттерн: 4 цифры-2 цифры-2 цифры (YYYY-MM-DD)
	re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return time.Time{}, fmt.Errorf("дата не найдена в имени файла: %s", filename)
	}
	
	dateStr := matches[1]
	return time.Parse("2006-01-02", dateStr)
}