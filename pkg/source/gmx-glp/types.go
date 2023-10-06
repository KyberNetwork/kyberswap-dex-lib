package gmxglp

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault      *Vault      `json:"vault"`
	GlpManager *GlpManager `json:"glpManager"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

const (
	secondaryPriceFeedVersion1 SecondaryPriceFeedVersion = 1
	secondaryPriceFeedVersion2 SecondaryPriceFeedVersion = 2
)
