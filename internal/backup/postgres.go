package backup

import (
	"backup-service/internal/config"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

const Postgres = "postgres"

var ErrDisabled = errors.New("postgres backup is disabled")

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
        return nil, ErrDisabled
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
	if err := checkDependencies(); err != nil {
        return "", err
    }

    fullPath = buildBackupPath(outputDir)

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

func checkDependencies() error {
    binaries := []string{"pg_dump"}

    for _, bin := range binaries {
        if _, err := lookPath(bin); err != nil {
            return fmt.Errorf("%s not found", bin)
        }
    }

    return nil
}

func buildBackupPath(dir string) string {
    timestamp := time.Now().Format("2006-01-02_15-04")
    return filepath.Join(dir, fmt.Sprintf("db_%s.sql.gz", timestamp))
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

    cmd.Env = append(cmd.Env,
        "PGPASSWORD="+p.Password,
    )

    return cmd
}

func compressTo(dst io.Writer, src io.Reader) (err error) {
    gz := gzip.NewWriter(dst)
    defer func () {
		if gzCloseErr := gz.Close(); gzCloseErr != nil {
        	err = fmt.Errorf("ошибка записывания gzip: %w", gzCloseErr)
    	}
	}()

    if _, err := io.Copy(gz, src); err != nil {
        return fmt.Errorf("ошибка сжатия бэкапа: %w", err)
    }

    return nil
}

func (p* PostgresBackup) GetBackupType() string {
	return Postgres
}