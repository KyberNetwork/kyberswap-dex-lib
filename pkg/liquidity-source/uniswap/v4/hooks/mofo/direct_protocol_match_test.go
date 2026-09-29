package mofo

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The local simulator must agree with Uniswap's canonical V4Quoter on Robinhood chain
// (0x8dc178efb8111bb0973dd9d722ebeff267c98f94, which runs the real MofoHook through PoolManager.unlock) for both
// pool layouts mofo creates: native ETH as currency0 with the coin as currency1 (MOFO/ETH), and the coin as
// currency0 against an ERC-20 (MOFO/STONKBROKER). Pool state -- slot0 and liquidity from StateView
// 0xf3334192d15450cdd385c8b70e03f9a6bd9e673b, the initialised ticks from getTickLiquidity at every edge of the
// LiquidityLocker's positions, and MofoHook.pools(id) -- and every expected quote were read at Robinhood block
// 72317939 (2026-09-25, timestamp 1790348471). Hand-built here, as in the b20 test, rather than bootstrapped
// through the tracker.
// Pure fixtures, no network: these run in CI.

const (
	native      = "0x0000000000000000000000000000000000000000"
	mofoToken   = "0xbce68856680a2043f02593a6d9705390ac5fa01f"
	stonkbroker = "0xe934e36a439c94017b64a3fece66af12099abf50"
	hookAddr    = "0x665C52D02Ddc506dfb3158C100Edc77FE412a8c0"
	fixtureTime = int64(1790348471)
)

func staticExtra(isNative0 bool, tickSpacing string) string {
	n := "false"
	if isNative0 {
		n = "true"
	}
	return `{"0x0":[` + n + `,false],"fee":8388608,"tS":` + tickSpacing + `,"hooks":"` + hookAddr +
		`","uR":"0x204FAca1764B154221e35c0d20aBb3c525710498","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`
}

func mofoEthSim(t *testing.T) *uniswapv4.PoolSimulator {
	t.Helper()
	pool := entity.Pool{
		Address:  "0x3ad7d794ef846fe0831223f373bb07e7b6021618be8bb0aba7b47f9e64a7c6bb",
		Exchange: string(valueobject.ExchangeUniswapV4Mofo),
		Type:     uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: native, Swappable: true}, {Address: mofoToken, Swappable: true}},
		// reserves only cap swap sizes in the simulator; the pool's real balances sit in the PoolManager
		Reserves:    entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: staticExtra(true, "200"),
		Extra: `{"liquidity":16501763971143611628933,"sqrtPriceX96":533030068703974672206693000384579,"tickSpacing":200,"tick":176288,` +
			`"ticks":[{"index":-887200,"liquidityGross":16501763971143611628933,"liquidityNet":16501763971143611628933},` +
			`{"index":202000,"liquidityGross":16501763971143611628933,"liquidityNet":-16501763971143611628933}],` +
			`"hX":{"f":20000,"l":1790284031}}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

func mofoStonkSim(t *testing.T) *uniswapv4.PoolSimulator {
	t.Helper()
	pool := entity.Pool{
		Address:     "0x1f8b90f9cad1130612e3c41c82b0a6454b988e5d5189edabffb68c1ad85d511d",
		Exchange:    string(valueobject.ExchangeUniswapV4Mofo),
		Type:        uniswapv4.DexType,
		Tokens:      []*entity.PoolToken{{Address: mofoToken, Swappable: true}, {Address: stonkbroker, Swappable: true}},
		Reserves:    entity.PoolReserves{"1000000000000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: staticExtra(false, "10"),
		Extra: `{"liquidity":1704769665038186390771842,"sqrtPriceX96":5855357205448406319175333015,"tickSpacing":10,"tick":-52103,` +
			`"ticks":[{"index":-77350,"liquidityGross":1704769665038186390771842,"liquidityNet":1704769665038186390771842},` +
			`{"index":887270,"liquidityGross":1704769665038186390771842,"liquidityNet":-1704769665038186390771842}],` +
			`"hX":{"f":20000,"l":1790284031}}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

func pinClock(t *testing.T, now int64) {
	t.Helper()
	orig := NowFn
	NowFn = func() int64 { return now }
	t.Cleanup(func() { NowFn = orig })
}

func bi(t *testing.T, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok)
	return v
}

type quoteCase struct{ name, tokenIn, tokenOut, amount, want string }

// maxDriftWei bounds the gap to the quoter for the cases below. The fee this hook sets is exact: one pip off would
// already be a 1e-6 relative error (1e17+ wei on these sizes), and TestSameAsStaticFeePool shows the hook adds
// nothing but the fee. What remains is inherited from the shared v3 tick math: v4-core steps through the tick bitmap
// one word (256 tick spacings) at a time and rounds every step, while the v3 simulator moves to the next initialised
// tick in one step. For these cases the gap is 0 to 271 wei and conservative (never more out, never less in, than
// the chain gives). A swap that crosses several bitmap words can land on either side, by about the value of one or
// two raw units of the pool's lower-precision token per word crossed; see TestDirectProtocolMatch_MultiWord.
const maxDriftWei = 1000

func assertConservative(t *testing.T, want string, got *big.Int, exactIn bool) {
	t.Helper()
	w := bi(t, want)
	diff := new(big.Int).Sub(w, got)
	if exactIn {
		require.GreaterOrEqualf(t, diff.Sign(), 0, "simulated output %s exceeds the chain's %s", got, w)
	} else {
		require.LessOrEqualf(t, diff.Sign(), 0, "simulated input %s is below the chain's %s", got, w)
	}
	require.LessOrEqualf(t, diff.Abs(diff).Int64(), int64(maxDriftWei), "want=%s got=%s: drift beyond %d wei", w, got, maxDriftWei)
}

func runExactIn(t *testing.T, sim *uniswapv4.PoolSimulator, cases []quoteCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: c.tokenIn, Amount: bi(t, c.amount)},
				TokenOut:      c.tokenOut,
			})
			require.NoError(t, err)
			assertConservative(t, c.want, res.TokenAmountOut.Amount, true)
			assert.Zero(t, res.RemainingTokenAmountIn.Amount.Sign())
		})
	}
}

func runExactOut(t *testing.T, sim *uniswapv4.PoolSimulator, cases []quoteCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := sim.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{Token: c.tokenOut, Amount: bi(t, c.amount)},
				TokenIn:        c.tokenIn,
			})
			require.NoError(t, err)
			assertConservative(t, c.want, res.TokenAmountIn.Amount, false)
		})
	}
}

func TestDirectProtocolMatch_MofoEth(t *testing.T) {
	pinClock(t, fixtureTime)
	sim := mofoEthSim(t)

	runExactIn(t, sim, []quoteCase{
		{"buy with 1e14 wei", native, mofoToken, "100000000000000", "4435606117448473343083"},
		{"buy with 1e15 wei", native, mofoToken, "1000000000000000", "44340117448329882037572"},
		{"buy with 1e16 wei", native, mofoToken, "10000000000000000", "441813083335030605211502"},
		{"sell 1e4 MOFO", mofoToken, native, "10000000000000000000000", "216492816976404"},
		{"sell 1e6 MOFO", mofoToken, native, "1000000000000000000000000", "21461745339796801"},
		{"sell 5e6 MOFO", mofoToken, native, "5000000000000000000000000", "103679939364200174"},
	})
	runExactOut(t, sim, []quoteCase{
		{"buy exactly 1e4 MOFO", native, mofoToken, "10000000000000000000000", "225459632467812"},
		{"buy exactly 1e6 MOFO", native, mofoToken, "1000000000000000000000000", "22748839429276590"},
	})
}

func TestDirectProtocolMatch_MofoStonkbroker(t *testing.T) {
	pinClock(t, fixtureTime)
	sim := mofoStonkSim(t)

	runExactIn(t, sim, []quoteCase{
		{"buy with 1 STONKBROKER", stonkbroker, mofoToken, "1000000000000000000", "179421745131579879654"},
		{"buy with 10 STONKBROKER", stonkbroker, mofoToken, "10000000000000000000", "1794091856890605953766"},
		{"buy with 100 STONKBROKER", stonkbroker, mofoToken, "100000000000000000000", "17928368790331474360466"},
		{"sell 1e4 MOFO", mofoToken, stonkbroker, "10000000000000000000000", "53504366536390342310"},
		{"sell 1e6 MOFO", mofoToken, stonkbroker, "1000000000000000000000000", "5134568367885977829737"},
		{"sell 5e6 MOFO", mofoToken, stonkbroker, "5000000000000000000000000", "22074408313483376322123"},
	})
	runExactOut(t, sim, []quoteCase{
		{"buy exactly 1e4 MOFO", stonkbroker, mofoToken, "10000000000000000000000", "55758343260213737405"},
		{"buy exactly 1e6 MOFO", stonkbroker, mofoToken, "1000000000000000000000000", "5825984576792630833434"},
	})
}

// afterSwap reverts BelowBirthPrice if a swap ends with the coin cheaper than the pool's birth price. The birth
// price is the pool's outermost initialised tick on the sell side (MOFO/ETH: tick 202000, the upper edge, since
// the coin is currency1; MOFO/STONKBROKER: tick -77350, the lower edge, since the coin is currency0), so:
//   - a sell bigger than the pool can pay is refused (no initialised tick past birth), where the chain would
//     revert BelowBirthPrice;
//   - the price limit handed to the executor sits one unit inside that tick, on the allowed side of birth.
func TestBirthFloor_SellsStopAtBirthPrice(t *testing.T) {
	pinClock(t, fixtureTime)

	for _, c := range []struct {
		name            string
		sim             *uniswapv4.PoolSimulator
		coin, asset     string
		birth           string
		coinIsCurrency1 bool
	}{
		{"MOFO/ETH", mofoEthSim(t), mofoToken, native, "1927678248329847372080333878109930", true},
		{"MOFO/STONKBROKER", mofoStonkSim(t), mofoToken, stonkbroker, "1657027252207809268604205414", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			birth := uint256.MustFromDecimal(c.birth)

			// the executor's price limit for a sell never lets the price cross birth
			meta := c.sim.GetMetaInfo(c.coin, c.asset).(uniswapv4.PoolMetaInfo)
			if c.coinIsCurrency1 {
				assert.True(t, meta.PriceLimit.Lt(birth), "sell limit %s must stay below birth %s", meta.PriceLimit, birth)
			} else {
				assert.True(t, meta.PriceLimit.Gt(birth), "sell limit %s must stay above birth %s", meta.PriceLimit, birth)
			}

			// a sell of 1e15 MOFO (more than the whole supply) would take the price past birth: refused
			huge := bi(t, "1000000000000000000000000000000000")
			_, err := c.sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: c.coin, Amount: huge},
				TokenOut:      c.asset,
			})
			if c.coinIsCurrency1 {
				require.ErrorIs(t, err, uniswapv3.ErrAtOrAboveLargest)
			} else {
				require.ErrorIs(t, err, uniswapv3.ErrBelowSmallest)
			}
		})
	}
}

// A swap that crosses tick-bitmap words: on MOFO/STONKBROKER an exact-output sell for 45168.19 STONKBROKER moves the
// price from tick -52103 to -60982, which v4-core walks in four word steps (2560 ticks each at tickSpacing 10). The
// chain's per-step rounding asks 1012 wei more MOFO than the simulator (V4Quoter at block 72317939:
// 13154204357411679494771900), a relative gap of 8e-23. This is the shared v3 math, not the hook, so the bound here
// is relative.
func TestDirectProtocolMatch_MultiWord(t *testing.T) {
	pinClock(t, fixtureTime)
	res, err := mofoStonkSim(t).CalcAmountIn(poolpkg.CalcAmountInParams{
		TokenAmountOut: poolpkg.TokenAmount{Token: stonkbroker, Amount: bi(t, "45168190962507613995785")},
		TokenIn:        mofoToken,
	})
	require.NoError(t, err)
	want := bi(t, "13154204357411679494771900")
	gap, _ := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Sub(want, res.TokenAmountIn.Amount)),
		new(big.Float).SetInt(want)).Float64()
	assert.InDelta(t, 0, gap, 1e-12, "want=%s got=%s", want, res.TokenAmountIn.Amount)
}

// The hook changes nothing but the fee: the same pool state as a plain, hookless pool whose key carries a static
// 2% fee (20000 pips) quotes identically, to the wei, in every direction and size above.
func TestSameAsStaticFeePool(t *testing.T) {
	pinClock(t, fixtureTime)
	hooked := mofoEthSim(t)
	plain := func() *uniswapv4.PoolSimulator {
		pool := entity.Pool{
			Address:  "0x3ad7d794ef846fe0831223f373bb07e7b6021618be8bb0aba7b47f9e64a7c6bb",
			Exchange: "uniswap-v4",
			Type:     uniswapv4.DexType,
			SwapFee:  20000,
			Tokens:   []*entity.PoolToken{{Address: native, Swappable: true}, {Address: mofoToken, Swappable: true}},
			Reserves: entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000000000000"},
			StaticExtra: `{"0x0":[true,false],"fee":20000,"tS":200,"hooks":"0x0000000000000000000000000000000000000000",` +
				`"uR":"0x204FAca1764B154221e35c0d20aBb3c525710498","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`,
			Extra: `{"liquidity":16501763971143611628933,"sqrtPriceX96":533030068703974672206693000384579,"tickSpacing":200,"tick":176288,` +
				`"ticks":[{"index":-887200,"liquidityGross":16501763971143611628933,"liquidityNet":16501763971143611628933},` +
				`{"index":202000,"liquidityGross":16501763971143611628933,"liquidityNet":-16501763971143611628933}]}`,
		}
		sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDRobinhood)
		require.NoError(t, err)
		return sim
	}()

	for _, c := range []struct{ tokenIn, tokenOut, amount string }{
		{native, mofoToken, "100000000000000"}, {native, mofoToken, "10000000000000000"},
		{mofoToken, native, "10000000000000000000000"}, {mofoToken, native, "5000000000000000000000000"},
	} {
		a, err := hooked.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: c.tokenIn, Amount: bi(t, c.amount)}, TokenOut: c.tokenOut})
		require.NoError(t, err)
		b, err := plain.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: c.tokenIn, Amount: bi(t, c.amount)}, TokenOut: c.tokenOut})
		require.NoError(t, err)
		assert.Equal(t, b.TokenAmountOut.Amount.String(), a.TokenAmountOut.Amount.String(), "%s -> %s %s", c.tokenIn, c.tokenOut, c.amount)
	}
	for _, out := range []string{"10000000000000000000000", "1000000000000000000000000"} {
		a, err := hooked.CalcAmountIn(poolpkg.CalcAmountInParams{TokenAmountOut: poolpkg.TokenAmount{Token: mofoToken, Amount: bi(t, out)}, TokenIn: native})
		require.NoError(t, err)
		b, err := plain.CalcAmountIn(poolpkg.CalcAmountInParams{TokenAmountOut: poolpkg.TokenAmount{Token: mofoToken, Amount: bi(t, out)}, TokenIn: native})
		require.NoError(t, err)
		assert.Equal(t, b.TokenAmountIn.Amount.String(), a.TokenAmountIn.Amount.String())
	}
}
