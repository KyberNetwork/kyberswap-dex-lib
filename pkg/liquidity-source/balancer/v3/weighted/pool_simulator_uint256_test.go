package weighted

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// TestCalcAmtOutParity checks the uint256 CalcAmtOut against the big.Int CalcAmountOut on the same
// base fixture. The engine picks CalcAmtOut, so any drift changes routes.
func TestCalcAmtOutParity(t *testing.T) {
	t.Parallel()
	const (
		weth = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
		tree = "0x77146784315ba81904d654466968e3a7c196d1f3"
	)
	amounts := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(10000),
		big.NewInt(1e18),
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e6)),
		new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1e12)),
	}
	pairs := [][2]string{{weth, tree}, {tree, weth}}

	for _, pair := range pairs {
		for _, amt := range amounts {
			want, wantErr := testutil.CalcAmountOutNative(poolSim, pool.TokenAmount{Token: pair[0], Amount: amt}, pair[1], nil)
			got, gotErr := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: pair[0], Amount: *uint256.MustFromBig(amt)},
				TokenOut:      pair[1],
			})
			if wantErr != nil {
				assert.Equal(t, wantErr, gotErr, "in=%s %s->%s", amt, pair[0], pair[1])
				continue
			}
			require.NoError(t, gotErr, "in=%s %s->%s", amt, pair[0], pair[1])
			assert.Equal(t, want.TokenAmountOut.Token, got.TokenAmountOut.Token)
			assert.Equal(t, want.TokenAmountOut.Amount.String(), got.TokenAmountOut.Amount.Dec(), "amountOut in=%s", amt)
			assert.Equal(t, want.Fee.Token, got.Fee.Token)
			assert.Equal(t, want.Fee.Amount.String(), got.Fee.Amount.Dec(), "fee in=%s", amt)
			assert.Equal(t, want.Gas, got.Gas)
			assert.Nil(t, want.RemainingTokenAmountIn)
			assert.Nil(t, got.RemainingTokenAmountIn)
			wantSwap := want.SwapInfo.(shared.SwapInfo)
			gotSwap := got.SwapInfo.(shared.SwapInfo)
			assert.Equal(t, wantSwap.AggregateFee.String(), gotSwap.AggregateFee.String(), "aggFee in=%s", amt)
			assert.Equal(t, wantSwap.AmountOutBeforeConvert.String(), gotSwap.AmountOutBeforeConvert.String(),
				"amountOutBeforeConvert in=%s", amt)
		}
	}

	res, err := poolSim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: weth, Amount: *uint256.NewInt(1e18)},
		TokenOut:      tree,
	})
	require.NoError(t, err)
	assert.Equal(t, "12858515052701213115427", res.TokenAmountOut.Amount.Dec())
}
