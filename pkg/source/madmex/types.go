package madmex

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

const POLYGON ChainID = 137

const (
	SecondaryPriceFeedVersion1       SecondaryPriceFeedVersion = 1
	SecondaryPriceFeedVersion2       SecondaryPriceFeedVersion = 2
	DefaultSecondaryPriceFeedVersion                           = SecondaryPriceFeedVersion2
)
