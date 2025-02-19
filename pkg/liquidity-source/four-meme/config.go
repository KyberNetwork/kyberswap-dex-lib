package fourmeme

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	ChainID              valueobject.ChainID `json:"chainID"`
	NewPoolLimit         int                 `json:"newPoolLimit"`
	TokenManagerV2       string              `json:"tokenManagerV2"`
	TokenManagerHelperV3 string              `json:"tokenManagerHelperV3"`
	DefaultQuoteToken    string              `json:"defaultQuoteToken"`
}
