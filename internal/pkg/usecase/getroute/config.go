package getroute

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID          valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress    string              `mapstructure:"routerAddress" json:"routerAddress"`
	GasTokenAddress  string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources []string            `mapstructure:"availableSources" json:"availableSources"`

	AmmAggregator AmmAggregatorConfig `mapstructure:"ammAggregator" json:"ammAggregator"`
	Cache         CacheConfig         `mapstructure:"cache" json:"cache"`
}

type AmmAggregatorConfig struct {
	WhitelistedTokenSet map[string]bool                 `mapstructure:"whitelistedTokenSet" json:"whitelistedTokenSet"`
	GetBestPoolsOptions valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions" json:"getBestPoolsOptions"`
	FinderOptions       valueobject.FinderOptions       `mapstructure:"finderOptions" json:"finderOptions"`
}

type CachePoint struct {
	Amount float64       `mapstructure:"amount" json:"amount"`
	TTL    time.Duration `mapstructure:"ttl" json:"ttl"`
}

type CacheRange struct {
	AmountUSDLowerBound float64       `mapstructure:"amountUSDLowerBound" json:"amountUSDLowerBound"`
	TTL                 time.Duration `mapstructure:"ttl" json:"ttl"`
}

type CacheConfig struct {
	// DefaultTTL default time to live of the cache
	DefaultTTL time.Duration `mapstructure:"defaultTtl" json:"defaultTtl"`

	// TTLByAmount time to live by amount
	// key is amount without decimals
	TTLByAmount []CachePoint `mapstructure:"ttlByAmount" json:"ttlByAmount"`

	// TTLByAmountUSDRange time to live by amount usd range
	// key is lower bound of the range
	TTLByAmountUSDRange []CacheRange `mapstructure:"ttlByAmountUsdRange" json:"ttlByAmountUsdRange"`

	PriceImpactThreshold float64 `mapstructure:"priceImpactThreshold" json:"priceImpactThreshold"`

	ShrinkFuncName   string  `mapstructure:"shrinkFuncName" json:"shrinkFuncName"`
	ShrinkFuncPowExp float64 `mapstructure:"shrinkFuncPowExp" json:"shrinkFuncPowExp"`
}
