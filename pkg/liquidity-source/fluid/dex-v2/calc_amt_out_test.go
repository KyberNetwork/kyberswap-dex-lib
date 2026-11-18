package dexv2

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// Native CalcAmtOut must match legacy CalcAmountOut on amount out, fee, gas and error, for the same limit.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	sim, p := snapshotSimulator(t)
	limit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(sim.CalculateLimit()))
	for dir := range 2 {
		for _, a := range []int64{10_000, 123_456, 100_000_000} {
			tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
			legacy, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(a)}, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(limit)})
			var amt uint256.Int
			amt.SetUint64(uint64(a))
			native, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amt}, TokenOut: tokenOut, Limit: limit})
			require.Equal(t, lerr, nerr, "dir=%d in=%d", dir, a)
			if lerr != nil {
				continue
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountOut dir=%d in=%d", dir, a)
			require.Equal(t, legacy.Fee.Token, native.Fee.Token, "fee token dir=%d in=%d", dir, a)
			require.Equal(t, legacy.Gas, native.Gas, "gas dir=%d in=%d", dir, a)
		}
	}
}

// A shared budget that cannot cover the quote must fail on both paths with the same error.
func TestCalcAmtOutLimitBindsLikeLegacy(t *testing.T) {
	sim, p := snapshotSimulator(t)
	balances := sim.CalculateLimit()
	for k := range balances {
		balances[k] = big.NewInt(0)
	}
	limit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(balances))
	var amt uint256.Int
	amt.SetUint64(100_000_000)
	tokenIn, tokenOut := p.Tokens[0].Address, p.Tokens[1].Address
	_, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(100_000_000)}, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(limit)})
	_, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amt}, TokenOut: tokenOut, Limit: limit})
	require.Error(t, lerr)
	require.Equal(t, lerr, nerr)
}
