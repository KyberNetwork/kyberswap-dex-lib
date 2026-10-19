package upshiftvaultv2

// Config lists the Tokenized Vault V2 addresses; one pool is made per vault, for its reference asset.
type Config struct {
	DexID  string   `json:"dexID"`
	Vaults []string `json:"vaults"`
}
