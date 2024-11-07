package swapbasedperp

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

const (
	secondaryPriceFeedVersion1 SecondaryPriceFeedVersion = 1
	secondaryPriceFeedVersion2 SecondaryPriceFeedVersion = 2
)
