package uniswapv3

import (
	"math/bits"
	"strings"
	"testing"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// refPool is an independent transcription of UniswapV3Pool.swap over a real TickBitmap. It shares
// only SwapMath/TickMath with the code under test, so it can serve as the oracle for word traversal.
type refPool struct {
	fee          FeeAmount
	tickSpacing  int
	tick         int
	sqrtPrice    uint256.Int
	liquidity    uint256.Int
	bitmap       map[int]*uint256.Int
	liquidityNet map[int]*int256.Int
}

func newRefPool(p *Pool) *refPool {
	r := &refPool{
		fee:          p.Fee,
		tickSpacing:  p.TickSpacing,
		tick:         p.TickCurrent,
		sqrtPrice:    p.SqrtRatioX96,
		liquidity:    p.Liquidity,
		bitmap:       map[int]*uint256.Int{},
		liquidityNet: map[int]*int256.Int{},
	}
	var bit uint256.Int
	for _, t := range p.Ticks {
		if t.LiquidityGross.IsZero() {
			continue
		}
		compressed := r.compress(t.Index)
		word := r.bitmap[compressed>>8]
		if word == nil {
			word = new(uint256.Int)
			r.bitmap[compressed>>8] = word
		}
		word.Or(word, bit.Lsh(uint256.NewInt(1), uint(compressed&255)))
		r.liquidityNet[t.Index] = t.LiquidityNet
	}
	return r
}

// compress floors like Solidity's `tick / tickSpacing; if (tick < 0 && tick % tickSpacing != 0) compressed--`.
func (r *refPool) compress(tick int) int {
	compressed := tick / r.tickSpacing
	if tick < 0 && tick%r.tickSpacing != 0 {
		compressed--
	}
	return compressed
}

// nextInitializedTickWithinOneWord is TickBitmap.nextInitializedTickWithinOneWord verbatim.
func (r *refPool) nextInitializedTickWithinOneWord(tick int, lte bool) (int, bool) {
	one := uint256.NewInt(1)
	compressed := r.compress(tick)
	var mask, masked uint256.Int
	if lte {
		bitPos := compressed & 255
		mask.Lsh(one, uint(bitPos)).Sub(&mask, one).Add(&mask, masked.Lsh(one, uint(bitPos)))
		masked.Clear()
		if word := r.bitmap[compressed>>8]; word != nil {
			masked.And(word, &mask)
		}
		if !masked.IsZero() {
			return (compressed - (bitPos - (masked.BitLen() - 1))) * r.tickSpacing, true
		}
		return (compressed - bitPos) * r.tickSpacing, false
	}
	compressed++
	bitPos := compressed & 255
	mask.Lsh(one, uint(bitPos)).Sub(&mask, one).Not(&mask)
	if word := r.bitmap[compressed>>8]; word != nil {
		masked.And(word, &mask)
	}
	if !masked.IsZero() {
		return (compressed + (lsb(&masked) - bitPos)) * r.tickSpacing, true
	}
	return (compressed + (255 - bitPos)) * r.tickSpacing, false
}

func lsb(x *uint256.Int) int {
	for i, limb := range x {
		if limb != 0 {
			return i*64 + bits.TrailingZeros64(limb)
		}
	}
	return 256
}

// swap mirrors the while-loop of UniswapV3Pool.swap with the default price limit.
func (r *refPool) swap(zeroForOne bool, amountSpecified uint256.Int) (SwapResult, error) {
	limit := *MaxSqrtRatioU256M1
	if zeroForOne {
		limit = *MinSqrtRatioU256P1
	}
	exactIn := amountSpecified.Sign() >= 0
	remaining, sqrtPrice, tick, liquidity := amountSpecified, r.sqrtPrice, r.tick, r.liquidity
	var calculated uint256.Int
	var crossed, emptyWords int
	for !remaining.IsZero() && !sqrtPrice.Eq(&limit) {
		tickNext, initialized := r.nextInitializedTickWithinOneWord(tick, zeroForOne)
		tickNext = min(max(tickNext, MinTick), MaxTick)

		var sqrtNext, target, next, in, out, fee uint256.Int
		if err := GetSqrtRatioAtTick(tickNext, &sqrtNext); err != nil {
			return SwapResult{}, err
		}
		if zeroForOne && sqrtNext.Lt(&limit) || !zeroForOne && sqrtNext.Gt(&limit) {
			target = limit
		} else {
			target = sqrtNext
		}
		if err := ComputeSwapStep(&sqrtPrice, &target, &liquidity, &remaining, r.fee, &next, &in, &out,
			&fee); err != nil {
			return SwapResult{}, err
		}

		in.Add(&in, &fee)
		if exactIn {
			remaining.Sub(&remaining, &in)
			calculated.Sub(&calculated, &out)
		} else {
			remaining.Add(&remaining, &out)
			calculated.Add(&calculated, &in)
		}

		if next.Eq(&sqrtNext) {
			if initialized {
				if zeroForOne {
					liquidity.Sub(&liquidity, (*uint256.Int)(r.liquidityNet[tickNext]))
				} else {
					liquidity.Add(&liquidity, (*uint256.Int)(r.liquidityNet[tickNext]))
				}
				crossed++
			} else {
				emptyWords++
			}
			if zeroForOne {
				tick = tickNext - 1
			} else {
				tick = tickNext
			}
		} else if !next.Eq(&sqrtPrice) {
			var err error
			if tick, err = GetTickAtSqrtRatio(&next); err != nil {
				return SwapResult{}, err
			}
		}
		sqrtPrice = next
	}
	return SwapResult{
		AmountCalculated:    calculated,
		SqrtRatioX96:        sqrtPrice,
		Liquidity:           liquidity,
		RemainingAmountIn:   remaining,
		CurrentTick:         tick,
		CrossInitTickLoops:  crossed,
		CrossEmptyWordLoops: emptyWords,
	}, nil
}

func fullRangeTicks(spacing int, liquidity string, extra ...TickU256) []TickU256 {
	edge := MaxTick / spacing * spacing
	ticks := []TickU256{{Index: -edge, LiquidityGross: uint256.MustFromDecimal(liquidity),
		LiquidityNet: int256.MustFromDec(liquidity)}}
	ticks = append(ticks, extra...)
	return append(ticks, TickU256{Index: edge, LiquidityGross: uint256.MustFromDecimal(liquidity),
		LiquidityNet: int256.MustFromDec("-" + liquidity)})
}

func sqrtAt(t *testing.T, tick int) uint256.Int {
	var s uint256.Int
	require.NoError(t, GetSqrtRatioAtTick(tick, &s))
	return s
}

// avaxDustPool is the Avalanche BTC.b/WETH.e 0.01% V4 pool at block 79724099: one full-range
// position of 4787383 liquidity, about 2 units of BTC.b and 1.05e13 of WETH.e. One bitmap word of
// price movement costs ~0.025 units of token0 but the contract charges 1 unit plus 1 unit of fee.
func avaxDustPool(t *testing.T, extra ...TickU256) *Pool {
	liquidity := uint256.NewInt(4787383)
	for _, e := range extra {
		if e.Index <= 291969 {
			liquidity.Add(liquidity, (*uint256.Int)(e.LiquidityNet))
		}
	}
	p, err := NewPool(100, *uint256.MustFromDecimal("173217865696950102619138552042419888"), *liquidity, 291969,
		fullRangeTicks(1, "4787383", extra...), 1)
	require.NoError(t, err)
	return p
}

func signed(amount string) uint256.Int {
	a := *uint256.MustFromDecimal(strings.TrimPrefix(amount, "-"))
	if strings.HasPrefix(amount, "-") {
		a.Neg(&a)
	}
	return a
}

// TestWordTraversalOnChain pins the oracle to a PoolManager.swap that actually ran, Tenderly
// simulation 9a8a0d70-28e2-4222-b54f-0d04392f167a: amount0 -19, amount1 1079518536551,
// sqrtPriceX96 155352516287591836139164025014435689, tick 289791. The approximation quoted
// 9331551613090 here — 8.6x the real output — and the route reverted on-chain.
func TestWordTraversalOnChain(t *testing.T) {
	t.Parallel()

	p := avaxDustPool(t)
	want, err := newRefPool(p).swap(true, signed("19"))
	require.NoError(t, err)
	require.Equal(t, "-1079518536551", signedDec(&want.AmountCalculated))
	require.Equal(t, "155352516287591836139164025014435689", want.SqrtRatioX96.Dec())
	require.Equal(t, 289791, want.CurrentTick)

	got, err := p.Swap(true, signed("19"), uint256.Int{})
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func signedDec(x *uint256.Int) string {
	if x.Sign() >= 0 {
		return x.Dec()
	}
	var n uint256.Int
	return "-" + n.Neg(x).Dec()
}

// TestWordTraversalMatchesContract covers shapes where the old single-jump approximation diverged
// from the on-chain loop; its quote is noted per case. In each, a run of empty bitmap words is cheap
// enough that the contract's per-word round-up of amountIn (plus fee on it) is material.
func TestWordTraversalMatchesContract(t *testing.T) {
	t.Parallel()

	dust := func(t *testing.T) *Pool { return avaxDustPool(t) }
	for _, tc := range []struct {
		name       string
		pool       func(t *testing.T) *Pool
		zeroForOne bool
		amount     string // positive: exact input, negative: exact output
		pinned     string
	}{
		// The dust exact-in values match #1591's separate transcription of the V4 loop; two
		// independent oracles agreeing rules out a shared transcription bug.
		{"dust exact-in 1 word", dust, true, "2", "-67380838795"},          // was -3281414713980
		{"dust exact-in 2 words", dust, true, "5", "-199637723176"},        // was -6763967348539
		{"dust exact-in 5 words", dust, true, "10", "-586401573504"},       // was -8418510341968
		{"dust exact-in 50 words", dust, true, "100", "-4912420720935"},    // was -10240240324215
		{"dust exact-in 394 words", dust, true, "1000", "-10399346476185"}, // was -10443844256719
		{"dust exact-out", dust, true, "-500000000000", "10"},              // was 4608

		// The step reaches a distant initialized tick (fully crossed), which is the case #1703's
		// first commit left on the old formula: the approximation takes 291969 -> 289000 in one step.
		{"dust fully crosses far initialized tick", func(t *testing.T) *Pool {
			return avaxDustPool(t,
				TickU256{Index: 289000, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(1)},
				TickU256{Index: 295000, LiquidityGross: uint256.NewInt(1), LiquidityNet: int256.NewInt(-1)})
		}, true, "100", "-4840872429419"}, // was -10236487160986

		// token1 is the dust side: one-for-zero, negative ticks, spacing 60 (words of 15360 ticks),
		// current tick not a multiple of the spacing so compression has to floor.
		{"spacing 60 negative tick one-for-zero", func(t *testing.T) *Pool {
			p, err := NewPool(3000, sqrtAt(t, -200030), *uint256.NewInt(1000), -200030,
				fullRangeTicks(60, "1000"), 60)
			require.NoError(t, err)
			return p
		}, false, "20", "-21897296"}, // was -21995942

		// Ordinary pool, one-for-zero exact-out: solidly-v3 AI/WETH, 10000 AI out. #1591 pins the
		// input independently; the approximation rounded amountIn up to a multiple of the words the
		// step spanned.
		{"solidly-v3 one-for-zero exact-out", func(t *testing.T) *Pool {
			ticks := []TickU256{
				{Index: -887200, LiquidityGross: uint256.MustFromDecimal("11420043566174051417"),
					LiquidityNet: int256.MustFromDec("11420043566174051417")},
				{Index: -191100, LiquidityGross: uint256.MustFromDecimal("84172845905035329535"),
					LiquidityNet: int256.MustFromDec("84172845905035329535")},
				{Index: -185000, LiquidityGross: uint256.MustFromDecimal("84172845905035329535"),
					LiquidityNet: int256.MustFromDec("-84172845905035329535")},
				{Index: 887200, LiquidityGross: uint256.MustFromDecimal("11420043566174051417"),
					LiquidityNet: int256.MustFromDec("-11420043566174051417")},
			}
			p, err := NewPool(10000, *uint256.MustFromDecimal("9374274824798812391640411"),
				*uint256.MustFromDecimal("11420043566174051417"), -180852, ticks, 100)
			require.NoError(t, err)
			return p
		}, false, "-10000000000000000000000", "157754838261358"}, // was 157754838261364
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			want, err := newRefPool(tc.pool(t)).swap(tc.zeroForOne, signed(tc.amount))
			require.NoError(t, err)
			require.Equal(t, tc.pinned, signedDec(&want.AmountCalculated))

			got, err := tc.pool(t).Swap(tc.zeroForOne, signed(tc.amount), uint256.Int{})
			require.NoError(t, err)
			require.Equal(t, want, got, "simulator must match the contract loop exactly")
		})
	}
}
