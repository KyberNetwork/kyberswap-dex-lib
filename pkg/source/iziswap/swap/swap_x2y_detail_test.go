package swap

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func getLiquiditiesDetailX2Y() []LiquidityPointU256 {
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

func getLimitOrdersDetailX2Y() []LimitOrderPointU256 {
	limitOrders := []LimitOrderPointU256{
		{SellingY: uint256.NewInt(100000000000), Point: -6200},
		{SellingY: uint256.NewInt(150000000000), Point: -1000},
		{SellingY: uint256.NewInt(120000000000), Point: 1200},
		{SellingX: uint256.NewInt(120000000000), Point: 1800},
	}
	return limitOrders
}

func getPoolInfoDetailX2Y() PoolInfoU256 {
	return PoolInfoU256{
		CurrentPoint: 1729,
		PointDelta:   40,
		LeftMostPt:   -800000,
		RightMostPt:  800000,
		Fee:          2000,
		// other test case may change following
		// liquidity and liquidityX value
		Liquidity:   uint256.NewInt(500000),
		LiquidityX:  uint256.NewInt(500000),
		Liquidities: getLiquiditiesDetailX2Y(),
		LimitOrders: getLimitOrdersDetailX2Y(),
	}
}

func TestSwapDetailX2Y1(t *testing.T) {
	t.Parallel()
	// x2y start partial y-liquidity,
	// end partial y-liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1729
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(134333)
	var amount uint256.Int
	_ = amount.SetFromDecimal("462592000000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462592000000")
	acquireY := uint256.MustFromDecimal("372866052521")
	finalPoint := -6786
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("151638")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y2(t *testing.T) {
	t.Parallel()
	// gocalc x2y start partial liquidity
	// end full liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1729
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(134333)
	var amount uint256.Int
	_ = amount.SetFromDecimal("100000000000000000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462592912170")
	acquireY := uint256.MustFromDecimal("372866514294")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("200000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y3(t *testing.T) {
	t.Parallel()
	// x2y start with full-x-liquidity
	// end full liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1731
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(500000)
	var amount uint256.Int
	_ = amount.SetFromDecimal("100000000000000000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462593495113")
	acquireY := uint256.MustFromDecimal("372867205930")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("200000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y4(t *testing.T) {
	t.Parallel()
	// x2y start with full-y-liquidity
	// end full liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1729
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(0)
	var amount uint256.Int
	_ = amount.SetFromDecimal("100000000000000000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462593035624")
	acquireY := uint256.MustFromDecimal("372866660757")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("200000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y5(t *testing.T) {
	t.Parallel()
	// x2y start with limitorder-y
	// full-x-liquidity, end partial liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1200
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(500000)
	var amount uint256.Int
	_ = amount.SetFromDecimal("462346200000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462346200000")
	acquireY := uint256.MustFromDecimal("372581497872")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("167919")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y6(t *testing.T) {
	t.Parallel()
	// x2y start with limitorder-x
	// full-y-liquidity, end partial liquidity
	poolInfo := getPoolInfoDetailX2Y()

	poolInfo.LimitOrders[2].SellingX = uint256.NewInt(120000000000)
	poolInfo.LimitOrders[2].SellingY = uint256.NewInt(0)

	poolInfo.CurrentPoint = 1200
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(0)

	var amount uint256.Int
	_ = amount.SetFromDecimal("355702300000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("355702300000")
	acquireY := uint256.MustFromDecimal("252582032779")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("173521")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y7(t *testing.T) {
	t.Parallel()
	// x2y start with limitorder-y, partial-liquidity,
	// end partial liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1200
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(383966)
	var amount uint256.Int
	_ = amount.SetFromDecimal("462346300000")
	lowPt := -6789
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("462346300000")
	acquireY := uint256.MustFromDecimal("372581616273")
	finalPoint := -6789
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("161169")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y8(t *testing.T) {
	t.Parallel()
	// x2y start with limitorder-y, partial-liquidity
	// end full-x-liquidityand partial or full limit order
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1200
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(383966)
	var amount uint256.Int
	_ = amount.SetFromDecimal("325923465573")
	lowPt := -6201
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("325923465573")
	acquireY := uint256.MustFromDecimal("299340764005")
	finalPoint := -6200
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("200000")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}

func TestSwapDetailX2Y9(t *testing.T) {
	t.Parallel()
	// gocalc x2y start with limitorder-y, partial-liquidity
	// end partial-x-liquidity
	poolInfo := getPoolInfoDetailX2Y()
	poolInfo.CurrentPoint = 1200
	poolInfo.Liquidity = uint256.NewInt(500000)
	poolInfo.LiquidityX = uint256.NewInt(383966)
	var amount uint256.Int
	_ = amount.SetFromDecimal("275923400000")
	lowPt := -6200
	swapResult, _ := SwapX2Y(&amount, lowPt, poolInfo)
	costX := uint256.MustFromDecimal("275923400000")
	acquireY := uint256.MustFromDecimal("272496469260")
	finalPoint := -6200
	if swapResult.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapResult.AmountX.String(), costX.String())
	}
	if swapResult.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapResult.AmountY.String(), acquireY.String())
	}
	if swapResult.CurrentPoint != finalPoint {
		t.Fatalf("result currentPoint not equal (%d, %d)", swapResult.CurrentPoint, finalPoint)
	}
	resultLiquidity := uint256.MustFromDecimal("200000")
	resultLiquidityX := uint256.MustFromDecimal("152001")
	if swapResult.Liquidity.Cmp(resultLiquidity) != 0 {
		t.Fatalf("Liquidity not equal (%s, %s)", swapResult.Liquidity.String(), resultLiquidity.String())
	}
	if swapResult.LiquidityX.Cmp(resultLiquidityX) != 0 {
		t.Fatalf("LiquidityX not equal (%s, %s)", swapResult.LiquidityX.String(), resultLiquidityX.String())
	}
}
