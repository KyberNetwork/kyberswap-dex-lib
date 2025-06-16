package genericarm

type Config struct {
	ArmAddress string   `json:"armAddress"`
	DexID      string   `json:"dexId"`
	SwapTypes  SwapType `json:"swapTypes"`
}
