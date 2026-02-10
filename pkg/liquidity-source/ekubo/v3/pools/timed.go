package pools

import (
	"time"

	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

const slotDuration = 12

type (
	TimedPoolState struct {
		*TimedPoolSwapState
		VirtualDeltas []TimeRateDelta `json:"virtualDeltas"`
	}

	TimedPoolSwapState struct {
		Token0Rate        *uint256.Int `json:"token0Rate"`
		Token1Rate        *uint256.Int `json:"token1Rate"`
		LastExecutionTime uint64       `json:"lastExecutionTime"`
	}

	TimeRateDelta struct {
		Time   uint64      `json:"time"`
		Delta0 *int256.Int `json:"delta0"`
		Delta1 *int256.Int `json:"delta1"`
	}
)

func (s *TimedPoolSwapState) Clone() *TimedPoolSwapState {
	return NewTimedPoolSwapState(s.Token0Rate.Clone(), s.Token1Rate.Clone(), s.LastExecutionTime)
}

func NewTimedPoolState(timedPoolSwapState *TimedPoolSwapState, virtualDeltas []TimeRateDelta) *TimedPoolState {
	return &TimedPoolState{
		TimedPoolSwapState: timedPoolSwapState,
		VirtualDeltas:      virtualDeltas,
	}
}

func NewTimedPoolSwapState(token0Rate, token1Rate *uint256.Int, lastExecutionTime uint64) *TimedPoolSwapState {
	return &TimedPoolSwapState{
		Token0Rate:        token0Rate,
		Token1Rate:        token1Rate,
		LastExecutionTime: lastExecutionTime,
	}
}

func NewTimeRateDelta(time uint64, delta0, delta1 *int256.Int) TimeRateDelta {
	return TimeRateDelta{Time: time, Delta0: delta0, Delta1: delta1}
}

func estimatedBlockTimestamp() uint64 {
	return uint64(time.Now().Unix()) + slotDuration
}

func realLastTime(now uint64, last uint32) uint64 {
	return now - ((now - uint64(last)) & 0xffffffff)
}

func approximateExtraDistinctTimeBitmapLookups(startTime, endTime uint64) int64 {
	return int64((endTime >> 16) - (startTime >> 16))
}
