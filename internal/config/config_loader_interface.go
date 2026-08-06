package config

//go:generate mockery
type ConfigLoader interface {
	GetString(path []string, defaultValue string) string
	GetInt(path []string, defaultValue int) int
	GetBool(path []string, defaultValue bool) bool
}
