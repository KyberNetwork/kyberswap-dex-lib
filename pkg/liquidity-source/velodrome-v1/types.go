package velodromev1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
)

type PoolStaticExtra struct {
	FeePrecision uint64       `json:"feePrecision"`
	Decimal0     *uint256.Int `json:"decimal0"`
	Decimal1     *uint256.Int `json:"decimal1"`
	Stable       bool         `json:"stable"`
}

type PoolExtra struct {
	IsPaused bool   `json:"isPaused"`
	Fee      uint64 `json:"fee"`
}

type PoolMeta struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
	BlockNumber  uint64 `json:"blockNumber"`
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
