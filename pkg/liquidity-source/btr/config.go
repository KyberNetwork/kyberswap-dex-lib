package btr

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config is the dex-lib exchange configuration. The factory watch list is the
// BTR PoolFactory on Monad; new cores appear as factory events. SamplePoints
// bounds the per-pair on-chain probe grid the tracker uses to build each ladder.
type Config struct {
	DexID        string              `json:"dexID"`
	ChainID      valueobject.ChainID `json:"chainId"`
	Factory      string              `json:"factory"`
	StartBlock   uint64              `json:"startBlock"`
	SamplePoints int                 `json:"samplePoints"`
	// LaneGranted reports that BTR governance has granted this source's
	// executor the SWAP_GATED lane (0x400) on the configured cores. A gated
	// leg stays refused while this is false, which is the safe default for
	// every core that has not been granted the lane. Set true only after the
	// on-chain grant; the tracker stamps it onto each pool's Extra.
	LaneGranted bool `json:"laneGranted"`
}

// NewConfig returns the Monad defaults. SamplePoints is capped by
// maxSamplePoints; a smaller grid lowers RPC load at the cost of interpolation
// error between samples.
func NewConfig(chainID valueobject.ChainID) *Config {
	return &Config{
		DexID:        DexType,
		ChainID:      chainID,
		Factory:      PoolFactory,
		StartBlock:   0,
		SamplePoints: defaultSamplePoints,
	}
}
