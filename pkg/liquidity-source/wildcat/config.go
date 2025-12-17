package wildcat

type Config struct {
	DexID          string `json:"dexID"`
	ChainID        int    `json:"chainID"`
	FactoryAddress string `json:"factoryAddress"`
	Fee            uint64 `json:"fee"`
	FeePrecision   uint64 `json:"feePrecision"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
