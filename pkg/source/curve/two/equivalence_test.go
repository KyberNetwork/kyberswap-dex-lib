package two

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

// On-chain state of three Ethereum curve-two pools read at block 26188397 (the values the pool tracker stores),
// with get_dy results from the same block.
type onchainDy struct {
	i, j   int
	dx, dy string
}

type twoState struct {
	pool                                                     string
	block, timestamp                                         int64
	coins                                                    [2]string
	decimals                                                 [2]int64
	lpToken                                                  string
	balances                                                 [2]string
	A, D, gamma, feeGamma, midFee, outFee                    string
	futureAGamma, initialAGamma, xcpProfit, virtualPrice     string
	allowedExtraProfit, adjustmentStep, maHalfTime           string
	priceScale, priceOracle, lastPrices, lpSupply            string
	futureAGammaTime, initialAGammaTime, lastPricesTimestamp int64
	getDy                                                    []onchainDy
}

var (
	// CVX/ETH, CurveCryptoSwap2ETH: 0xb576491f1e6e5e62f1d8f26062ee822b40b0e0d4
	cvxETH = twoState{
		pool: "0xb576491f1e6e5e62f1d8f26062ee822b40b0e0d4", block: 26188397, timestamp: 1791952691,
		coins:    [2]string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x4e3fbd56cd56c3e72c1403e103b45db9da5b9d2b"},
		decimals: [2]int64{18, 18}, lpToken: "0x3a283d9c08e8b55966afb64c515f5143cf907611",
		balances:           [2]string{"1213926003558609983857", "1582278078919869531746642"},
		A:                  "400000",
		D:                  "2473891443596100529676",
		gamma:              "145000000000000",
		feeGamma:           "230000000000000",
		midFee:             "26000000",
		outFee:             "45000000",
		futureAGamma:       "136112946768375385385349842972852284582400000",
		initialAGamma:      "136112946768375385385349842972852284582400000",
		xcpProfit:          "1232511904047188462",
		virtualPrice:       "1116260734821589802",
		allowedExtraProfit: "2000000000000",
		adjustmentStep:     "146000000000000",
		maHalfTime:         "600",
		priceScale:         "796372805131279",
		priceOracle:        "779859124398978",
		lastPrices:         "779859076866219",
		lpSupply:           "39266915138093884385106",
		futureAGammaTime:   0, initialAGammaTime: 0, lastPricesTimestamp: 1791952679,
		getDy: []onchainDy{
			{0, 1, "1213926003558609", "1544961038298828400"},
			{1, 0, "1582278078919869531", "1233952607454755"},
			{0, 1, "12139260035586099", "15449478805538624105"},
			{1, 0, "15822780789198695317", "12339407838557035"},
			{0, 1, "121392600355860998", "154481647885117063924"},
			{1, 0, "158227807891986953174", "123382246769845212"},
			{0, 1, "1213926003558609983", "1543520789195791583321"},
			{1, 0, "1582278078919869531746", "1232631850402409412"},
			{0, 1, "12139260035586099838", "15331859948210450648747"},
			{1, 0, "15822780789198695317466", "12204011453163291830"},
			{0, 1, "121392600355860998385", "144515220649127717984766"},
			{1, 0, "158227807891986953174664", "111341310925026695694"},
		},
	}
	// T/ETH, crypto factory pool: 0x752ebeb79963cf0732e9c0fec72a49fd1defaeac
	tETH = twoState{
		pool: "0x752ebeb79963cf0732e9c0fec72a49fd1defaeac", block: 26188397, timestamp: 1791952691,
		coins:    [2]string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xcdf7028ceab81fa0c6971208e83fa7872994bee5"},
		decimals: [2]int64{18, 18}, lpToken: "0xcb08717451aae9ef950a2524e33b6dcaba60147b",
		balances:           [2]string{"102820238915372036552", "49295165570242258229298028"},
		A:                  "400000",
		D:                  "208222150230658987133",
		gamma:              "145000000000000",
		feeGamma:           "230000000000000",
		midFee:             "26000000",
		outFee:             "45000000",
		futureAGamma:       "136112946768375385385349842972852284582400000",
		initialAGamma:      "136112946768375385385349842972852284582400000",
		xcpProfit:          "1212817056019433730",
		virtualPrice:       "1106414915847837419",
		allowedExtraProfit: "2000000000000",
		adjustmentStep:     "146000000000000",
		maHalfTime:         "600",
		priceScale:         "2138229209617",
		priceOracle:        "2119094242354",
		lastPrices:         "2119094227997",
		lpSupply:           "64350478459486602888152",
		futureAGammaTime:   0, initialAGammaTime: 0, lastPricesTimestamp: 1791941903,
		getDy: []onchainDy{
			{0, 1, "102820238915372", "48235294480784807825"},
			{1, 0, "49295165570242258229", "104373990965996"},
			{0, 1, "1028202389153720", "482350404719587745801"},
			{1, 0, "492951655702422582292", "1043732906958764"},
			{0, 1, "10282023891537203", "4823251215725789066930"},
			{1, 0, "4929516557024225822929", "10436626589748990"},
			{0, 1, "102820238915372036", "48208414726144712807294"},
			{1, 0, "49295165570242258229298", "104293862016079653"},
			{0, 1, "1028202389153720365", "480719068087354120988132"},
			{1, 0, "492951655702422582292980", "1034160752208749027"},
			{0, 1, "10282023891537203655", "4511190232289908067892044"},
			{1, 0, "4929516557024225822929802", "9435841165831345125"},
		},
	}
	// USDC/EURS, 6 and 2 decimals: 0x98a7f18d4e56cfe84e3d081b40001b3d5bd3eb8b
	eursUSDC = twoState{
		pool: "0x98a7f18d4e56cfe84e3d081b40001b3d5bd3eb8b", block: 26188397, timestamp: 1791952691,
		coins:    [2]string{"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "0xdb25f211ab05b1c97d595516f45794528a807ad8"},
		decimals: [2]int64{6, 2}, lpToken: "0x3d229e1b4faab62f621ef2f6a610961f7bd7b23b",
		balances:           [2]string{"1943402316156", "208398055"},
		A:                  "20000000",
		D:                  "4443113759734943250042194",
		gamma:              "1000000000000000",
		feeGamma:           "5000000000",
		midFee:             "3000000",
		outFee:             "45000000",
		futureAGamma:       "6805647338418769269267492148636364229120000000",
		initialAGamma:      "6805647338418769269267492148635464229120000000",
		xcpProfit:          "1020579340164674926",
		virtualPrice:       "1010298993353292894",
		allowedExtraProfit: "10000000000",
		adjustmentStep:     "300000000000000",
		maHalfTime:         "2000",
		priceScale:         "1202412830544881456",
		priceOracle:        "1095090085021067714",
		lastPrices:         "1094186769010889337",
		lpSupply:           "2005306289975023031336447",
		futureAGammaTime:   0, initialAGammaTime: 1689276695, lastPricesTimestamp: 1791932015,
		getDy: []onchainDy{
			{0, 1, "1943402", "177"},
			{1, 0, "208", "2265662"},
			{0, 1, "19434023", "1769"},
			{1, 0, "2083", "22689067"},
			{0, 1, "194340231", "17680"},
			{1, 0, "20839", "226965517"},
			{0, 1, "1943402316", "176642"},
			{1, 0, "208398", "2267420684"},
			{0, 1, "19434023161", "1751537"},
			{1, 0, "2083980", "22440110043"},
			{0, 1, "194340231615", "16592762"},
			{1, 0, "20839805", "202077307491"},
		},
	}
)

var allStates = map[string]twoState{"cvxETH": cvxETH, "tETH": tETH, "eursUSDC": eursUSDC}

func (s twoState) entityPool() entity.Pool {
	extra := fmt.Sprintf(`{"A":%q,"D":%q,"gamma":%q,"priceScale":%q,"lastPrices":%q,"priceOracle":%q,"feeGamma":%q,`+
		`"midFee":%q,"outFee":%q,"futureAGammaTime":%d,"futureAGamma":%q,"initialAGammaTime":%d,"initialAGamma":%q,`+
		`"lastPricesTimestamp":%d,"lpSupply":%q,"xcpProfit":%q,"virtualPrice":%q,"allowedExtraProfit":%q,`+
		`"adjustmentStep":%q,"maHalfTime":%q}`,
		s.A, s.D, s.gamma, s.priceScale, s.lastPrices, s.priceOracle, s.feeGamma, s.midFee, s.outFee,
		s.futureAGammaTime, s.futureAGamma, s.initialAGammaTime, s.initialAGamma, s.lastPricesTimestamp, s.lpSupply,
		s.xcpProfit, s.virtualPrice, s.allowedExtraProfit, s.adjustmentStep, s.maHalfTime)
	precision := func(decimals int64) string {
		return new(big.Int).Exp(big.NewInt(10), big.NewInt(18-decimals), nil).String()
	}
	return entity.Pool{
		Address:  s.pool,
		Exchange: "curve",
		Type:     "curve-two",
		Reserves: entity.PoolReserves{s.balances[0], s.balances[1]},
		Tokens: []*entity.PoolToken{
			{Address: s.coins[0], Decimals: uint8(s.decimals[0]), Swappable: true},
			{Address: s.coins[1], Decimals: uint8(s.decimals[1]), Swappable: true},
		},
		Extra: extra,
		StaticExtra: fmt.Sprintf(`{"lpToken":%q,"precisionMultipliers":[%q,%q]}`, s.lpToken,
			precision(s.decimals[0]), precision(s.decimals[1])),
	}
}

// pinClock makes quotes and UpdateBalance use *now as the block timestamp for the rest of the (sequential) test,
// which advances time by updating *now.
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

// requireSameState checks every field that the contract state (and therefore later quotes) depends on.
func requireSameState(t *testing.T, msg string, old *legacyPoolSimulator, nw *PoolSimulator) {
	t.Helper()
	for k := range nCoins {
		require.Equal(t, old.Info.Reserves[k].String(), nw.Info.Reserves[k].String(), "%s: balance %d", msg, k)
	}
	require.Equal(t, old.D.String(), nw.D.Dec(), "%s: D", msg)
	require.Equal(t, old.PriceScalePacked.String(), nw.PriceScale.Dec(), "%s: price_scale", msg)
	require.Equal(t, old.PriceOraclePacked.String(), nw.PriceOracle.Dec(), "%s: price_oracle", msg)
	require.Equal(t, old.LastPricesPacked.String(), nw.LastPrices.Dec(), "%s: last_prices", msg)
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

// TestMatchesOnchainGetDy pins the simulator to the contracts' get_dy at the same block. The older pools (CVX/ETH,
// USDC/EURS) match to the wei. The factory pool (T/ETH) runs a newer contract with 2-coin specific newton_y /
// geometric_mean rounding; the simulator keeps the previous implementation's generic N-coin math there, which is
// within 1e-11 of the chain, and must stay identical to the previous implementation.
func TestMatchesOnchainGetDy(t *testing.T) {
	now := int64(cvxETH.timestamp)
	pinClock(t, &now)
	for name, s := range allStates {
		old, nw := newPair(t, s.entityPool())
		exact := 0
		for _, q := range s.getDy {
			dx, _ := new(big.Int).SetString(q.dx, 10)
			res, err := nw.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: s.coins[q.i], Amount: dx}, TokenOut: s.coins[q.j]})
			require.NoError(t, err, "%s %d->%d dx=%s", name, q.i, q.j, q.dx)
			got := res.TokenAmountOut.Amount
			legacyDy, _, err := old.GetDy(q.i, q.j, dx)
			require.NoError(t, err)
			require.Equal(t, legacyDy.String(), got.String(), "%s %d->%d dx=%s: new vs previous implementation", name,
				q.i, q.j, q.dx)
			want, _ := new(big.Int).SetString(q.dy, 10)
			if name != "tETH" {
				require.Equal(t, q.dy, got.String(), "%s %d->%d dx=%s: on-chain get_dy", name, q.i, q.j, q.dx)
			}
			diff := new(big.Int).Abs(new(big.Int).Sub(got, want))
			require.LessOrEqual(t, new(big.Int).Mul(diff, big.NewInt(1e11)).Cmp(want), 0,
				"%s %d->%d dx=%s: on-chain %s, simulated %s", name, q.i, q.j, q.dx, q.dy, got)
			if diff.Sign() == 0 {
				exact++
			}
		}
		t.Logf("%s: %d/%d quotes equal on-chain get_dy", name, exact, len(s.getDy))
	}
}

// TestEquivalence_CalcAmountOut compares quotes of both implementations over a grid of amounts (from 1 wei to
// twice the reserve, where quotes start failing) and random amounts, for both directions of every pool.
func TestEquivalence_CalcAmountOut(t *testing.T) {
	now := int64(cvxETH.timestamp)
	pinClock(t, &now)
	fracs := []float64{1e-15, 1e-12, 1e-9, 1e-6, 1e-4, 1e-3, 1e-2, 0.05, 0.1, 0.25, 0.5, 0.75, 1, 2}
	rng := rand.New(rand.NewPCG(1, 2))
	var ok, failed int
	for name, s := range allStates {
		old, nw := newPair(t, s.entityPool())
		for i := range nCoins {
			j := 1 - i
			reserve := nw.Info.Reserves[i]
			amounts := []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(10), big.NewInt(1000)}
			for _, f := range fracs {
				a, _ := new(big.Float).Mul(new(big.Float).SetInt(reserve), big.NewFloat(f)).Int(nil)
				amounts = append(amounts, a)
			}
			for range 500 {
				amounts = append(amounts, randomAmount(rng, reserve, 1.5))
			}
			for _, amt := range amounts {
				msg := fmt.Sprintf("%s %d->%d dx=%s", name, i, j, amt)
				_, success := requireSameQuote(t, msg, old, nw, pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: s.coins[i], Amount: amt},
					TokenOut:      s.coins[j],
				})
				if success {
					ok++
				} else {
					failed++
				}
			}
		}
	}
	// Both outcomes must be exercised, otherwise the comparison proves nothing.
	require.Greater(t, ok, 2000, "successful quotes compared")
	require.Greater(t, failed, 10, "failing quotes compared")
}

// TestEquivalence_SwapSequences runs random swap sequences (quote, then UpdateBalance with the quoted amount, as
// the router does) on both implementations with an advancing block timestamp, comparing every quote and the full
// pool state after every swap. It asserts that the state-changing branches of tweak_price were all exercised.
func TestEquivalence_SwapSequences(t *testing.T) {
	var now int64
	pinClock(t, &now)
	var swaps, priceScaleMoves, notAdjustedSet, noPriceSwaps, sameTimestamp int
	for name, s := range allStates {
		for seed := range uint64(8) {
			now = s.timestamp
			rng := rand.New(rand.NewPCG(seed, 7))
			old, nw := newPair(t, s.entityPool())
			for step := range 300 {
				dt := int64(rng.IntN(121))
				if dt == 0 {
					sameTimestamp++
				}
				now += dt

				i := rng.IntN(nCoins)
				j := 1 - i
				amt := randomAmount(rng, nw.Info.Reserves[i], 0.3)
				msg := fmt.Sprintf("%s seed %d step %d %d->%d dx=%s", name, seed, step, i, j, amt)
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.coins[i], Amount: amt},
					TokenOut: s.coins[j]}
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
	}
	t.Logf("swaps=%d price_scale moves=%d not_adjusted=%d no-price swaps=%d same-timestamp=%d",
		swaps, priceScaleMoves, notAdjustedSet, noPriceSwaps, sameTimestamp)
	require.Greater(t, swaps, 4000)
	require.Positive(t, priceScaleMoves, "price_scale adjustment branch not exercised")
	require.Positive(t, notAdjustedSet, "not_adjusted branch not exercised")
	require.Positive(t, noPriceSwaps, "p_i == 0 branch not exercised")
	require.Positive(t, sameTimestamp, "same-timestamp (no EMA update) branch not exercised")
}

// TestEquivalence_AGammaRamp runs quotes and swap sequences while an A/gamma ramp is in progress and past its end.
// Unlike tricrypto, curve-two quotes (get_dy) also use the interpolated A/gamma, and exchange() recomputes D and
// clears future_A_gamma_time (1 -> 0) once the ramp is over.
func TestEquivalence_AGammaRamp(t *testing.T) {
	start := cvxETH.timestamp
	now := start
	pinClock(t, &now)
	state := cvxETH
	// Ramp from A=200_000, gamma=1e14 (started 30 minutes ago) to the pool's current A/gamma (in 30 minutes).
	initial := new(big.Int).Or(new(big.Int).Lsh(big.NewInt(200_000), 128), big.NewInt(100_000_000_000_000))
	state.initialAGamma = initial.String()
	state.initialAGammaTime = start - 1800
	state.futureAGammaTime = start + 1800

	var duringRamp, afterRamp int
	for seed := range uint64(5) {
		rng := rand.New(rand.NewPCG(seed, 11))
		now = start
		old, nw := newPair(t, state.entityPool())
		for step := range 300 {
			now += 12
			i := rng.IntN(nCoins)
			j := 1 - i
			amt := randomAmount(rng, nw.Info.Reserves[i], 0.1)
			msg := fmt.Sprintf("ramp seed %d step %d %d->%d dx=%s", seed, step, i, j, amt)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: state.coins[i], Amount: amt},
				TokenOut: state.coins[j]}
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

// Before a scheduled A/gamma ramp starts, the contract's _A_gamma() underflows (block.timestamp - t0), so both
// get_dy and exchange revert; the simulator must fail instead of extrapolating A/gamma backwards.
func TestBeforeRampStartFails(t *testing.T) {
	now := cvxETH.timestamp
	pinClock(t, &now)
	state := cvxETH
	state.initialAGammaTime = now + 60
	state.futureAGammaTime = now + 3600
	p, err := NewPoolSimulator(state.entityPool())
	require.NoError(t, err)

	params := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: state.coins[0], Amount: big.NewInt(1e18)}, TokenOut: state.coins[1]}
	_, err = p.CalcAmountOut(params)
	require.ErrorIs(t, err, ErrUnderflow)
	_, err = p.Exchange(0, 1, big.NewInt(1e18))
	require.ErrorIs(t, err, ErrUnderflow)

	now = state.initialAGammaTime // once the ramp has started, the same swap succeeds
	_, err = p.CalcAmountOut(params)
	require.NoError(t, err)
	_, err = p.Exchange(0, 1, big.NewInt(1e18))
	require.NoError(t, err)
}

// Malformed pools and amounts are rejected with a specific error rather than simulated.
func TestRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	ep := cvxETH.entityPool()

	threeCoins := ep
	threeCoins.Tokens = append(append([]*entity.PoolToken{}, ep.Tokens...), &entity.PoolToken{Address: "0x01"})
	_, err := NewPoolSimulator(threeCoins)
	require.ErrorIs(t, err, errNotTwoCoins)

	missingReserve := ep
	missingReserve.Reserves = ep.Reserves[:1]
	_, err = NewPoolSimulator(missingReserve)
	require.ErrorIs(t, err, errMissingPoolData)

	p, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: cvxETH.coins[0], Amount: big.NewInt(-1)}, TokenOut: cvxETH.coins[1]})
	require.ErrorIs(t, err, errNegativeAmount)
	_, err = p.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: cvxETH.coins[0], Amount: new(big.Int).Lsh(big.NewInt(1), 256)},
		TokenOut:      cvxETH.coins[1]})
	require.ErrorIs(t, err, ErrOverflow)
	_, _, err = p.GetDy(0, 2, big.NewInt(1))
	require.ErrorIs(t, err, errTokenIndexOutOfRange)
}

// virtual_price * 2 - 10**18 is a checked subtraction in tweak_price: it reverts when virtual_price < 0.5e18,
// but only while not_adjusted is false (the contract short-circuits `not needs_adjustment and ...`).
func TestTweakPriceVirtualPriceBelowHalf(t *testing.T) {
	now := cvxETH.timestamp
	pinClock(t, &now)
	newTiny := func(notAdjusted bool) *PoolSimulator {
		p, err := NewPoolSimulator(cvxETH.entityPool())
		require.NoError(t, err)
		p.LpSupply = *uint256.MustFromDecimal("1000000000000000000000000000000") // 1e30: virtual_price ~ 1e9
		p.VirtualPrice.SetUint64(1)
		p.NotAdjusted = notAdjusted
		return p
	}
	_, err := newTiny(false).Exchange(0, 1, big.NewInt(1e18))
	require.ErrorIs(t, err, ErrUnderflow)
	_, err = newTiny(true).Exchange(0, 1, big.NewInt(1e18))
	require.NoError(t, err)
}

// A reverted exchange must not leave a half-applied swap (input added, output not removed, oracle moved).
func TestFailedExchangeLeavesPoolUnchanged(t *testing.T) {
	now := cvxETH.timestamp + 12
	pinClock(t, &now)
	p, err := NewPoolSimulator(cvxETH.entityPool())
	require.NoError(t, err)
	p.VirtualPrice.SetUint64(2e18) // the swap lowers virtual_price below this: "Loss" after balances/D/oracle changed
	before := p.CloneState().(*PoolSimulator)

	_, err = p.Exchange(0, 1, big.NewInt(1e18))
	require.ErrorIs(t, err, errLoss)
	require.Equal(t, before, p)

	p.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: cvxETH.coins[0], Amount: big.NewInt(1e18)},
		TokenAmountOut: pool.TokenAmount{Token: cvxETH.coins[1], Amount: big.NewInt(1)},
	})
	require.Equal(t, before, p)
}
