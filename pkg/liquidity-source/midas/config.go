package midas

type Config struct {
	DexId   string                  `json:"dexId"`
	MTokens map[string]MTokenConfig `json:"mTokens"`
}

type MTokenConfig struct {
	DepositVaultType    *depositVaultType    `json:"depositVaultType"`
	DepositVault        string               `json:"depositVault"`
	RedemptionVaultType *redemptionVaultType `json:"redemptionVaultType"`
	RedemptionVault     string               `json:"redemptionVault"`
}
