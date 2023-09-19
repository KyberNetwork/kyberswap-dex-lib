package iziswap

import "github.com/KyberNetwork/blockchain-toolkit/time/durationjson"

type Config struct {
	DexID        string
	ChainID      int `json:"chainID"`
	NewPoolLimit int `json:"newPoolLimit"`

	// for pool tracker
	// liquidity/limit order snapshot range is within
	// [currentPoint - PointRange, currentPoint + PointRange)
	// we recommend a value not more than 10000
	//     due to the fact that larger PointRange will take more time to fetch snapshot data
	//     and our limit order may frequently change after each exchange,
	//     so you may need to track limit order snapshot frequently via `GetNewPoolState`
	//     method of pool tracker
	// a non-positive value will be set to 2000 by default,
	// so the default range of liquidity/limitOrder distribution
	// is [currentPrice/1.2, currentPrice * 1.2)
	PointRange int `mapstructure:"point_range" json:"point_range,omitempty"`

	HTTP HTTPConfig `mapstructure:"http" json:"http,omitempty"`

	// //todo: we may use it in the future for speed up
	// preGenesisPoolAddrs []string
}

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}
