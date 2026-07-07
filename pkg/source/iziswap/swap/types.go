package swap

import (
	"math/big"

	"github.com/holiman/uint256"
)

type SwapResult struct {
	AmountX       *uint256.Int
	AmountY       *uint256.Int
	CurrentPoint  int
	Liquidity     *uint256.Int
	LiquidityX    *uint256.Int
	CrossedPoints int64
}

type PoolInfo struct {
	CurrentPoint int
	PointDelta   int
	LeftMostPt   int
	RightMostPt  int
	Fee          int
	Liquidity    *big.Int
	LiquidityX   *big.Int
	Liquidities  []LiquidityPoint
	LimitOrders  []LimitOrderPoint
}

type PoolInfoU256 struct {
	CurrentPoint int
	PointDelta   int
	LeftMostPt   int
	RightMostPt  int
	Fee          int
	Liquidity    *uint256.Int
	LiquidityX   *uint256.Int
	Liquidities  []LiquidityPointU256
	LimitOrders  []LimitOrderPointU256
}
