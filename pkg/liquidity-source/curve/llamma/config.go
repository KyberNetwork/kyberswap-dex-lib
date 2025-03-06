package llamma

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID                 string                `json:"dexID"`
	FactoryAddress        string                `json:"factoryAddress"`
	StableCoin            string                `json:"stableCoin"`
	NewPoolLimit          int                   `json:"newPoolLimit"`
	AllowSubgraphError    bool                  `json:"allowSubgraphError"`
	FetchPoolsMinDuration durationjson.Duration `json:"fetch_pools_min_duration,omitempty"`
}
