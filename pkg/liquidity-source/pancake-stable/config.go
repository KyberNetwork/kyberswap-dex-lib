package pancakestable

type Config struct {
	DexID        string `json:"dexID"`
	ChainID      int    `json:"chainID"`
	NewPoolLimit int    `json:"newPoolLimit"`
	// SkipInitFactory include dexes that don't have factory.
	SkipInitFactory bool   `json:"skipInitFactory"`
	FactoryAddress  string `json:"factoryAddress"`
}
