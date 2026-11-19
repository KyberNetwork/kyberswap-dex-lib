package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestCloneState(t testing.TB, sim pool.IPoolSimulator, params pool.CalcAmountOutParams, swapLimit pool.SwapLimit) {
	t.Helper()

	clone := sim.CloneState()

	result1, err := pool.DefaultCalcAmtOut(context.Background(), sim, params.TokenAmountIn, params.TokenOut, swapLimit)
	require.NoError(t, err)

	clone.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  params.TokenAmountIn,
		TokenAmountOut: *result1.TokenAmountOut,
		Fee:            *result1.Fee,
		SwapInfo:       result1.SwapInfo,
		SwapLimitBig:   pool.AsSwapLimitBig(swapLimit),
	})

	result2, err := pool.DefaultCalcAmtOut(context.Background(), sim, params.TokenAmountIn, params.TokenOut, swapLimit)
	require.NoError(t, err)

	assert.Equal(t, result1.TokenAmountOut.Amount, result2.TokenAmountOut.Amount, "CloneState: UpdateBalance on clone mutated the original")
}
