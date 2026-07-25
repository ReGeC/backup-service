package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
	"backup-service/internal/config"
)

const Postgres BackupType = "postgres"

func init() {
	cfg, enabled, err := config.NewPostgresConfig()
	if enabled {
		// Автоматическая регистрация при импорте
		Register(Postgres, func() (Backupper, error) {
			if err != nil {
				return nil, fmt.Errorf("Неверная конфигурация для PostgreSQL: %w", err)
			}
			return NewPostgresBackupFromConfig(cfg), nil
		})
	}
}

type PostgresBackup struct {
	Host string
	Port int
	User string
	Password string
	Database string
}

func NewPostgresBackup(host string, port int, user, password, database string) *PostgresBackup {
	return &PostgresBackup{
		Host: host,
		Port: port,
		User: user,
		Password: password,
		Database: database,
	}
}

func NewPostgresBackupFromConfig(cfg *config.PostgresConfig) *PostgresBackup {
	return &PostgresBackup{
        Host:     cfg.PGHost,
        Port:     cfg.PGPort,
        User:     cfg.PGUser,
        Password: cfg.PGPassword,
        Database: cfg.PGDatabase,
    }
}

func (p *PostgresBackup) Create(outputDir string) (string, error) {
	// Проверка, что pg_dump и gzip установлены
	if _, err := exec.LookPath("pg_dump"); err != nil {
	    return "", fmt.Errorf("pg_dump not found: %w", err)
	}
	if _, err := exec.LookPath("gzip"); err != nil {
	    return "", fmt.Errorf("gzip not found: %w", err)
	}
	
	// Формирум имя файла: db_2026-07-10_15-30.sql.gz
	timestamp := time.Now().Format("2006-01-02_15-04")
	filename := fmt.Sprintf("db_%s.sql.gz", timestamp)
	fullPath := filepath.Join(outputDir, filename)

	// Команда pg_dump
	// pg_dump -h localhost -p 5432 -U postgres -d testdb | gzip > backup.sql.gz
	pgDumpCmd := exec.Command(
		"pg_dump", 
		"-h", p.Host,
		"-p", fmt.Sprintf("%d", p.Port),
		"-U", p.User,
		"-d", p.Database,
		"-F", "p",
	)

	// Безопасная перадча пароля в команду
	pgDumpCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", p.Password))

	// Создаем файл для записи
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("Ошибка создания файла: %w", err)
	}
	// Обработка ошибки записи в файл
	var createErr error
    defer func() {
        if closeErr := file.Close(); closeErr != nil && createErr == nil {
            createErr = fmt.Errorf("failed to close file: %w", closeErr)
        }
        if createErr != nil {
            os.Remove(fullPath)  // Удаляем битый файл
        }
		// TODO добавить логирование и обработку ошибок
    }()

	// Пишем вывод pg_dump в файл через gzip
	gzipCmd := exec.Command("gzip")
	gzipCmd.Stdin, err = pgDumpCmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("Ошибка получения входного потока: %w", err)
	}
	gzipCmd.Stdout = file

	// Запуск gzip
	if err := gzipCmd.Start(); err != nil {
		return "", fmt.Errorf("Ошибка запуска gzip: %w", err)
	}

	// Запуск pg_dump
	if err := pgDumpCmd.Run(); err != nil {
		return "", fmt.Errorf("Ошибка pg_dump: %w", err)
	}

	// Ждем завершения gzip
	if err := gzipCmd.Wait(); err != nil {
		return "", fmt.Errorf("Ошибка gzip: %w", err)
	}

	return fullPath, nil
}