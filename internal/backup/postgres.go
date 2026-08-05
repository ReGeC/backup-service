package backup

import (
	"backup-service/internal/config"
	"context"
	"errors"
	"fmt"
	"io"
	"compress/gzip"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const Postgres = "postgres"

var ErrPostgresDisabled = errors.Join(ErrDisabled, errors.New(Postgres))

var (
	commandContext = exec.CommandContext
	lookPath       = exec.LookPath
)

func init() {
	// Автоматическая регистрация при импорте
	Register(Postgres, newPostgresBackupper)
}

func newPostgresBackupper() (Backupper, error) {
    cfg, enabled, err := config.NewPostgresConfig()
    if err != nil {
        return nil, err
    }
	
	if !enabled {
        return nil, ErrPostgresDisabled
    }

    return NewPostgresBackup(cfg.PGHost, cfg.PGPort, cfg.PGUser, cfg.PGPassword, cfg.PGDatabase), nil
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


func (p *PostgresBackup) CreateBackup(ctx context.Context, outputDir string) (fullPath string, err error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	
	if err := p.checkDependencies(); err != nil {
        return "", err
    }

    fullPath = buildBackupPath(Postgres, outputDir)

	// Создаем файл для записи
	file, err := os.Create(fullPath)
	if err != nil {
	    return "", fmt.Errorf("Ошибка создания файла: %w", err)
	}
	// Обработка ошибки записи в файл
	defer func() {
	    if closeErr := file.Close(); err == nil && closeErr != nil {
	        err = fmt.Errorf("ошибка закрытия файла: %w", closeErr)
	    }

	    if err != nil {
	        _ = os.Remove(file.Name())
	    }
	}()

	// Пишем вывод pg_dump в файл через gzip
	if err = p.dumpTo(ctx, file); err != nil {
        return "", err
    }

	return fullPath, nil
}

func (p *PostgresBackup) checkDependencies() error {
    binaries := []string{"pg_dump"}

    for _, bin := range binaries {
        if _, err := lookPath(bin); err != nil {
            return fmt.Errorf("%s not found", bin)
        }
    }

    return nil
}

func (p *PostgresBackup) dumpTo(ctx context.Context, dst io.Writer) error {
    // Команда pg_dump
	// pg_dump -h localhost -p 5432 -U postgres -d testdb
	cmd := p.newDumpCommand(ctx)
	
	stdout, err := cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("ошибка потока вывода: %w", err)
    }

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("ошибка запуска pg_dump: %w", err)
    }

    if err := compressTo(dst, stdout); err != nil {
        _ = cmd.Process.Kill()
        _ = cmd.Wait()
        return err
    }

    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("ошибка ожидания pg_dump: %w", err)
    }

    return nil
}

func (p *PostgresBackup) newDumpCommand(ctx context.Context) *exec.Cmd {
    cmd := commandContext(
		ctx,
        "pg_dump",
        "-h", p.Host,
        "-p", strconv.Itoa(p.Port),
        "-U", p.User,
        "-d", p.Database,
        "-F", "p",
    )

    cmd.Env = append(cmd.Environ(),
        "PGPASSWORD="+p.Password,
    )

    return cmd
}

// RestoreBackup восстанавливает бэкап из файла (сжатого gzip) в новую БД.
// Возвращает имя созданной БД или ошибку.
func (p *PostgresBackup) RestoreBackup(ctx context.Context, backupPath string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// 1. Проверяем наличие psql
	if err := p.checkRestoreDependencies(); err != nil {
		return "", err
	}

	// 2. Генерируем имя новой БД
	newDBName := buildRestoredPath(p.Database)

	// 3. Создаём новую БД
	if err := p.createDatabase(ctx, newDBName); err != nil {
		return "", fmt.Errorf("не удалось создать БД %s: %w", newDBName, err)
	}

	// Если восстановление провалится, удалим созданную БД
	var restoreErr error
	defer func() {
		if restoreErr != nil {
			_ = p.dropDatabase(ctx, newDBName)
		}
	}()

	// 4. Восстанавливаем данные
	if restoreErr = p.restoreFromFile(ctx, backupPath, newDBName); restoreErr != nil {
		return "", fmt.Errorf("ошибка восстановления: %w", restoreErr)
	}

	// 5. Всё ок
	return newDBName, nil
}

// checkRestoreDependencies проверяет наличие psql
func (p *PostgresBackup) checkRestoreDependencies() error {
	binaries := []string{"psql"}
	for _, bin := range binaries {
		if _, err := lookPath(bin); err != nil {
			return fmt.Errorf("%s not found", bin)
		}
	}
	return nil
}



// createDatabase создаёт новую БД через psql
func (p *PostgresBackup) createDatabase(ctx context.Context, dbName string) error {
	// Используем psql -c "CREATE DATABASE ..."
	cmd := commandContext(ctx, "psql",
		"-h", p.Host,
		"-p", strconv.Itoa(p.Port),
		"-U", p.User,
		"-d", "postgres", // подключаемся к стандартной БД, чтобы создать новую
		"-c", "CREATE DATABASE "+dbName+";",
	)
	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+p.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql create database failed: %w, output: %s", err, string(output))
	}
	return nil
}

// dropDatabase удаляет БД (используется при ошибке)
func (p *PostgresBackup) dropDatabase(ctx context.Context, dbName string) error {
	cmd := commandContext(ctx, "psql",
		"-h", p.Host,
		"-p", strconv.Itoa(p.Port),
		"-U", p.User,
		"-d", "postgres",
		"-c", "DROP DATABASE IF EXISTS "+dbName+";",
	)
	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+p.Password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка удаления БД: %w, output: %s", err, string(output))
	}
	return nil
}

// restoreFromFile открывает gzip-файл и подаёт распакованный поток в psql
func (p *PostgresBackup) restoreFromFile(ctx context.Context, backupPath, dbName string) error {
	// Открываем файл бэкапа
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл бэкапа: %w", err)
	}
	defer file.Close()

	// Создаём gzip-ридер
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("не удалось распаковать gzip: %w", err)
	}
	defer gzReader.Close()

	// Запускаем psql с перенаправлением stdin
	cmd := commandContext(ctx, "psql",
		"-h", p.Host,
		"-p", strconv.Itoa(p.Port),
		"-U", p.User,
		"-d", dbName,
		"-q",                 // тихий режим (не выводить лишнего)
		"-v", "ON_ERROR_STOP=1", // останавливаться при ошибке SQL
	)
	cmd.Env = append(cmd.Environ(), "PGPASSWORD="+p.Password)
	cmd.Stdin = gzReader

	// Собираем stdout и stderr для диагностики
	var stderr strings.Builder
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard // нам не нужен вывод, можно игнорировать

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql восстановление завершилось ошибкой: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

func (p* PostgresBackup) GetBackupType() string {
	return Postgres
}