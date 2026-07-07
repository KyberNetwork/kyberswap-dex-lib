package swap

import (
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

type LiquidityPoint struct {
	LiqudityDelta *big.Int
	Point         int
}

type LiquidityPointU256 struct {
	LiqudityDelta *int256.Int
	Point         int
}

type LimitOrderPoint struct {
	SellingX *big.Int
	SellingY *big.Int
	Point    int
}

type LimitOrderPointU256 struct {
	SellingX *uint256.Int
	SellingY *uint256.Int
	Point    int
}

type OrderData struct {
	Liquidities   []LiquidityPointU256
	LiquidityIdx  int
	LimitOrders   []LimitOrderPointU256
	LimitOrderIdx int
}

func (orderData *OrderData) IsLiquidity(point int) bool {
	if orderData.LiquidityIdx < 0 || orderData.LiquidityIdx >= len(orderData.Liquidities) {
		return false
	}
	return orderData.Liquidities[orderData.LiquidityIdx].Point == point
}

func (orderData *OrderData) IsLimitOrder(point int) bool {
	if orderData.LimitOrderIdx < 0 || orderData.LimitOrderIdx >= len(orderData.LimitOrders) {
		return false
	}
	return orderData.LimitOrders[orderData.LimitOrderIdx].Point == point
}

func (orderData *OrderData) UnsafeGetDeltaLiquidity() *uint256.Int {
	return (*uint256.Int)(orderData.Liquidities[orderData.LiquidityIdx].LiqudityDelta)
}

func (orderData *OrderData) UnsafeGetLimitSellingX() *uint256.Int {
	return orderData.LimitOrders[orderData.LimitOrderIdx].SellingX
}

func (orderData *OrderData) UnsafeGetLimitSellingY() *uint256.Int {
	return orderData.LimitOrders[orderData.LimitOrderIdx].SellingY
}
