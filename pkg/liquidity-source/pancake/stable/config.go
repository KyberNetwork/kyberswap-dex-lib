package pancakestable

type Config struct {
	DexID          string `json:"dexID"`
	ChainID        int    `json:"chainID"`
	NewPoolLimit   int    `json:"newPoolLimit"`
	FactoryAddress string `json:"factoryAddress"`
}
