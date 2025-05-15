package brownfi

type Config struct {
	DexID          string `json:"dexID"`
	Fee            uint64 `json:"fee"`
	FeePrecision   uint64 `json:"feePrecision"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
