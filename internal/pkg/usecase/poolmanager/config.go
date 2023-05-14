package poolmanager

type Config struct {
	BlacklistedPoolSet map[string]bool `mapstructure:"blacklistedPoolSet"`
}
