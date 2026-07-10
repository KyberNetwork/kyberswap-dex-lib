package umbraedlmm

// Config drives DLMM discovery and tracking.
//   - BinWindow overrides how many bins on each side of the active bin the tracker samples
//     (0 -> defaultBinWindow).
type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	ViewerAddress  string `json:"viewerAddress"` // PairViewer: fronts bins/quotes (the pair routes them to an extension)
	RouterAddress  string `json:"routerAddress"` // DLMM Router: the swap entry point and token spender (the pair reverts on direct calls)
	NewPoolLimit   int    `json:"newPoolLimit"`
}
