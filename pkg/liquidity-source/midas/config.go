package midas

type Config struct {
	DexId      string `json:"dexId"`
	ConfigPath string `json:"configPath"`
}

type MTokenConfig struct {
	MToken              string `json:"token"`
	DepositVaultType    string `json:"depositVaultType"`
	DepositVault        string `json:"depositVault"`
	RedemptionVaultType string `json:"redemptionVaultType"`
	RedemptionVault     string `json:"redemptionVault"`
}
