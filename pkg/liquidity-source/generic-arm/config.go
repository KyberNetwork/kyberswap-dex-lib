package genericarm

type Config struct {
	DexID string            `json:"dexId"`
	Arms  map[string]ArmCfg `json:"arms"`
}

type ArmCfg struct {
	Gas                GasCfg   `json:"gas"`
	SwapType           SwapType `json:"swapType"`
	ArmType            ArmType  `json:"armType"`
	HasWithdrawalQueue bool     `json:"hasWithdrawalQueue"`
}

type GasCfg struct {
	Deposit uint64 `json:"deposit"`
	Redeem  uint64 `json:"redeem"`
}
