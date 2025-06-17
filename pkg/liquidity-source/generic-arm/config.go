package genericarm

type Config struct {
	ArmAddress string   `json:"armAddress"`
	DexID      string   `json:"dexId"`
	SwapType   SwapType `json:"swapType"`
	ArmType    ArmType  `json:"armType"`
}
