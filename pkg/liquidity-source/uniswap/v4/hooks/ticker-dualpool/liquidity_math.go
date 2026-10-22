package tickerdualpool

import (
	"sort"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

const totalWeightBps = 10000
const maxSwapFee = 1000000

var uMaxSwapFee = uint256.NewInt(maxSwapFee)
var uMaxUint128 = new(uint256.Int).Sub(new(uint256.Int).Lsh(uint256.NewInt(1), 128), uint256.NewInt(1))

type position = dualpool.Position

type swapResult struct {
	amountIn, amountOut *uint256.Int // amountIn includes the fee (what the swapper pays)
	sqrtPriceX96        *uint256.Int
	tick                int
	positionFees        []*uint256.Int
}

// swapExactInWithSpacing tracks V4 word boundaries and collected position fees.
func swapExactInWithSpacing(positions []position, sqrtP *uint256.Int, tick int, zeroForOne bool,
	amountIn *uint256.Int, feePips uint64, spacing int, protocolPips uint64) (*swapResult, error) {
	// initialized ticks and their net liquidity
	net := map[int]*uint256.Int{}
	sign := map[int]int{}
	for _, p := range positions {
		addNet(net, sign, p.TickLower, p.Liquidity, +1)
		addNet(net, sign, p.TickUpper, p.Liquidity, -1)
	}
	ticks := make([]int, 0, len(net))
	for t := range net {
		ticks = append(ticks, t)
	}
	sort.Ints(ticks)

	liquidity := new(uint256.Int)
	for _, p := range positions {
		if p.TickLower <= tick && tick < p.TickUpper {
			liquidity.Add(liquidity, p.Liquidity)
		}
	}

	var limit uint256.Int
	if zeroForOne {
		limit.AddUint64(v3Utils.MinSqrtRatioU256, 1)
	} else {
		limit.SubUint64(v3Utils.MaxSqrtRatioU256, 1)
	}

	growth := make([]*uint256.Int, len(positions))
	for i := range growth {
		growth[i] = new(uint256.Int)
	}
	q128 := new(uint256.Int).Lsh(uint256.NewInt(1), 128)
	remaining := amountIn.Clone()
	totalIn, totalOut := new(uint256.Int), new(uint256.Int)
	sqrt := sqrtP.Clone()

	for !remaining.IsZero() && sqrt.Cmp(&limit) != 0 {
		tNext, found := nextTickWithinWord(ticks, tick, spacing, zeroForOne)
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
			sqrt.Set(target)
		} else {
			sqrtNext, stepIn, stepOut, fee, err := dualpool.ComputeSwapStepExactIn(sqrt, target, liquidity, remaining, feePips)
			if err != nil {
				return nil, err
			}
			var paid uint256.Int
			paid.Add(stepIn, fee)
			// Each active position earns the same per-liquidity fee growth.
			// v4 rounds growth per step, then rounds each position's collection.
			lpFee := fee.Clone()
			if protocolPips != 0 {
				protocol := new(uint256.Int)
				if feePips == protocolPips {
					protocol.Set(fee)
				} else {
					protocol.Mul(&paid, uint256.NewInt(protocolPips)).Div(protocol, uMaxSwapFee)
				}
				lpFee.Sub(lpFee, protocol)
			}
			stepGrowth, err := v3Utils.MulDiv(lpFee, q128, liquidity)
			if err != nil {
				return nil, err
			}
			for i, p := range positions {
				if p.TickLower <= tick && tick < p.TickUpper {
					growth[i].Add(growth[i], stepGrowth)
				}
			}
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
						return nil, dualpool.ErrInsufficientLiquidity
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
	}
	if !remaining.IsZero() {
		return nil, dualpool.ErrInsufficientLiquidity
	}
	if totalOut.IsZero() {
		return nil, dualpool.ErrZeroOutput
	}
	collected := make([]*uint256.Int, len(positions))
	for i, p := range positions {
		// All buckets use LP_SALT. Identical ranges therefore share one position:
		// its first removal collects fees on their combined liquidity, rounded once.
		collected[i] = new(uint256.Int)
		first := true
		for j := range i {
			if positions[j].TickLower == p.TickLower && positions[j].TickUpper == p.TickUpper {
				first = false
				break
			}
		}
		if !first {
			continue
		}
		combined := p.Liquidity.Clone()
		for j := i + 1; j < len(positions); j++ {
			if positions[j].TickLower == p.TickLower && positions[j].TickUpper == p.TickUpper {
				if _, overflow := combined.AddOverflow(combined, positions[j].Liquidity); overflow {
					return nil, dualpool.ErrInsufficientLiquidity
				}
			}
		}
		var err error
		collected[i], err = v3Utils.MulDiv(growth[i], combined, q128)
		if err != nil {
			return nil, err
		}
	}
	return &swapResult{amountIn: totalIn, amountOut: totalOut, sqrtPriceX96: sqrt, tick: tick, positionFees: collected}, nil
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

// nextTickWithinWord follows v4 TickBitmap: search only the current 256-bit
// word, returning its edge when it contains no initialized tick in this direction.
// Division must round down for negative ticks, rather than Go's truncation to zero.
func nextTickWithinWord(ticks []int, tick, spacing int, zeroForOne bool) (int, bool) {
	compressed := tick / spacing
	if tick < 0 && tick%spacing != 0 {
		compressed--
	}
	if zeroForOne {
		boundary := (compressed >> 8) * 256 * spacing
		i := sort.SearchInts(ticks, tick+1) - 1
		if i >= 0 && ticks[i] >= boundary {
			return ticks[i], true
		}
		return boundary, false
	}
	compressed++
	boundary := (((compressed>>8)+1)*256 - 1) * spacing
	i := sort.SearchInts(ticks, tick+1)
	if i < len(ticks) && ticks[i] <= boundary {
		return ticks[i], true
	}
	return boundary, false
}
