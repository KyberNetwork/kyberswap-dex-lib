package fulcrom

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

type ChainID uint

type SecondaryPriceFeedVersion int
