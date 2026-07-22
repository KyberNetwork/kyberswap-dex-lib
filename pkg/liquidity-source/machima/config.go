package machima

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
	// RouterAddress is the MachimaAggregatorRouter the executor approves and calls.
	RouterAddress string `json:"routerAddress"`

	// Counter assets, mirroring the router's _isCounterAsset set.
	WETH string `json:"weth"`
	USDC string `json:"usdc"`
	XMA  string `json:"xma"`
}
