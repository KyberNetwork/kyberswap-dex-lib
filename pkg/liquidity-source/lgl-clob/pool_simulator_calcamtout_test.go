package lglclob

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCalcAmtOutMatchesLegacy checks the native quote against the legacy one in both book directions, with no limit,
// a loose limit, and a limit that binds on the output token. The pool has a market maker, so the limit applies.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	t.Parallel()

	const maker = "0x125f12a1938b97f11fdc35b1a5fb4d5217cda50b"
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(wxtzWethPoolJSON), &ep))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))
	extra.Maker = maker
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(extraBytes)
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	wxtz, weth := sim.Info.Tokens[0], sim.Info.Tokens[1]
	cases := []struct {
		name, tokenIn, tokenOut, amount string
	}{
		{"x-to-y", wxtz, weth, "12000000000000000000000"},
		{"x-to-y-small", wxtz, weth, "1000000000000000000"},
		{"y-to-x", weth, wxtz, "1000000000000000"},
	}

	limits := map[string]*uint256.Int{
		"none":      nil,
		"loose":     uint256.MustFromDecimal("1000000000000000000000000"),
		"binds-out": uint256.NewInt(1),
	}
	for name, lim := range limits {
		for _, c := range cases {
			in, ok := new(big.Int).SetString(c.amount, 10)
			require.True(t, ok)

			var swapLimit pool.SwapLimit
			if lim != nil {
				swapLimit = swaplimit.NewInventory(valueobject.ExchangeHanji, map[string]*uint256.Int{
					sim.limitKey(c.tokenOut): lim,
				})
			}
			legacy, legacyErr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: c.tokenIn, Amount: in},
				TokenOut:      c.tokenOut,
				Limit:         pool.AsSwapLimitBig(swapLimit),
			})
			native, nativeErr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: c.tokenIn, Amount: *uint256.MustFromBig(in)},
				TokenOut:      c.tokenOut,
				Limit:         swapLimit,
			})
			require.ErrorIs(t, nativeErr, legacyErr, "%s/%s error", name, c.name)
			if name == "binds-out" {
				require.ErrorIs(t, nativeErr, pool.ErrNotEnoughInventory, "%s limit must bind", c.name)
			}
			if legacyErr != nil {
				continue
			}
			require.Equal(t, 0, legacy.TokenAmountOut.Amount.Cmp(native.TokenAmountOut.Amount.ToBig()), "%s/%s amount out", name, c.name)
			require.Equal(t, 0, legacy.Fee.Amount.Cmp(native.Fee.Amount.ToBig()), "%s/%s fee", name, c.name)
			require.Equal(t, 0, legacy.RemainingTokenAmountIn.Amount.Cmp(native.RemainingTokenAmountIn.Amount.ToBig()),
				"%s/%s remaining in", name, c.name)
			require.Equal(t, legacy.Gas, native.Gas, "%s/%s gas", name, c.name)
			require.Equal(t, legacy.SwapInfo, native.SwapInfo, "%s/%s swap info", name, c.name)
		}
	}
}
