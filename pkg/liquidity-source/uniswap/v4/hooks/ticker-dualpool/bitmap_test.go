package tickerdualpool

import (
	"testing"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestBitmapStepBoundaries(t *testing.T) {
	for _, tc := range []struct {
		tick, spacing int
		left          bool
		next          int
	}{
		{-256400, 200, false, -256200},
		{-256401, 200, false, -256200},
		{-1, 10, true, -2560},
		{-1, 10, false, 2550},
		{0, 10, true, 0},
		{2550, 10, false, 5110},
	} {
		n, initialized := nextTickWithinWord(nil, tc.tick, tc.spacing, tc.left)
		require.Equal(t, tc.next, n)
		require.False(t, initialized)
	}
	n, initialized := nextTickWithinWord([]int{-100, 100}, 0, 10, false)
	require.Equal(t, 100, n)
	require.True(t, initialized)
}

func TestBitmapQuoteMatchesExplicitTwoSteps(t *testing.T) {
	var price uint256.Int
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(-256400, &price))
	positions, err := dualpool.AllocatePositions([]dualpool.Bucket{{TickLower: -887200, TickUpper: 887200, WeightBps: 10000}}, &price, uint256.MustFromDecimal("1000000000000000000000"), uint256.NewInt(10000000000))
	require.NoError(t, err)
	res, err := swapExactInWithSpacing(positions, &price, -256400, false, uint256.NewInt(100000000), 2000, 200, 0)
	require.NoError(t, err)
	require.Equal(t, "13428054105542917337", res.amountOut.Dec())
	require.Equal(t, "217457268350574018262156", res.sqrtPriceX96.Dec())
}
