package ambient

import (
	"math/big"
	"sort"
)

// SnapshotBitmapView is an in-memory implementation of BitmapView backed by a
// tracked pool snapshot. It avoids rebuilding the full on-chain bitmap words
// while preserving the bump-selection behavior that SweepSwap needs.
type SnapshotBitmapView struct {
	activeTicks []int32
	levels      map[int32]BookLevel
}

func NewSnapshotBitmapView(state *TrackerExtra) *SnapshotBitmapView {
	levels := make(map[int32]BookLevel, len(state.Levels))
	for _, level := range state.Levels {
		levels[level.Tick] = cloneBookLevel(level.Level)
	}

	return &SnapshotBitmapView{
		activeTicks: append([]int32(nil), state.ActiveTicks...),
		levels:      levels,
	}
}

func (v *SnapshotBitmapView) PinBitmap(isBuy bool, startTick int32) (int32, bool) {
	tickMezz := MezzKey(startTick)

	if isBuy {
		i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] > startTick })
		if i < len(v.activeTicks) && MezzKey(v.activeTicks[i]) == tickMezz {
			return v.activeTicks[i], false
		}
		return spillOverPin(true, tickMezz), true
	}

	i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] > startTick })
	j := i - 1
	if j >= 0 && MezzKey(v.activeTicks[j]) == tickMezz {
		return v.activeTicks[j], false
	}
	return spillOverPin(false, tickMezz), true
}

func (v *SnapshotBitmapView) SeekMezzSpill(borderTick int32, isBuy bool) int32 {
	if isBuy {
		i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] >= borderTick })
		if i < len(v.activeTicks) {
			return v.activeTicks[i]
		}
		return zeroTick(true)
	}

	i := sort.Search(len(v.activeTicks), func(i int) bool { return v.activeTicks[i] >= borderTick })
	if i > 0 {
		return v.activeTicks[i-1]
	}
	return zeroTick(false)
}

func (v *SnapshotBitmapView) QueryLevel(tick int32) (bidLots, askLots *big.Int) {
	level, ok := v.levels[tick]
	if !ok {
		return new(big.Int), new(big.Int)
	}
	return copyBigInt(level.BidLots), copyBigInt(level.AskLots)
}
