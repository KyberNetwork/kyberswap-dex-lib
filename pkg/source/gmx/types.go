package gmx

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

const (
	SecondaryPriceFeedVersion1 SecondaryPriceFeedVersion = 1
	SecondaryPriceFeedVersion2 SecondaryPriceFeedVersion = 2
)
