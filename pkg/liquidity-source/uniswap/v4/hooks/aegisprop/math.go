package aegisprop

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var (
	uPriceScale = u256.TenPow(18)
	uBpsDenom   = uint256.NewInt(bpsDenom)
)

// stalenessHaircutBps linearly ramps from 0 at age==freshThresholdSec up to haircutBpsAtStale at
// age==staleThresholdSec, matching the on-chain hook's ceil-rounded interpolation. filled is false
// once age exceeds staleThresholdSec, mirroring the hook's own reject-on-stale behavior.
func stalenessHaircutBps(age, freshThresholdSec, staleThresholdSec uint64, haircutBpsAtStale uint16) (bps uint64, ok bool) {
	if age > staleThresholdSec {
		return 0, false
	}
	if age <= freshThresholdSec || staleThresholdSec <= freshThresholdSec {
		return 0, true
	}
	span := staleThresholdSec - freshThresholdSec
	num := (age - freshThresholdSec) * uint64(haircutBpsAtStale)
	return (num + span - 1) / span, true // ceil division
}

// applyFeeOut applies the combined taker fee + staleness haircut to a raw ladder-walk output
// amount, rounding down in the venue's favor: out = floor(raw * (10000-totalBps) / 10000).
func applyFeeOut(raw *uint256.Int, totalBps uint64) *uint256.Int {
	mult := uint256.NewInt(bpsDenom - totalBps)
	return u256.MulDivDown(new(uint256.Int), raw, mult, uBpsDenom)
}

// applyFeeIn is the exact-out counterpart of applyFeeOut: it inflates a raw ladder-walk input
// requirement so the taker pays for the fee/haircut, rounding up in the venue's favor:
// in = ceil(raw * 10000 / (10000-totalBps)).
func applyFeeIn(raw *uint256.Int, totalBps uint64) *uint256.Int {
	mult := uint256.NewInt(bpsDenom - totalBps)
	return u256.MulDivUp(new(uint256.Int), raw, uBpsDenom, mult)
}

// walkExactIn replicates the reference off-chain ladder traversal (integration guide §4 B3):
// walk levels best-price-first, filling amountIn and accumulating the counter amount.
// zeroForOne selects which side of the ladder is read: true walks Bids (selling token0 for
// token1), false walks Asks (selling token1 for token0). Level.Amplitude is always denominated
// in raw token0 units, on both sides.
// filled is false if the ladder (as read) is too thin to fill the whole amountIn.
func walkExactIn(levels []Level, amountIn *uint256.Int, zeroForOne bool) (amountOut *uint256.Int, updated []Level, filled bool) {
	remaining := new(uint256.Int).Set(amountIn)
	out := new(uint256.Int)
	updated = make([]Level, len(levels))
	copy(updated, levels)

	for i := range updated {
		if remaining.IsZero() {
			break
		}
		lvl := updated[i]
		if lvl.Price == nil || lvl.Amplitude == nil || lvl.Amplitude.IsZero() {
			continue
		}

		if zeroForOne {
			take := u256.Min(remaining, lvl.Amplitude).Clone()
			out.Add(out, u256.MulDivDown(new(uint256.Int), take, lvl.Price, uPriceScale))
			remaining.Sub(remaining, take)
			updated[i].Amplitude = new(uint256.Int).Sub(lvl.Amplitude, take)
		} else {
			cap1 := u256.MulDivDown(new(uint256.Int), lvl.Amplitude, lvl.Price, uPriceScale)
			take := u256.Min(remaining, cap1).Clone()
			levelOut := u256.MulDivDown(new(uint256.Int), take, uPriceScale, lvl.Price)
			out.Add(out, levelOut)
			remaining.Sub(remaining, take)
			updated[i].Amplitude = new(uint256.Int).Sub(lvl.Amplitude, levelOut)
		}
	}

	return out, updated, remaining.IsZero()
}

// walkExactOut is the exact-out counterpart of walkExactIn: given a desired amountOut, it finds
// the required amountIn walking the same side, per integration guide §4 ("Exact-output quoting is
// the same walk with the roles of the specified/derived legs swapped"). Required input at each
// level is rounded up so the taker never underpays the venue.
func walkExactOut(levels []Level, amountOut *uint256.Int, zeroForOne bool) (amountIn *uint256.Int, updated []Level, filled bool) {
	remaining := new(uint256.Int).Set(amountOut)
	in := new(uint256.Int)
	updated = make([]Level, len(levels))
	copy(updated, levels)

	for i := range updated {
		if remaining.IsZero() {
			break
		}
		lvl := updated[i]
		if lvl.Price == nil || lvl.Amplitude == nil || lvl.Amplitude.IsZero() {
			continue
		}

		if zeroForOne {
			cap1 := u256.MulDivDown(new(uint256.Int), lvl.Amplitude, lvl.Price, uPriceScale)
			takeOut := u256.Min(remaining, cap1).Clone()
			takeIn := u256.Min(u256.MulDivUp(new(uint256.Int), takeOut, uPriceScale, lvl.Price), lvl.Amplitude).Clone()
			in.Add(in, takeIn)
			remaining.Sub(remaining, takeOut)
			updated[i].Amplitude = new(uint256.Int).Sub(lvl.Amplitude, takeIn)
		} else {
			takeOut := u256.Min(remaining, lvl.Amplitude).Clone()
			takeIn := u256.MulDivUp(new(uint256.Int), takeOut, lvl.Price, uPriceScale)
			in.Add(in, takeIn)
			remaining.Sub(remaining, takeOut)
			updated[i].Amplitude = new(uint256.Int).Sub(lvl.Amplitude, takeOut)
		}
	}

	return in, updated, remaining.IsZero()
}
