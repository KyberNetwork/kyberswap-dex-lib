package bancorv21

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"

type ExtraInner struct {
	AnchorAddress string `json:"AnchorAddress"`
	ConversionFee uint64 `json:"ConversionFee"`
}

type Extra struct {
	InnerPoolByAnchor         map[string]*entity.Pool `json:"innerPoolByAnchor"`
	AnchorsByConvertibleToken map[string][]string     `json:"anchorsByConvertibleToken"`
	InnerPools                []entity.Pool           `json:"innerPools"`
	TokensByLpAddress         map[string][]string     `json:"tokensByLpAddress"`
}

type Gas struct {
	Swap int64
}

type PoolMetaInner struct {
	Fee         uint64 `json:"fee"`
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	ConversionPath []string `json:"conversionPath"`
}
