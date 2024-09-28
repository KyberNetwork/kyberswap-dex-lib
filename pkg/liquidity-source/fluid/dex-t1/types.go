package dexT1

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type PoolMeta struct {
	BlockNumber uint64 `json:"blockNumber"`
}

type PoolExtra struct {
	CollateralReserves CollateralReserves
	DebtReserves       DebtReserves
}

type Pool struct {
	PoolAddress   common.Address `json:"poolAddress"`
	Token0Address common.Address `json:"token0Address"`
	Token1Address common.Address `json:"token1Address"`
	Fee           *big.Int       `json:"fee"`
}

type CollateralReserves struct {
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

type DebtReserves struct {
	Token0Debt              *big.Int `json:"token0Debt"`
	Token1Debt              *big.Int `json:"token1Debt"`
	Token0RealReserves      *big.Int `json:"token0RealReserves"`
	Token1RealReserves      *big.Int `json:"token1RealReserves"`
	Token0ImaginaryReserves *big.Int `json:"token0ImaginaryReserves"`
	Token1ImaginaryReserves *big.Int `json:"token1ImaginaryReserves"`
}

type PoolWithReserves struct {
	PoolAddress        common.Address     `json:"poolAddress"`
	Token0Address      common.Address     `json:"token0Address"`
	Token1Address      common.Address     `json:"token1Address"`
	Fee                *big.Int           `json:"fee"`
	CollateralReserves CollateralReserves `json:"collateralReserves"`
	DebtReserves       DebtReserves       `json:"debtReserves"`
}

type Gas struct {
	Swap int64
}

type StaticExtra struct {
	DexReservesResolver string `json:"dexReservesResolver"`
}
