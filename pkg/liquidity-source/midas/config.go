package midas

type Config struct {
	DexId   string                  `json:"dexId"`
	MTokens map[string]MTokenConfig `json:"mTokens"`
}

type MTokenConfig struct {
	MToken              string              `json:"mToken"`
	DepositVaultType    depositVaultType    `json:"dVT"`
	DepositVault        string              `json:"dV"`
	RedemptionVaultType redemptionVaultType `json:"rVT"`
	RedemptionVault     string              `json:"rV"`
}
