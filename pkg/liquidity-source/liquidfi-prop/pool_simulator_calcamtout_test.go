package liquidfiprop

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

	sim := newSim(t, nil, false)
	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.NewInt(1 << 62),
		"binds-out": uint256.NewInt(1),
	}
	amounts := []string{usdtIn(1), usdtIn(1500), usdtIn(3000), usdtIn(3500), usdtIn(5000), usdtIn(5001)}

	for name, lim := range limits {
		for _, amt := range amounts {
			in, ok := new(big.Int).SetString(amt, 10)
			require.True(t, ok)

			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(DexType, map[string]*uint256.Int{vault + ":" + wbnb: lim})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: usdt, Amount: in},
				TokenOut:      wbnb,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: usdt, Amount: *uint256.MustFromBig(in)},
				TokenOut:      wbnb,
				Limit:         swapLimit,
			})
			require.ErrorIs(t, nativeErr, legacyErr, "%s/%s error", name, amt)
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%s amount out", name, amt)
			require.Equal(t, 0, legacy.Fee.Amount.Sign(), "%s/%s fee", name, amt)
			require.True(t, native.Fee.Amount.IsZero(), "%s/%s fee", name, amt)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%s gas", name, amt)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo, "%s/%s swap info", name, amt)
		}
	}
}

func TestCalcAmtOutLimitBinds(t *testing.T) {
	t.Parallel()

	sim := newSim(t, nil, false)
	limit := swaplimit.NewInventory(DexType, map[string]*uint256.Int{vault + ":" + wbnb: uint256.NewInt(1)})
	_, err := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: usdt, Amount: *uint256.MustFromDecimal(usdtIn(1500))},
		TokenOut:      wbnb,
		Limit:         limit,
	})
	require.ErrorIs(t, err, pool.ErrNotEnoughInventory)
}
