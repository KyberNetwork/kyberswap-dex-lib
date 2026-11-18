package ghost

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one over several inputs and limits.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	sim, err := NewPoolSimulator(makePool(10_000, 5_000_000, 10_000_000_000, "2", "1"))
	require.NoError(t, err)

	amounts := []int64{1_000_000, 20_000_000, 300_000}
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1_000_000_000_000),
		"binds-out": uint256.NewInt(500_000),
	}

	for name, lim := range limits {
		for _, amt := range amounts {
			var swapLimit poolPkg.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{token1: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(poolPkg.CalcAmountOutParams{
				TokenAmountIn: poolPkg.TokenAmount{Token: token0, Amount: big.NewInt(amt)},
				TokenOut:      token1,
				Limit:         poolPkg.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(poolPkg.CalcAmtOutParams{
				TokenAmountIn: poolPkg.TokenAmt{Token: token0, Amount: *uint256.NewInt(uint64(amt))},
				TokenOut:      token1,
				Limit:         swapLimit,
			})
			require.Equal(t, legacyErr, nativeErr, "%s/%d error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%d amount out", name, amt)
			require.Equal(t, 0, legacy.Fee.Amount.Cmp(native.Fee.Amount.ToBig()), "%s/%d fee", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%d gas", name, amt)
		}
	}
}

// TestCalcAmtOutGolden pins outputs to the hand-worked values in TestCalcAmountOut, so a core math change fails.
func TestCalcAmtOutGolden(t *testing.T) {
	t.Parallel()

	sim, err := NewPoolSimulator(makePool(10_000, 5_000_000, 10_000_000_000, "2", "1"))
	require.NoError(t, err)

	res, err := sim.CalcAmtOut(poolPkg.CalcAmtOutParams{
		TokenAmountIn: poolPkg.TokenAmt{Token: token0, Amount: *uint256.NewInt(1_000_000)},
		TokenOut:      token1,
	})
	require.NoError(t, err)
	require.Equal(t, "1998002", res.TokenAmountOut.Amount.Dec())
	require.Equal(t, "999", res.Fee.Amount.Dec())
}
