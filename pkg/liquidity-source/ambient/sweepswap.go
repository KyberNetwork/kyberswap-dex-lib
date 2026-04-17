package ambient

import (
	"math"
	"math/big"
)

// TickInfinityUpper and TickInfinityLower are Solidity's `type(int24).max/min`,
// used as sentinel "zero horizon" bump ticks by Bitmaps.zeroTick().
const (
	TickInfinityUpper int32 = (1 << 23) - 1 // 8_388_607
	TickInfinityLower int32 = -(1 << 23)    // -8_388_608
)

// BitmapView is the minimum read surface SweepSwap needs from the 3-layer
// tick bitmap + per-level data. Mirrors CrocImpact.sol's view-side helpers.
type BitmapView interface {
	// PinBitmap mirrors Bitmaps.pinBitmap/pinTermMezz. It returns the next
	// bump tick in the LOCAL mezz neighborhood of startTick, and spillsOver
	// when no bit is found locally.
	PinBitmap(isBuy bool, startTick int32) (bumpTick int32, spillsOver bool)

	// SeekMezzSpill mirrors Bitmaps.seekMezzSpill. It finds the next active
	// tick across the entire bitmap starting from borderTick, falling back
	// to zeroTick(isBuy) when nothing is found.
	SeekMezzSpill(borderTick int32, isBuy bool) int32

	// QueryLevel returns (bidLots, askLots) at tick for adjTickLiq.
	// For pools without concentrated liquidity at the tick, both are zero.
	QueryLevel(tick int32) (bidLots, askLots *big.Int)
}

// EmptyBitmapView implements BitmapView for a pool whose bitmap is empty.
// Every PinBitmap call spills over to the mezz boundary; SeekMezzSpill
// always returns the int24 infinity sentinel; QueryLevel returns zeros.
type EmptyBitmapView struct{}

func (EmptyBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	mezz := MezzKey(startTick)
	return spillOverPin(isBuy, mezz), true
}

func (EmptyBitmapView) SeekMezzSpill(borderTick int32, isBuy bool) int32 {
	return zeroTick(isBuy)
}

func (EmptyBitmapView) QueryLevel(tick int32) (*big.Int, *big.Int) {
	return new(big.Int), new(big.Int)
}

// spillOverPin replicates Bitmaps.spillOverPin: it returns the tick that
// sits one mezz bucket away from startTick in the swap direction.
func spillOverPin(isBuy bool, tickMezz int16) int32 {
	if isBuy {
		if tickMezz == math.MaxInt16 {
			return zeroTick(true)
		}
		// weldMezzTerm(tickMezz+1, zeroTerm(!isBuy)=0)
		return int32(tickMezz+1) << 8
	}
	// weldMezzTerm(tickMezz, 0)
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

// SweepSwap runs the outer loop that CrocImpact.sol's sweepSwap() performs:
// it repeatedly calls SwapToLimit against the next bitmap-bounded bump tick
// until the swap is exhausted or the limit price is reached. Fees are
// accumulated segment-by-segment (matching the on-chain fee-charging).
func SweepSwap(
	curve *CurveState,
	swap *SwapDirective,
	pool *PoolParams,
	bmp BitmapView,
) (*SwapAccum, error) {
	accum := NewSwapAccum()

	// Solidity short-circuit: if the current price is already past the limit
	// in the swap direction, produce zero flow.
	if swap.IsBuy && curve.PriceRoot.Cmp(swap.LimitPrice) >= 0 {
		return accum, nil
	}
	if !swap.IsBuy && curve.PriceRoot.Cmp(swap.LimitPrice) <= 0 {
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

// sweepTrace is an optional debug callback. Tests can set it to observe the
// per-segment state evolution. Leaving it nil disables tracing.
var sweepTrace func(label string, tick int32, swap *SwapDirective, curve *CurveState)

func hasSwapLeft(curve *CurveState, swap *SwapDirective) bool {
	var inLimit bool
	if swap.IsBuy {
		inLimit = curve.PriceRoot.Cmp(swap.LimitPrice) < 0
	} else {
		inLimit = curve.PriceRoot.Cmp(swap.LimitPrice) > 0
	}
	return inLimit && swap.Qty.Sign() > 0
}

// adjTickLiq mirrors CrocImpact.sol's adjTickLiq: when we cross a
// concentrated-liquidity bump, update concLiq, shave one unit of token
// precision across the bump, and return the updated midTick.
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
	if HasKnockoutLiq(crossedLots) {
		accum.KnockoutCrossLoops++
	}

	// crossDelta = netLotsOnLiquidity(bid, ask) = (int128(bid) - int128(ask)) * LOT_SIZE.
	// For EmptyBitmapView both are zero, so liqDelta = 0 and concLiq is
	// unchanged.
	crossDelta := new(big.Int).Sub(LotsToLiquidity(bidLots), LotsToLiquidity(askLots))

	liqDelta := new(big.Int).Set(crossDelta)
	if !swap.IsBuy {
		liqDelta.Neg(liqDelta)
	}
	curve.ConcLiq = new(big.Int).Add(curve.ConcLiq, liqDelta)

	paidBase, paidQuote, burnSwap, err := ShaveAtBump(curve, swap.InBaseQty, swap.IsBuy, swap.Qty)
	if err != nil {
		return 0, err
	}
	accum.Accumulate(paidBase, paidQuote, new(big.Int))
	swap.Qty = new(big.Int).Sub(swap.Qty, burnSwap)
	accum.CrossInitTickLoops++

	if swap.IsBuy {
		return bumpTick, nil
	}
	return bumpTick - 1, nil
}

// LotSize mirrors LiquidityMath.LOT_SIZE_LIQ (Ambient uses 1024-unit lots for
// concentrated positions).
var LotSize = big.NewInt(1024)
