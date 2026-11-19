package ambient

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one with no limit, a loose limit, and a
// limit that binds on the output token.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	amounts := []string{"1000000", "1000000000000000", "100000000000000000"}
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.MustFromDecimal("1000000000000000000000000"),
		"binds-out": uint256.NewInt(1),
	}

	for name, lim := range limits {
		for _, amt := range amounts {
			in, ok := new(big.Int).SetString(amt, 10)
			require.True(t, ok)

			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{testUSDC: lim, testNative: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: testWETH, Amount: in},
				TokenOut:      testUSDC,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: testWETH, Amount: *uint256.MustFromBig(in)},
				TokenOut:      testUSDC,
				Limit:         swapLimit,
			})
			require.Equal(t, legacyErr, nativeErr, "%s/%s error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%s amount out", name, amt)
			require.Equal(t, 0, legacy.Fee.Amount.Sign(), "%s/%s fee", name, amt)
			require.True(t, native.Fee.Amount.IsZero(), "%s/%s fee", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%s gas", name, amt)
			require.Equal(t, 0, legacy.RemainingTokenAmountIn.Amount.Cmp(native.RemainingTokenAmountIn.Amount.ToBig()),
				"%s/%s remaining in", name, amt)
		}
	}
}

func TestCalcAmtOutLimitBinds(t *testing.T) {
	t.Parallel()

	sim := newTestPoolSimulator(t)
	in := uint256.NewInt(1_000_000)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{testUSDC: uint256.NewInt(1)})

	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: testWETH, Amount: *in},
		TokenOut:      testUSDC,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}
