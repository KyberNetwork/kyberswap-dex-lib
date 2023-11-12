package velodrome

type Config struct {
	DexID          string `json:"dexID"`
	FeePrecision   uint64 `json:"feePrecision"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
