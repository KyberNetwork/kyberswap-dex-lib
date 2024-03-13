package poolmanager

import "time"

type Config struct {
	BlacklistedPoolSet         map[string]bool `mapstructure:"blacklistedPoolSet"`
	Capacity                   int             `mapstructure:"capacity" json:"capacity"`
	PoolRenewalInterval        time.Duration   `mapstructure:"poolRenewalInterval" json:"poolRenewalInterval"`
	UseAEVM                    bool            `mapstructure:"useAEVM" json:"useAEVM"`
	FaultyPoolsExpireThreshold time.Duration   `mapstructure:"faultyPoolsExpireThreshold" json:"faultyPoolsExpireThreshold"`
	MaxFaultyPoolSize          int64           `mapstructure:"maxFaultyPoolSize" json:"maxFaultyPoolSize"`
	//StallingPMMThreshold determine the duration a PMM pool is updated before it is marked as stalled
	// non-configured stalling threshold is treat as non-enabling stalling threshold
	StallingPMMThreshold time.Duration `mapstructure:"stallingPMMThreshold" json:"stallingPMMThreshold"`
}
