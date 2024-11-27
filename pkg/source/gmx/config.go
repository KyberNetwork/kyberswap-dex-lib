package gmx

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID                 valueobject.ChainID `json:"chainID"`
	DexID                   string              `json:"dexID"`
	VaultAddress            string              `json:"vaultAddress"`
	UseSecondaryPriceFeedV1 bool                `json:"useSecondaryPriceFeedV1"`
}
