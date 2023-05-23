package madmex

type Config struct {
	DexID     string `json:"-"`
	VaultPath string `json:"vaultPath"`
	ChainID   int    `json:"chainID"`
}
