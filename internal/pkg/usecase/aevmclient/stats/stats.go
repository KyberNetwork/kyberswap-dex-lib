package stats

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxHistoricalStatLength = 100_000
)

type SortedDatapoints[T any] []T

func PN[T any](datapoints SortedDatapoints[T], percentile int) *T {
	var index int
	if percentile == 100 {
		index = len(datapoints) - 1
	} else {
		index = int(percentile) * len(datapoints) / 100
	}
	if index >= 0 && index < len(datapoints) {
		return &datapoints[index]
	}
	return nil
}

type HistoricalStat[T any] struct {
	ts         [maxHistoricalStatLength]int64
	datapoints [maxHistoricalStatLength]T
	len        int
	next       int
}

func (h *HistoricalStat[T]) Push(timestamp int64, datapoint T) {
	h.ts[h.next] = timestamp
	h.datapoints[h.next] = datapoint
	h.next = (h.next + 1) % maxHistoricalStatLength
	if h.len < maxHistoricalStatLength {
		h.len++
	}
}

func (h *HistoricalStat[T]) Query(fromTimestampUs int64, toTimestampExclusiveUs int64) []T {
	if h.len == 0 || fromTimestampUs >= toTimestampExclusiveUs {
		return nil
	}

	var (
		index      = (h.next - 1 + maxHistoricalStatLength) % maxHistoricalStatLength
		datapoints []T
	)
	for i := 0; i < h.len; i++ {
		if h.ts[index] >= fromTimestampUs && h.ts[index] < toTimestampExclusiveUs {
			datapoints = append(datapoints, h.datapoints[index])
		}

		index = (index - 1 + maxHistoricalStatLength) % maxHistoricalStatLength
	}

	// reverse datapoints
	for i, j := 0, len(datapoints)-1; i < j; i, j = i+1, j-1 {
		datapoints[i], datapoints[j] = datapoints[j], datapoints[i]
	}

	return datapoints
}

type ShardedStats[T any] struct {
	n             int
	counterShards []*atomic.Uint64
	dataShards    []*HistoricalStat[T]
	shardLock     []sync.Mutex
	globalLock    sync.RWMutex
	shardCounter  atomic.Uint64
	cmp           func(T, T) int
}

func NewShardedStats[T any](n int, cmp func(T, T) int) *ShardedStats[T] {
	counterShards := make([]*atomic.Uint64, n)
	dataShards := make([]*HistoricalStat[T], n)
	for i := 0; i < n; i++ {
		counterShards[i] = new(atomic.Uint64)
		dataShards[i] = new(HistoricalStat[T])
	}
	return &ShardedStats[T]{
		n:             n,
		counterShards: counterShards,
		dataShards:    dataShards,
		shardLock:     make([]sync.Mutex, n),
		cmp:           cmp,
	}
}

func (s *ShardedStats[T]) Add(ts int64, datapoint T) {
	index := (s.shardCounter.Add(1) - 1) % uint64(s.n)
	s.shardLock[index].Lock()
	s.globalLock.RLock()
	s.counterShards[index].Add(1)
	s.dataShards[index].Push(ts, datapoint)
	s.globalLock.RUnlock()
	s.shardLock[index].Unlock()
}

func (s *ShardedStats[T]) Query(windowStart, windowEnd time.Time) (count uint64, datapoints SortedDatapoints[T]) {
	for _, s := range s.counterShards {
		count += uint64(s.Load())
	}

	s.globalLock.Lock()
	for _, s := range s.dataShards {
		datapoints = append(datapoints, s.Query(windowStart.UnixMicro(), windowEnd.UnixMicro())...)
	}
	s.globalLock.Unlock()

	slices.SortFunc(datapoints, s.cmp)

	return count, datapoints
}
