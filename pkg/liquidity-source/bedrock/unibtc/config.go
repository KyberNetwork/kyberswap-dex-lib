package unibtc

type Config struct {
	Vaults map[string]VaultCfg `json:"vaults"`
}

type VaultType string

const (
	VaultTypeUniBTC VaultType = "unibtc"
	VaultTypeBrBTC  VaultType = "brbtc"
)

type VaultCfg struct {
	Type   VaultType `json:"type"`
	Tokens []string  `json:"tokens"`
}
