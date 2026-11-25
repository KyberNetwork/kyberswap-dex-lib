package spireprop

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one on both sides, with no limit,
// a loose limit, and a limit that binds on the output token.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	s := flat(t)
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1 << 62),
		"binds-out": uint256.NewInt(1),
	}
	cases := []struct {
		side   int
		amount string
	}{{0, "10000000000000000"}, {0, "5000000000000000"}, {1, "25000000"}, {1, "12500000"}}

	for name, lim := range limits {
		for _, c := range cases {
			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{
					s.limitKey(baseToken): lim, s.limitKey(quoteToken): lim,
				})
			}
			params := quoteParams(c.side, c.amount)
			params.Limit = pool.AsSwapLimitBig(swapLimit)
			legacy, legacyErr := s.CalcAmountOut(params)

			native, nativeErr := s.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: params.TokenAmountIn.Token, Amount: *uint256.MustFromBig(params.TokenAmountIn.Amount)},
				TokenOut:      params.TokenOut,
				Limit:         swapLimit,
			})
			require.ErrorIs(t, nativeErr, legacyErr, "%s/%d/%s error", name, c.side, c.amount)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%d/%s amount out", name, c.side, c.amount)
			require.Equal(t, 0, legacy.Fee.Amount.Sign(), "%s/%d/%s fee", name, c.side, c.amount)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%d/%s gas", name, c.side, c.amount)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo, "%s/%d/%s swap info", name, c.side, c.amount)
		}
	}
}

func TestCalcAmtOutLimitBindsNative(t *testing.T) {
	t.Parallel()

	s := flat(t)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{s.limitKey(quoteToken): uint256.NewInt(1)})
	_, err := s.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: baseToken, Amount: *uint256.NewInt(10_000_000_000_000_000)},
		TokenOut:      quoteToken,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}
