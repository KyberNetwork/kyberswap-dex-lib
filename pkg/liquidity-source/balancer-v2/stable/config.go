package stable

type Config struct {
	DexID        string `json:"dexID"`
	VaultAddress string `json:"vaultAddress"`
	SubgraphAPI  string `json:"subgraphAPI"`
	NewPoolLimit int    `json:"newPoolLimit"`
}
