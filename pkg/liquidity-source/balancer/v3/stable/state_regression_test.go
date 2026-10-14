package stable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Vault._swap updates raw pool-token balances before BatchRouter unwraps output.
// The oracle uses those raw amounts, not the final user's underlying amounts.
func TestUpdateBalanceQuotedRawAmounts(t *testing.T) {
	t.Run("exact_in_unwrapped_output", func(t *testing.T) {
		s := poolSim.CloneState().(*base.PoolSimulator)
		in := pool.TokenAmount{Token: s.GetTokens()[2], Amount: big.NewInt(2e18)}
		r, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: in, TokenOut: s.GetTokens()[1]})
		require.NoError(t, err)
		info := r.SwapInfo.(shared.SwapInfo)
		require.NotNil(t, info.AmountOutBeforeConvert)
		require.NotEqual(t, info.AmountOutBeforeConvert.String(), r.TokenAmountOut.Amount.String())
		wantIn := new(big.Int).Sub(new(big.Int).Add(s.GetReserves()[0], in.Amount), info.AggregateFee)
		wantOut := new(big.Int).Sub(s.GetReserves()[1], info.AmountOutBeforeConvert)
		s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: in, TokenAmountOut: *r.TokenAmountOut, Fee: *r.Fee, SwapInfo: r.SwapInfo})
		require.Equal(t, wantIn.String(), s.GetReserves()[0].String())
		require.Equal(t, wantOut.String(), s.GetReserves()[1].String())
	})
	t.Run("exact_out_wrapped_input", func(t *testing.T) {
		s := poolSim.CloneState().(*base.PoolSimulator)
		out := pool.TokenAmount{Token: s.GetTokens()[3], Amount: big.NewInt(1e18)}
		r, err := s.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: out, TokenIn: s.GetTokens()[0]})
		require.NoError(t, err)
		info := r.SwapInfo.(shared.SwapInfo)
		require.NotNil(t, info.AmountInBeforeConvert)
		require.NotEqual(t, info.AmountInBeforeConvert.String(), r.TokenAmountIn.Amount.String())
		wantIn := new(big.Int).Sub(new(big.Int).Add(s.GetReserves()[0], info.AmountInBeforeConvert), info.AggregateFee)
		wantOut := new(big.Int).Sub(s.GetReserves()[1], out.Amount)
		s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: *r.TokenAmountIn, TokenAmountOut: out, Fee: *r.Fee, SwapInfo: r.SwapInfo})
		require.Equal(t, wantIn.String(), s.GetReserves()[0].String())
		require.Equal(t, wantOut.String(), s.GetReserves()[1].String())
	})
}
