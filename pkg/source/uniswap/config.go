package uniswap

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID              string                   `json:"dexID"`
	SwapFee            float64                  `json:"swapFee"`
	FactoryAddress     string                   `json:"factoryAddress"`
	NewPoolLimit       int                      `json:"newPoolLimit"`
	TrackInactivePools TrackInactivePoolsConfig `json:"trackInactivePools"`
}

type TrackInactivePoolsConfig struct {
	Enabled       bool                  `json:"enabled"`
	TimeThreshold durationjson.Duration `json:"timeThreshold"`
}
