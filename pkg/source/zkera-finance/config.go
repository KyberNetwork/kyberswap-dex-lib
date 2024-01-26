package zkerafinance

type Config struct {
	DexID                   string `json:"dexID"`
	VaultAddress            string `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool   `json:"useSecondaryPriceFeedV1"`
}
