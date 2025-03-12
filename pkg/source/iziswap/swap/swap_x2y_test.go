package swap

import (
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

func getLiquiditiesX2Y() []LiquidityPointU256 {
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

func getLimitOrdersY() []LimitOrderPointU256 {
	limitOrders := []LimitOrderPointU256{
		{SellingY: uint256.NewInt(100000000000), Point: -3000},
		{SellingY: uint256.NewInt(150000000000), Point: -1000},
		{SellingY: uint256.NewInt(120000000000), Point: 1200},
	}
	return limitOrders
}

func getPoolInfoX2Y() PoolInfoU256 {
	return PoolInfoU256{
		CurrentPoint: 1887,
		PointDelta:   40,
		LeftMostPt:   -800000,
		RightMostPt:  800000,
		Fee:          2000,
		Liquidity:    uint256.NewInt(700000),
		LiquidityX:   uint256.NewInt(246660),
		Liquidities:  getLiquiditiesX2Y(),
		LimitOrders:  getLimitOrdersY(),
	}
}

func TestSwapX2Y1(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("100000000000000000000000")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("410079196782")
	acquireY := uint256.MustFromDecimal("371715048235")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}
func TestSwapX2Y2(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("410079196782")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("410079196782")
	acquireY := uint256.MustFromDecimal("371715048235")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}
func TestSwapX2Y3(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("399624951498")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("399624951497")
	acquireY := uint256.MustFromDecimal("364135750158")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}

func TestSwapX2Y4(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("368662456348")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("368662456348")
	acquireY := uint256.MustFromDecimal("341243701400")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}

func TestSwapX2Y5(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("245774970898")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("245774970898")
	acquireY := uint256.MustFromDecimal("245800546380")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}

func TestSwapX2Y6(t *testing.T) {
	poolInfo := getPoolInfoX2Y()
	var amount uint256.Int
	_ = amount.SetFromDecimal("122887485449")
	swapAmount, _ := SwapX2Y(&amount, -6123, poolInfo)
	costX := uint256.MustFromDecimal("122887485449")
	acquireY := uint256.MustFromDecimal("134829182908")
	if swapAmount.AmountX.Cmp(costX) != 0 {
		t.Fatalf("amount x not equal (%s, %s)", swapAmount.AmountX.String(), costX.String())
	}
	if swapAmount.AmountY.Cmp(acquireY) != 0 {
		t.Fatalf("amount y not equal (%s, %s)", swapAmount.AmountY.String(), acquireY.String())
	}
}
