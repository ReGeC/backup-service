package backup

import (
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

func buildBackupPath(typ, dir string) string {
    timestamp := time.Now().Format("2006-01-02_15-04")
    return filepath.Join(dir, fmt.Sprintf("db_%s_%s.sql.gz", timestamp, typ))
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