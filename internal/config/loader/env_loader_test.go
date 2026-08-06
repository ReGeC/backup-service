package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvLoader_GetEnv(t *testing.T) {
	loader := &EnvLoader{}

	t.Run("returns environment value", func(t *testing.T) {
		t.Setenv("TEST_KEY", "hello")

		got := loader.GetString([]string{"test", "key"}, "default")

		assert.Equal(t, "hello", got)
	})

	t.Run("returns default value when environment variable does not exist", func(t *testing.T) {
		got := loader.GetString([]string{"test", "key", "not", "exists"}, "default")

		assert.Equal(t, "default", got)
	})
}

func TestEnvLoader_GetEnvAsInt(t *testing.T) {
	loader := &EnvLoader{}

	tests := []struct {
		name      string
		envValue  string
		envExists bool
		defaultV  int
		want      int
	}{
		{
			name:      "valid integer",
			envValue:  "42",
			envExists: true,
			defaultV:  10,
			want:      42,
		},
		{
			name:      "missing variable",
			envExists: false,
			defaultV:  10,
			want:      10,
		},
		{
			name:      "invalid integer",
			envValue:  "hello",
			envExists: true,
			defaultV:  10,
			want:      10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envExists {
				t.Setenv("TEST_INT", tt.envValue)
			}

			got := loader.GetInt([]string{"test", "int"}, tt.defaultV)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEnvLoader_GetEnvAsBool(t *testing.T) {
	loader := &EnvLoader{}

	tests := []struct {
		name      string
		envValue  string
		envExists bool
		defaultV  bool
		want      bool
	}{
		{
			name:      "returns parsed true",
			envValue:  "true",
			envExists: true,
			defaultV:  false,
			want:      true,
		},
		{
			name:      "returns parsed false",
			envValue:  "false",
			envExists: true,
			defaultV:  true,
			want:      false,
		},
		{
			name:      "returns default when variable does not exist",
			envExists: false,
			defaultV:  true,
			want:      true,
		},
		{
			name:      "returns default when value is invalid",
			envValue:  "hello",
			envExists: true,
			defaultV:  true,
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envExists {
				t.Setenv("TEST_BOOL", tt.envValue)
			}

			got := loader.GetBool([]string{"test", "bool"}, tt.defaultV)

			assert.Equal(t, tt.want, got)
		})
	}
}