package machima

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"

// Config is the upstream-supplied config. The subgraph URL is not here: pool-service builds the
// GraphQL client from its own subgraph config block and injects it into the factories.
type Config struct {
	DexID string `json:"dexID"`

	// ClankNow is the launchpad holding the per-token tax config.
	ClankNow string `json:"clankNow"`
	// SwapAdapter exposes xmaSellSqrtPriceLimit, the launch-tick floor for XMA sells.
	SwapAdapter string `json:"swapAdapter"`
	// TickLensAddress is the UniV3 TickLens used to enumerate initialized ticks.
	TickLensAddress string `json:"tickLensAddress"`

	// UseLens, LensGas and LensGuard are the UniV3 tracker's (see uniswapv3.Config): the UniV3
	// half of a pool is read whole through V3Lens, in place of the TickLens sweep.
	UseLens   bool                  `json:"useLens,omitempty"`
	LensGas   uint64                `json:"lensGas,omitempty"`
	LensGuard ephemeral.GuardConfig `json:"lensGuard"`
	// RouterAddress is the MachimaAggregatorRouter the executor approves and calls.
	RouterAddress string `json:"routerAddress"`

	// Counter assets, mirroring the router's _isCounterAsset set.
	WETH string `json:"weth"`
	USDC string `json:"usdc"`
	XMA  string `json:"xma"`
}
