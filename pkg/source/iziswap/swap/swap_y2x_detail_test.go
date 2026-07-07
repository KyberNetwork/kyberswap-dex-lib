package swap

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func getLiquiditiesDetailY2X() []LiquidityPointU256 {
	liquidities := []LiquidityPointU256{
		{LiqudityDelta: int256.NewInt(200000), Point: -7000},
		{LiqudityDelta: int256.NewInt(300000), Point: -5000},
		{LiqudityDelta: int256.NewInt(-300000), Point: -2000},
		{LiqudityDelta: int256.NewInt(-200000), Point: -240},

		{LiqudityDelta: int256.NewInt(600000), Point: -200},
		{LiqudityDelta: int256.NewInt(-600000), Point: 40},
		{LiqudityDelta: int256.NewInt(500000), Point: 80},
		{LiqudityDelta: int256.NewInt(-500000), Point: 2000},
	}
	return liquidities
}

func getLimitOrdersDetailY2X() []LimitOrderPointU256 {
	limitOrders := []LimitOrderPointU256{
		{SellingY: uint256.NewInt(80000000000), Point: -6400},
		// some test case may change order at -6200
		{SellingX: uint256.NewInt(100000000000), Point: -6200},
		{SellingX: uint256.NewInt(150000000000), Point: -1000},
		{SellingX: uint256.NewInt(120000000000), Point: 1200},
	}
	return limitOrders
}

func getPoolInfoDetailY2X() PoolInfoU256 {
	return PoolInfoU256{
		CurrentPoint: -6216,
		PointDelta:   40,
		LeftMostPt:   -800000,
		RightMostPt:  800000,
		Fee:          2000,
		// other test case may change following
		// liquidity and liquidityX value
		Liquidity:   uint256.NewInt(200000),
		LiquidityX:  uint256.NewInt(31891),
		Liquidities: getLiquiditiesDetailY2X(),
		LimitOrders: getLimitOrdersDetailY2X(),
	}
}

func TestSwapDetailY2X1(t *testing.T) {
	t.Parallel()
	// y2x start partial x-liquidity,
	// end partial x-liquidity
	poolInfo := getPoolInfoDetailY2X()
	poolInfo.CurrentPoint = -6215
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(31891)
	var amount uint256.Int
	_ = amount.SetFromDecimal("328168800000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328168800000")
	acquireX := uint256.MustFromDecimal("373337423211")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1559 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1559)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("59052")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X2(t *testing.T) {
	t.Parallel()
	// y2x start partial liquidity,
	// end full liquidity
	poolInfo := getPoolInfoDetailY2X()
	poolInfo.CurrentPoint = -6215
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(31891)
	var amount uint256.Int
	_ = amount.SetFromDecimal("1000000000000000000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328168863966")
	acquireX := uint256.MustFromDecimal("373337477835")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1560 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1560)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("500000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X3(t *testing.T) {
	t.Parallel()
	// y2x start with full-y-liquidity
	// end full liquidity
	poolInfo := getPoolInfoDetailY2X()
	poolInfo.CurrentPoint = -6216
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(0)
	var amount uint256.Int
	_ = amount.SetFromDecimal("1000000000000000000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328168987421")
	acquireX := uint256.MustFromDecimal("373337707209")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1560 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1560)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("500000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X4(t *testing.T) {
	t.Parallel()
	// y2x start with full-x-liquidity
	// end full liquidity
	poolInfo := getPoolInfoDetailY2X()
	poolInfo.CurrentPoint = -6216
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(200000)
	var amount uint256.Int
	_ = amount.SetFromDecimal("1000000000000000000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328169134289")
	acquireX := uint256.MustFromDecimal("373337980108")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1560 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1560)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("500000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}
func TestSwapDetailY2X5(t *testing.T) {
	t.Parallel()
	// y2x start with limitorder-x, full-x-liquidity
	// end partial liquidity
	poolInfo := getPoolInfoDetailY2X()
	poolInfo.CurrentPoint = -6200
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(200000)
	var amount uint256.Int
	_ = amount.SetFromDecimal("328166700000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328166700000")
	acquireX := uint256.MustFromDecimal("373333544041")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1559 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1559)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("77101")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X6(t *testing.T) {
	t.Parallel()
	// y2x start with limitorder-y, full-x-liquidity
	// end partial liquidity
	poolInfo := getPoolInfoDetailY2X()

	poolInfo.LimitOrders[1].SellingX = uint256.NewInt(0)
	poolInfo.LimitOrders[1].SellingY = uint256.NewInt(100000000000)

	poolInfo.CurrentPoint = -6200
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(200000)
	var amount uint256.Int
	_ = amount.SetFromDecimal("274262809000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("274262809000")
	acquireX := uint256.MustFromDecimal("273333568073")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1559 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1559)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("51121")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X7(t *testing.T) {
	t.Parallel()
	// y2x start with limitorder-y, full-x-liquidity
	// end partial liquidity
	poolInfo := getPoolInfoDetailY2X()

	poolInfo.CurrentPoint = -6200
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(198640)
	var amount uint256.Int
	_ = amount.SetFromDecimal("328166700000")
	swapResult, _ := SwapY2X(&amount, 1560, poolInfo)
	costY := uint256.MustFromDecimal("328166700000")
	acquireX := uint256.MustFromDecimal("373333543040")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1559 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1559)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("76178")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X8(t *testing.T) {
	t.Parallel()
	// y2x start with limitorder-y, full-x-liquidity
	// end partial liquidity
	poolInfo := getPoolInfoDetailY2X()

	poolInfo.CurrentPoint = -6200
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(198640)
	var amount uint256.Int
	_ = amount.SetFromDecimal("250000000000")
	swapResult, _ := SwapY2X(&amount, 1201, poolInfo)
	costY := uint256.MustFromDecimal("249999999999")
	acquireX := uint256.MustFromDecimal("304147179726")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1200 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1200)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("500000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailY2X9(t *testing.T) {
	t.Parallel()
	// y2x start with limitorder-y, full-x-liquidity
	// end partial liquidity
	poolInfo := getPoolInfoDetailY2X()

	poolInfo.CurrentPoint = -6200
	poolInfo.Liquidity = uint256.NewInt(200000)
	poolInfo.LiquidityX = uint256.NewInt(198640)
	var amount uint256.Int
	_ = amount.SetFromDecimal("327974071999")
	swapResult, _ := SwapY2X(&amount, 1201, poolInfo)
	costY := uint256.MustFromDecimal("327974071999")
	acquireX := uint256.MustFromDecimal("373166078203")

	if swapResult.AmountY.Cmp(costY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), costY.String())
	}
	if swapResult.AmountX.Cmp(acquireX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), acquireX.String())
	}
	if swapResult.CurrentPoint != 1200 {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, 1200)
	}
	resultLiquidity := uint256.MustFromDecimal("500000")
	resultLiquidityX := uint256.MustFromDecimal("358")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}
