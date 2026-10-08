package fablecurve

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
// (0x8dc178efb8111bb0973dd9d722ebeff267c98f94, which runs the real FableHook through PoolManager.unlock) for both
// launch-pool layouts Fable creates: native ETH as currency0 with the coin as currency1 (FABLE/ETH), and the coin as
// currency0 against an ERC-20 (FABLE/STONKBROKER). Pool state -- slot0 and liquidity from StateView
// 0xf3334192d15450cdd385c8b70e03f9a6bd9e673b, the initialised ticks from getTickLiquidity at both edges of the
// LiquidityLocker's one position in each pool (positionAt), and FableHook.pools(id) -- and every expected quote were
// read at Robinhood block 81852934 (2026-10-06, timestamp 1791312410). Hand-built here, as in the mofo test, rather
// than bootstrapped through the tracker.
// Pure fixtures, no network: these run in CI.

const (
	native      = "0x0000000000000000000000000000000000000000"
	fableToken  = "0x606370c0b28fe9fa6581114ebf3b78ae541180d9"
	stonkbroker = "0xe934e36a439c94017b64a3fece66af12099abf50"
	hookAddr    = "0x376aA4C2e48AC2498CaaAc06bd7A6b6ef373f8E0"
	fixtureTime = int64(1791312410)
)

func staticExtra(isNative0 bool, tickSpacing string, hooks string) string {
	n := "false"
	if isNative0 {
		n = "true"
	}
	fee := "8388608" // the dynamic-fee flag every Fable pool key carries
	if hooks != hookAddr {
		fee = "10000"
	}
	return `{"0x0":[` + n + `,false],"fee":` + fee + `,"tS":` + tickSpacing + `,"hooks":"` + hooks +
		`","uR":"0x204FAca1764B154221e35c0d20aBb3c525710498","pm2":"0x000000000022D473030F116dDEE9F6B43aC78BA3","mc3":"0xcA11bde05977b3631167028862bE2a173976CA11"}`
}

// fableEthState is FABLE/ETH at the fixture block: the coin is currency1, so its birth price (tick 202000) is the
// upper edge of the pool's one range.
const fableEthState = `"liquidity":13357598883239533669812,"sqrtPriceX96":1849867144864904834059615894201144,"tickSpacing":200,"tick":201175,` +
	`"ticks":[{"index":-887200,"liquidityGross":13357598883239533669812,"liquidityNet":13357598883239533669812},` +
	`{"index":202000,"liquidityGross":13357598883239533669812,"liquidityNet":-13357598883239533669812}]`

// fableStonkState is FABLE/STONKBROKER at the fixture block: the coin is currency0, so its birth price
// (tick -70860) is the lower edge of the pool's one range.
const fableStonkState = `"liquidity":1880550227242885107588209,"sqrtPriceX96":2384261363483876107843925372,"tickSpacing":10,"tick":-70073,` +
	`"ticks":[{"index":-70860,"liquidityGross":1880550227242885107588209,"liquidityNet":1880550227242885107588209},` +
	`{"index":887270,"liquidityGross":1880550227242885107588209,"liquidityNet":-1880550227242885107588209}]`

func newSim(t *testing.T, address string, tokens [2]string, static, extra string, plainFee uint64) *uniswapv4.PoolSimulator {
	t.Helper()
	pool := entity.Pool{
		Address:  address,
		Exchange: string(valueobject.ExchangeUniswapV4FableCurve),
		Type:     uniswapv4.DexType,
		Tokens:   []*entity.PoolToken{{Address: tokens[0], Swappable: true}, {Address: tokens[1], Swappable: true}},
		// reserves only cap swap sizes in the simulator; the pool's real balances sit in the PoolManager
		Reserves:    entity.PoolReserves{"1000000000000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: static,
		Extra:       extra,
	}
	if plainFee != 0 {
		pool.Exchange, pool.SwapFee = "uniswap-v4", float64(plainFee)
	}
	sim, err := uniswapv4.NewPoolSimulator(pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

func fableEthSim(t *testing.T) *uniswapv4.PoolSimulator {
	return newSim(t, "0xe017805f7daddd43228b54fda857ca180c1062f3cd53cf8a73736e9a20c36698", [2]string{native, fableToken},
		staticExtra(true, "200", hookAddr), `{`+fableEthState+`,"hX":{"f":10000,"l":1791309607}}`, 0)
}

func fableStonkSim(t *testing.T) *uniswapv4.PoolSimulator {
	return newSim(t, "0x173abfde2df7ab21c20375e83551dbacf04636caee9c92d3700d810a88897ff0", [2]string{fableToken, stonkbroker},
		staticExtra(false, "10", hookAddr), `{`+fableStonkState+`,"hX":{"f":10000,"l":1791309607}}`, 0)
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

// Every case below matches the quoter to the wei. The fee this hook sets is exact (one pip off would already be a
// 1e-6 relative error, 1e9+ wei on these sizes), TestSameAsStaticFeePool shows the hook adds nothing but the fee,
// and none of these swaps leaves its tick-bitmap word, so the shared v3 tick math takes the same single step as
// v4-core.
func runExactIn(t *testing.T, sim *uniswapv4.PoolSimulator, cases []quoteCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: c.tokenIn, Amount: bi(t, c.amount)},
				TokenOut:      c.tokenOut,
			})
			require.NoError(t, err)
			assert.Equal(t, c.want, res.TokenAmountOut.Amount.String())
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
			assert.Equal(t, c.want, res.TokenAmountIn.Amount.String())
		})
	}
}

func TestDirectProtocolMatch_FableEth(t *testing.T) {
	pinClock(t, fixtureTime)
	sim := fableEthSim(t)

	runExactIn(t, sim, []quoteCase{
		{"buy with 1e14 wei", native, fableToken, "100000000000000", "53961244564085978457537"},
		{"buy with 1e15 wei", native, fableToken, "1000000000000000", "538773485498360367523952"},
		{"buy with 1e16 wei", native, fableToken, "10000000000000000", "5305251681318689684859173"},
		{"buy with 1e17 wei", native, fableToken, "100000000000000000", "46008825782886310536769488"},
		{"sell 1e4 FABLE", fableToken, native, "10000000000000000000000", "18159316518144"},
		{"sell 1e6 FABLE", fableToken, native, "1000000000000000000000000", "1810243068230388"},
		{"sell 5e6 FABLE", fableToken, native, "5000000000000000000000000", "8938086331038739"},
	})
	runExactOut(t, sim, []quoteCase{
		{"buy exactly 1e4 FABLE", native, fableToken, "10000000000000000000000", "18529206442737"},
		{"buy exactly 1e6 FABLE", native, fableToken, "1000000000000000000000000", "1858821260889459"},
		{"sell for exactly 1e15 wei", fableToken, native, "1000000000000000", "551628265637969791497251"},
	})
}

func TestDirectProtocolMatch_FableStonkbroker(t *testing.T) {
	pinClock(t, fixtureTime)
	sim := fableStonkSim(t)

	runExactIn(t, sim, []quoteCase{
		{"buy with 1 STONKBROKER", stonkbroker, fableToken, "1000000000000000000", "1093148196174464366875"},
		{"buy with 10 STONKBROKER", stonkbroker, fableToken, "10000000000000000000", "10929761196740128726432"},
		{"buy with 100 STONKBROKER", stonkbroker, fableToken, "100000000000000000000", "109125832957840148740295"},
		{"sell 1e4 FABLE", fableToken, stonkbroker, "10000000000000000000000", "8964270471784453730"},
		{"sell 1e5 FABLE", fableToken, stonkbroker, "100000000000000000000000", "89515091838488286849"},
		{"sell 1e6 FABLE", fableToken, stonkbroker, "1000000000000000000000000", "882586658402471771110"},
	})
	runExactOut(t, sim, []quoteCase{
		{"buy exactly 1e4 FABLE", stonkbroker, fableToken, "10000000000000000000000", "9149194580407114824"},
		{"buy exactly 1e6 FABLE", stonkbroker, fableToken, "1000000000000000000000000", "929649820322281831802"},
	})
}

// On a launch pool, afterSwap reverts BelowBirthPrice if a swap ends with the coin cheaper than the pool's birth
// price. The birth price is the pool's outermost initialised tick on the sell side (FABLE/ETH: tick 202000, the
// upper edge, since the coin is currency1; FABLE/STONKBROKER: tick -70860, the lower edge, since the coin is
// currency0), so:
//   - a sell bigger than the pool can pay is refused (no initialised tick past birth). The two sells below are
//     past each pool's capacity at the fixture block (about 13.3M and 2.5M FABLE), and V4Quoter reverts both with
//     BelowBirthPrice (FableHook.afterSwap, selector 0x21e88689);
//   - the price limit handed to the executor sits one unit inside that tick, on the allowed side of birth.
func TestBirthFloor_SellsStopAtBirthPrice(t *testing.T) {
	pinClock(t, fixtureTime)

	for _, c := range []struct {
		name            string
		sim             *uniswapv4.PoolSimulator
		coin, asset     string
		birth           string // FableHook.pools(id).birthSqrtPriceX96
		tooBig          string
		coinIsCurrency1 bool
	}{
		{"FABLE/ETH", fableEthSim(t), fableToken, native, "1927678248329847372080333878109930",
			"100000000000000000000000000", true},
		{"FABLE/STONKBROKER", fableStonkSim(t), fableToken, stonkbroker, "2292192908003631255009281043",
			"5000000000000000000000000", false},
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

			_, err := c.sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: c.coin, Amount: bi(t, c.tooBig)},
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

// A mesh pool (none is live yet) holds one full-range position and has no floor: the same FABLE/ETH price and
// liquidity spread over the full range quotes a sell far past the launch pool's birth price, identical to a
// hookless full-range pool with a static 1% fee.
func TestMeshPool_NoFloor(t *testing.T) {
	pinClock(t, fixtureTime)
	const meshState = `"liquidity":13357598883239533669812,"sqrtPriceX96":1849867144864904834059615894201144,"tickSpacing":200,"tick":201175,` +
		`"ticks":[{"index":-887200,"liquidityGross":13357598883239533669812,"liquidityNet":13357598883239533669812},` +
		`{"index":887200,"liquidityGross":13357598883239533669812,"liquidityNet":-13357598883239533669812}]`
	mesh := newSim(t, "0x00000000000000000000000000000000000000000000000000000000000000fa", [2]string{native, fableToken},
		staticExtra(true, "200", hookAddr), `{`+meshState+`,"hX":{"f":10000,"l":1791309607}}`, 0)
	plain := newSim(t, "0x00000000000000000000000000000000000000000000000000000000000000fa", [2]string{native, fableToken},
		staticExtra(true, "200", native), `{`+meshState+`}`, 10000)

	tooBigForLaunchPool := bi(t, "100000000000000000000000000") // 1e8 FABLE, refused by the launch pool above
	a, err := mesh.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: fableToken, Amount: tooBigForLaunchPool}, TokenOut: native})
	require.NoError(t, err)
	b, err := plain.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: fableToken, Amount: tooBigForLaunchPool}, TokenOut: native})
	require.NoError(t, err)
	assert.Positive(t, a.TokenAmountOut.Amount.Sign())
	assert.Equal(t, b.TokenAmountOut.Amount.String(), a.TokenAmountOut.Amount.String())
	assert.Equal(t, int64(gasBeforeSwap), a.Gas-b.Gas, "the hooked pool costs the locker checkpoint on top")
}

// The hook changes nothing but the fee: the same pool state as a plain, hookless pool whose key carries a static
// 1% fee (10000 pips) quotes identically, to the wei, in every direction and size above, and costs gasBeforeSwap
// more.
func TestSameAsStaticFeePool(t *testing.T) {
	pinClock(t, fixtureTime)
	for _, c := range []struct {
		name          string
		hooked, plain *uniswapv4.PoolSimulator
		asset         string
		in, out       []string
	}{
		{"FABLE/ETH", fableEthSim(t), newSim(t, "0xe017805f7daddd43228b54fda857ca180c1062f3cd53cf8a73736e9a20c36698",
			[2]string{native, fableToken}, staticExtra(true, "200", native), `{`+fableEthState+`}`, 10000), native,
			[]string{"100000000000000", "100000000000000000", "-10000000000000000000000", "-5000000000000000000000000"},
			[]string{"10000000000000000000000", "1000000000000000000000000"}},
		{"FABLE/STONKBROKER", fableStonkSim(t), newSim(t, "0x173abfde2df7ab21c20375e83551dbacf04636caee9c92d3700d810a88897ff0",
			[2]string{fableToken, stonkbroker}, staticExtra(false, "10", native), `{`+fableStonkState+`}`, 10000), stonkbroker,
			[]string{"1000000000000000000", "100000000000000000000", "-10000000000000000000000", "-1000000000000000000000000"},
			[]string{"10000000000000000000000", "1000000000000000000000000"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			// in: a buy of the coin with that much of the asset, or (negative) a sell of that much of the coin
			for _, amount := range c.in {
				tokenIn, tokenOut, amt := c.asset, fableToken, bi(t, amount)
				if amt.Sign() < 0 {
					tokenIn, tokenOut, amt = fableToken, c.asset, amt.Neg(amt)
				}
				a, err := c.hooked.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut})
				require.NoError(t, err)
				b, err := c.plain.CalcAmountOut(poolpkg.CalcAmountOutParams{TokenAmountIn: poolpkg.TokenAmount{Token: tokenIn, Amount: amt}, TokenOut: tokenOut})
				require.NoError(t, err)
				assert.Equal(t, b.TokenAmountOut.Amount.String(), a.TokenAmountOut.Amount.String(), "%s -> %s %s", tokenIn, tokenOut, amt)
				assert.Equal(t, int64(gasBeforeSwap), a.Gas-b.Gas)
			}
			// out: exactly that much of the coin bought with the asset
			for _, amount := range c.out {
				a, err := c.hooked.CalcAmountIn(poolpkg.CalcAmountInParams{TokenAmountOut: poolpkg.TokenAmount{Token: fableToken, Amount: bi(t, amount)}, TokenIn: c.asset})
				require.NoError(t, err)
				b, err := c.plain.CalcAmountIn(poolpkg.CalcAmountInParams{TokenAmountOut: poolpkg.TokenAmount{Token: fableToken, Amount: bi(t, amount)}, TokenIn: c.asset})
				require.NoError(t, err)
				assert.Equal(t, b.TokenAmountIn.Amount.String(), a.TokenAmountIn.Amount.String())
			}
		})
	}
}
