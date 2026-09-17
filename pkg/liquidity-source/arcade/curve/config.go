package curve

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

// Config wires the pool factory and tracker to one ArcadeHook deployment.
type Config struct {
	DexID   string              `json:"dexId"`
	ChainID valueobject.ChainID `json:"chainId"`
	// Hook is the ArcadeHook address: it emits LaunchCreated for every launch and
	// exposes buy/sell for the bonding-curve phase of PUMP launches.
	Hook string `json:"hook"`
}
