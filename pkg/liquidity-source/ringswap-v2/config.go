package ringswapv2

type Config struct {
	DexID          string `json:"dexID"`
	Fee            uint64 `json:"fee"`
	FeePrecision   uint64 `json:"feePrecision"`
	FeeTracker     string `json:"feeTracker"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
	FewFactory     string `json:"fewFactory"`
}
