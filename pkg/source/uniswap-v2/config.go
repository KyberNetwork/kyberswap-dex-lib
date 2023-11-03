package uniswapv2

type Config struct {
	DexID          string `json:"dexID"`
	Fee            int64  `json:"fee"`
	FeePrecision   int64  `json:"feePrecision"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
