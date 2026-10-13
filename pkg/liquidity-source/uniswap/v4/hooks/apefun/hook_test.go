package apefun

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const launched = int64(1791414828) // ARC2REWARD's launch, 2026-10-08

// Every creator tax the tests cover: none, 1% (all taxed live coins) and register's 10% ceiling.
var taxes = []int64{0, 100, 1000}

// solidityFees is a literal port of LaunchFeeProtectionHook._fees, kept independent of the package code:
//
//	rate = buy && !exempt ? (elapsed == 0 ? 9900 : elapsed == 1 ? 618 : elapsed == 2 ? 19 : 0) : 0;
//	cap = 9800 - p.tax; if (rate > cap) rate = cap;
//	snipe = gross * rate / 10000; fee = gross * 100 / 10000 + snipe; tax = gross * p.tax / 10000;
func solidityFees(gross *big.Int, taxBps, elapsed int64, buy bool) (fee, tax *big.Int) {
	var rate int64
	if buy {
		switch elapsed {
		case 0:
			rate = 9900
		case 1:
			rate = 618
		case 2:
			rate = 19
		}
	}
	if c := 9800 - taxBps; rate > c {
		rate = c
	}
	floor := func(r int64) *big.Int {
		return new(big.Int).Quo(new(big.Int).Mul(gross, big.NewInt(r)), big.NewInt(10000))
	}
	return new(big.Int).Add(floor(100), floor(rate)), floor(taxBps)
}

func hookAt(tax int64, quote0 bool) *Hook {
	return &Hook{Extra: Extra{Live: true, QuoteIsCurrency0: quote0, TaxBps: tax, Launched: launched}}
}

func pinClock(t testing.TB, now int64) {
	t.Helper()
	orig := NowFn
	NowFn = func() int64 { return now }
	t.Cleanup(func() { NowFn = orig })
}

func bi(t testing.TB, s string) *big.Int {
	t.Helper()
	v, ok := new(big.Int).SetString(s, 10)
	require.True(t, ok)
	return v
}

// chainSecond is the quoting time whose snipe rate is the chain's at elapsed second e (clock margin applied).
func chainSecond(e int64) int64 { return launched + e + clockMarginSeconds }

// chainRate is the hook's rate at block timestamp launched + elapsed (elapsed >= 0), capped at 9800 - tax.
func chainRate(tax, elapsed int64) int64 {
	if elapsed >= 3 {
		return 0
	}
	return min([]int64{9_900, 618, 19}[elapsed], 9_800-tax)
}

// The rate at each second of the window: 99% is capped at 9800 - tax, so the hook never keeps more than 99%
// of a buy (100 base + 9800 - tax snipe + tax).
func TestSnipeRate_EachSecond(t *testing.T) {
	for _, c := range []struct {
		tax  int64
		want [3]int64 // chain elapsed 0, 1, 2
	}{
		{0, [3]int64{9_800, 618, 19}},
		{100, [3]int64{9_700, 618, 19}},
		{1000, [3]int64{8_800, 618, 19}},
	} {
		e := Extra{TaxBps: c.tax, Launched: launched}
		for elapsed, want := range c.want {
			assert.Equalf(t, want, e.snipeRate(chainSecond(int64(elapsed))), "tax %d elapsed %d", c.tax, elapsed)
		}
		for _, elapsed := range []int64{3, 4, 60, 86_400} {
			assert.Zero(t, e.snipeRate(chainSecond(elapsed)), "tax %d elapsed %d", c.tax, elapsed)
		}
		// a clock behind the launch reads as the launch second (most expensive)
		assert.Equal(t, c.want[0], e.snipeRate(launched-7))
		assert.Equal(t, int64(9_900), baseFeeBps+c.want[0]+c.tax, "99%% at most")
	}
}

// The quoting clock may run up to clockMarginSeconds ahead of the block: whatever the skew in that range, the
// quoted rate is never below the rate the chain charges, and it is exact from 3 + margin seconds on.
func TestSnipeRate_ClockAheadNeverUndercharges(t *testing.T) {
	for _, tax := range taxes {
		e := Extra{TaxBps: tax, Launched: launched}
		for raw := int64(-2); raw <= 10; raw++ { // quoting clock - launch
			quoted := e.snipeRate(launched + raw)
			for skew := int64(0); skew <= clockMarginSeconds; skew++ {
				if block := raw - skew; block >= 0 {
					assert.GreaterOrEqualf(t, quoted, chainRate(tax, block), "tax %d clock %d skew %d", tax, raw, skew)
				}
			}
			if raw >= 3+clockMarginSeconds {
				assert.Zero(t, quoted, "exact past the window plus margin")
			}
		}
	}
}

// fees must equal the Solidity port bit for bit over the boundary amounts of every floor, every tax the tests
// cover, every second of the window, buys and sells.
func TestFees_MatchSolidity(t *testing.T) {
	amounts := []string{"1", "99", "100", "101", "161", "162", "199", "200", "526", "527", "9999", "10000", "10001",
		"123456789", "1000000000000000000", "170141183460469231731687303715884105727"}
	for gross := int64(0); gross <= 20_000; gross += 7 {
		amounts = append(amounts, big.NewInt(gross).String())
	}
	for _, tax := range taxes {
		for elapsed := int64(0); elapsed <= 4; elapsed++ {
			for _, buy := range []bool{true, false} {
				e := Extra{TaxBps: tax, Launched: launched}
				for _, a := range amounts {
					gross := bi(t, a)
					fee, taxAmt := solidityFees(gross, tax, elapsed, buy)
					want := new(big.Int).Add(fee, taxAmt)
					if got := e.fees(gross, buy, chainSecond(elapsed)); got.Cmp(want) != 0 {
						t.Fatalf("tax %d elapsed %d buy %v gross %s: got %s want %s", tax, elapsed, buy, a, got, want)
					}
				}
			}
		}
	}
}

// Each part is floored on its own, so small amounts keep less than the summed rate would: 150 at 1% + 1% keeps
// 1 + 1, not 3; 99 keeps nothing.
func TestFees_BoundaryRounding(t *testing.T) {
	after := chainSecond(3)
	for _, c := range []struct {
		gross string
		tax   int64
		now   int64
		buy   bool
		want  int64
	}{
		{"99", 0, after, true, 0},
		{"100", 0, after, true, 1},
		{"150", 100, after, true, 2},
		{"199", 100, after, false, 2},
		{"200", 100, after, false, 4},
		{"999", 1000, after, true, 108},      // 9 + 99
		{"100", 0, chainSecond(0), true, 99}, // 1 + 98 (9800 cap)
		{"101", 0, chainSecond(0), true, 99}, // 1 + floor(98.98)
		{"161", 0, chainSecond(1), true, 10},
		{"162", 0, chainSecond(1), true, 11}, // 1 + floor(10.0116)
		{"526", 0, chainSecond(2), true, 5},  // 5 + 0
		{"527", 0, chainSecond(2), true, 6},  // 5 + floor(1.0013)
		{"10000", 0, chainSecond(1), false, 100},
	} {
		e := Extra{TaxBps: c.tax, Launched: launched}
		assert.Equalf(t, c.want, e.fees(bi(t, c.gross), c.buy, c.now).Int64(), "%+v", c)
	}
}

// BeforeSwap takes a buy's fee and tax off the gross input (specified delta) at several sizes; a sell passes
// through, its fee comes in AfterSwap. Works whichever side the quote asset is.
func TestBeforeSwap_BuyTakesFromInput(t *testing.T) {
	for _, quote0 := range []bool{true, false} {
		buyZeroForOne := quote0 // quote in
		for _, tax := range taxes {
			h := hookAt(tax, quote0)
			for elapsed := int64(0); elapsed <= 3; elapsed++ {
				pinClock(t, chainSecond(elapsed))
				for _, a := range []string{"5000", "495000", "1000000000000000", "7900236222200242", "10000000000000000000"} {
					res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: buyZeroForOne,
						AmountSpecified: bi(t, a)})
					require.NoError(t, err)
					require.NoError(t, uniswapv4.ValidateBeforeSwapResult(res))
					fee, taxAmt := solidityFees(bi(t, a), tax, elapsed, true)
					assert.Equal(t, new(big.Int).Add(fee, taxAmt).String(), res.DeltaSpecified.String())
					assert.Zero(t, res.DeltaUnspecified.Sign())
					assert.Zero(t, res.SwapFee, "pool fee 0, no override")
					assert.Equal(t, int64(gasBeforeSwapBuy), res.Gas)

					res, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: !buyZeroForOne,
						AmountSpecified: bi(t, a)})
					require.NoError(t, err)
					assert.Zero(t, res.DeltaSpecified.Sign(), "sell: nothing taken before the swap")
					assert.Equal(t, int64(gasBeforeSwapSell), res.Gas)
				}
			}
		}
	}
}

// AfterSwap takes a sell's fee and tax off the pool's quote output, never a snipe fee (sells are not sniped).
func TestAfterSwap_SellTakesFromOutput(t *testing.T) {
	for _, quote0 := range []bool{true, false} {
		sellZeroForOne := !quote0 // coin in
		for _, tax := range taxes {
			h := hookAt(tax, quote0)
			for elapsed := int64(0); elapsed <= 3; elapsed++ {
				pinClock(t, chainSecond(elapsed))
				for _, q := range []string{"319", "485149", "3327787", "475998250000001", "7587386867801113"} {
					res, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
						BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: sellZeroForOne},
						AmountIn:         bignumber.TenPowInt(24), AmountOut: bi(t, q)})
					require.NoError(t, err)
					require.NoError(t, uniswapv4.ValidateAfterSwapResult(res))
					fee, taxAmt := solidityFees(bi(t, q), tax, elapsed, false)
					assert.Equal(t, new(big.Int).Add(fee, taxAmt).String(), res.HookFee.String())
					assert.Equal(t, int64(gasAfterSwapSell), res.Gas)

					res, err = h.AfterSwap(&uniswapv4.AfterSwapParams{
						BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: !sellZeroForOne},
						AmountIn:         bi(t, q), AmountOut: bignumber.TenPowInt(24)})
					require.NoError(t, err)
					assert.Zero(t, res.HookFee.Sign(), "buy: nothing taken after the swap")
				}
			}
		}
	}
	h := hookAt(100, true)
	h.BuybackVault = true
	res, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountIn: bignumber.TenPowInt(18), AmountOut: bignumber.TenPowInt(6)})
	require.NoError(t, err)
	assert.Equal(t, int64(gasAfterSwapSell+gasObservation), res.Gas, "a buyback pool records an observation")
}

// The hook only accepts exact-input swaps (amountSpecified < 0) of at most type(int128).max. The executor always
// swaps exact-in, CalcAmountIn only reverses it (see TestCalcAmountIn_ReversesExactIn); an input above int128
// max is refused as the hook would revert UnsupportedSwap.
func TestBeforeSwap_ExactInputBound(t *testing.T) {
	pinClock(t, launched+60)
	h := hookAt(100, true)
	for _, zeroForOne := range []bool{true, false} {
		_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne, AmountSpecified: maxInt128})
		require.NoError(t, err)
		_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: zeroForOne,
			AmountSpecified: new(big.Int).Add(maxInt128, bignumber.One)})
		assert.ErrorIs(t, err, ErrAmountTooLarge)
	}
}

// CalcIn's gross-up is the smallest gross whose net covers the request, in the window and after, for every tax:
// net(gross) >= want and net(gross - 1) < want.
func TestGrossFor_Minimal(t *testing.T) {
	for _, tax := range taxes {
		e := Extra{TaxBps: tax, Launched: launched}
		for elapsed := int64(0); elapsed <= 3; elapsed++ {
			now := launched + elapsed
			for _, buy := range []bool{true, false} {
				for _, n := range []int64{1, 2, 3, 50, 99, 100, 101, 999, 1000, 4_851, 485_149, 3_327_787, 1_000_000_007} {
					want := big.NewInt(n)
					gross, err := e.grossFor(want, buy, now)
					require.NoError(t, err)
					net := func(g *big.Int) *big.Int { return new(big.Int).Sub(g, e.fees(g, buy, now)) }
					assert.GreaterOrEqualf(t, net(gross).Cmp(want), 0, "tax %d el %d buy %v n %d", tax, elapsed, buy, n)
					assert.Negativef(t, net(new(big.Int).Sub(gross, bignumber.One)).Cmp(want), "tax %d el %d buy %v n %d",
						tax, elapsed, buy, n)
				}
			}
		}
	}
}

// CalcIn hook results: a sell grosses the pool's output up before the swap, a buy grosses its input up after it.
func TestCalcIn_GrossUps(t *testing.T) {
	pinClock(t, launched+1)
	h := hookAt(100, true)
	want := big.NewInt(3_327_787)

	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: false, AmountSpecified: want})
	require.NoError(t, err)
	q := new(big.Int).Add(want, res.DeltaSpecified)
	assert.Equal(t, want.String(), new(big.Int).Sub(q, h.fees(q, false, NowFn())).String())
	after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{ZeroForOne: false},
		AmountIn: bignumber.TenPowInt(24), AmountOut: q})
	require.NoError(t, err)
	assert.Zero(t, after.HookFee.Sign())

	res, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: true, AmountSpecified: bignumber.TenPowInt(24)})
	require.NoError(t, err)
	assert.Zero(t, res.DeltaSpecified.Sign())
	after, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{ZeroForOne: true},
		AmountIn: want, AmountOut: bignumber.TenPowInt(24)})
	require.NoError(t, err)
	gross := new(big.Int).Add(want, after.HookFee)
	assert.GreaterOrEqual(t, new(big.Int).Sub(gross, h.fees(gross, true, NowFn())).Cmp(want), 0)
}

// A pool Track did not confirm (unregistered, still inside its creation transaction, other chain) is never priced.
func TestUntrackedPoolRefusesToQuote(t *testing.T) {
	h := &Hook{}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: bignumber.TenPowInt(6)})
	assert.ErrorIs(t, err, ErrNotTracked)
	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountIn: bignumber.TenPowInt(6), AmountOut: bignumber.TenPowInt(6)})
	assert.ErrorIs(t, err, ErrNotTracked)
}

func TestExtraFrom(t *testing.T) {
	id := common.HexToHash("0x9aad681373e1ef45fe09c32db8acd2f395472e8c414bcf8f13d0adc7469911c4")
	vault := common.HexToAddress("0x1")
	policy := policyRaw{Destination: common.HexToAddress("0x2"), Tax: 100, Launched: uint64(launched)}
	launch := launchRaw{PoolId: id, Stage: stageTrades}

	assert.Equal(t, Extra{Live: true, QuoteIsCurrency0: true, TaxBps: 100, Launched: launched, BuybackVault: true},
		extraFrom(&policy, &launch, id, true, vault))
	assert.Equal(t, Extra{Live: true, TaxBps: 100, Launched: launched}, extraFrom(&policy, &launch, id, false, common.Address{}))

	for name, mutate := range map[string]func(p *policyRaw, l *launchRaw){
		"unregistered":         func(p *policyRaw, _ *launchRaw) { p.Destination = common.Address{} },
		"creation in progress": func(_ *policyRaw, l *launchRaw) { l.Stage = 2 },
		"no launch":            func(_ *policyRaw, l *launchRaw) { *l = launchRaw{} },
		"other pool":           func(_ *policyRaw, l *launchRaw) { l.PoolId = common.HexToHash("0x1") },
		"tax above register":   func(p *policyRaw, _ *launchRaw) { p.Tax = maxTaxBps + 1 },
	} {
		p, l := policy, launch
		mutate(&p, &l)
		assert.Equal(t, Extra{}, extraFrom(&p, &l, id, true, vault), name)
	}
}

// The factory loads a pool's extra only on the hook's own chain: the same address on another chain (addresses
// repeat across ape.fun's chains) or an unknown chain stays unpriceable. The address bits select beforeSwap,
// afterSwap and both return deltas (0x20cc, checked in the hook's constructor).
func TestFactory_ChainKeyedAndPermissions(t *testing.T) {
	extra := uniswapv4.HookExtra(`{"v":true,"q0":true,"t":100,"l":1791414828}`)
	for addr, d := range Deployments {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: d.ChainID}, HookExtra: extra})
		require.True(t, ok)
		h, isApe := hook.(*Hook)
		require.True(t, isApe)
		assert.Equal(t, Extra{Live: true, QuoteIsCurrency0: true, TaxBps: 100, Launched: launched}, h.Extra)
		assert.Equal(t, string(valueobject.ExchangeUniswapV4ApeFun), h.GetExchange())
		assert.True(t, h.CanBeforeSwap(addr))
		assert.True(t, h.CanAfterSwap(addr))
		clone, isApe := h.CloneState().(*Hook)
		require.True(t, isApe, "a clone keeps the apefun hook, not the embedded base")
		assert.Equal(t, h.Extra, clone.Extra)

		for _, cfg := range []*uniswapv4.Config{nil, {ChainID: valueobject.ChainIDEthereum}, {ChainID: otherChain(d.ChainID)}} {
			hook, _ = uniswapv4.GetHook(addr, &uniswapv4.HookParam{Cfg: cfg, HookExtra: extra})
			assert.False(t, hook.(*Hook).Live, "chain %v", cfg)
		}

		const (
			beforeInitialize      = 1 << 13
			beforeSwap            = 1 << 7
			afterSwap             = 1 << 6
			beforeSwapReturnDelta = 1 << 3
			afterSwapReturnDelta  = 1 << 2
		)
		flags := (uint16(addr[18])<<8 | uint16(addr[19])) & 0x3fff
		assert.Equal(t, uint16(beforeInitialize|beforeSwap|afterSwap|beforeSwapReturnDelta|afterSwapReturnDelta), flags)
	}
	assert.Len(t, HookAddresses, len(Deployments))
}

func otherChain(c valueobject.ChainID) valueobject.ChainID {
	if c == valueobject.ChainIDArc {
		return valueobject.ChainIDRobinhood
	}
	return valueobject.ChainIDArc
}
