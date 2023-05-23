package dystopia

type Config struct {
	DexID          string  `json:"dexID"`
	FactoryAddress string  `json:"factoryAddress"`
	NewPoolLimit   int     `json:"newPoolLimit"`
	SwapFee        float64 `json:"swapFee"`
}
