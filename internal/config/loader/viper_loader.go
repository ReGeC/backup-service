package loader

import (
	"strings"

	"github.com/spf13/viper"
)

type ViperLoader struct {
	viper *viper.Viper
}

func NewViperLoader(configPath string) (*ViperLoader, error) {
	v := viper.New()

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// ENV тоже оставляем как fallback
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	return &ViperLoader{
		viper: v,
	}, nil
}

func (v *ViperLoader) GetString(path []string, defaultValue string) string {
	key := strings.Join(path, ".")

	if !v.viper.IsSet(key) {
		return defaultValue
	}

	return v.viper.GetString(key)
}

func (v *ViperLoader) GetInt(path []string, defaultValue int) int {
	key := strings.Join(path, ".")

	if !v.viper.IsSet(key) {
		return defaultValue
	}

	return v.viper.GetInt(key)
}

func (v *ViperLoader) GetBool(path []string, defaultValue bool) bool {
	key := strings.Join(path, ".")

	if !v.viper.IsSet(key) {
		return defaultValue
	}

	return v.viper.GetBool(key)
}