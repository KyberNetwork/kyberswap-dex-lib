package biswap

type Config struct {
	DexID          string
	SwapFee        float64 `json:"swapFee"`
	FactoryAddress string  `json:"factoryAddress"`
	NewPoolLimit   int     `json:"newPoolLimit"`
}
