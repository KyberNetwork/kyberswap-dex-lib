package getroutev2

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID          valueobject.ChainID `mapstructure:"chainID"`
	RouterAddress    string              `mapstructure:"routerAddress"`
	GasTokenAddress  string              `mapstructure:"gasTokenAddress"`
	AvailableSources []string            `mapstructure:"availableSources"`

	AmmAggregator AmmAggregatorConfig `mapstructure:"ammAggregator"`
	Cache         CacheConfig         `mapstructure:"cache"`
	PoolManager   PoolManagerConfig   `mapstructure:"poolManager"`
	PoolFactory   PoolFactoryConfig   `mapstructure:"poolFactory"`
}

type AmmAggregatorConfig struct {
	WhitelistedTokenSet map[string]struct{}       `mapstructure:"whitelistedTokenSet"`
	GetBestPoolsOptions types.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions"`
	FinderOptions       FinderOptions             `mapstructure:"finderOptions"`
}

type CachePoint struct {
	Amount float64       `mapstructure:"amount"`
	TTL    time.Duration `mapstructure:"ttl"`
}

type CacheRange struct {
	AmountUSDLowerBound float64       `mapstructure:"amountUSDLowerBound"`
	TTL                 time.Duration `mapstructure:"ttl"`
}

type CacheConfig struct {
	// DefaultTTL default time to live of the cache
	DefaultTTL time.Duration `mapstructure:"defaultTtl"`

	// TTLByAmount time to live by amount
	// key is amount without decimals
	TTLByAmount []CachePoint `mapstructure:"ttlByAmount"`

	// TTLByAmountUSDRange time to live by amount usd range
	// key is lower bound of the range
	TTLByAmountUSDRange []CacheRange `mapstructure:"ttlByAmountUsdRange"`

	PriceImpactThreshold float64 `mapstructure:"priceImpactThreshold"`

	ShrinkFuncName   string  `mapstructure:"shrinkFuncName"`
	ShrinkFuncPowExp float64 `mapstructure:"shrinkFuncPowExp"`
}

type PoolManagerConfig struct {
	BlacklistedPoolSet map[string]struct{} `mapstructure:"blacklistedPoolSet"`
}

type FinderOptions struct {
	MaxHops                 uint32  `mapstructure:"maxHops"`
	DistributionPercent     uint32  `mapstructure:"distributionPercent"`
	MaxPathsInRoute         uint32  `mapstructure:"maxPathsInRoute"`
	MaxPathsToGenerate      uint32  `mapstructure:"maxPathsToGenerate"`
	MaxPathsToReturn        uint32  `mapstructure:"maxPathsToReturn"`
	MinPartUSD              float64 `mapstructure:"minPartUSD"`
	MinThresholdAmountInUSD float64 `mapstructure:"minThresholdAmountInUSD"`
	MaxThresholdAmountInUSD float64 `mapstructure:"maxThresholdAmountInUSD"`
}

type PoolFactoryConfig struct {
	ChainID int `mapstructure:"chainId"`
}
