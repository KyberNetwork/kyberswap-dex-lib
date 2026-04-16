package lunarbase

import "github.com/holiman/uint256"

type Metadata struct {
	Initialized bool `json:"initialized"`
}

type Extra struct {
	PriceX96          *uint256.Int `json:"p,omitempty"`
	FeeQ48            uint64       `json:"f,omitempty"`
	LatestUpdateBlock uint64       `json:"b,omitempty"`
	Paused            bool         `json:"0,omitempty"`
	BlockDelay        uint64       `json:"d,omitempty"`
	ConcentrationK    uint32       `json:"k,omitempty"`
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

type PoolParams struct {
	SqrtPriceX96   *uint256.Int
	FeeQ48         uint64
	ReserveX       *uint256.Int
	ReserveY       *uint256.Int
	ConcentrationK uint32
}

type QuoteResult struct {
	AmountOut     *uint256.Int
	SqrtPriceNext *uint256.Int
	Fee           *uint256.Int
}
