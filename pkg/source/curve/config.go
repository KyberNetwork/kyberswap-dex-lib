package curve

type Config struct {
	DexID        string `json:"dexID"`
	ChainID      int    `json:"chainID"`
	PoolPath     string `json:"poolPath"`
	NewPoolLimit int    `json:"newPoolLimit"`
	// SkipInitFactory include dexes that don't have factory.
	SkipInitFactory bool `json:"skipInitFactory"`

	AddressProvider            string `json:"addressProvider"`
	MainRegistryAddress        string `json:"mainRegistryAddress"`
	MetaPoolsFactoryAddress    string `json:"metaPoolsFactoryAddress"`
	CryptoPoolsRegistryAddress string `json:"cryptoPoolsRegistryAddress"`
	CryptoPoolsFactoryAddress  string `json:"cryptoPoolsFactoryAddress"`
}
