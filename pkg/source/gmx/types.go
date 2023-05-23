package gmx

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault *Vault `json:"vault"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

const ARBITRUM ChainID = 42161
const AVALANCHE ChainID = 43114

const (
	secondaryPriceFeedVersion1       SecondaryPriceFeedVersion = 1
	secondaryPriceFeedVersion2       SecondaryPriceFeedVersion = 2
	defaultSecondaryPriceFeedVersion                           = secondaryPriceFeedVersion2
)
