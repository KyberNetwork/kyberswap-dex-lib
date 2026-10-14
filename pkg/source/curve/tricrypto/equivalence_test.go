package tricrypto

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Curve tricrypto2 (USDT/WBTC/WETH, 0xD51a44d3FaE010294C616388b506AcdA1bfAAE46) storage read on-chain at two
// consecutive blocks. Block 26181453 contains exactly one swap on the pool (tx 0x8ebd6cb5...b196): 2134.233857
// USDT -> WETH at timestamp 1791868991.
const (
	tri2USDT = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	tri2WBTC = "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
	tri2WETH = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"

	tri2Block52Timestamp = 1791868775 // last_prices_timestamp at block 26181452
	tri2Block53Timestamp = 1791868991 // timestamp of block 26181453
)

type tri2State struct {
	balances                              [3]string
	D, xcpProfit, virtualPrice            string
	priceScale, priceOracle, lastPrices   [2]string
	lastPricesTimestamp, futureAGammaTime int64
	futureAGamma, initialAGamma           string
	initialAGammaTime                     int64
}

var (
	tri2AtBlock26181452 = tri2State{
		balances:            [3]string{"3947114941094", "4687873211", "1554787014549222229616"},
		D:                   "11722711763398766437016767",
		xcpProfit:           "1145801814313530617",
		virtualPrice:        "1072902665505676632",
		priceScale:          [2]string{"82724804211703587475975", "2506887709019354722973"},
		priceOracle:         [2]string{"83374526269551916167919", "2520068483911547726169"},
		lastPrices:          [2]string{"83557713878122097375341", "2526812438494472454161"},
		lastPricesTimestamp: tri2Block52Timestamp,
		futureAGamma:        "581076037942835227425498917514114728328226821",
		initialAGamma:       "183752478137306770270222288013175834186240000",
		initialAGammaTime:   1633548703,
	}
	tri2AtBlock26181453 = tri2State{
		balances:            [3]string{"3949249174951", "4687873211", "1553942938614753907878"},
		D:                   "11722713272985875482879069",
		xcpProfit:           "1145801961863660937",
		virtualPrice:        "1072902803668250769",
		priceScale:          [2]string{"82724804211703587475975", "2506887709019354722973"},
		priceOracle:         [2]string{"83414980580639546189580", "2521557787903165499647"},
		lastPrices:          [2]string{"83557713878122097375341", "2528485613494407747182"},
		lastPricesTimestamp: tri2Block53Timestamp,
		futureAGamma:        "581076037942835227425498917514114728328226821",
		initialAGamma:       "183752478137306770270222288013175834186240000",
		initialAGammaTime:   1633548703,
	}
)

func (s tri2State) entityPool() entity.Pool {
	extra := fmt.Sprintf(`{"A":"1707629","D":%q,"gamma":"11809167828997","priceScale":[%q,%q,""],`+
		`"lastPrices":[%q,%q,""],"priceOracle":[%q,%q,""],"feeGamma":"500000000000000","midFee":"3000000",`+
		`"outFee":"30000000","futureAGammaTime":%d,"futureAGamma":%q,"initialAGammaTime":%d,"initialAGamma":%q,`+
		`"lastPricesTimestamp":%d,"lpSupply":"6153038484735248206702","xcpProfit":%q,"virtualPrice":%q,`+
		`"allowedExtraProfit":"2000000000000","adjustmentStep":"490000000000000","maHalfTime":"600"}`,
		s.D, s.priceScale[0], s.priceScale[1], s.lastPrices[0], s.lastPrices[1], s.priceOracle[0], s.priceOracle[1],
		s.futureAGammaTime, s.futureAGamma, s.initialAGammaTime, s.initialAGamma, s.lastPricesTimestamp,
		s.xcpProfit, s.virtualPrice)
	return entity.Pool{
		Address:  "0xd51a44d3fae010294c616388b506acda1bfaae46",
		Exchange: "curve",
		Type:     "curve-tricrypto",
		Reserves: entity.PoolReserves{s.balances[0], s.balances[1], s.balances[2]},
		Tokens: []*entity.PoolToken{
			{Address: tri2USDT, Decimals: 6, Swappable: true},
			{Address: tri2WBTC, Decimals: 8, Swappable: true},
			{Address: tri2WETH, Decimals: 18, Swappable: true},
		},
		Extra:       extra,
		StaticExtra: `{"lpToken":"0xc4ad29ba4b3c580e6d59105fff484999997675ff","precisionMultipliers":["1000000000000","10000000000","1"]}`,
	}
}

// pinClock makes UpdateBalance use *now as the block timestamp for the rest of the (sequential) test, which
// advances time by updating *now.
func pinClock(t testing.TB, now *int64) {
	t.Helper()
	prev := nowUnix
	nowUnix = func() int64 { return *now }
	t.Cleanup(func() { nowUnix = prev })
}

func newPair(t testing.TB, ep entity.Pool) (*legacyPoolSimulator, *PoolSimulator) {
	t.Helper()
	old, err := newLegacyPoolSimulator(ep)
	require.NoError(t, err)
	nw, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return old, nw
}

func legacyUnpack(p *big.Int, k uint) string {
	return new(big.Int).And(new(big.Int).Rsh(p, legacyPriceSize*k), legacyPriceMask).String()
}

// requireSameState checks every field that the contract state (and therefore later quotes) depends on.
func requireSameState(t *testing.T, msg string, old *legacyPoolSimulator, nw *PoolSimulator) {
	t.Helper()
	for k := range nCoins {
		require.Equal(t, old.Info.Reserves[k].String(), nw.Info.Reserves[k].String(), "%s: balance %d", msg, k)
	}
	require.Equal(t, old.D.String(), nw.D.Dec(), "%s: D", msg)
	for k := range nCoins - 1 {
		require.Equal(t, legacyUnpack(old.PriceScalePacked, uint(k)), nw.PriceScale[k].Dec(), "%s: price_scale %d", msg, k)
		require.Equal(t, legacyUnpack(old.PriceOraclePacked, uint(k)), nw.PriceOracle[k].Dec(), "%s: price_oracle %d", msg, k)
		require.Equal(t, legacyUnpack(old.LastPricesPacked, uint(k)), nw.LastPrices[k].Dec(), "%s: last_prices %d", msg, k)
	}
	require.Equal(t, old.LastPricesTimestamp, nw.LastPricesTimestamp, "%s: last_prices_timestamp", msg)
	require.Equal(t, old.XcpProfit.String(), nw.XcpProfit.Dec(), "%s: xcp_profit", msg)
	require.Equal(t, old.VirtualPrice.String(), nw.VirtualPrice.Dec(), "%s: virtual_price", msg)
	require.Equal(t, old.NotAdjusted, nw.NotAdjusted, "%s: not_adjusted", msg)
	require.Equal(t, old.FutureAGammaTime, nw.FutureAGammaTime, "%s: future_A_gamma_time", msg)
}

// requireSameQuote checks that both implementations agree on a quote: same amount, fee and gas, or both fail.
// It returns whether the quote succeeded.
func requireSameQuote(t *testing.T, msg string, old *legacyPoolSimulator, nw *PoolSimulator,
	params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, bool) {
	t.Helper()
	oldRes, oldErr := old.CalcAmountOut(params)
	newRes, newErr := nw.CalcAmountOut(params)
	require.Equal(t, oldErr == nil, newErr == nil, "%s: error mismatch: legacy=%v new=%v", msg, oldErr, newErr)
	if oldErr != nil {
		return nil, false
	}
	require.Equal(t, oldRes.TokenAmountOut.Amount.String(), newRes.TokenAmountOut.Amount.String(), "%s: amount out", msg)
	require.Equal(t, oldRes.TokenAmountOut.Token, newRes.TokenAmountOut.Token, "%s: token out", msg)
	require.Equal(t, oldRes.Fee.Amount.String(), newRes.Fee.Amount.String(), "%s: fee", msg)
	require.Equal(t, oldRes.Gas, newRes.Gas, "%s: gas", msg)
	return newRes, true
}

// randomAmount draws an amount log-uniformly between 1e-12 and maxFrac of the reserve.
func randomAmount(rng *rand.Rand, reserve *big.Int, maxFrac float64) *big.Int {
	exp := -12 + rng.Float64()*(12+math.Log10(maxFrac))
	f := new(big.Float).Mul(new(big.Float).SetInt(reserve), big.NewFloat(math.Pow(10, exp)))
	amt, _ := f.Int(nil)
	if amt.Sign() == 0 {
		amt.SetInt64(1)
	}
	return amt
}

// TestMatchesOnchainGetDy pins the new implementation to on-chain CurveCryptoSwap.get_dy results.
func TestMatchesOnchainGetDy(t *testing.T) {
	t.Parallel()
	tokens := []string{tri2USDT, tri2WBTC, tri2WETH}
	cases := []struct {
		state  tri2State
		i, j   int
		dx, dy string
	}{
		{tri2AtBlock26181452, 2, 0, "1000000000000000000", "2522518214"},
		{tri2AtBlock26181452, 2, 0, "4000000000000000000", "10076058580"},
		{tri2AtBlock26181452, 2, 0, "26500000000000000000", "66178118182"},
		{tri2AtBlock26181452, 0, 2, "2134233857", "844075934468321738"}, // the swap executed in block 26181453
		{tri2AtBlock26181452, 0, 2, "100000000000", "38528654097308071648"},
		{tri2AtBlock26181453, 2, 0, "1000000000000000000", "2524755889"},
		{tri2AtBlock26181453, 2, 0, "4000000000000000000", "10084225374"},
		{tri2AtBlock26181453, 2, 0, "26500000000000000000", "66228690602"},
		{tri2AtBlock26181453, 0, 2, "2134233857", "843246389517857108"},
		{tri2AtBlock26181453, 0, 2, "100000000000", "38485950028925892794"},
	}
	for _, tc := range cases {
		p, err := NewPoolSimulator(tc.state.entityPool())
		require.NoError(t, err)
		dx, _ := new(big.Int).SetString(tc.dx, 10)
		res, err := p.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokens[tc.i], Amount: dx},
			TokenOut:      tokens[tc.j],
		})
		require.NoError(t, err)
		require.Equal(t, tc.dy, res.TokenAmountOut.Amount.String(), "%d->%d dx=%s", tc.i, tc.j, tc.dx)
	}
}

// TestExchangeReproducesNextBlock applies the only swap of block 26181453 to the block 26181452 state and
// expects exactly the on-chain storage of block 26181453: balances, D, price oracle, last prices, xcp profit and
// virtual price all come out of exchange() + tweak_price().
func TestExchangeReproducesNextBlock(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	old, nw := newPair(t, tri2AtBlock26181452.entityPool())
	dx := big.NewInt(2134233857)

	dy, err := nw.Exchange(0, 2, dx)
	require.NoError(t, err)
	require.Equal(t, "844075934468321738", dy.String(), "amount received on-chain")
	_, err = old.Exchange(0, 2, dx)
	require.NoError(t, err)

	want, err := NewPoolSimulator(tri2AtBlock26181453.entityPool())
	require.NoError(t, err)
	for k := range nCoins {
		require.Equal(t, want.Info.Reserves[k].String(), nw.Info.Reserves[k].String(), "balance %d", k)
	}
	require.Equal(t, want.D, nw.D, "D")
	require.Equal(t, want.PriceScale, nw.PriceScale, "price_scale")
	require.Equal(t, want.PriceOracle, nw.PriceOracle, "price_oracle")
	require.Equal(t, want.LastPrices, nw.LastPrices, "last_prices")
	require.Equal(t, want.LastPricesTimestamp, nw.LastPricesTimestamp, "last_prices_timestamp")
	require.Equal(t, want.XcpProfit, nw.XcpProfit, "xcp_profit")
	require.Equal(t, want.VirtualPrice, nw.VirtualPrice, "virtual_price")

	requireSameState(t, "legacy vs new after the block 26181453 swap", old, nw)
}

// TestEquivalence_CalcAmountOut compares quotes of both implementations over a grid of amounts (from 1 wei to
// twice the reserve, where quotes start failing) and random amounts, for all 6 directions and several states.
func TestEquivalence_CalcAmountOut(t *testing.T) {
	t.Parallel()
	states := map[string]entity.Pool{
		"tricrypto2@26181452": tri2AtBlock26181452.entityPool(),
		"tricrypto2@26181453": tri2AtBlock26181453.entityPool(),
	}
	tokens := []string{tri2USDT, tri2WBTC, tri2WETH}
	fracs := []float64{1e-15, 1e-12, 1e-9, 1e-6, 1e-4, 1e-3, 1e-2, 0.05, 0.1, 0.25, 0.5, 0.75, 1, 2}
	rng := rand.New(rand.NewPCG(1, 2))
	var ok, failed int
	for name, ep := range states {
		old, nw := newPair(t, ep)
		for i := range nCoins {
			for j := range nCoins {
				if i == j {
					continue
				}
				reserve := nw.Info.Reserves[i]
				amounts := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(10), big.NewInt(1000)}
				for _, f := range fracs {
					a, _ := new(big.Float).Mul(new(big.Float).SetInt(reserve), big.NewFloat(f)).Int(nil)
					amounts = append(amounts, a)
				}
				for range 300 {
					amounts = append(amounts, randomAmount(rng, reserve, 1.5))
				}
				for _, amt := range amounts {
					msg := fmt.Sprintf("%s %d->%d dx=%s", name, i, j, amt)
					_, success := requireSameQuote(t, msg, old, nw, pool.CalcAmountOutParams{
						TokenAmountIn: pool.TokenAmount{Token: tokens[i], Amount: amt},
						TokenOut:      tokens[j],
					})
					if success {
						ok++
					} else {
						failed++
					}
				}
			}
		}
	}
	// Both outcomes must be exercised, otherwise the comparison proves nothing.
	require.Greater(t, ok, 3000, "successful quotes compared")
	require.Greater(t, failed, 10, "failing quotes compared")
}

// TestEquivalence_SwapSequences runs random swap sequences (quote, then UpdateBalance with the quoted amount, as
// the router does) on both implementations with an advancing block timestamp, comparing every quote and the full
// pool state after every swap. It asserts that the state-changing branches of tweak_price were all exercised.
func TestEquivalence_SwapSequences(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	tokens := []string{tri2USDT, tri2WBTC, tri2WETH}
	var swaps, priceScaleMoves, notAdjustedSet, noPriceSwaps, sameTimestamp int
	for seed := range uint64(20) {
		rng := rand.New(rand.NewPCG(seed, 7))
		old, nw := newPair(t, tri2AtBlock26181453.entityPool())
		for step := range 300 {
			dt := int64(rng.IntN(121))
			if dt == 0 {
				sameTimestamp++
			}
			now += dt

			i := rng.IntN(nCoins)
			j := (i + 1 + rng.IntN(nCoins-1)) % nCoins
			amt := randomAmount(rng, nw.Info.Reserves[i], 0.3)
			msg := fmt.Sprintf("seed %d step %d %d->%d dx=%s", seed, step, i, j, amt)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[i], Amount: amt},
				TokenOut: tokens[j]}
			res, success := requireSameQuote(t, msg, old, nw, params)
			if !success {
				continue
			}
			before := nw.PriceScale
			update := pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
				Fee: *res.Fee}
			old.UpdateBalance(update)
			nw.UpdateBalance(update)
			requireSameState(t, msg, old, nw)

			swaps++
			if nw.PriceScale != before {
				priceScaleMoves++
			}
			if nw.NotAdjusted {
				notAdjustedSet++
			}
			if amt.Cmp(big.NewInt(100000)) <= 0 || res.TokenAmountOut.Amount.Cmp(big.NewInt(100000)) <= 0 {
				noPriceSwaps++ // p_i == 0: last prices re-derived with newton_y
			}
		}
	}
	t.Logf("swaps=%d price_scale moves=%d not_adjusted=%d no-price swaps=%d same-timestamp=%d",
		swaps, priceScaleMoves, notAdjustedSet, noPriceSwaps, sameTimestamp)
	require.Greater(t, swaps, 4000)
	require.Positive(t, priceScaleMoves, "price_scale adjustment branch not exercised")
	require.Positive(t, notAdjustedSet, "not_adjusted branch not exercised")
	require.Positive(t, noPriceSwaps, "p_i == 0 branch not exercised")
	require.Positive(t, sameTimestamp, "same-timestamp (no EMA update) branch not exercised")
}

// TestEquivalence_AGammaRamp runs swap sequences while an A/gamma ramp is in progress and past its end, so that
// exchange() recomputes D with the interpolated A/gamma and future_A_gamma_time goes 1 -> 0.
func TestEquivalence_AGammaRamp(t *testing.T) {
	start := int64(tri2Block53Timestamp)
	now := start
	pinClock(t, &now)
	state := tri2AtBlock26181453
	// Ramp from A=1_000_000, gamma=2e13 (started 30 minutes ago) to the pool's current A/gamma (in 30 minutes).
	initial := new(big.Int).Or(new(big.Int).Lsh(big.NewInt(1_000_000), 128), big.NewInt(20_000_000_000_000))
	state.initialAGamma = initial.String()
	state.initialAGammaTime = start - 1800
	state.futureAGammaTime = start + 1800
	tokens := []string{tri2USDT, tri2WBTC, tri2WETH}

	var duringRamp, afterRamp int
	for seed := range uint64(5) {
		rng := rand.New(rand.NewPCG(seed, 11))
		now = start
		old, nw := newPair(t, state.entityPool())
		for step := range 300 {
			now += 12
			i := rng.IntN(nCoins)
			j := (i + 1 + rng.IntN(nCoins-1)) % nCoins
			amt := randomAmount(rng, nw.Info.Reserves[i], 0.1)
			msg := fmt.Sprintf("ramp seed %d step %d %d->%d dx=%s", seed, step, i, j, amt)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[i], Amount: amt},
				TokenOut: tokens[j]}
			res, success := requireSameQuote(t, msg, old, nw, params)
			if !success {
				continue
			}
			update := pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *res.TokenAmountOut,
				Fee: *res.Fee}
			old.UpdateBalance(update)
			nw.UpdateBalance(update)
			requireSameState(t, msg, old, nw)
			if now < state.futureAGammaTime {
				duringRamp++
			} else {
				afterRamp++
			}
		}
		require.Zero(t, nw.FutureAGammaTime, "ramp should be finished and cleared")
	}
	require.Positive(t, duringRamp)
	require.Positive(t, afterRamp)
}

// Before a scheduled A/gamma ramp starts, the contract's _A_gamma() underflows (block.timestamp - t0) and the swap
// reverts, so the simulator must fail the swap instead of extrapolating A/gamma backwards.
func TestExchangeBeforeRampStartFails(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	state := tri2AtBlock26181453
	state.initialAGammaTime = now + 60
	state.futureAGammaTime = now + 3600
	p, err := NewPoolSimulator(state.entityPool())
	require.NoError(t, err)

	_, err = p.Exchange(0, 2, big.NewInt(1_000_000_000))
	require.ErrorIs(t, err, ErrUnderflow)

	now = state.initialAGammaTime // once the ramp has started, the same swap succeeds
	_, err = p.Exchange(0, 2, big.NewInt(1_000_000_000))
	require.NoError(t, err)
}

// Malformed pools and amounts are rejected with a specific error rather than simulated.
func TestRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	ep := tri2AtBlock26181452.entityPool()

	twoCoins := ep
	twoCoins.Tokens = ep.Tokens[:2]
	_, err := NewPoolSimulator(twoCoins)
	require.ErrorIs(t, err, errNotThreeCoins)

	missingReserve := ep
	missingReserve.Reserves = ep.Reserves[:2]
	_, err = NewPoolSimulator(missingReserve)
	require.ErrorIs(t, err, errMissingPoolData)

	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tri2WETH, Amount: big.NewInt(-1)}, TokenOut: tri2USDT})
	require.ErrorIs(t, err, errNegativeAmount)
	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tri2WETH, Amount: new(big.Int).Lsh(big.NewInt(1), 256)}, TokenOut: tri2USDT})
	require.ErrorIs(t, err, ErrOverflow)
}

// tweak_price asserts every price it packs into storage is < PRICE_MASK (2**128-1) and reverts otherwise.
func TestExchangeRejectsPriceAtMask(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	dx := big.NewInt(1e18) // WETH -> USDT leaves last_prices[0] unchanged

	for _, tc := range []struct {
		lastPrice uint256.Int
		wantErr   error
	}{
		{*new(uint256.Int).SubUint64(PriceMask, 1), nil},
		{*PriceMask, errPriceTooLarge},
	} {
		p, err := NewPoolSimulator(tri2AtBlock26181452.entityPool())
		require.NoError(t, err)
		p.LastPrices[0] = tc.lastPrice
		_, err = p.Exchange(2, 0, dx)
		if tc.wantErr == nil {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, tc.wantErr)
		}
	}
}

// virtual_price * 2 - 10**18 is a checked subtraction in tweak_price: it reverts when virtual_price < 0.5e18,
// but only while not_adjusted is false (the contract short-circuits `not needs_adjustment and ...`).
func TestTweakPriceVirtualPriceBelowHalf(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	newTiny := func(notAdjusted bool) *PoolSimulator {
		p, err := NewPoolSimulator(tri2AtBlock26181452.entityPool())
		require.NoError(t, err)
		p.LpSupply = *uint256.MustFromDecimal("100000000000000000000000000") // 1e26: virtual_price ~ 0.04e18
		p.VirtualPrice.SetUint64(1)
		p.NotAdjusted = notAdjusted
		return p
	}
	_, err := newTiny(false).Exchange(0, 2, big.NewInt(1_000_000_000))
	require.ErrorIs(t, err, ErrUnderflow)
	_, err = newTiny(true).Exchange(0, 2, big.NewInt(1_000_000_000))
	require.NoError(t, err)
}

// A reverted exchange must not leave a half-applied swap (input added, output not removed, oracle moved).
func TestFailedExchangeLeavesPoolUnchanged(t *testing.T) {
	now := int64(tri2Block53Timestamp)
	pinClock(t, &now)
	p, err := NewPoolSimulator(tri2AtBlock26181452.entityPool())
	require.NoError(t, err)
	p.VirtualPrice.SetUint64(2e18) // the swap lowers virtual_price below this: "Loss" after balances/D/oracle changed
	before := p.CloneState().(*PoolSimulator)

	_, err = p.Exchange(0, 2, big.NewInt(2134233857))
	require.ErrorIs(t, err, errLoss)
	require.Equal(t, before, p)

	p.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tri2USDT, Amount: big.NewInt(2134233857)},
		TokenAmountOut: pool.TokenAmount{Token: tri2WETH, Amount: big.NewInt(1)},
	})
	require.Equal(t, before, p)
}

// get_dy asserts i != j, both indexes < 3 and dx > 0, like exchange.
func TestGetDyRejectsInvalidSwap(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(tri2AtBlock26181452.entityPool())
	require.NoError(t, err)
	_, _, err = p.GetDy(1, 1, big.NewInt(1e6))
	require.ErrorIs(t, err, errSameCoin)
	_, _, err = p.GetDy(0, 3, big.NewInt(1e6))
	require.ErrorIs(t, err, errCoinIndexOutOfRange)
	_, _, err = p.GetDy(-1, 0, big.NewInt(1e6))
	require.ErrorIs(t, err, errCoinIndexOutOfRange)
	_, _, err = p.GetDy(0, 2, big.NewInt(0))
	require.ErrorIs(t, err, errZeroAmount)
}
