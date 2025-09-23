package midas

type Config struct {
	DexId      string `json:"dexId"`
	Executor   string `json:"executor"`
	ConfigPath string `json:"configPath"`
}

type MTokenConfig struct {
	MToken              string    `json:"token"`
	DepositVaultType    VaultType `json:"depositVaultType"`
	DepositVault        string    `json:"depositVault"`
	RedemptionVaultType VaultType `json:"redemptionVaultType"`
	RedemptionVault     string    `json:"redemptionVault"`
}
