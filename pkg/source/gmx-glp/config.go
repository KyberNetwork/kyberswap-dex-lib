package gmxglp

type Config struct {
	DexID                   string `json:"-"`
	RewardRouterAddress     string `json:"rewardRouterAddress"`
	GlpManagerAddress       string `json:"glpManagerAddress"`
	StakeGLPAddress         string `json:"stakeGLPAddress"`
	YearnTokenVaultAddress  string `json:"yearnTokenVaultAddress"`
	VaultAddress            string `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool   `json:"useSecondaryPriceFeedV1"`
}
