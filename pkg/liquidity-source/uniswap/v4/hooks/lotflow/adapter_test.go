package lotflow

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// hookOf builds the adapter from a fixture's pre-state, passing the state through the JSON extra the
// way pool-service does (Track -> HookExtra -> factory).
func hookOf(t *testing.T, c *Case, m *Market) *Hook {
	t.Helper()
	raw, err := json.Marshal(Extra{Tracked: true, Lot: c.Pre.Lot.Address, Usdg: c.Pre.Hook.Usdg,
		V3Factory: c.Pre.Hook.V3Factory, Market: m})
	require.NoError(t, err)
	f := uniswapv4.HookFactories[HookAddresses[0]]
	require.NotNil(t, f, "factory registered for the live V17 hook")
	require.NotNil(t, uniswapv4.HookFactories[HookAddresses[1]], "factory registered for MQ hook #2")
	h := f(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)}).(*Hook)
	require.Equal(t, string(Exchange), h.GetExchange())
	return h
}

// The adapter's BeforeSwap over every prefunded fixture (base 85 + MQ 14), state through JSON: a fill
// is exact unless the quote is Inexact, and then never above the chain; a chain revert is refused.
func TestAdapterBeforeSwapFixtures(t *testing.T) {
	_, base := loadAll(t)
	all := append(base, loadMq(t)...)
	exact, refused, inexactLower, limitRefused := 0, 0, 0, 0
	for _, c := range all {
		if c.Input.Settlement != "prefund" {
			continue
		}
		h := hookOf(t, c, marketOf(t, c))
		r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: c.Input.ZeroForOne,
			AmountSpecified: c.Input.AmountIn.Copy()})
		if !c.IsFill() {
			require.Error(t, err, c.ID)
			refused++
			continue
		}
		if err != nil {
			// The recorded swaps use an extreme price limit. The adapter now sends the buffered
			// NAV band from #1740, and must refuse fills that cannot complete at that limit.
			limit := h.SqrtPriceLimit(c.Input.ZeroForOne)
			if limit != nil {
				in := inputOf(c)
				in.SqrtPriceLimitX96 = limit.ToBig()
				bounded, boundErr := QuoteConservative(h.cfg(), h.Market, in)
				if boundErr != nil || bounded.Venue.AmountIn.Cmp(in.AmountIn) != 0 {
					limitRefused++
					continue
				}
			}
			res, _ := Quote(fixtureConfig(c, FinalisedRules), marketOf(t, c), inputOf(c))
			require.True(t, res != nil && res.Inexact, "%s: %v", c.ID, err)
			inexactLower++
			continue
		}
		eq(t, c.Input.AmountIn.Copy(), r.DeltaSpecified, c.ID)
		out := new(big.Int).Neg(r.DeltaUnspecified)
		if out.Cmp(c.Outcome.AmountOut.Copy()) == 0 {
			exact++
		} else {
			require.Equal(t, -1, out.Cmp(c.Outcome.AmountOut.Copy()), "%s: adapter over-quotes", c.ID)
			inexactLower++
		}
		buy := c.Input.ZeroForOne == h.Market.Venue.UsdgIs0
		require.Equal(t, SwapGasFor(gasFeatures(h.Market, buy, r.SwapInfo.(*SwapInfo).res)),
			r.Gas+uniswapv4.DefaultGas.BaseGas, c.ID)
	}
	t.Logf("adapter BeforeSwap: %d exact fills, %d chain reverts refused, %d Inexact quoted lower/refused, %d NAV-limit refusals, 0 over",
		exact, refused, inexactLower, limitRefused)
}

// UpdateBalance threads state: on the A5 sequential fixtures the adapter's second and third quotes,
// made on the state its own UpdateBalance produced, equal the chain's.
func TestAdapterUpdateBalance(t *testing.T) {
	_, multis := loadA5(t)
	n := 0
	for _, mc := range multis {
		if mc.ID != "a5-amznaapl-seq3" && mc.ID != "a5-amznaapl-standing-seq2" {
			continue
		}
		h := hookOf(t, mc.Swaps[0], marketOf(t, mc.Swaps[0]))
		for k, sc := range mc.Swaps {
			// later swaps keep the hook's threaded state; only the block timestamp moves
			h.Market.Timestamp = sc.Pin.BlockTimestamp.Uint64()
			hs := h.CloneState().(*Hook)
			r, err := hs.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: sc.Input.ZeroForOne,
				AmountSpecified: sc.Input.AmountIn.Copy()})
			require.NoError(t, err, mc.ID, k)
			eq(t, sc.Outcome.AmountOut.Copy(), new(big.Int).Neg(r.DeltaUnspecified), mc.ID, k)
			h.UpdateBalance(r.SwapInfo)
			n++
		}
	}
	require.Equal(t, 5, n)
}

func TestAdapterRefusesExactOut(t *testing.T) {
	_, base := loadAll(t)
	h := hookOf(t, base[0], marketOf(t, base[0]))
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: true, AmountSpecified: big.NewInt(1)})
	require.ErrorIs(t, err, ErrExactOut)
}

// The reported gas covers every minimum passing gas limit we measured with Kyber's REAL router +
// executor on V17 (docs/kyber-delivery/B-EXECUTOR.md B3, MEASURED-ON-FORK): 2-stock 2.85-3.46M,
// Mag 5 (5 stocks) 4.68-5.39M, MQ 3.44-3.77M. Pinned values: 2 -> 3.9M, 5 -> 6.9M, MQ -> 4.2M.
func TestSwapGasCoversMeasured(t *testing.T) {
	require.Equal(t, int64(3_900_000), SwapGas(2, false))
	require.Equal(t, int64(6_900_000), SwapGas(5, false))
	require.Equal(t, int64(4_200_000), SwapGas(3, true))
	require.GreaterOrEqual(t, SwapGas(2, false), int64(3_460_000))
	require.GreaterOrEqual(t, SwapGas(5, false), int64(5_390_000))
	require.GreaterOrEqual(t, SwapGas(3, true), int64(3_770_000))
	for n := 1; n <= 5; n++ {
		require.Less(t, SwapGas(n, false), int64(32_000_000), "under the per-tx cap")
	}
}
