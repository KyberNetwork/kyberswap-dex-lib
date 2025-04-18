package valueobject

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"

type IndexedRFQParams struct {
	pool.RFQParams
	PathIdx    int
	SwapIdx    int
	ExecutedId int32
}
