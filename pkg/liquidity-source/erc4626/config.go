package erc4626

type Config struct {
	DexId  string              `json:"dexId"`
	Vaults map[string]VaultCfg `json:"vaults"`
}

type VaultCfg struct {
	Gas       Gas      `json:"gas"`
	SwapTypes SwapType `json:"swapTypes"`
}
