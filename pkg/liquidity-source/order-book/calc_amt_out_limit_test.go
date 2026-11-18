package orderbook

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// A binding limit must reject on both the native and legacy paths. A limit at the quoted output must not bind.
func TestPoolSimulator_CalcAmtOut_BindingLimit(t *testing.T) {
	t.Parallel()
	sim, err := NewPoolSimulator(pool.FactoryParams{EntityPool: entityPool})
	require.NoError(t, err)

	tokenIn, tokenOut := entityPool.Tokens[1].Address, entityPool.Tokens[0].Address
	amountIn := *uint256.NewInt(10_000_000_000)

	free, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	out := free.TokenAmountOut.Amount

	var tight uint256.Int
	tight.Sub(&out, uint256.NewInt(1))
	tightLimit := swaplimit.NewInventory("pmm-1", map[string]*uint256.Int{tokenOut: &tight})
	native, errNative := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         tightLimit,
	})
	legacy, errLegacy := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		TokenOut:      tokenOut,
		Limit:         pool.AsSwapLimitBig(tightLimit),
	})
	assert.ErrorIs(t, errNative, ErrSwapLimitExceeded)
	assert.ErrorIs(t, errLegacy, ErrSwapLimitExceeded)
	assert.Nil(t, native)
	assert.Nil(t, legacy)

	var exactOut uint256.Int
	exactOut.Set(&out)
	exact := swaplimit.NewInventory("pmm-1", map[string]*uint256.Int{tokenOut: &exactOut})
	got, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
		Limit:         exact,
	})
	require.NoError(t, err)
	assert.Equal(t, out.Dec(), got.TokenAmountOut.Amount.Dec())
	assert.Equal(t, free.Gas, got.Gas)
}
