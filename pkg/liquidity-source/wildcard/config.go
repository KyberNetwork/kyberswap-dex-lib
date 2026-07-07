package wildcard

type Config struct {
	DexID            string `json:"dexID"`
	ChainID          int    `json:"chainID"`
	MulticallAddress string `json:"multicallAddress"`
	FactoryAddress   string `json:"factoryAddress"`
	Fee              uint64 `json:"fee"`
	FeePrecision     uint64 `json:"feePrecision"`
	NewPoolLimit     int    `json:"newPoolLimit"`
	PriceTolerance   int64  `json:"priceTolerance"`
}
