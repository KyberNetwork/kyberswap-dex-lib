package erc4626

type Config struct {
	DexId  string              `json:"dexId"`
	Vaults map[string]VaultCfg `json:"vaults"`
}

type VaultCfg struct {
	Gas       GasCfg         `json:"gas"`
	SwapTypes SwapType       `json:"swapTypes"`
	Meta      map[string]any `json:"meta,omitempty"`
}

type GasCfg struct {
	Deposit uint64 `json:"deposit"`
	Redeem  uint64 `json:"redeem"`
}
