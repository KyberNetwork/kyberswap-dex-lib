package alphix

import (
	"errors"
	"math/big"
	"testing"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// The reference below is Uniswap v3-core's SqrtPriceMath, SwapMath.computeSwapStep and v3-periphery's
// LiquidityAmounts, written in math/big from the Solidity, apart from the hook and from uniswapv3-sdk-uint256.

var (
	refQ96  = new(big.Int).Lsh(big.NewInt(1), 96)
	ref2256 = new(big.Int).Lsh(big.NewInt(1), 256)
	ref2160 = new(big.Int).Lsh(big.NewInt(1), 160)
	pipsMax = big.NewInt(1_000_000)
)

func bMulDiv(a, b, c *big.Int) *big.Int { return new(big.Int).Div(new(big.Int).Mul(a, b), c) }

func bMulDivUp(a, b, c *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(new(big.Int).Mul(a, b), c, new(big.Int))
	if r.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	return q
}

func bDivUp(a, b *big.Int) *big.Int { return bMulDivUp(a, big.NewInt(1), b) }

func sorted(a, b *big.Int) (*big.Int, *big.Int) {
	if a.Cmp(b) > 0 {
		return b, a
	}
	return a, b
}

// SqrtPriceMath.getAmount0Delta
func amount0Delta(a, b, l *big.Int, up bool) *big.Int {
	a, b = sorted(a, b)
	n1, n2 := new(big.Int).Lsh(l, 96), new(big.Int).Sub(b, a)
	if up {
		return bDivUp(bMulDivUp(n1, n2, b), a)
	}
	return new(big.Int).Div(bMulDiv(n1, n2, b), a)
}

// SqrtPriceMath.getAmount1Delta
func amount1Delta(a, b, l *big.Int, up bool) *big.Int {
	a, b = sorted(a, b)
	if up {
		return bMulDivUp(l, new(big.Int).Sub(b, a), refQ96)
	}
	return bMulDiv(l, new(big.Int).Sub(b, a), refQ96)
}

// SqrtPriceMath.getNextSqrtPriceFromAmount0RoundingUp
func nextFromAmount0(p, l, amt *big.Int, add bool) *big.Int {
	if amt.Sign() == 0 {
		return p
	}
	n1 := new(big.Int).Lsh(l, 96)
	product := new(big.Int).Mul(amt, p)
	if add {
		if den := new(big.Int).Add(n1, product); product.Cmp(ref2256) < 0 && den.Cmp(ref2256) < 0 {
			return bMulDivUp(n1, p, den)
		}
		return bDivUp(n1, new(big.Int).Add(new(big.Int).Div(n1, p), amt))
	}
	return bMulDivUp(n1, p, new(big.Int).Sub(n1, product))
}

// SqrtPriceMath.getNextSqrtPriceFromAmount1RoundingDown
func nextFromAmount1(p, l, amt *big.Int, add bool) *big.Int {
	if add {
		var q *big.Int
		if amt.Cmp(ref2160) < 0 {
			q = new(big.Int).Div(new(big.Int).Lsh(amt, 96), l)
		} else {
			q = bMulDiv(amt, refQ96, l)
		}
		return new(big.Int).Add(p, q)
	}
	var q *big.Int
	if amt.Cmp(ref2160) < 0 {
		q = bDivUp(new(big.Int).Lsh(amt, 96), l)
	} else {
		q = bMulDivUp(amt, refQ96, l)
	}
	return new(big.Int).Sub(p, q)
}

// SwapMath.computeSwapStep; amountRemaining >= 0 is exact in.
func refSwapStep(p, target, l, amountRemaining *big.Int, feePips int64) (next, in, out, fee *big.Int) {
	zfo := p.Cmp(target) >= 0
	exactIn := amountRemaining.Sign() >= 0
	fp := big.NewInt(feePips)
	rest := new(big.Int).Abs(amountRemaining)
	if exactIn {
		lessFee := bMulDiv(amountRemaining, new(big.Int).Sub(pipsMax, fp), pipsMax)
		if zfo {
			in = amount0Delta(target, p, l, true)
		} else {
			in = amount1Delta(p, target, l, true)
		}
		if lessFee.Cmp(in) >= 0 {
			next = target
		} else if zfo {
			next = nextFromAmount0(p, l, lessFee, true)
		} else {
			next = nextFromAmount1(p, l, lessFee, true)
		}
	} else {
		if zfo {
			out = amount1Delta(target, p, l, false)
		} else {
			out = amount0Delta(p, target, l, false)
		}
		if rest.Cmp(out) >= 0 {
			next = target
		} else if zfo {
			next = nextFromAmount1(p, l, rest, false)
		} else {
			next = nextFromAmount0(p, l, rest, false)
		}
	}
	reached := next.Cmp(target) == 0
	if zfo {
		if !reached || !exactIn {
			in = amount0Delta(next, p, l, true)
		}
		if !reached || exactIn {
			out = amount1Delta(next, p, l, false)
		}
	} else {
		if !reached || !exactIn {
			in = amount1Delta(p, next, l, true)
		}
		if !reached || exactIn {
			out = amount0Delta(p, next, l, false)
		}
	}
	if !exactIn && out.Cmp(rest) > 0 {
		out = rest
	}
	if exactIn && !reached {
		fee = new(big.Int).Sub(amountRemaining, in)
	} else {
		fee = bMulDivUp(in, fp, new(big.Int).Sub(pipsMax, fp))
	}
	return next, in, out, fee
}

// LiquidityAmounts.getLiquidityForAmounts
func refLiquidity(p, lower, upper, a0, a1 *big.Int) *big.Int {
	liq0 := func(a, b *big.Int) *big.Int { return bMulDiv(a0, bMulDiv(a, b, refQ96), new(big.Int).Sub(b, a)) }
	liq1 := func(a, b *big.Int) *big.Int { return bMulDiv(a1, refQ96, new(big.Int).Sub(b, a)) }
	switch {
	case p.Cmp(lower) <= 0:
		return liq0(lower, upper)
	case p.Cmp(upper) >= 0:
		return liq1(lower, upper)
	}
	l0, l1 := liq0(p, upper), liq1(lower, p)
	if l1.Cmp(l0) < 0 {
		return l1
	}
	return l0
}

type refJit struct {
	ds, du, next *big.Int
	gas          int64
	err          error
}

// refBeforeSwap is the Alphix hook's JIT step (AlphixHook._beforeSwap): it mints getLiquidityForAmounts of the
// yield sources' amounts on [tickLower, tickUpper] and swaps against it with one computeSwapStep toward the range
// edge in the swap direction; a price already at or past that edge swaps nothing. The hook pays the swap's input
// out of the pool manager's balance of the input token first, so an input above it fails. The range's sqrt
// prices are TickMath's (TestFactory_JitLiquidityMatchesLiquidityAmounts checks them against 1.0001^(tick/2)).
func refBeforeSwap(t *testing.T, h *Hook, zfo, exactIn bool, amt *big.Int) refJit {
	t.Helper()
	var lo, up uint256.Int
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(h.TickLower, &lo))
	require.NoError(t, v3Utils.GetSqrtRatioAtTickV2(h.TickUpper, &up))
	p, lower, upper := h.SqrtPriceX96.ToBig(), lo.ToBig(), up.ToBig()
	l := refLiquidity(p, lower, upper, h.Amount0Available.ToBig(), h.Amount1Available.ToBig())
	target, past := lower, p.Cmp(lower) <= 0
	if !zfo {
		target, past = upper, p.Cmp(upper) >= 0
	}
	if l.Sign() == 0 { // no position: only the dynamic fee
		return refJit{ds: new(big.Int), du: new(big.Int)}
	}
	if past { // a step of nothing, at the current price
		return refJit{ds: new(big.Int), du: new(big.Int), next: p, gas: 248527}
	}
	rem := new(big.Int).Set(amt)
	if !exactIn {
		rem.Neg(rem)
	}
	next, in, out, fee := refSwapStep(p, target, l, rem, int64(h.SwapFee))
	ds := new(big.Int).Add(in, fee)
	balance := h.PoolManagerBalances[0]
	if !zfo {
		balance = h.PoolManagerBalances[1]
	}
	if ds.Cmp(balance.ToBig()) > 0 {
		return refJit{err: uniswapv4.ErrInvalidAmountOut}
	}
	return refJit{ds: ds, du: new(big.Int).Neg(out), next: next, gas: 248527}
}

func refPrices() map[string]*uint256.Int {
	return map[string]*uint256.Int{
		"tick0":   q96(),
		"low":     new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(9997)), uint256.NewInt(10000)),
		"high":    new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(10003)), uint256.NewInt(10000)),
		"outside": new(uint256.Int).Mul(q96(), uint256.NewInt(2)),
		"below":   new(uint256.Int).Div(q96(), uint256.NewInt(2)),
	}
}

func checkJitAgainstRef(t *testing.T, h *Hook, zfo, exactIn bool, amt *big.Int) {
	t.Helper()
	want := refBeforeSwap(t, h, zfo, exactIn, amt)
	res, err := beforeSwap(h, zfo, exactIn, amt)
	msg := []any{"price %s zfo=%v exactIn=%v %s", h.SqrtPriceX96.Dec(), zfo, exactIn, amt}
	if want.err != nil {
		assert.Truef(t, errors.Is(err, want.err), "%v: err %v, want %v", msg, err, want.err)
		return
	}
	if !assert.NoError(t, err, msg...) {
		return
	}
	assert.Equal(t, uniswapv4.FeeAmount(3000), res.SwapFee, msg...)
	assert.Equal(t, want.ds.String(), res.DeltaSpecified.String(), msg...)
	assert.Equal(t, want.du.String(), res.DeltaUnspecified.String(), msg...)
	if want.next == nil {
		assert.Nil(t, res.SwapInfo, msg...)
		return
	}
	assert.Equal(t, want.gas, res.Gas, msg...)
	assert.Equal(t, want.next.String(), res.SwapInfo.(*uint256.Int).Dec(), msg...)
}

// TestBeforeSwap_MatchesReferenceModel runs the JIT fixture at five prices (on tick 0, inside the range either
// side, above and below it), both directions, exact in and exact out, from 1 wei to 1e15, against refBeforeSwap.
func TestBeforeSwap_MatchesReferenceModel(t *testing.T) {
	t.Parallel()
	amounts := []*big.Int{big.NewInt(1)}
	for e := 3; e <= 15; e += 2 {
		amounts = append(amounts, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil))
	}
	for _, price := range refPrices() {
		for _, zfo := range []bool{true, false} {
			for _, exactIn := range []bool{true, false} {
				for _, amt := range amounts {
					checkJitAgainstRef(t, jitHook(price), zfo, exactIn, amt)
				}
			}
		}
	}
}

// Spot values of the reference step, by hand from SwapMath: at P = Q96 with L = 1e12 and a 0.3% fee, 1000 token0
// in exact in leaves 997 after the fee, moves the price to ceil(L*Q96*P / (L*Q96 + 997*P)) and pays 996 token1 (one
// below 997 from the rounding down); the hook reports 1000 in and -996 out.
func TestReferenceModel_SpotStep(t *testing.T) {
	t.Parallel()
	l := big.NewInt(1e12)
	p := refQ96
	next, in, out, fee := refSwapStep(p, new(big.Int).Rsh(p, 1), l, big.NewInt(1000), 3000)
	want := bMulDivUp(new(big.Int).Lsh(l, 96), p, new(big.Int).Add(new(big.Int).Lsh(l, 96), new(big.Int).Mul(big.NewInt(997), p)))
	assert.Equal(t, want.String(), next.String())
	assert.Equal(t, "1000", new(big.Int).Add(in, fee).String())
	assert.Equal(t, "996", out.String())
}

// TestInvariant_JitCloneMonotonicBounded: UpdateBalance on a clone moves the clone's price and not the
// original's; exact-in output is non-decreasing in the input and never above the yield source's amount of the
// output token.
func TestInvariant_JitCloneMonotonicBounded(t *testing.T) {
	t.Parallel()
	for name, price := range refPrices() {
		h := jitHook(price)
		before := *h
		for _, zfo := range []bool{true, false} {
			res, err := beforeSwap(h, zfo, true, big.NewInt(1_000_000))
			require.NoError(t, err, name)
			clone := h.CloneState()
			clone.UpdateBalance(res.SwapInfo)
			assert.Equal(t, before, *h, "%s: UpdateBalance on the clone moved the original", name)

			prev := new(big.Int)
			for _, a := range []int64{1, 10, 1_000, 99_999, 1_000_000, 100_000_000, 999_999_999} {
				r, err := beforeSwap(h, zfo, true, big.NewInt(a))
				require.NoError(t, err)
				out := new(big.Int).Neg(r.DeltaUnspecified)
				assert.GreaterOrEqual(t, out.Cmp(prev), 0, "%s zfo=%v: not monotonic at %d", name, zfo, a)
				avail := h.Amount1Available
				if !zfo {
					avail = h.Amount0Available
				}
				assert.LessOrEqual(t, out.Cmp(avail.ToBig()), 0, "%s: out above the yield source", name)
				prev = out
			}
		}
	}
}

// FuzzBeforeSwap checks the hook against the reference model on random prices inside and around the range,
// directions, exactness and amounts.
func FuzzBeforeSwap(f *testing.F) {
	f.Add(uint64(10000), true, true, uint64(1000))
	f.Add(uint64(9997), false, true, uint64(1e9))
	f.Add(uint64(10003), true, false, uint64(1))
	f.Add(uint64(20000), false, false, uint64(1e15))
	f.Fuzz(func(t *testing.T, bps uint64, zfo, exactIn bool, amt uint64) {
		bps = 9_990 + bps%21 // price within ±0.1% of tick 0: in, at and past the [-8, 8] range
		if amt == 0 {
			amt = 1
		}
		price := new(uint256.Int).Div(new(uint256.Int).Mul(q96(), uint256.NewInt(bps)), uint256.NewInt(10000))
		checkJitAgainstRef(t, jitHook(price), zfo, exactIn, new(big.Int).SetUint64(amt))
	})
}
