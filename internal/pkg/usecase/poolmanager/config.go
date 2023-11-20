package poolmanager

import "time"

type Config struct {
	BlacklistedPoolSet  map[string]bool `mapstructure:"blacklistedPoolSet"`
	Capacity            int             `mapstructure:"capacity" json:"capacity"`
	PoolRenewalInterval time.Duration   `mapstructure:"poolRenewalInterval" json:"poolRenewalInterval"`
	UseAEVM             bool            `mapstructure:"useAEVM" json:"useAEVM"`
}
