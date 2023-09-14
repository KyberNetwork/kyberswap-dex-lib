package biswap

type Config struct {
	DexID          string
	SwapFee        float64 `json:"swapFee"`
	FeePrecision   int     `json:"feePrecision"`
	FactoryAddress string  `json:"factoryAddress"`
	NewPoolLimit   int     `json:"newPoolLimit"`
}
