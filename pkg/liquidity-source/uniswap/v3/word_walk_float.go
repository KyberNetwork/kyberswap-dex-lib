package uniswapv3

import (
	"math"

	"github.com/holiman/uint256"
)

// floatWalkAfterWords is how many empty words a swap walks exactly at nonzero liquidity before it
// prices the rest of a run of empty words in float64 instead of one ComputeSwapStep per word.
const floatWalkAfterWords = 16

// floatBias widens every float amount against the swapper, so float64 error can only make the
// quote slightly worse than the exact walk, never better.
const floatBias = 1e-14

const q96F = 1 << 96

// walkWordsFloat advances a swap that stands exactly on a word edge across further empty words at
// constant liquidity. Each word mirrors the contract's rounding (amountIn and fee up, amountOut
// down) in float64; at zero liquidity every amount is zero, so the run is jumped exactly. It stops
// before listTick, the price limit, the tick bounds and any word the remaining amount may not fully
// pay for; the exact loop prices those. Returns the new tick and the words walked.
func (p *Pool) walkWordsFloat(zeroForOne, exactInput bool, tick, listTick int,
	liquidity, sqrtPriceLimitX96, sqrtPriceX96, amountSpecifiedRemaining, amountCalculated *uint256.Int,
) (int, int, error) {
	limitTick, err := GetTickAtSqrtRatio(sqrtPriceLimitX96)
	if err != nil {
		return tick, 0, err
	}
	wordTicks := p.TickSpacing << 8
	var edge, n int
	if zeroForOne {
		edge = tick + 1
		n = (edge - max(listTick, limitTick, MinTick-1) - 1) / wordTicks
	} else {
		edge = tick
		n = (min(listTick, limitTick, MaxTick+1) - edge - 1) / wordTicks
	}
	if n <= 0 {
		return tick, 0, nil
	}

	var upEdges *upEdgeTable
	if !zeroForOne {
		upEdges = upEdgeTableFor(p.TickSpacing)
	}
	if liquidity.IsZero() {
		return p.landOnWordEdge(zeroForOne, edge, n, wordTicks, upEdges, sqrtPriceX96)
	}
	var tmp uint256.Int
	rem := tmp.Abs(amountSpecifiedRemaining).Float64()
	l := liquidity.Float64()
	l0, l1 := l*q96F, l/q96F
	cur := splitFloat(sqrtPriceX96)
	fee, feeDenom := float64(p.Fee), float64(maxFeeInt-p.Fee)
	invFeeDenom := 1 / feeDenom
	lessFee := feeDenom / maxFeeInt * (1 - floatBias)

	// Word edges come from the exact tables, split into hi+lo floats: a float price model drifts over
	// thousands of words, and the price difference across one word amplifies that drift ~80x.
	var sumIn, sumOut float64
	j := 0
	for ; j < n; j++ {
		var next edgeF
		if zeroForOne {
			next = downEdgeFloats[(edge-(j+1)*wordTicks)>>8-minDownEdge]
		} else if e := edge + (j+1)*wordTicks; upEdges != nil {
			next = upEdges.floats[(e/p.TickSpacing)>>8-upEdges.minWord]
		} else if err = GetSqrtRatioAtTick(e, &tmp); err != nil {
			return tick, 0, err
		} else {
			next = splitFloat(&tmp)
		}
		d := (cur.hi - next.hi) + (cur.lo - next.lo)
		if !zeroForOne {
			d = -d
		}
		amount0, amount1 := l0*d*cur.inv*next.inv, l1*d
		in, out := amount0, amount1
		if !zeroForOne {
			in, out = amount1, amount0
		}
		in = math.Ceil(in * (1 + floatBias))
		if exactInput && rem*lessFee-1 < in || !exactInput && rem*(1-floatBias) < out*(1+floatBias)+1 {
			break
		}
		in += ceilFee(in, fee, feeDenom, invFeeDenom)
		out = math.Floor(out * (1 - floatBias))
		if exactInput {
			rem -= in
		} else {
			rem -= out
		}
		sumIn, sumOut = sumIn+in, sumOut+out
		cur = next
	}
	if j == 0 {
		return tick, 0, nil
	}

	var in, out uint256.Int
	setFloat(&in, sumIn)
	setFloat(&out, sumOut)
	if exactInput {
		if in.Gt(amountSpecifiedRemaining) {
			return tick, 0, nil
		}
		amountSpecifiedRemaining.Sub(amountSpecifiedRemaining, &in)
		amountCalculated.Sub(amountCalculated, &out)
	} else {
		if out.Gt(tmp.Neg(amountSpecifiedRemaining)) {
			return tick, 0, nil
		}
		amountSpecifiedRemaining.Add(amountSpecifiedRemaining, &out)
		amountCalculated.Add(amountCalculated, &in)
	}

	return p.landOnWordEdge(zeroForOne, edge, j, wordTicks, upEdges, sqrtPriceX96)
}

// landOnWordEdge moves the price exactly onto the edge j words past edge, so the exact loop resumes
// from the contract's own price. Returns the new tick and j.
func (p *Pool) landOnWordEdge(zeroForOne bool, edge, j, wordTicks int, upEdges *upEdgeTable,
	sqrtPriceX96 *uint256.Int) (int, int, error) {
	if zeroForOne {
		edge -= j * wordTicks
		*sqrtPriceX96 = downEdgeSqrtPrice(edge)
		return edge - 1, j, nil
	}
	edge += j * wordTicks
	if upEdges != nil {
		*sqrtPriceX96 = upEdges.sqrtPrice(edge, p.TickSpacing)
	} else if err := GetSqrtRatioAtTick(edge, sqrtPriceX96); err != nil {
		return 0, 0, err
	}
	return edge, j, nil
}

// ceilFee returns ceil(in * fee / feeDenom), given invFeeDenom = 1/feeDenom. Below 2^52 the product
// is an exact integer, so the reciprocal's estimate is off by at most one and exact integer checks
// correct it.
func ceilFee(in, fee, feeDenom, invFeeDenom float64) float64 {
	f := in * fee
	if f >= 1<<52 {
		return math.Ceil(f * invFeeDenom * (1 + floatBias))
	}
	c := math.Ceil(f * invFeeDenom)
	if (c-1)*feeDenom >= f {
		return c - 1
	} else if c*feeDenom < f {
		return c + 1
	}
	return c
}

// setFloat sets z to the non-negative integer-valued f.
func setFloat(z *uint256.Int, f float64) *uint256.Int {
	if f < 1<<64 {
		return z.SetUint64(uint64(f))
	}
	frac, exp := math.Frexp(f)
	z.SetUint64(uint64(math.Ldexp(frac, 64)))
	return z.Lsh(z, uint(exp-64))
}

// edgeF carries a sqrt price as hi+lo float64s, ~106 bits, so the difference of two nearby prices
// keeps full float precision; inv is 1/hi.
type edgeF struct{ hi, lo, inv float64 }

func splitFloat(x *uint256.Int) edgeF {
	var h, r uint256.Int
	hi := x.Float64()
	if setFloat(&h, hi); x.Lt(&h) {
		return edgeF{hi, -r.Sub(&h, x).Float64(), 1 / hi}
	}
	return edgeF{hi, r.Sub(x, &h).Float64(), 1 / hi}
}
