package whlp

type Config struct {
	VaultAddress      string `json:"vaultAddress"`
	AccountantAddress string `json:"accountantAddress"`
	QuoteAssetAddress string `json:"quoteAssetAddress"`
	DepositorAddress  string `json:"depositorAddress"`
}
