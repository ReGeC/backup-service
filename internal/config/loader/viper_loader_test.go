package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewViperLoader(t *testing.T) {
	dir := t.TempDir()

	config := `
				backup:
				  storage: local
				  retention_days: 14

				cron:
				  enable: true
				`

	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewViperLoader(configPath)

	require.NoError(t, err)
	require.NotNil(t, loader)
}

func TestNewViperLoader_FileNotFound(t *testing.T) {
	loader, err := NewViperLoader("does-not-exist.yaml")

	require.Error(t, err)
	require.Nil(t, loader)
}

func TestViperLoader_GetValues(t *testing.T) {
	dir := t.TempDir()

	config := `
				backup:
				  storage: local
				  retention_days: 30

				cron:
				  enable: true
				`

	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewViperLoader(configPath)
	require.NoError(t, err)

	require.Equal(t,
		"local",
		loader.GetString([]string{"backup", "storage"}, ""),
	)

	require.Equal(t,
		30,
		loader.GetInt([]string{"backup", "retention_days"}, 0),
	)

	require.True(t,
		loader.GetBool([]string{"cron", "enable"}, false),
	)
}

func TestViperLoader_DefaultValues(t *testing.T) {
	dir := t.TempDir()

	config := "{}"

	configPath := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	loader, err := NewViperLoader(configPath)
	require.NoError(t, err)

	require.Equal(t,
		"default",
		loader.GetString([]string{"backup", "storage"}, "default"),
	)

	require.Equal(t,
		7,
		loader.GetInt([]string{"backup", "retention_days"}, 7),
	)

	require.True(t,
		loader.GetBool([]string{"cron", "enable"}, true),
	)
}
