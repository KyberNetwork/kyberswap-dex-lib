package curve

type Config struct {
	DexID        string `json:"dexID"`
	ChainID      int    `json:"chainID"`
	PoolPath     string `json:"poolPath"`
	NewPoolLimit int    `json:"newPoolLimit"`

	AddressProvider            string `json:"addressProvider"`
	MainRegistryAddress        string `json:"mainRegistryAddress"`
	MetaPoolsFactoryAddress    string `json:"metaPoolsFactoryAddress"`
	CryptoPoolsRegistryAddress string `json:"cryptoPoolsRegistryAddress"`
	CryptoPoolsFactoryAddress  string `json:"cryptoPoolsFactoryAddress"`
}
