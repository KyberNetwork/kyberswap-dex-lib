package ambient

import (
	"math"

	"github.com/holiman/uint256"
)

const (
	TickInfinityUpper int32 = (1 << 23) - 1
	TickInfinityLower int32 = -(1 << 23)
)

// BitmapView is the read surface SweepSwap needs.
type BitmapView interface {
	PinBitmap(isBuy bool, startTick int32) (bumpTick int32, spillsOver bool)
	SeekMezzSpill(borderTick int32, isBuy bool) int32
	QueryLevel(tick int32) (bidLots, askLots uint256.Int)
}

type EmptyBitmapView struct{}

func (EmptyBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	mezz := MezzKey(startTick)
	return spillOverPin(isBuy, mezz), true
}

func (EmptyBitmapView) SeekMezzSpill(_ int32, isBuy bool) int32 {
	return zeroTick(isBuy)
}

func (EmptyBitmapView) QueryLevel(_ int32) (uint256.Int, uint256.Int) {
	return uint256.Int{}, uint256.Int{}
}

func spillOverPin(isBuy bool, tickMezz int16) int32 {
	if isBuy {
		if tickMezz == math.MaxInt16 {
			return zeroTick(true)
		}
		return int32(tickMezz+1) << 8
	}
	return int32(tickMezz) << 8
}

func zeroTick(isBuy bool) int32 {
	if isBuy {
		return TickInfinityUpper
	}
	return TickInfinityLower
}

func isTickFinite(tick int32) bool {
	return tick > TickInfinityLower && tick < TickInfinityUpper
}

func SweepSwap(
	curve *CurveState,
	swap *SwapDirective,
	pool *PoolParams,
	bmp BitmapView,
) (*SwapAccum, error) {
	accum := NewSwapAccum()

	if swap.IsBuy && curve.PriceRoot.Cmp(&swap.LimitPrice) >= 0 {
		return accum, nil
	}
	if !swap.IsBuy && curve.PriceRoot.Cmp(&swap.LimitPrice) <= 0 {
		return accum, nil
	}

	midTick := GetTickAtSqrtRatio(curve.PriceRoot)
	if sweepTrace != nil {
		sweepTrace("start", midTick, swap, curve)
	}

	for doMore := true; doMore; {
		bumpTick, spillsOver := bmp.PinBitmap(swap.IsBuy, midTick)
		if sweepTrace != nil {
			sweepTrace("pin", bumpTick, swap, curve)
		}
		SwapToLimit(curve, accum, swap, pool, bumpTick)
		if sweepTrace != nil {
			sweepTrace("post-swap1", bumpTick, swap, curve)
		}

		doMore = hasSwapLeft(curve, swap)
		if !doMore {
			break
		}

		if spillsOver {
			liqTick := bmp.SeekMezzSpill(bumpTick, swap.IsBuy)
			tightSpill := bumpTick == liqTick
			bumpTick = liqTick
			if !tightSpill {
				SwapToLimit(curve, accum, swap, pool, bumpTick)
				if sweepTrace != nil {
					sweepTrace("post-swap2", bumpTick, swap, curve)
				}
				accum.PinSpillLoops++
				doMore = hasSwapLeft(curve, swap)
			}
		}

		if doMore {
			next, err := adjTickLiq(accum, bumpTick, curve, swap, bmp)
			if err != nil {
				return nil, err
			}
			midTick = next
			if sweepTrace != nil {
				sweepTrace("adj", midTick, swap, curve)
			}
		}
	}

	return accum, nil
}

var sweepTrace func(label string, tick int32, swap *SwapDirective, curve *CurveState)

func hasSwapLeft(curve *CurveState, swap *SwapDirective) bool {
	var inLimit bool
	if swap.IsBuy {
		inLimit = curve.PriceRoot.Lt(&swap.LimitPrice)
	} else {
		inLimit = curve.PriceRoot.Gt(&swap.LimitPrice)
	}
	return inLimit && !swap.Qty.IsZero()
}

func adjTickLiq(
	accum *SwapAccum,
	bumpTick int32,
	curve *CurveState,
	swap *SwapDirective,
	bmp BitmapView,
) (int32, error) {
	if !isTickFinite(bumpTick) {
		return bumpTick, nil
	}

	bidLots, askLots := bmp.QueryLevel(bumpTick)
	crossedLots := askLots
	if !swap.IsBuy {
		crossedLots = bidLots
	}
	if HasKnockoutLiq(&crossedLots) {
		accum.KnockoutCrossLoops++
	}

	// Apply liqDelta = (bidLiq - askLiq) * sign to ConcLiq (uint256 two's complement int128).
	var bidLiq, askLiq uint256.Int
	LotsToLiquidity(&bidLiq, &bidLots)
	LotsToLiquidity(&askLiq, &askLots)

	if swap.IsBuy {
		// delta = bidLiq - askLiq
		if bidLiq.Cmp(&askLiq) >= 0 {
			var diff uint256.Int
			diff.Sub(&bidLiq, &askLiq)
			curve.ConcLiq.Add(&curve.ConcLiq, &diff)
		} else {
			var diff uint256.Int
			diff.Sub(&askLiq, &bidLiq)
			curve.ConcLiq.Sub(&curve.ConcLiq, &diff)
		}
	} else {
		// delta = -(bidLiq - askLiq) = askLiq - bidLiq
		if askLiq.Cmp(&bidLiq) >= 0 {
			var diff uint256.Int
			diff.Sub(&askLiq, &bidLiq)
			curve.ConcLiq.Add(&curve.ConcLiq, &diff)
		} else {
			var diff uint256.Int
			diff.Sub(&bidLiq, &askLiq)
			curve.ConcLiq.Sub(&curve.ConcLiq, &diff)
		}
	}

	paidBase, paidQuote, burnSwap, err := ShaveAtBump(curve, swap.InBaseQty, swap.IsBuy, swap.Qty)
	if err != nil {
		return 0, err
	}
	accum.Accumulate(paidBase, paidQuote, uint256.Int{})
	swap.Qty.Sub(&swap.Qty, &burnSwap)
	accum.CrossInitTickLoops++

	if swap.IsBuy {
		return bumpTick, nil
	}
	return bumpTick - 1, nil
}
