package gmx

type Config struct {
	DexID                   string `json:"-"`
	ChainID                 int    `json:"chainID"`
	VaultAddress            string `json:"vaultAddress"`
	UseSecondaryPriceFeedV2 bool   `json:"useSecondaryPriceFeedV2"`
}
