package storage_test

import (
	"backup-service/internal/storage"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewLocalStorage тестирует создание локального хранилища
func TestNewLocalStorage(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "with valid path",
			path: t.TempDir(),
		},
		{
			name: "with empty path",
			path: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := storage.NewLocalStorage(tt.path)
			assert.NotNil(t, s)
		})
	}
}

// TestLocalStorage_Save тестирует метод Save
func TestLocalStorage_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("save file by moving", func(t *testing.T) {
		srcDir := t.TempDir()
		destDir := t.TempDir()
		
		srcFile := filepath.Join(srcDir, "backup.zip")
		err := os.WriteFile(srcFile, []byte("test data"), 0644)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		filename, err := s.Save(ctx, srcFile)
		require.NoError(t, err)
		
		destFile := filepath.Join(destDir, "backup.zip")
		assert.Equal(t, destFile, filename)
		
		_, err = os.Stat(destFile)
		assert.NoError(t, err)
		
		_, err = os.Stat(srcFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("save file when source equals destination", func(t *testing.T) {
		destDir := t.TempDir()
		destFile := filepath.Join(destDir, "backup.zip")
		err := os.WriteFile(destFile, []byte("test data"), 0644)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		filename, err := s.Save(ctx, destFile)
		require.NoError(t, err)
		assert.Equal(t, destFile, filename)
		
		_, err = os.Stat(destFile)
		assert.NoError(t, err)
	})

	t.Run("save file creates backup directory if not exists", func(t *testing.T) {
		srcDir := t.TempDir()
		srcFile := filepath.Join(srcDir, "backup.zip")
		err := os.WriteFile(srcFile, []byte("test data"), 0644)
		require.NoError(t, err)

		destDir := filepath.Join(t.TempDir(), "new", "backup", "path")
		
		s := storage.NewLocalStorage(destDir)
		
		filename, err := s.Save(ctx, srcFile)
		require.NoError(t, err)
		
		_, err = os.Stat(destDir)
		assert.NoError(t, err)
		
		_, err = os.Stat(filename)
		assert.NoError(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		destDir := t.TempDir()
		s := storage.NewLocalStorage(destDir)
		
		nonExistentFile := filepath.Join(t.TempDir(), "non-existent.zip")
		filename, err := s.Save(ctx, nonExistentFile)
		
		require.Error(t, err)
		assert.Empty(t, filename)
		assert.ErrorContains(t, err, "Файл бэкапа не найден")
	})
}

// TestLocalStorage_List тестирует метод List
func TestLocalStorage_List(t *testing.T) {
	ctx := context.Background()

	t.Run("list files successfully", func(t *testing.T) {
		destDir := t.TempDir()
		
		files := []string{"file1.zip", "file2.zip", "file3.txt"}
		for _, name := range files {
			path := filepath.Join(destDir, name)
			err := os.WriteFile(path, []byte("test"), 0644)
			require.NoError(t, err)
		}
		
		subDir := filepath.Join(destDir, "subdir")
		err := os.Mkdir(subDir, 0755)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		list, err := s.List(ctx)
		require.NoError(t, err)
		assert.Len(t, list, 3)
		
		foundFiles := make(map[string]bool)
		for _, info := range list {
			foundFiles[info.Name] = true
			assert.Greater(t, info.Size, int64(0))
			assert.False(t, info.CreatedAt.IsZero())
		}
		
		for _, name := range files {
			assert.True(t, foundFiles[name], "File %s not found", name)
		}
	})

	t.Run("list empty directory", func(t *testing.T) {
		destDir := t.TempDir()
		s := storage.NewLocalStorage(destDir)
		
		list, err := s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("list directory does not exist", func(t *testing.T) {
		nonExistentDir := filepath.Join(t.TempDir(), "non-existent")
		s := storage.NewLocalStorage(nonExistentDir)
		
		list, err := s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})

	t.Run("list directory with only subdirectories", func(t *testing.T) {
		destDir := t.TempDir()
		
		err := os.Mkdir(filepath.Join(destDir, "sub1"), 0755)
		require.NoError(t, err)
		err = os.Mkdir(filepath.Join(destDir, "sub2"), 0755)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		list, err := s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, list)
	})
}

// TestLocalStorage_Delete тестирует метод Delete
func TestLocalStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("delete file by absolute path", func(t *testing.T) {
		destDir := t.TempDir()
		filePath := filepath.Join(destDir, "file-to-delete.zip")
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		err = s.Delete(ctx, filePath)
		require.NoError(t, err)
		
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete file by relative path", func(t *testing.T) {
		destDir := t.TempDir()
		fileName := "file-to-delete.zip"
		filePath := filepath.Join(destDir, fileName)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)

		s := storage.NewLocalStorage(destDir)
		
		err = s.Delete(ctx, fileName)
		require.NoError(t, err)
		
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		destDir := t.TempDir()
		s := storage.NewLocalStorage(destDir)
		
		err := s.Delete(ctx, "non-existent.zip")
		require.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("delete file outside storage by absolute path", func(t *testing.T) {
		storageDir := t.TempDir()
		outsideDir := t.TempDir()
		filePath := filepath.Join(outsideDir, "outside-file.zip")
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err)

		s := storage.NewLocalStorage(storageDir)
		
		err = s.Delete(ctx, filePath)
		require.NoError(t, err)
		
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})
}

// TestLocalStorage_FullLifecycle тест полного жизненного цикла
func TestLocalStorage_FullLifecycle(t *testing.T) {
	ctx := context.Background()

	t.Run("full lifecycle: save -> list -> delete", func(t *testing.T) {
		destDir := t.TempDir()
		srcDir := t.TempDir()
		
		s := storage.NewLocalStorage(destDir)
		
		// 1. Сохраняем файл
		srcFile := filepath.Join(srcDir, "backup.zip")
		err := os.WriteFile(srcFile, []byte("test data"), 0644)
		require.NoError(t, err)
		
		filename, err := s.Save(ctx, srcFile)
		require.NoError(t, err)
		
		destFile := filepath.Join(destDir, "backup.zip")
		assert.Equal(t, destFile, filename)
		
		// 2. Получаем список файлов
		files, err := s.List(ctx)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "backup.zip", files[0].Name)
		
		// 3. Удаляем файл
		err = s.Delete(ctx, "backup.zip")
		require.NoError(t, err)
		
		// 4. Проверяем, что список пуст
		files, err = s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("save multiple files and list", func(t *testing.T) {
		destDir := t.TempDir()
		s := storage.NewLocalStorage(destDir)
		
		filenames := []string{"backup1.zip", "backup2.zip", "backup3.zip"}
		for _, name := range filenames {
			srcDir := t.TempDir()
			srcFile := filepath.Join(srcDir, name)
			err := os.WriteFile(srcFile, []byte("test "+name), 0644)
			require.NoError(t, err)
			
			_, err = s.Save(ctx, srcFile)
			require.NoError(t, err)
		}
		
		files, err := s.List(ctx)
		require.NoError(t, err)
		assert.Len(t, files, 3)
		
		foundNames := []string{}
		for _, f := range files {
			foundNames = append(foundNames, f.Name)
		}
		for _, name := range filenames {
			assert.Contains(t, foundNames, name)
		}
		
		for _, name := range filenames {
			err = s.Delete(ctx, name)
			require.NoError(t, err)
		}
		
		files, err = s.List(ctx)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

// TestLocalStorage_EdgeCases тестирует краевые случаи
func TestLocalStorage_EdgeCases(t *testing.T) {
	ctx := context.Background()

	// Табличный тест для разных имен файлов
	t.Run("save with various filenames", func(t *testing.T) {
		tests := []struct {
			name     string
			filename string
		}{
			{
				name:     "with special characters",
				filename: "backup-2026-01-01_12-30-45.zip",
			},
			{
				name:     "with spaces",
				filename: "backup 2026.zip",
			},
			{
				name:     "with dots",
				filename: "backup.v1.0.zip",
			},
			{
				name:     "with underscores",
				filename: "backup_2026_01_01.zip",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				destDir := t.TempDir()
				srcDir := t.TempDir()
				
				srcFile := filepath.Join(srcDir, tt.filename)
				err := os.WriteFile(srcFile, []byte("test"), 0644)
				require.NoError(t, err)
				
				s := storage.NewLocalStorage(destDir)
				
				filename, err := s.Save(ctx, srcFile)
				require.NoError(t, err)
				
				expected := filepath.Join(destDir, tt.filename)
				assert.Equal(t, expected, filename)
				
				_, err = os.Stat(expected)
				assert.NoError(t, err)
			})
		}
	})

	// Табличный тест для разных размеров файлов
	t.Run("list files of different sizes", func(t *testing.T) {
		tests := []struct {
			name     string
			size     int
			expected int64
		}{
			{
				name:     "empty file",
				size:     0,
				expected: 0,
			},
			{
				name:     "small file",
				size:     1024,
				expected: 1024,
			},
			{
				name:     "medium file",
				size:     1024 * 1024,
				expected: 1024 * 1024,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				destDir := t.TempDir()
				
				path := filepath.Join(destDir, "file.bin")
				data := make([]byte, tt.size)
				err := os.WriteFile(path, data, 0644)
				require.NoError(t, err)
				
				s := storage.NewLocalStorage(destDir)
				files, err := s.List(ctx)
				require.NoError(t, err)
				assert.Len(t, files, 1)
				assert.Equal(t, tt.expected, files[0].Size)
			})
		}
	})

	t.Run("delete with various paths", func(t *testing.T) {
		tests := []struct {
			name     string
			filename string
		}{
			{
				name:     "simple name",
				filename: "file.zip",
			},
			{
				name:     "with spaces",
				filename: "my backup.zip",
			},
			{
				name:     "with special chars",
				filename: "backup_2026-01-01.zip",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				destDir := t.TempDir()
				filePath := filepath.Join(destDir, tt.filename)
				err := os.WriteFile(filePath, []byte("test"), 0644)
				require.NoError(t, err)

				s := storage.NewLocalStorage(destDir)
				
				err = s.Delete(ctx, tt.filename)
				require.NoError(t, err)
				
				_, err = os.Stat(filePath)
				assert.True(t, os.IsNotExist(err))
			})
		}
	})
}