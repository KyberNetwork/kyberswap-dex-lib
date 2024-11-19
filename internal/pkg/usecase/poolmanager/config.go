package poolmanager

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	BlacklistedPoolSet         map[string]bool `mapstructure:"blacklistedPoolSet"`
	Capacity                   int             `mapstructure:"capacity" json:"capacity"`
	PoolRenewalInterval        time.Duration   `mapstructure:"poolRenewalInterval" json:"poolRenewalInterval"`
	BlackListRenewalInterval   time.Duration   `mapstructure:"blackListRenewalInterval" json:"blackListRenewalInterval"`
	FaultyPoolsRenewalInterval time.Duration   `mapstructure:"faultyPoolsRenewalInterval" json:"faultyPoolsRenewalInterval"`
	UseAEVMRemoteFinder        bool            `mapstructure:"useAEVMRemoteFinder" json:"useAEVMRemoteFinder"`
	//StallingPMMThreshold determine the duration a PMM pool is updated before it is marked as stalled
	// non-configured stalling threshold is treat as non-enabling stalling threshold
	StallingPMMThreshold time.Duration            `mapstructure:"stallingPMMThreshold" json:"stallingPMMThreshold"`
	FeatureFlags         valueobject.FeatureFlags `mapstructure:"featureFlags"`
}
