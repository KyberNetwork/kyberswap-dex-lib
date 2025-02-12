package algebra

import (
	"context"
	"math/big"
	"sort"

	"github.com/KyberNetwork/ethrpc"
	"golang.org/x/exp/constraints"
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
	blockNumber *big.Int, yesterday uint32, currentIndex uint16, timepoints map[uint16]Timepoint) (map[uint16]Timepoint,
	error) {
	if timepoints == nil {
		timepoints = make(map[uint16]Timepoint, maxTimepointPageSize)
	}

	req := d.EthrpcClient.NewRequest().SetContext(ctx)
	if blockNumber != nil && blockNumber.Sign() > 0 {
		req.SetBlockNumber(blockNumber)
	}

	timepointPageSize := maxTimepointPageSize / 4 // optimistically fetch fewer the first time
	req.Calls = make([]*ethrpc.Call, 0, timepointPageSize)
	page := make([]TimepointRPC, timepointPageSize)
	end := currentIndex // current last tp index of the page (inclusive)
	var enoughAtIdx uint16
	for range maxTimepointPages { // page backwards missing timepoints until we reach uninitialized or older than 1 day
		tpIdx := end // current start tp index of the page. can underflow (wrap back to end of buffer)
		var enough bool
		req.Calls = req.Calls[:0]
		tpIdxToPageIdxMap := make(map[uint16]uint16, timepointPageSize)
		for i := range timepointPageSize {
			for tp := timepoints[tpIdx]; tp.GetInitialized(); tpIdx-- { // skip refetching for existing timepoints
				if tp.GetBlockTimestamp() < yesterday { // stop right away if we found a timepoint older than 1 day
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
			tpIdx--
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
				enoughAtIdx = tpIdx + uint16(smallestUsableTpIdxOffset)
			}
		}
		for i := enoughAtIdx; i != end+1; i++ {
			if !timepoints[i].GetInitialized() {
				timepoints[i] = page[tpIdxToPageIdxMap[i]].ToTimepoint()
			}
		}

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

		end = tpIdx - 1 // next page, can be underflow back to end of buffer
		timepointPageSize = min(maxTimepointPageSize, timepointPageSize*2)
	}

	// remove old timepoints before enoughAtIdx
	for idx := range timepoints {
		if LteConsideringOverflow(idx, enoughAtIdx-1, currentIndex+2) {
			delete(timepoints, idx)
		}
	}

	if !timepoints[currentIndex].GetInitialized() {
		return nil, nil // some new pools don't have timepoints initialized yet, ignore them
	}
	return timepoints, nil
}

// LteConsideringOverflow returns true if a <= b with c as greatest value anchor for overflow checking.
// a <= b <= c | true
// b <= c <  a | true
// c <  a <= b | true
// a <= c <  b | false
// b <  a <= c | false
// c <  b <  a | false
func LteConsideringOverflow[T constraints.Ordered](a, b, currentTime T) bool {
	res := a > currentTime
	if res == (b > currentTime) {
		res = a <= b
	}
	return res
}
