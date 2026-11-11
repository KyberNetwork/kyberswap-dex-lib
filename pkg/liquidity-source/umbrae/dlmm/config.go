package umbraedlmm

// Config drives DLMM discovery and tracking.
type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`
	ViewerAddress  string `json:"viewerAddress"` // PairViewer: fronts bins/quotes (the pair routes them to an extension)
	RouterAddress  string `json:"routerAddress"` // DLMM Router: the swap entry point and token spender (the pair reverts on direct calls)
	NewPoolLimit   int    `json:"newPoolLimit"`
	// SwapRevision is the pair build this deployment runs (SwapRevisionV2 on Base,
	// SwapRevisionV2Oct2026 on Robinhood Chain). Zero keeps the Base behaviour.
	SwapRevision uint8 `json:"swapRevision"`
}
