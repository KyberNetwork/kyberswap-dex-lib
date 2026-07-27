package aegisprop

import (
	"github.com/holiman/uint256"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var uPriceScale = u256.TenPow(18)

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
