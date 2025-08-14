package uniswapv2

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type Config struct {
	DexID              string                    `json:"dexID"`
	FactoryAddress     string                    `json:"factoryAddress"`
	OldReserveMethods  bool                      `json:"oldReserveMethods"`
	Fee                uint64                    `json:"fee"`
	FeePrecision       uint64                    `json:"feePrecision"`
	FeeTracker         *FeeTrackerCfg            `json:"feeTracker"`
	NewPoolLimit       int                       `json:"newPoolLimit"`
	TrackInactivePools *TrackInactivePoolsConfig `json:"trackInactivePools,omitempty"`
}

type FeeTrackerCfg struct {
	Target   string   `json:"target"`
	Selector uint32   `json:"selector"`
	Args     []string `json:"args"`
}

type TrackInactivePoolsConfig struct {
	Enabled       bool                  `json:"enabled"`
	TimeThreshold durationjson.Duration `json:"timeThreshold"`
}
