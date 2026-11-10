package velodromev1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
)

type PoolStaticExtra struct {
	FeePrecision uint64 `json:"feePrecision"`
	Stable       bool   `json:"stable,omitempty"`
	Gas          int64  `json:"g,omitempty"`
}

type PoolExtra struct {
	IsPaused bool   `json:"isPaused,omitempty"`
	Fee      uint64 `json:"fee"`

	// Balance0 and Balance1 are balanceOf(pair) of the pool tokens, read at the block of the reserves. They differ
	// from the reserves when the pair was donated to (above) or lost tokens (below). Both nil (an extra saved before
	// these fields existed, or a balanceOf that failed) means unknown: the swap is then not checked against them.
	Balance0 *big.Int `json:"balance0,omitempty"`
	Balance1 *big.Int `json:"balance1,omitempty"`
}

type PoolMeta struct {
	Fee             uint64 `json:"f"`
	FeePrecision    uint64 `json:"p"`
	Stable          bool   `json:"s,omitempty"`
	ApprovalAddress string `json:"approvalAddress"`
	BlockNumber     uint64 `json:"blockNumber"`
}

type PairMetadata struct {
	Dec0 *big.Int
	Dec1 *big.Int
	R0   *big.Int
	R1   *big.Int
	St   bool
	T0   common.Address
	T1   common.Address
	Fee  uint64
}

type ReserveData = uniswapv2.ReserveData

type PairFactoryData struct {
	AllPairsLength int64
	IsPaused       bool
}
