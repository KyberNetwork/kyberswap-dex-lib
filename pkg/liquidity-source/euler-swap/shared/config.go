package shared

type Config struct {
	DexID          string `json:"dexID"`
	FactoryAddress string `json:"factoryAddress"`

	// UseLens reads euler-swap-v2 pools in batches through EulerSwapV2Lens,
	// in as few eth_calls as the node takes, where each pool otherwise takes up to four
	// multicalls of its own.
	UseLens bool `json:"useLens,omitempty"`
}
