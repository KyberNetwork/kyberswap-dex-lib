package algebra

import (
	"cmp"
	"context"
	"math/big"
	"slices"
	"sort"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

const (
	maxTimepointPageSize = uint16(300)
	maxTimepointPages    = 16
)

type (
	Timepoint interface {
		GetInitialized() bool
		GetBlockTimestamp() uint32
	}
	TimepointRPC[T Timepoint] interface {
		ToTimepoint() T
		Timepoint
	}
)

type PoolTracker[T Timepoint, R TimepointRPC[T]] struct {
	EthrpcClient *ethrpc.Client
}

func (d *PoolTracker[Timepoint, TimepointRPC]) GetTimepoints(ctx context.Context, callPrototype *ethrpc.Call,
	blockNumber *big.Int, yesterday uint32, currentIndex uint16, currentTimestamp uint32,
	timepoints map[uint16]Timepoint, overrides map[common.Address]gethclient.OverrideAccount) (map[uint16]Timepoint,
	error) {
	if timepoints == nil {
		timepoints = make(map[uint16]Timepoint, maxTimepointPageSize)
	} else {
		dropOverwrittenTimepoints(timepoints, currentIndex, currentTimestamp)
	}

	req := d.EthrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	timepointPageSize := maxTimepointPageSize / 4 // optimistically fetch fewer the first time
	req.Calls = make([]*ethrpc.Call, 0, timepointPageSize)
	end := currentIndex + 1 // current last tp index of the page (exclusive)
	var enoughAtIdx uint16
	for range maxTimepointPages { // page backwards missing timepoints until we reach uninitialized or older than 1 day
		tpIdx := end // current start tp index of the page (inclusive). can underflow (wrap back to end of buffer)
		var enough bool
		req.Calls = req.Calls[:0]
		page := make([]TimepointRPC, timepointPageSize)
		tpIdxToPageIdxMap := make(map[uint16]uint16, timepointPageSize)
		for i := range timepointPageSize {
			for tpIdx--; ; tpIdx-- { // skip refetching for existing timepoints
				if tp := timepoints[tpIdx]; !tp.GetInitialized() {
					break
				} else if tp.GetBlockTimestamp() < yesterday { // stop right away if we found a timepoint older than 1 day
					enough = true
					break
				}
			}
			if enough {
				break
			}
			call := *callPrototype
			call.Params = []any{big.NewInt(int64(tpIdx))}
			req.AddCall(&call, []any{&page[i]})
			tpIdxToPageIdxMap[tpIdx] = i
		}
		if len(req.Calls) > 0 {
			if _, err := req.Aggregate(); err != nil {
				return nil, err
			}
		}

		enoughAtIdx = tpIdx
		if !enough {
			smallestUsableTpIdxOffset := sort.Search(int(end-tpIdx), func(i int) bool {
				tpSearchIdx := tpIdx + uint16(i) // with overflow
				if tp := timepoints[tpSearchIdx]; tp.GetInitialized() {
					return tp.GetBlockTimestamp() >= yesterday
				}
				tp := page[tpIdxToPageIdxMap[tpSearchIdx]]
				return tp.GetInitialized() && tp.GetBlockTimestamp() >= yesterday
			})
			if enough = smallestUsableTpIdxOffset > 0; enough {
				enoughAtIdx = tpIdx + uint16(smallestUsableTpIdxOffset) - 1
			}
		}
		for i := enoughAtIdx; i != end; i++ {
			if !timepoints[i].GetInitialized() {
				timepoints[i] = page[tpIdxToPageIdxMap[i]].ToTimepoint()
			}
		}
		logger.Debugf("fetched %v timepoints from %v to %v, enough=%v, enoughAtIdx=%v, ts=%v, ytd=%v",
			len(req.Calls), tpIdx, end, enough, enoughAtIdx, timepoints[enoughAtIdx].GetBlockTimestamp(), yesterday)

		if enough { // fetch some additional timepoints
			req.Calls = req.Calls[:0]
			additionalIndices := []uint16{0, currentIndex + 1, currentIndex + 2, enoughAtIdx, currentIndex - 1}
			tps := make([]TimepointRPC, len(additionalIndices))
			for i, x := range additionalIndices {
				if !timepoints[x].GetInitialized() {
					call := *callPrototype
					call.Params = []any{big.NewInt(int64(x))}
					req.AddCall(&call, []any{&tps[i]})
				}
			}
			if len(req.Calls) > 0 {
				if _, err := req.Aggregate(); err != nil {
					return nil, err
				}
				for i, x := range additionalIndices {
					if !timepoints[x].GetInitialized() {
						timepoints[x] = tps[i].ToTimepoint()
					}
				}
			}
			break
		}

		end = tpIdx // next page, can be underflow back to end of buffer
		timepointPageSize = min(maxTimepointPageSize, timepointPageSize*2)
	}

	// remove old timepoints before enoughAtIdx
	for idx := range timepoints {
		if idx > 0 && LteConsideringOverflow(idx, enoughAtIdx-1, currentIndex+2) {
			delete(timepoints, idx)
		}
	}

	if !timepoints[currentIndex].GetInitialized() {
		return nil, nil // some new pools don't have timepoints initialized yet, ignore them
	}
	return timepoints, nil
}

// dropOverwrittenTimepoints removes cached timepoints that the plugin has overwritten since they were cached.
// The timepoints form a uint16 ring buffer, so a slot is not immutable: after wrap-around it is rewritten with a new
// timepoint, while the cached copy still holds the one from the previous cycle (older timestamp). Besides the slots
// below the head, the cache holds a few slots ahead of it (the oldest ones, fetched by GetTimepoints), which become
// the next slots to be written.
//
// Slots written since the last refresh form the run (prevIndex, currentIndex]. Timestamps increase along the buffer
// up to the head, so going backwards from currentIndex through the cached slots, they must not increase. A cached slot
// that is newer than the one after it marks the end of the run: everything visited before it is stale. The head must
// also carry currentTimestamp (the plugin's lastTimepointTimestamp), otherwise it is stale itself.
// Cost is linear in the cache size and involves no RPC. currentTimestamp == 0 means unknown, so the cache is trusted.
func dropOverwrittenTimepoints[T Timepoint](timepoints map[uint16]T, currentIndex uint16, currentTimestamp uint32) {
	if currentTimestamp == 0 {
		return
	}
	indices := make([]uint16, 0, len(timepoints))
	for idx, tp := range timepoints {
		if tp.GetInitialized() {
			indices = append(indices, idx)
		}
	}
	// distance walked backwards from the head
	slices.SortFunc(indices, func(a, b uint16) int { return cmp.Compare(currentIndex-a, currentIndex-b) })

	headStale := false
	if tp, ok := timepoints[currentIndex]; ok && tp.GetInitialized() {
		headStale = tp.GetBlockTimestamp() != currentTimestamp
	}
	for i := 1; i < len(indices); i++ {
		if timepoints[indices[i]].GetBlockTimestamp() > timepoints[indices[i-1]].GetBlockTimestamp() {
			for _, idx := range indices[:i] {
				delete(timepoints, idx)
			}
			return
		}
	}
	if headStale { // no older slot proves otherwise, so the whole cache is from a previous cycle
		clear(timepoints)
	}
}

// LteConsideringOverflow returns true if a <= b with c as greatest value anchor for overflow checking.
// a <= b <= c | true
// b <= c <  a | true
// c <  a <= b | true
// a <= c <  b | false
// b <  a <= c | false
// c <  b <  a | false
func LteConsideringOverflow[T cmp.Ordered](a, b, currentTime T) bool {
	res := a > currentTime
	if res == (b > currentTime) {
		res = a <= b
	}
	return res
}
