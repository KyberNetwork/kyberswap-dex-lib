package hyperamm

// Config is the per-DEX configuration loaded from the aggregator config file.
type Config struct {
	// DexId is the identifier used by the aggregator (matches DexType).
	DexId string `json:"dexId"`
	// Factory is the HyperAMMFactory contract address.
	Factory string `json:"factory"`
	// Lens is the HyperAMMLens contract address.
	// Obtain it once via HyperAMMSwapRouter.hyperAMMLens().
	Lens string `json:"lens"`
	// SwapRouter is the HyperAMMSwapRouter contract address.
	SwapRouter string `json:"swapRouter"`
}
