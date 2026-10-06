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

// liveBpsBound is the acceptance bound on |plugin - V4Quoter| / V4Quoter, in bps. It includes the
// 1.5 bps buy haircut on top of the chord's own (under-quoting) error.
const liveBpsBound = 5.0

// sweepFracs places sizes strictly inside every ladder segment (and below the first rung), so the
// interpolation itself is what is measured, not the rungs it was built from.
var sweepFracs = []float64{0.1, 0.25, 0.4, 0.55, 0.7, 0.9}

// TestLive_PluginVsV4Quoter runs each live venue through Kyber's own pipeline (PoolTracker ->
// hook Track -> NewPoolSimulator -> CalcAmountOut) and compares the result with a fresh V4Quoter
// call pinned to the same block, at six sizes inside every ladder segment in both directions.
// The plugin must never quote more than the V4Quoter (beyond one base unit of rounding).
func TestLive_PluginVsV4Quoter(t *testing.T) {
	skipInCI(t)
	ctx := context.Background()
	rpc := liveRPC()
	log := newLiveLog(t)
	log.Printf("# plugin vs V4Quoter, %s", time.Now().UTC().Format(time.RFC3339))

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

		usdg, lot := strings.ToLower(v.Currency0), strings.ToLower(v.Currency1)
		for _, d := range []struct {
			name              string
			ladder            []Rung
			tokenIn, tokenOut string
			zeroForOne        bool
		}{{"buy", hx.Buy, usdg, lot, true}, {"sell", hx.Sell, lot, usdg, false}} {
			var worstOver, worstUnder float64
			n := 0
			prev := new(big.Int)
			for _, r := range d.ladder {
				hi := r.In.ToBig()
				span := new(big.Int).Sub(hi, prev)
				for _, f := range sweepFracs {
					amt, _ := new(big.Float).Mul(big.NewFloat(f), new(big.Float).SetInt(span)).Int(nil)
					amt.Add(amt, prev)
					if amt.Sign() <= 0 {
						continue
					}
					res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: d.tokenIn, Amount: new(big.Int).Set(amt)},
						TokenOut:      d.tokenOut,
					})
					ref, ok := v4Quote(ctx, rpc, block, v, d.zeroForOne, amt)
					if !ok {
						log.Printf("%s %s in=%s quoter=REVERT plugin_err=%v", v.Name, d.name, amt, err)
						assert.Error(t, err, "plugin must refuse what the quoter refuses")
						continue
					}
					require.NoError(t, err, "%s %s %s", v.Name, d.name, amt)
					got := res.TokenAmountOut.Amount
					e := errBps(got, ref)
					n++
					worstOver, worstUnder = math.Max(worstOver, e), math.Min(worstUnder, e)
					log.Printf("%s %s in=%s plugin=%s quoter=%s err_bps=%+.4f gas=%d", v.Name, d.name, amt,
						got, ref, e, res.Gas)
					assert.LessOrEqual(t, math.Abs(e), liveBpsBound, "%s %s %s", v.Name, d.name, amt)
					assert.LessOrEqual(t, got.Cmp(new(big.Int).Add(ref, big.NewInt(1))), 0,
						"%s %s %s: plugin must not over-quote the real fill (%+.4f bps)", v.Name, d.name, amt, e)
				}
				prev = hi
			}
			log.Printf("SUMMARY %s %s block=%d sizes=%d worst_over_bps=%+.4f worst_under_bps=%+.4f",
				v.Name, d.name, hx.Block, n, worstOver, worstUnder)
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
