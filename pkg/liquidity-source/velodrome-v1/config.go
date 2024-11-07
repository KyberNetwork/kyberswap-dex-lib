package velodromev1

type Config struct {
	DexID          string `json:"dexID"`
	FeePrecision   uint64 `json:"feePrecision"`
	FeeTracker     string `json:"feeTracker"`
	FactoryAddress string `json:"factoryAddress"`
	NewPoolLimit   int    `json:"newPoolLimit"`
}
