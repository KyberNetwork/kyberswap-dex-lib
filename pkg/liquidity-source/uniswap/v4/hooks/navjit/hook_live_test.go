package navjit

import (
	"context"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The three live NavJitHookV15 venues on Robinhood chain (USDG is currency0 for all of them).
var liveVenues = []poolSpec{
	{Name: "LOT-two", ID: "0x8327ac000f8b77d690b8315290ba7dcb4834b12b377b0367234b3d6661f20056",
		Currency0: usdgAddr, Currency1: "0xF5e660dC904b5017fF2cecC22608567c2bE6D7c6"},
	{Name: "NAVJIT1", ID: "0xc979ff8f442939c2dc77163985864882ff533f6da29a3fc2c3e763b520bc2e10",
		Currency0: usdgAddr, Currency1: "0xa4F3F662584B415Cb5bD56Bfab0Db32d0b33E3d7"},
	{Name: "NAVJIT2", ID: "0x17bd247937ea62b1a68c81ed9c8c13986a1ceb6c206ca42048292dfc5d11e431",
		Currency0: usdgAddr, Currency1: "0xBD8b2E17194b7470bB06B5a76964804fa10f5e2f"},
}

func init() {
	for i := range liveVenues {
		v := &liveVenues[i]
		v.Dec0, v.Dec1, v.Fee, v.TickSpacing, v.Hooks = usdgDecimals, 18, 0, 1, HookAddresses[0].Hex()
	}
}

// liveBpsBound is the brief's acceptance bound: the plugin's quote within 5 bps of V4Quoter.
const liveBpsBound = 5.0

// TestLive_PluginVsV4Quoter runs each live venue through Kyber's own pipeline (PoolTracker ->
// hook Track -> NewPoolSimulator -> CalcAmountOut) and compares the result with a fresh V4Quoter
// call pinned to the same block, at five sizes per direction chosen BETWEEN the ladder rungs, so
// the interpolation itself is what is measured.
func TestLive_PluginVsV4Quoter(t *testing.T) {
	skipInCI(t)
	ctx := context.Background()
	rpc := liveRPC()
	log := newLiveLog(t)
	log.Printf("# plugin vs V4Quoter, %s", time.Now().UTC().Format(time.RFC3339))

	buyUsd := []float64{2.5, 37, 420, 2_600, 12_000}
	sellFrac := []float64{0.004, 0.06, 0.3, 0.65, 0.97} // of the top sell rung

	for _, v := range liveVenues {
		p := trackPool(ctx, t, rpc, v)
		require.Equal(t, valueobject.ExchangeUniswapV4NavJit, p.Exchange)
		sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDRobinhood)
		require.NoError(t, err)

		var ex uniswapv4.Extra
		require.NoError(t, json.Unmarshal([]byte(p.Extra), &ex))
		var hx Extra
		require.NoError(t, uniswapv4.HookExtra(ex.HookExtra).Unmarshal(&hx))
		require.True(t, hx.Tracked)
		block := new(big.Int).SetUint64(hx.Block)
		log.Printf("## %s pool %s block %d nav18 %v float %v buyRungs %d sellRungs %d reserves %v",
			v.Name, v.ID[:10], hx.Block, hx.Nav, hx.Float, len(hx.Buy), len(hx.Sell), p.Reserves)
		require.NotEmpty(t, hx.Buy)
		require.NotEmpty(t, hx.Sell)

		check := func(dir string, tokenIn, tokenOut string, zeroForOne bool, amt *big.Int) {
			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amt)},
				TokenOut:      tokenOut,
			})
			ref, ok := v4Quote(ctx, rpc, block, v, zeroForOne, amt)
			if !ok {
				log.Printf("%s %s in=%s quoter=REVERT plugin_err=%v", v.Name, dir, amt, err)
				assert.Error(t, err, "plugin must refuse what the quoter refuses")
				return
			}
			require.NoError(t, err, "%s %s %s", v.Name, dir, amt)
			e := errBps(res.TokenAmountOut.Amount, ref)
			log.Printf("%s %s in=%s plugin=%s quoter=%s err_bps=%+.4f gas=%d", v.Name, dir, amt,
				res.TokenAmountOut.Amount, ref, e, res.Gas)
			assert.LessOrEqual(t, math.Abs(e), liveBpsBound, "%s %s %s", v.Name, dir, amt)
			// Not exactly concave (whole-unit mint rounding): allow a sliver over, never a real over-quote.
			assert.LessOrEqual(t, e, 0.5, "%s %s: plugin must not over-quote the real fill", v.Name, dir)
		}

		usdg, lot := strings.ToLower(v.Currency0), strings.ToLower(v.Currency1)
		for _, usd := range buyUsd {
			amt, _ := new(big.Float).Mul(big.NewFloat(usd), new(big.Float).SetInt(bignumber.TenPowInt(usdgDecimals))).Int(nil)
			check("buy", usdg, lot, true, amt)
		}
		top := hx.Sell[len(hx.Sell)-1].In.ToBig()
		for _, f := range sellFrac {
			amt, _ := new(big.Float).Mul(big.NewFloat(f), new(big.Float).SetInt(top)).Int(nil)
			check("sell", lot, usdg, false, amt)
		}

		// Beyond the float: the hook reverts SellExceedsFloat, the plugin must refuse.
		if hx.Float != nil {
			over := new(big.Int).Add(hx.Float.ToBig(), bignumber.TenPowInt(18))
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: lot, Amount: over}, TokenOut: usdg})
			_, ok := v4Quote(ctx, rpc, block, v, false, over)
			log.Printf("%s sell-over-float in=%s quoter_ok=%v plugin_err=%v", v.Name, over, ok, err)
			assert.Error(t, err)
			assert.False(t, ok)
		}
	}
}
