package ambient

import (
	"sort"

	"github.com/holiman/uint256"
)

// SnapshotBitmapView is an in-memory BitmapView backed by a tracked pool snapshot.
type SnapshotBitmapView struct {
	activeTicks      []int32
	levels           map[int32]BookLevel
	minTick, maxTick int32
	boundaryExceeded bool
}

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

func (v *SnapshotBitmapView) BoundaryExceeded() bool { return v.boundaryExceeded }

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

// QueryLevel returns bidLots/askLots for tick as value copies (zero if tick absent).
func (v *SnapshotBitmapView) QueryLevel(tick int32) (bidLots, askLots uint256.Int) {
	level, ok := v.levels[tick]
	if !ok {
		return
	}
	return level.BidLots, level.AskLots
}
