package gmx

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID                 valueobject.ChainID `json:"-"`
	DexID                   string              `json:"-"`
	VaultAddress            string              `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool                `json:"useSecondaryPriceFeedV1"`
}
