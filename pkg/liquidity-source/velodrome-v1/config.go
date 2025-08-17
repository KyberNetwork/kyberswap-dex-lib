package velodromev1

import (
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type Config struct {
	DexID          string         `json:"dexID"`
	FactoryAddress string         `json:"factoryAddress"`
	FeePrecision   uint64         `json:"feePrecision"`
	FeeTracker     *FeeTrackerCfg `json:"feeTracker"`
	NewPoolLimit   int            `json:"newPoolLimit"`

	TrackInactivePools *pooltrack.TrackInactivePoolsConfig `json:"trackInactivePools"`
}

type FeeTrackerCfg struct {
	Target   string   `json:"target"`
	Selector uint32   `json:"selector"`
	Args     []string `json:"args"`
}
