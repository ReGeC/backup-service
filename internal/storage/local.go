package storage

import (
	"backup-service/internal/config"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const Local = "local"

func init() {
	Register(Local, func() (Storage, error) {
		cfg, _, err := config.NewLocalConfig()
		if err != nil {
			return nil, fmt.Errorf("Неверная конфигурация для PostgreSQL: %w", err)
		}
		return NewLocalStorage(cfg.LocalStoragePath), nil
	})
}

type LocalStorage struct {
	localStoragePath string
}

func NewLocalStorage(localStoragePath string) *LocalStorage {
	return &LocalStorage{
		localStoragePath: localStoragePath,
	}
}

func (l *LocalStorage) Save(ctx context.Context, localPath string) (string, error) {
	// Проверка существования файла
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("Файл бэкапа не найден: %v", err)
	}

	// Формирование пути назначение
	filename := filepath.Base(localPath)
	destPath := filepath.Join(l.localStoragePath, filename)

	// Проверка, если файл уже на месте (В случае если tmp бэкап папка = localStorage BackupPath)
	if localPath == destPath {
		return localPath, nil
	}

	// Создание папки бэкапов если её нет
	if err := os.MkdirAll(l.localStoragePath, 0755); err != nil {
		return "", fmt.Errorf("Ошибка создания папки %s: %v", l.localStoragePath, err)
	}

	// Перемещение файла
	if err := os.Rename(localPath, destPath); err != nil {
		// Если Rename не сработал - копируем и удаляем
		if err := l.copyAndRemove(localPath, destPath); err != nil {
			return "", fmt.Errorf("Ошибка перемещения файла: %w", err)
		}
	}

	return destPath, nil
}

func (l *LocalStorage) copyAndRemove(src, dst string) error {
	// Открытие исходного файла
	in, err := os.OpenFile(src, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("Открытие файла не удалось %s: %w", src, err)
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("Закрытие файла прервано: %w", closeErr)
		}
		// TODO добавить обработку ошибок
	}()

	// Создание файла
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Ошибка создания файла бэкапа: %w", err)
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("Закрытие файла прервано: %w", closeErr)
		}
		// TODO добавить обработку ошибок
	}()

	// Копируем содержимое
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("Ошибка копирования: %w", err)
	}

	if err := out.Sync(); err != nil {
		// Логируем, но продолжаем
		slog.Warn("sync не завершен", "dst_file", dst, "error", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("Ошибка удаления файла %s: %w", src, err)
	}

	return nil
}

func (l *LocalStorage) List(ctx context.Context) ([]FileInfo, error) {
	entries, err := os.ReadDir(l.localStoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, fmt.Errorf("Ошибка чтения папки %s: %w", l.localStoragePath, err)
	}

	var files []FileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:      entry.Name(),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	return files, nil
}

func (l *LocalStorage) Delete(ctx context.Context, path string) error {
	if filepath.IsAbs(path) {
		return os.Remove(path)
	}
	return os.Remove(filepath.Join(l.localStoragePath, path))
}

func (l *LocalStorage) Download(ctx context.Context, path string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var fullPath string

	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(l.localStoragePath, path)
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("файл бэкапа не найден: %s", fullPath)
		}
		return "", fmt.Errorf("проверка файла: %w", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("указанный путь является директорией: %s", fullPath)
	}

	return fullPath, nil
}
