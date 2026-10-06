package daosworld

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Every swap ever made through DaosWorldLaunchHook on Robinhood (both launches: TPULP and TEST, 5000 -> 300 bips
// over 300 s), read from PoolManager Swap and hook TaxCollected events. Both pools hold one full-range position
// (ticks +-887220), so the pre-swap state is the previous Swap event's (or Initialize's) price and liquidity.
// Rows span the decay (4671 .. 1256 bips) and the floor (300), so they pin _feeBips and the tax rounding to the
// chain at each block's timestamp. a0/a1 are the pool's own swap deltas, before the hook's deltas.
type swapRow struct {
	poolID          string
	launchTime, ts  int64
	sqrtPriceX96, l string
	tick            int
	a0, a1, tax     string
	bips            uint64
}

var swapRows = []swapRow{
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188718, "1372272028650297976024856118884715", "8660254037844386467637", 195202, "-5275710000000000", "1566187497910794089033093", "4624290000000000", 4671},  // block 80642781
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188747, "1357943793350266111174091206094941", "8660254037844386467637", 194992, "5275709999999999", "-1566187497910794089033093", "2224766906999999", 4217},  // block 80643068
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188848, "1372272028650297976024856109834350", "8660254037844386467637", 195202, "-2668307506995595", "796243009698336823000461", "954649053758777", 2635},    // block 80644043
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188853, "1364987615248848938971811319335477", "8660254037844386467637", 195096, "-27149171594964730", "7645573354826168435692138", "9322042261785310", 2556}, // block 80644090
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188857, "1295042238863369134805743686005733", "8660254037844386467637", 194044, "2963472749621255", "-796243009698336823000460", "739090103755540", 2494},    // block 80644130
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188910, "1302326652264818171858788458279636", "8660254037844386467637", 194156, "-36468403477159436", "9215750300484574048014412", "7274433847009252", 1663}, // block 80644641
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188927, "1218016543195614441328516302886313", "8660254037844386467637", 192817, "-8516970000000000", "1982966112976561269160810", "1383030000000000", 1397},  // block 80644809
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188927, "1199875416991620163334568532302072", "8660254037844386467637", 192517, "-12775455000000000", "2866115900987976412441922", "2074545000000000", 1397}, // block 80644813
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188927, "1173654812446315850237347970266135", "8660254037844386467637", 192075, "-17033940000000000", "3632147764404891632939018", "2766060000000000", 1397}, // block 80644813
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188927, "1140426180572865548943552645317065", "8660254037844386467637", 191501, "-18737334000000000", "3764991154149494970497619", "3042665999999999", 1397}, // block 80644813
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188932, "1105982233579831187572648576958373", "8660254037844386467637", 190887, "10011803951521084", "-1982966112976561269160810", "1320556941205630", 1319}, // block 80644859
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791188936, "1124123359783825465566596346109609", "8660254037844386467637", 191213, "17524403317231660", "-3632147764404891632939018", "2201065056644296", 1256}, // block 80644896
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791189055, "1157351991657275766860391667159168", "8660254037844386467637", 191795, "-54495822469663609", "10649848730209928692072496", "1685437808340111", 300}, // block 80646073
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791189086, "1059922061511782574952174976962216", "8660254037844386467637", 190036, "-9700000000000000", "1710413519674018027742481", "300000000000000", 300},    // block 80646365
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791189113, "1044274377308622834499852561968727", "8660254037844386467637", 189739, "-4801500000000000", "828103941289301247433964", "148500000000000", 300},     // block 80646634
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791189121, "1036698484805646990003333856923995", "8660254037844386467637", 189593, "4801499999999999", "-828103941289301247433964", "144044999999999", 300},     // block 80646705
	{"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce", 1791188697, 1791189137, "1044274377308622834499852555992681", "8660254037844386467637", 189739, "-43016586700745459", "7013986404724708087708843", "1330409897961199", 300},  // block 80646863
	{"0xcd4db9ecea6b2c2cfa1371b0e2682379cd8ac953033705925952fc6ed2ccf5ec", 1791254793, 1791255063, "10330088552455568551261429092103210", "1150447483271055597992", 235576, "6122448979591835", "-339999999999999999999999999", "471428571428571", 770}, // block 81293365
	{"0xcd4db9ecea6b2c2cfa1371b0e2682379cd8ac953033705925952fc6ed2ccf5ec", 1791254793, 1791255459, "33744955938021523934126120318166067", "1150447483271055597992", 259253, "1106466683058764", "-340000000000000000000000000", "33194000491762", 300},  // block 81297240
	{"0xcd4db9ecea6b2c2cfa1371b0e2682379cd8ac953033705925952fc6ed2ccf5ec", 1791254793, 1791255459, "57159823323587479316990811590239852", "1150447483271055597992", 269794, "-1697500000000", "882612962091135985414510", "52500000000", 300},           // block 81297243
}

var weth = valueobject.LowerWrapped(valueobject.ChainIDRobinhood)

func bi(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }

func pinClock(t *testing.T, now int64) {
	orig := NowFn
	NowFn = func() int64 { return now }
	t.Cleanup(func() { NowFn = orig })
}

func newSim(t *testing.T, r swapRow, token string) *uniswapv4.PoolSimulator {
	sim, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address:     r.poolID,
		Exchange:    valueobject.ExchangeUniswapV4DaosWorld,
		Type:        uniswapv4.DexType,
		Tokens:      []*entity.PoolToken{{Address: weth, Swappable: true}, {Address: token, Swappable: true}},
		Reserves:    entity.PoolReserves{"1000000000000000000000000", "1000000000000000000000000000000000"},
		StaticExtra: `{"0x0":[true,false],"fee":0,"tS":60,"hooks":"` + HookAddresses[0].Hex() + `"}`,
		Extra: fmt.Sprintf(`{"liquidity":%s,"sqrtPriceX96":%s,"tickSpacing":60,"tick":%d,"ticks":[`+
			`{"index":-887220,"liquidityGross":%[1]s,"liquidityNet":%[1]s},`+
			`{"index":887220,"liquidityGross":%[1]s,"liquidityNet":-%[1]s}],"hX":{"s":5000,"e":300,"d":300,"l":%[4]d}}`,
			r.l, r.sqrtPriceX96, r.tick, r.launchTime),
	}, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

func TestHistoricalSwaps(t *testing.T) {
	tokens := map[string]string{
		"0x2a1f19a0872eb22a69ccf19fd7afc0232d562ae7ba499d0e0e670530f39190ce": "0x08ff85024b399d229ec636aa5f842c413739b6b7",
		"0xcd4db9ecea6b2c2cfa1371b0e2682379cd8ac953033705925952fc6ed2ccf5ec": "0x61cea6c382f22b836b69d5d1c69aeb34243e2d6a",
	}
	for i, r := range swapRows {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			pinClock(t, r.ts)
			h := &Hook{Extra: Extra{StartFeeBips: 5000, EndFeeBips: 300, DecayDuration: 300, LaunchTime: uint32(r.launchTime)}}
			require.Equal(t, r.bips, h.feeBips(r.ts), "_feeBips")

			sim := newSim(t, r, tokens[r.poolID])
			a0, a1, tax := bi(r.a0), bi(r.a1), bi(r.tax)
			buy := a0.Sign() < 0
			// user amounts: a buy sends pool input + tax; a sell receives pool output - tax
			tokenIn, tokenOut, amountIn, wantOut := weth, tokens[r.poolID], new(big.Int).Sub(tax, a0), a1
			if !buy {
				tokenIn, tokenOut, amountIn, wantOut = tokenOut, weth, new(big.Int).Neg(a1), new(big.Int).Sub(a0, tax)
				fee, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
					AmountOut: a0})
				require.NoError(t, err)
				assert.Equal(t, tax, fee.HookFee, "sell tax on the pool's ETH output")
			} else {
				res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: amountIn})
				require.NoError(t, err)
				assert.Equal(t, tax, res.DeltaSpecified, "buy tax on the ETH input")
			}

			res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
			require.NoError(t, err)
			assert.Equal(t, wantOut, res.TokenAmountOut.Amount, "amountOut")

			// a clone keeps the hook state (a BaseHook clone would quote tax-free)
			res2, err := sim.CloneState().(*uniswapv4.PoolSimulator).CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn}, TokenOut: tokenOut})
			require.NoError(t, err)
			assert.Equal(t, wantOut, res2.TokenAmountOut.Amount, "clone amountOut")

			// CalcAmountIn inverts exact-in: the min input that still yields wantOut
			in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: wantOut}, TokenIn: tokenIn})
			require.NoError(t, err)
			assert.LessOrEqual(t, in.TokenAmountIn.Amount.Cmp(amountIn), 0, "CalcAmountIn above the real input")
			out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: in.TokenAmountIn.Amount}, TokenOut: tokenOut})
			require.NoError(t, err)
			assert.GreaterOrEqual(t, out.TokenAmountOut.Amount.Cmp(wantOut), 0, "CalcAmountIn's input falls short")
		})
	}
}

func TestFeeBips(t *testing.T) {
	e := Extra{StartFeeBips: 5000, EndFeeBips: 300, DecayDuration: 300, LaunchTime: 1000}
	for now, want := range map[int64]uint64{
		900: 5000, 1000: 5000, 1001: 4985 /* 5000-4700/300 */, 1150: 2650, 1299: 316, 1300: 300, 9999: 300} {
		assert.Equalf(t, want, e.feeBips(now), "now=%d", now)
	}
	flat := Extra{StartFeeBips: 100, EndFeeBips: 100, LaunchTime: 1000} // decayDuration 0: endFeeBips at once
	assert.Equal(t, uint64(100), flat.feeBips(900))
	assert.Equal(t, uint64(100), flat.feeBips(1000))
}

func TestUntrackedPoolRefusesToQuote(t *testing.T) {
	_, err := (&Hook{}).BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1)})
	assert.ErrorIs(t, err, ErrPoolNotRegistered)
}
