package velodromev1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolStaticExtra struct {
	FeePrecision uint64 `json:"feePrecision"`
	Stable       bool   `json:"stable,omitempty"`
}

type PoolExtra struct {
	IsPaused bool   `json:"isPaused,omitempty"`
	Fee      uint64 `json:"fee"`
}

type PoolMeta struct {
	Fee          uint64 `json:"f"`
	FeePrecision uint64 `json:"p"`
	Stable       bool   `json:"s,omitempty"`
	pool.ApprovalInfo
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
	AllPairsLength *big.Int
	IsPaused       bool
}
