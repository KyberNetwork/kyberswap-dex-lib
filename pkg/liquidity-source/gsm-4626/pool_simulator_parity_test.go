package gsm4626

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// TestCalcAmtOutMatchesLegacy checks the native CalcAmtOut against the legacy CalcAmountOut on amount out,
// fee, gas, error, and swap info. The GHO-limit case binds, so the native path must reject it too.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		tokenIn  int
		tokenOut int
		rate     *big.Int
		amountIn *big.Int
		wantErr  error
	}{
		{"buy asset", 0, 1, bignumber.NewBig("1146696576337460102970261542"), big.NewInt(1000000000000000000), nil},
		{"sell asset", 1, 0, bignumber.NewBig("1146698616999179571600457092"), big.NewInt(100000000000), nil},
		{"sell asset exceeds gho limit", 1, 0, bignumber.NewBig("1146698616999179571600457092"),
			bignumber.NewBig("200000000000"), ErrGhoLimitExceeded},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sim := newPoolSim(t, tc.rate)
			tokenIn := tokens[tc.tokenIn].Address
			tokenOut := tokens[tc.tokenOut].Address

			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: tc.amountIn},
				TokenOut:      tokenOut,
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *uint256.MustFromBig(tc.amountIn)},
				TokenOut:      tokenOut,
			})

			require.ErrorIs(t, legacyErr, tc.wantErr)
			require.ErrorIs(t, nativeErr, tc.wantErr)
			if tc.wantErr != nil {
				return
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec())
			require.Equal(t, legacy.Gas, native.Gas)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo)
		})
	}
}

// TestCalcAmtOutPinnedBuy anchors the native path to the amount the pre-refactor legacy path returned. Parity
// alone cannot catch a change to the shared core, since legacy forwards native.
func TestCalcAmtOutPinnedBuy(t *testing.T) {
	t.Parallel()

	sim := newPoolSim(t, bignumber.NewBig("1146696576337460102970261542"))
	native, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokens[0].Address, Amount: *uint256.NewInt(1000000000000000000)},
		TokenOut:      tokens[1].Address,
	})
	require.NoError(t, err)
	require.Equal(t, "870763", native.TokenAmountOut.Amount.Dec())
	require.Equal(t, getAssetAmountForBuyAssetGas+buyAssetGas, native.Gas)
}
