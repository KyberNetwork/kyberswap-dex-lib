package zkswap

type Config struct {
	DexID          string  `json:"dexID"`
	SwapFee        float64 `json:"swapFee"`
	FactoryAddress string  `json:"factoryAddress"`
	NewPoolLimit   int     `json:"newPoolLimit"`
}
