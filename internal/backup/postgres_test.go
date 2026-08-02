package backup

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHelperPGDumpProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	_, _ = os.Stdout.Write([]byte("test postgres dump"))
	os.Exit(0)
}

func TestHelperPGDumpProcessError(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	_, _ = os.Stderr.Write([]byte("pg_dump failed"))
	os.Exit(1)
}

func mockExecCommand(t *testing.T, helper string) {
	t.Helper()

	oldCommand := commandContext
	oldLookPath := lookPath

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(
			ctx,
			os.Args[0],
			"-test.run=^"+helper+"$",
		)

		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
		)

		return cmd
	}

	lookPath = func(string) (string, error) {
		return "pg_dump", nil
	}

	t.Cleanup(func() {
		commandContext = oldCommand
		lookPath = oldLookPath
	})
}

func TestCreateBackup_Success(t *testing.T) {
	mockExecCommand(t, "TestHelperPGDumpProcess")

	tmpDir := t.TempDir()

	p := NewPostgresBackup(
		"localhost",
		5432,
		"postgres",
		"password",
		"testdb",
	)

	path, err := p.CreateBackup(context.Background(), tmpDir)

	require.NoError(t, err)
	require.FileExists(t, path)

	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	gz, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer gz.Close()

	data, err := io.ReadAll(gz)
	require.NoError(t, err)

	assert.Equal(t, "test postgres dump", string(data))
}

func TestCreateBackup_PgDumpError(t *testing.T) {
	mockExecCommand(t, "TestHelperPGDumpProcessError")

	tmpDir := t.TempDir()

	p := NewPostgresBackup(
		"localhost",
		5432,
		"postgres",
		"password",
		"testdb",
	)

	path, err := p.CreateBackup(context.Background(), tmpDir)

	require.Error(t, err)
	assert.Empty(t, path)

	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	assert.Empty(t, files)
}

func TestCreateBackup_PgDumpStartError(t *testing.T) {
	oldCommand := commandContext
	oldLookPath := lookPath

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(
			ctx,
			os.Args[1],
			"1",
		)

		cmd.Env = append(
			os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
		)

		return cmd
	}

	lookPath = func(string) (string, error) {
		return "pg_dump", nil
	}

	t.Cleanup(func() {
		commandContext = oldCommand
		lookPath = oldLookPath
	})

	tmpDir := t.TempDir()

	p := NewPostgresBackup(
		"localhost",
		5432,
		"postgres",
		"password",
		"testdb",
	)

	path, err := p.CreateBackup(context.Background(), tmpDir)

	require.Error(t, err)
	assert.Empty(t, path)

	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	assert.Empty(t, files)
}

func TestCheckDependencies_Success(t *testing.T) {
	old := lookPath

	lookPath = func(string) (string, error) {
		return "pg_dump", nil
	}

	t.Cleanup(func() {
		lookPath = old
	})

	require.NoError(t, checkDependencies())
}

func TestCheckDependencies_Error(t *testing.T) {
	old := lookPath

	lookPath = func(string) (string, error) {
		return "", errors.New("not found")
	}

	t.Cleanup(func() {
		lookPath = old
	})

	p := NewPostgresBackup(
		"localhost",
		5432,
		"postgres",
		"password",
		"testdb",
	)

	_, err := p.CreateBackup(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pg_dump")
}

func TestCompressTo_Success(t *testing.T) {
	var compressed bytes.Buffer

	err := compressTo(
		&compressed,
		strings.NewReader("hello world"),
	)

	require.NoError(t, err)

	gz, err := gzip.NewReader(&compressed)
	require.NoError(t, err)
	defer gz.Close()

	data, err := io.ReadAll(gz)
	require.NoError(t, err)

	assert.Equal(t, "hello world", string(data))
}

func TestBuildBackupPath(t *testing.T) {
	dir := t.TempDir()

	path := buildBackupPath(dir)

	assert.True(t, strings.HasPrefix(path, dir))
	assert.True(t, strings.HasSuffix(path, ".sql.gz"))

	base := filepath.Base(path)
	assert.NotEmpty(t, base)
}

func TestGetBackupType(t *testing.T) {
	p := NewPostgresBackup(
		"localhost",
		5432,
		"user",
		"pass",
		"db",
	)

	assert.Equal(t, Postgres, p.GetBackupType())
}

func TestNewPostgresBackupper(t *testing.T) {
    t.Run("success", func(t *testing.T) {
        t.Setenv("PG_ENABLE", "true")
        
        backupper, err := newPostgresBackupper()
        require.NoError(t, err)
        assert.NotNil(t, backupper)
    })

    t.Run("disabled", func(t *testing.T) {
        t.Setenv("PG_ENABLE", "false")
        
        backupper, err := newPostgresBackupper()
        assert.ErrorIs(t, err, ErrDisabled)
        assert.Nil(t, backupper)
    })

    t.Run("config error", func(t *testing.T) {
        t.Setenv("PG_ENABLE", "true")
        t.Setenv("PG_HOST", "") // вызываем ошибку валидации
        
        backupper, err := newPostgresBackupper()
        assert.Error(t, err)
        assert.Nil(t, backupper)
    })
}