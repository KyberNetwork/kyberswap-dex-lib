package gmx

type Config struct {
	DexID                   string `json:"-"`
	VaultAddress            string `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool   `json:"useSecondaryPriceFeedV1"`
}
