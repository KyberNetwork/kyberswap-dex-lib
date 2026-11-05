package dexv2

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	DexID       string
	ChainID     valueobject.ChainID `json:"chainID"`
	Dex         string              `json:"dex"`
	Resolver    string              `json:"resolver"`
	Liquidity   string              `json:"liquidity"`
	SubgraphAPI string              `json:"subgraphAPI"`

	// UseLens reads pools through FluidDexV2Lens: their state and every tick
	// from one block in one eth_call, or a few when the ticks do not fit. A bootstrap then asks no
	// subgraph, a pool without state, such as one the factory built from its LogInitialize, gets it
	// whole on its next update, and ticks accumulated from logs that turn out invalid are replaced
	// by the lens's set, which finds a tick the stored set lacked.
	ephemeral.LensConfig
}
