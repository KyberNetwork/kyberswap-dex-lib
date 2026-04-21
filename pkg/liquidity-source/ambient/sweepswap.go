package ambient

import (
	"math"
	"math/big"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// int24 sentinels used by Bitmaps.zeroTick() as "zero horizon" bump ticks.
const (
	TickInfinityUpper int32 = (1 << 23) - 1
	TickInfinityLower int32 = -(1 << 23)
)

// BitmapView is the read surface SweepSwap needs. Mirrors CrocImpact.sol.
type BitmapView interface {
	// PinBitmap returns the next bump tick in the local mezz of startTick;
	// spillsOver when no bit is found locally.
	PinBitmap(isBuy bool, startTick int32) (bumpTick int32, spillsOver bool)
	// SeekMezzSpill returns the next active tick across the whole bitmap,
	// falling back to zeroTick(isBuy).
	SeekMezzSpill(borderTick int32, isBuy bool) int32
	// QueryLevel returns (bidLots, askLots) at tick; zeros if none.
	QueryLevel(tick int32) (bidLots, askLots *big.Int)
}

// EmptyBitmapView is the zero-liquidity implementation of BitmapView.
type EmptyBitmapView struct{}

func (EmptyBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	mezz := MezzKey(startTick)
	return spillOverPin(isBuy, mezz), true
}

func (EmptyBitmapView) SeekMezzSpill(borderTick int32, isBuy bool) int32 {
	return zeroTick(isBuy)
}

func (EmptyBitmapView) QueryLevel(tick int32) (*big.Int, *big.Int) {
	return bignum.ZeroBI, bignum.ZeroBI
}

// spillOverPin mirrors Bitmaps.spillOverPin: next mezz bucket in swap direction.
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

// SweepSwap mirrors CrocImpact.sol sweepSwap: repeatedly SwapToLimit to the
// next bump tick until swap exhausts or hits limit price; fees accumulate
// segment-by-segment.
func SweepSwap(
	curve *CurveState,
	swap *SwapDirective,
	pool *PoolParams,
	bmp BitmapView,
) (*SwapAccum, error) {
	accum := NewSwapAccum()

	// Solidity short-circuit: already past limit → zero flow.
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

// sweepTrace is an optional debug callback for tests; nil disables tracing.
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

// adjTickLiq mirrors CrocImpact.sol adjTickLiq: on bump crossing, update
// concLiq, shave one unit of precision, and return new midTick.
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

	// liqDelta = (int128(bid) - int128(ask)) * LOT_SIZE, negated on sell side.
	liqDelta := new(big.Int).Sub(LotsToLiquidity(bidLots), LotsToLiquidity(askLots))
	if !swap.IsBuy {
		liqDelta.Neg(liqDelta)
	}
	curve.ConcLiq = liqDelta.Add(curve.ConcLiq, liqDelta)

	paidBase, paidQuote, burnSwap, err := ShaveAtBump(curve, swap.InBaseQty, swap.IsBuy, swap.Qty)
	if err != nil {
		return 0, err
	}
	accum.Accumulate(paidBase, paidQuote, bignum.ZeroBI)
	swap.Qty.Sub(swap.Qty, burnSwap)
	accum.CrossInitTickLoops++

	if swap.IsBuy {
		return bumpTick, nil
	}
	return bumpTick - 1, nil
}
