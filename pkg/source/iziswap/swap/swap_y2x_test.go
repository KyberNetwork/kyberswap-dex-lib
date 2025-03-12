package swap

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func getLiquiditiesY2X() []LiquidityPointU256 {
	liquidities := []LiquidityPointU256{
		{LiqudityDelta: int256.NewInt(200000), Point: -9000},
		{LiqudityDelta: int256.NewInt(300000), Point: -8000},
		{LiqudityDelta: int256.NewInt(-300000), Point: -5000},
		{LiqudityDelta: int256.NewInt(-200000), Point: -4000},

		{LiqudityDelta: int256.NewInt(100000), Point: -2000},
		{LiqudityDelta: int256.NewInt(500000), Point: -1200},
		{LiqudityDelta: int256.NewInt(-500000), Point: -800},
		{LiqudityDelta: int256.NewInt(-100000), Point: 800},

		{LiqudityDelta: int256.NewInt(700000), Point: 1000},
		{LiqudityDelta: int256.NewInt(-700000), Point: 2000},
	}
	return liquidities
}

func getLimitOrdersX() []LimitOrderPointU256 {
	limitOrders := []LimitOrderPointU256{
		{SellingX: uint256.NewInt(100000000000), Point: -3000},
		{SellingX: uint256.NewInt(150000000000), Point: -1000},
		{SellingX: uint256.NewInt(120000000000), Point: 1200},
	}
	return limitOrders
}

func getPoolInfoY2X() PoolInfoU256 {
	return PoolInfoU256{
		CurrentPoint: -6182,
		PointDelta:   40,
		LeftMostPt:   -800000,
		RightMostPt:  800000,
		Fee:          2000,
		Liquidity:    uint256.NewInt(500000),
		LiquidityX:   uint256.NewInt(202614),
		Liquidities:  getLiquiditiesY2X(),
		LimitOrders:  getLimitOrdersX(),
	}
}

func TestSwapY2X1(t *testing.T) {
	poolInfo := getPoolInfoY2X()
	var amount uint256.Int
	_ = amount.SetFromDecimal("100000000000000000000000")
	swapAmount, _ := SwapY2X(&amount, 1100, poolInfo)
	costY := uint256.MustFromDecimal("211374358247")
	acquireX := uint256.MustFromDecimal("251597283132")
	if swapAmount.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), "211374358247")
	}
	if swapAmount.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), "251597283132")
	}
}

func TestSwapY2X2(t *testing.T) {
	poolInfo := getPoolInfoY2X()
	var amount uint256.Int
	_ = amount.SetFromDecimal("211374358247")
	swapAmount, _ := SwapY2X(&amount, 1100, poolInfo)
	costY := uint256.MustFromDecimal("211374358247")
	acquireX := uint256.MustFromDecimal("251597283132")
	if swapAmount.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), "211374358247")
	}
	if swapAmount.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), "251597283132")
	}
}

func TestSwapY2X3(t *testing.T) {
	poolInfo := getPoolInfoY2X()
	var amount uint256.Int
	_ = amount.SetFromDecimal("190236922422")
	swapAmount, _ := SwapY2X(&amount, 1100, poolInfo)
	costY := uint256.MustFromDecimal("190236922422")
	acquireX := uint256.MustFromDecimal("228316826682")
	if swapAmount.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), "211374358247")
	}
	if swapAmount.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), "251597283132")
	}
}

func TestSwapY2X4(t *testing.T) {
	poolInfo := getPoolInfoY2X()
	var amount uint256.Int
	_ = amount.SetFromDecimal("126824614948")
	swapAmount, _ := SwapY2X(&amount, 1100, poolInfo)
	costY := uint256.MustFromDecimal("126824614948")
	acquireX := uint256.MustFromDecimal("158375901172")
	if swapAmount.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), "211374358247")
	}
	if swapAmount.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), "251597283132")
	}
}

func TestSwapY2X5(t *testing.T) {
	poolInfo := getPoolInfoY2X()
	var amount uint256.Int
	_ = amount.SetFromDecimal("63412307474")
	swapAmount, _ := SwapY2X(&amount, 1100, poolInfo)
	costY := uint256.MustFromDecimal("63412307474")
	acquireX := uint256.MustFromDecimal("85638433523")
	if swapAmount.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), "211374358247")
	}
	if swapAmount.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), "251597283132")
	}
}
