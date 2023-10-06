package gmxglp

type Config struct {
	DexID                   string `json:"-"`
	RewardRouterAddress     string `json:"rewardRouterAddress"`
	VaultAddress            string `json:"vaultAddress"`
	GlpManagerAddress       string `json:"glpManagerAddress"`
	UseSecondaryPriceFeedV1 bool   `json:"useSecondaryPriceFeedV1"`
}
