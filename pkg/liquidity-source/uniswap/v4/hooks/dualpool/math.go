package dualpool

import (
	"sort"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	q96         = new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	uMaxUint128 = new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 128), uint256.NewInt(1))
	uMaxSwapFee = uint256.NewInt(maxSwapFee)
)

// ---- v3-periphery LiquidityAmounts, exact port -------------------------------

func liquidityForAmount0(sqrtA, sqrtB, amount0 *uint256.Int) (*uint256.Int, error) {
	if sqrtA.Cmp(sqrtB) > 0 {
		sqrtA, sqrtB = sqrtB, sqrtA
	}
	intermediate, err := v3Utils.MulDiv(sqrtA, sqrtB, q96)
	if err != nil {
		return nil, err
	}
	var diff uint256.Int
	diff.Sub(sqrtB, sqrtA)
	liq, err := v3Utils.MulDiv(amount0, intermediate, &diff)
	if err != nil {
		return nil, err
	}
	return toUint128(liq)
}

func liquidityForAmount1(sqrtA, sqrtB, amount1 *uint256.Int) (*uint256.Int, error) {
	if sqrtA.Cmp(sqrtB) > 0 {
		sqrtA, sqrtB = sqrtB, sqrtA
	}
	var diff uint256.Int
	diff.Sub(sqrtB, sqrtA)
	liq, err := v3Utils.MulDiv(amount1, q96, &diff)
	if err != nil {
		return nil, err
	}
	return toUint128(liq)
}

func liquidityForAmounts(sqrtP, sqrtA, sqrtB, amount0, amount1 *uint256.Int) (*uint256.Int, error) {
	if sqrtA.Cmp(sqrtB) > 0 {
		sqrtA, sqrtB = sqrtB, sqrtA
	}
	switch {
	case sqrtP.Cmp(sqrtA) <= 0:
		return liquidityForAmount0(sqrtA, sqrtB, amount0)
	case sqrtP.Cmp(sqrtB) < 0:
		l0, err := liquidityForAmount0(sqrtP, sqrtB, amount0)
		if err != nil {
			return nil, err
		}
		l1, err := liquidityForAmount1(sqrtA, sqrtP, amount1)
		if err != nil {
			return nil, err
		}
		if l0.Cmp(l1) < 0 {
			return l0, nil
		}
		return l1, nil
	default:
		return liquidityForAmount1(sqrtA, sqrtB, amount1)
	}
}

func toUint128(x *uint256.Int) (*uint256.Int, error) {
	if x.Cmp(uMaxUint128) > 0 {
		return nil, ErrInsufficientLiquidity
	}
	return x, nil
}

// ---- Distribution.computeAllocations, per-bucket liquidity ---------------------

type position struct {
	tickLower, tickUpper int
	sqrtLower, sqrtUpper *uint256.Int
	liquidity            *uint256.Int
}

// allocate mirrors DualPool's computeAllocations: each bucket is sized from its
// weighted share of both balances with getLiquidityForAmounts at the current price.
func allocate(buckets []Bucket, sqrtP, bal0, bal1 *uint256.Int) ([]position, error) {
	out := make([]position, 0, len(buckets))
	weight, w0, w1 := new(uint256.Int), new(uint256.Int), new(uint256.Int)
	denom := uint256.NewInt(totalWeightBps)
	for _, b := range buckets {
		var sqrtLower, sqrtUpper uint256.Int
		if err := v3Utils.GetSqrtRatioAtTickV2(int(b.TickLower), &sqrtLower); err != nil {
			return nil, err
		}
		if err := v3Utils.GetSqrtRatioAtTickV2(int(b.TickUpper), &sqrtUpper); err != nil {
			return nil, err
		}
		weight.SetUint64(uint64(b.WeightBps))
		w0.Mul(bal0, weight)
		w0.Div(w0, denom)
		w1.Mul(bal1, weight)
		w1.Div(w1, denom)
		liq, err := liquidityForAmounts(sqrtP, &sqrtLower, &sqrtUpper, w0, w1)
		if err != nil {
			return nil, err
		}
		if liq.IsZero() {
			continue
		}
		out = append(out, position{
			tickLower: int(b.TickLower), tickUpper: int(b.TickUpper),
			sqrtLower: sqrtLower.Clone(), sqrtUpper: sqrtUpper.Clone(), liquidity: liq,
		})
	}
	return out, nil
}

// ---- v4-core SwapMath.computeSwapStep, exact-input branch, exact port ----------

func computeSwapStepExactIn(sqrtCurrent, sqrtTarget, liquidity, amountRemaining *uint256.Int, feePips uint64) (
	sqrtNext, amountIn, amountOut, feeAmount *uint256.Int, err error) {
	zeroForOne := sqrtCurrent.Cmp(sqrtTarget) >= 0
	fee := uint256.NewInt(feePips)
	var feeDelta uint256.Int
	feeDelta.Sub(uMaxSwapFee, fee)
	amountRemainingLessFee, err := v3Utils.MulDiv(amountRemaining, &feeDelta, uMaxSwapFee)
	if err != nil {
		return
	}
	amountIn = new(uint256.Int)
	if zeroForOne {
		err = v3Utils.GetAmount0DeltaV2(sqrtTarget, sqrtCurrent, liquidity, true, amountIn)
	} else {
		err = v3Utils.GetAmount1DeltaV2(sqrtCurrent, sqrtTarget, liquidity, true, amountIn)
	}
	if err != nil {
		return
	}
	sqrtNext = new(uint256.Int)
	feeAmount = new(uint256.Int)
	if amountRemainingLessFee.Cmp(amountIn) >= 0 {
		sqrtNext.Set(sqrtTarget)
		if feePips == maxSwapFee {
			feeAmount.Set(amountIn)
		} else {
			if feeAmount, err = v3Utils.MulDivRoundingUp(amountIn, fee, &feeDelta); err != nil {
				return
			}
		}
	} else {
		amountIn = amountRemainingLessFee
		if err = v3Utils.GetNextSqrtPriceFromInput(sqrtCurrent, liquidity, amountIn, zeroForOne, sqrtNext); err != nil {
			return
		}
		feeAmount.Sub(amountRemaining, amountIn)
	}
	amountOut = new(uint256.Int)
	if zeroForOne {
		err = v3Utils.GetAmount1DeltaV2(sqrtNext, sqrtCurrent, liquidity, false, amountOut)
	} else {
		err = v3Utils.GetAmount0DeltaV2(sqrtCurrent, sqrtNext, liquidity, false, amountOut)
	}
	return
}

// ---- ProtocolFeeLibrary / FeeLib.effectiveSwapFee -------------------------------

// effectiveSwapFee returns the fee pips v4 charges for this direction: the pool's
// lp fee, or the protocol fee compounded with it when a protocol fee is set.
func effectiveSwapFee(lpFee uint32, protocolFee uint32, zeroForOne bool) uint64 {
	var directional uint64
	if zeroForOne {
		directional = uint64(protocolFee & 0xfff)
	} else {
		directional = uint64(protocolFee >> 12)
	}
	if directional == 0 {
		return uint64(lpFee)
	}
	// ProtocolFeeLibrary.calculateSwapFee: p + lp - p*lp/1e6
	lp := uint64(lpFee)
	return directional + lp - directional*lp/pipsDenom
}

// ---- the swap itself: a v4 swap across the freshly deployed positions -----------

type swapResult struct {
	amountIn, amountOut *uint256.Int // amountIn includes the fee (what the swapper pays)
	sqrtPriceX96        *uint256.Int
	tick                int
}

// swapExactIn runs Pool.swap over the bucket positions exactly as the PoolManager
// would once the hook has deployed them: constant liquidity between initialized
// ticks, liquidityNet crossings at bucket edges, fee taken per step.
func swapExactIn(positions []position, sqrtP *uint256.Int, tick int, zeroForOne bool,
	amountIn *uint256.Int, feePips uint64) (*swapResult, error) {
	// initialized ticks and their net liquidity
	net := map[int]*uint256.Int{}
	sign := map[int]int{}
	for _, p := range positions {
		addNet(net, sign, p.tickLower, p.liquidity, +1)
		addNet(net, sign, p.tickUpper, p.liquidity, -1)
	}
	ticks := make([]int, 0, len(net))
	for t := range net {
		ticks = append(ticks, t)
	}
	sort.Ints(ticks)

	liquidity := new(uint256.Int)
	for _, p := range positions {
		if p.tickLower <= tick && tick < p.tickUpper {
			liquidity.Add(liquidity, p.liquidity)
		}
	}

	var limit uint256.Int
	if zeroForOne {
		limit.AddUint64(v3Utils.MinSqrtRatioU256, 1)
	} else {
		limit.SubUint64(v3Utils.MaxSqrtRatioU256, 1)
	}

	remaining := amountIn.Clone()
	totalIn, totalOut := new(uint256.Int), new(uint256.Int)
	sqrt := sqrtP.Clone()

	for !remaining.IsZero() && sqrt.Cmp(&limit) != 0 {
		tNext, found := nextTick(ticks, tick, zeroForOne)
		if tNext < v3Utils.MinTick {
			tNext = v3Utils.MinTick
		} else if tNext > v3Utils.MaxTick {
			tNext = v3Utils.MaxTick
		}
		var sqrtNextTick uint256.Int
		if err := v3Utils.GetSqrtRatioAtTickV2(tNext, &sqrtNextTick); err != nil {
			return nil, err
		}
		target := &sqrtNextTick
		if (zeroForOne && sqrtNextTick.Cmp(&limit) < 0) || (!zeroForOne && sqrtNextTick.Cmp(&limit) > 0) {
			target = &limit
		}
		if liquidity.IsZero() {
			// nothing to trade against here: jump to the next initialized tick without output
			if !found {
				break
			}
			sqrt.Set(&sqrtNextTick)
		} else {
			sqrtNext, stepIn, stepOut, fee, err := computeSwapStepExactIn(sqrt, target, liquidity, remaining, feePips)
			if err != nil {
				return nil, err
			}
			var paid uint256.Int
			paid.Add(stepIn, fee)
			remaining.Sub(remaining, &paid)
			totalIn.Add(totalIn, &paid)
			totalOut.Add(totalOut, stepOut)
			sqrt.Set(sqrtNext)
		}
		if sqrt.Cmp(&sqrtNextTick) == 0 {
			if found {
				delta := net[tNext]
				if (sign[tNext] > 0) != zeroForOne { // crossing into the range adds, out of it removes
					liquidity.Add(liquidity, delta)
				} else {
					if liquidity.Cmp(delta) < 0 {
						return nil, ErrInsufficientLiquidity
					}
					liquidity.Sub(liquidity, delta)
				}
			}
			if zeroForOne {
				tick = tNext - 1
			} else {
				tick = tNext
			}
		} else if sqrt.Cmp(sqrtP) != 0 {
			t, err := v3Utils.GetTickAtSqrtRatioV2(sqrt)
			if err != nil {
				return nil, err
			}
			tick = t
		}
		if !found && liquidity.IsZero() {
			break
		}
	}
	if !remaining.IsZero() {
		return nil, ErrInsufficientLiquidity
	}
	if totalOut.IsZero() {
		return nil, ErrZeroOutput
	}
	return &swapResult{amountIn: totalIn, amountOut: totalOut, sqrtPriceX96: sqrt, tick: tick}, nil
}

// addNet accumulates |liquidityNet| per tick with its sign; buckets never overlap
// an edge in opposite directions in practice, but the sum is kept exact anyway.
func addNet(net map[int]*uint256.Int, sign map[int]int, tick int, liq *uint256.Int, s int) {
	cur, ok := net[tick]
	if !ok {
		net[tick] = liq.Clone()
		sign[tick] = s
		return
	}
	if sign[tick] == s {
		cur.Add(cur, liq)
		return
	}
	if cur.Cmp(liq) >= 0 {
		cur.Sub(cur, liq)
	} else {
		cur.Sub(liq, cur)
		sign[tick] = s
	}
}

// nextTick mirrors nextInitializedTickWithinOneWord semantics over the full set:
// zeroForOne looks for the greatest initialized tick <= current, otherwise the
// smallest initialized tick > current. found=false means none in that direction.
func nextTick(ticks []int, tick int, zeroForOne bool) (int, bool) {
	if zeroForOne {
		i := sort.SearchInts(ticks, tick+1) - 1
		if i < 0 {
			return v3Utils.MinTick, false
		}
		return ticks[i], true
	}
	i := sort.SearchInts(ticks, tick+1)
	if i >= len(ticks) {
		return v3Utils.MaxTick, false
	}
	return ticks[i], true
}
