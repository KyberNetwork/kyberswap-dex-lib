package gmx

type Config struct {
	DexID                   string        `json:"dexID"`
	VaultAddress            string        `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool          `json:"useSecondaryPriceFeedV1"`
	PriceFeedType           PriceFeedType `json:"priceFeedType"`
	UsdgForkName            string        `json:"usdgForkName"`
}
