package poe

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

func TestApplyFeeCeil(t *testing.T) {
	amount := uint256.NewInt(1_000_000)

	require.Equal(t, uint256.NewInt(3000), applyFeeCeil(amount, uint256.NewInt(3000)))
	require.Equal(t, uint256.NewInt(3001), applyFeeCeil(uint256.NewInt(1_000_001), uint256.NewInt(3000)))
	require.Equal(t, u256.New0(), applyFeeCeil(amount, u256.New0()))
}

func TestComputeVirtualReserves(t *testing.T) {
	price := uint256.MustFromDecimal("1000000000000000000000000")
	reserveX := uint256.NewInt(1_000_000_000)
	reserveY := uint256.NewInt(1_000_000_000)

	vr := computeVirtualReserves(reserveX, reserveY, price, uint256.NewInt(10100))

	require.True(t, vr.xv.Gt(reserveX))
	require.True(t, vr.yv.Gt(reserveY))

	diff := new(uint256.Int)
	if vr.xv.Gt(vr.yv) {
		diff.Sub(vr.xv, vr.yv)
	} else {
		diff.Sub(vr.yv, vr.xv)
	}
	require.True(t, diff.Lt(new(uint256.Int).Div(vr.xv, uint256.NewInt(100))))
}

func TestComputeVirtualReserves_ZeroReserves(t *testing.T) {
	price, _ := uint256.FromDecimal("1000000000000000000000000")

	vr := computeVirtualReserves(u256.New0(), u256.New0(), price, uint256.NewInt(10100))

	require.True(t, vr.xv.IsZero())
	require.True(t, vr.yv.IsZero())
}

func TestCalcAmountOutCPMM(t *testing.T) {
	xv := uint256.NewInt(100_000_000_000)
	yv := uint256.NewInt(100_000_000_000)
	netIn := uint256.NewInt(1_000_000)

	amountOut := calcAmountOutCPMM(xv, yv, netIn)

	require.True(t, amountOut.Gt(uint256.NewInt(999_000)))
	require.True(t, amountOut.Lt(netIn))
}

func TestCalcAmountInCPMM(t *testing.T) {
	xv := uint256.NewInt(100_000_000_000)
	yv := uint256.NewInt(100_000_000_000)
	desiredOut := uint256.NewInt(999_990)

	netIn := calcAmountInCPMM(xv, yv, desiredOut)
	require.NotNil(t, netIn)

	require.True(t, calcAmountOutCPMM(xv, yv, netIn).Cmp(desiredOut) >= 0)
}

func TestCalcAmountInCPMM_ExceedsReserve(t *testing.T) {
	require.Nil(t, calcAmountInCPMM(uint256.NewInt(100_000_000), uint256.NewInt(100_000_000), uint256.NewInt(100_000_001)))
}

func TestSwapXtoY(t *testing.T) {
	price, _ := uint256.FromDecimal("2000000000000000")
	reserveX := uint256.MustFromDecimal("10000000000000000000")
	reserveY := uint256.MustFromDecimal("20000000000")

	vr := computeVirtualReserves(reserveX, reserveY, price, uint256.NewInt(10500))
	require.True(t, vr.xv.Gt(reserveX))
	require.True(t, vr.yv.Gt(reserveY))

	amountIn := uint256.MustFromDecimal("100000000000000000")
	netIn := new(uint256.Int).Sub(amountIn, applyFeeCeil(amountIn, uint256.NewInt(3000)))
	amountOut := calcAmountOutCPMM(vr.xv, vr.yv, netIn)

	t.Logf("Swap 0.1 ETH -> USDC: amountOut = %s (expected ~200e6)", amountOut.String())
	require.True(t, amountOut.Gt(uint256.NewInt(190_000_000)))
	require.True(t, amountOut.Lt(uint256.NewInt(210_000_000)))
}
