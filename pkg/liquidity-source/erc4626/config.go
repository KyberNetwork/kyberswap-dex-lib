package erc4626

type Config struct {
	DexId     string   `json:"dexId"`
	Vault     string   `json:"vault"`
	Gas       Gas      `json:"gas"`
	SwapTypes SwapType `json:"swapTypes"`
}
