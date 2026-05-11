package ambient

import (
	"math/big"
	"sort"

	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// SnapshotBitmapView is an in-memory implementation of BitmapView backed by a
// tracked pool snapshot. It avoids rebuilding the full on-chain bitmap words
// while preserving the bump-selection behavior that SweepSwap needs.
//
// When MinTick/MaxTick are narrower than the full int24 range, the view
// reports boundary-exceeded via BoundaryExceeded after any PinBitmap or
// SeekMezzSpill call that would have returned a tick outside the window.
type SnapshotBitmapView struct {
	activeTicks      []int32
	levels           map[int32]BookLevel
	minTick, maxTick int32
	boundaryExceeded bool
}

// NewSnapshotBitmapView builds a read-only view over state. Callers must not
// mutate state.ActiveTicks or state.Levels for the lifetime of the view; the
// view borrows them without copying.
func NewSnapshotBitmapView(state *TrackerExtra) *SnapshotBitmapView {
	levels := make(map[int32]BookLevel, len(state.Levels))
	for _, level := range state.Levels {
		levels[level.Tick] = level.Level
	}

	minTick := state.MinTick
	maxTick := state.MaxTick
	if minTick == 0 && maxTick == 0 {
		minTick = FullTickWindow.MinTick
		maxTick = FullTickWindow.MaxTick
	}

	return &SnapshotBitmapView{
		activeTicks: state.ActiveTicks,
		levels:      levels,
		minTick:     minTick,
		maxTick:     maxTick,
	}
}

func (v *SnapshotBitmapView) BoundaryExceeded() bool {
	return v.boundaryExceeded
}

func (v *SnapshotBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	tickMezz := MezzKey(startTick)

	if isBuy {
		i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] > startTick })
		if i < len(v.activeTicks) && MezzKey(v.activeTicks[i]) == tickMezz {
			return v.activeTicks[i], false
		}
		pin := spillOverPin(true, tickMezz)
		if pin > v.maxTick {
			v.boundaryExceeded = true
		}
		return pin, true
	}

	i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] > startTick })
	j := i - 1
	if j >= 0 && MezzKey(v.activeTicks[j]) == tickMezz {
		return v.activeTicks[j], false
	}
	pin := spillOverPin(false, tickMezz)
	if pin < v.minTick {
		v.boundaryExceeded = true
	}
	return pin, true
}

func (v *SnapshotBitmapView) SeekMezzSpill(borderTick int32, isBuy bool) int32 {
	if isBuy {
		i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] >= borderTick })
		if i < len(v.activeTicks) {
			return v.activeTicks[i]
		}
		v.boundaryExceeded = true
		return zeroTick(true)
	}

	i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] >= borderTick })
	if i > 0 {
		return v.activeTicks[i-1]
	}
	v.boundaryExceeded = true
	return zeroTick(false)
}

// QueryLevel returns bid/ask lots for tick. Callers must treat the returned
// *big.Int as read-only; misses share bignum.ZeroBI, hits share the level's
// own pointers.
func (v *SnapshotBitmapView) QueryLevel(tick int32) (bidLots, askLots *big.Int) {
	level, ok := v.levels[tick]
	if !ok {
		return bignum.ZeroBI, bignum.ZeroBI
	}
	return level.BidLots, level.AskLots
}
