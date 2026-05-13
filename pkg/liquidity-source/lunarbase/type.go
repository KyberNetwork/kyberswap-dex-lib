package lunarbase

import "github.com/holiman/uint256"

type Metadata struct {
	Initialized bool `json:"initialized"`
}

// Extra is the per-pool state cached in the entity. JSON tags are kept short
// to minimise on-disk size; semantics map to the on-chain `state()` view:
//
//	SqrtPriceX48 — `anchorPrice` (uint80, Q32.48 canonical price)
//	FeeAskX24    — fee on Y → X (uint24, Q24 where 2^24 == 100%)
//	FeeBidX24    — fee on X → Y (uint24, Q24)
type Extra struct {
	SqrtPriceX48      *uint256.Int `json:"p,omitempty"`
	FeeAskX24         uint32       `json:"fa,omitempty"`
	FeeBidX24         uint32       `json:"fb,omitempty"`
	LatestUpdateBlock uint64       `json:"b,omitempty"`
	Paused            bool         `json:"0,omitempty"`
	BlockDelay        uint64       `json:"d,omitempty"`
	// ConcentrationK is Q20.12 (effective K = ConcentrationK / 2^12).
	ConcentrationK uint32 `json:"k,omitempty"`
}

func (e *Extra) IsStale(blockNumber uint64) bool {
	if e.BlockDelay == 0 || e.LatestUpdateBlock == 0 || blockNumber <= e.LatestUpdateBlock {
		return false
	}
	return blockNumber-e.LatestUpdateBlock > e.BlockDelay
}

type StaticExtra struct {
	HasNative bool `json:"n,omitempty"`
}

type PoolMeta struct {
	BlockNumber     uint64 `json:"blockNumber,omitempty"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
	HasNative       bool   `json:"n,omitempty"`
}

// PoolParams is the snapshot consumed by quoteXToY / quoteYToX. Mirrors the
// shape of `math/go/lunarbasepmm.PoolParams` (single-price Q32.48, asymmetric
// directional fees in Q24).
type PoolParams struct {
	SqrtPriceX48 *uint256.Int
	FeeAskX24    uint32
	FeeBidX24    uint32
	ReserveX     *uint256.Int
	ReserveY     *uint256.Int
	// ConcentrationK is Q20.12 (effective K = ConcentrationK / 2^12).
	ConcentrationK uint32
}

type QuoteResult struct {
	AmountOut     *uint256.Int
	SqrtPriceNext *uint256.Int
	Fee           *uint256.Int
}
