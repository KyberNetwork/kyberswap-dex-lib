package oracle

import (
	"errors"
)

const (
	MAX_ABS_TICK_MOVE = 9116
)

type Observation struct {
	BlockTimestamp uint32 `json:"bt"`
	PrevTick       int    `json:"pt"`
	TickCumulative int64  `json:"tc"`
	Initialized    bool   `json:"i"`
}

type ObservationStorage struct {
	data []*Observation
}

func NewObservationStorage(observations []*Observation) ObservationStorage {
	return ObservationStorage{
		data: observations,
	}
}

func lte(time, a, b uint32) bool {
	if a <= time && b <= time {
		return a <= b
	}

	var aAdjusted uint64
	if a > time {
		aAdjusted = uint64(a)
	} else {
		aAdjusted = uint64(a) + (1 << 32)
	}

	var bAdjusted uint64
	if b > time {
		bAdjusted = uint64(b)
	} else {
		bAdjusted = uint64(b) + (1 << 32)
	}

	return aAdjusted <= bAdjusted
}

func (o *ObservationStorage) binarySearch(time, target uint32, index, cardinality uint32) (*Observation, *Observation) {
	l := uint64((index + 1) % cardinality)
	r := l + uint64(cardinality) - 1

	var i uint64
	var beforeOrAt, atOrAfter *Observation
	for l <= r {
		i = (l + r) / 2

		beforeOrAt = o.data[uint32(i)%cardinality]

		if !beforeOrAt.Initialized {
			l = i + 1
			continue
		}

		atOrAfter = o.data[(uint32(i)+1)%cardinality]

		targetAtOrAfter := lte(time, beforeOrAt.BlockTimestamp, target)

		if targetAtOrAfter && lte(time, target, atOrAfter.BlockTimestamp) {
			break
		}

		if !targetAtOrAfter {
			r = i - 1
		} else {
			l = i + 1
		}
	}

	return beforeOrAt, atOrAfter
}

func (o *ObservationStorage) getSurroundingObservations(
	intermediate *Observation,
	time uint32,
	target uint32,
	tick int,
	index uint32,
	cardinality uint32,
) (*Observation, *Observation, error) {
	beforeOrAt := intermediate

	if lte(time, beforeOrAt.BlockTimestamp, target) {
		if beforeOrAt.BlockTimestamp == target {
			return beforeOrAt, nil, nil
		} else {
			return beforeOrAt, transform(beforeOrAt, target, tick), nil
		}
	}

	beforeOrAt = o.data[index]
	atOrAfter := intermediate

	if lte(time, beforeOrAt.BlockTimestamp, target) {
		return beforeOrAt, atOrAfter, nil
	}

	next := (index + 1) % cardinality
	beforeOrAt = o.data[next]
	if !beforeOrAt.Initialized {
		beforeOrAt = o.data[0]
	}

	if !lte(time, beforeOrAt.BlockTimestamp, target) {
		return nil, nil, errors.New("TargetPredatesOldestObservation")
	}

	beforeOrAt, atOrAfter = o.binarySearch(time, target, index, cardinality)

	return beforeOrAt, atOrAfter, nil
}

func (o *ObservationStorage) ObserveDouble(intermediate *Observation, time uint32, secondsAgos []uint32,
	tick int, index uint32, cardinality uint32) ([]int64, error) {
	if cardinality == 0 {
		return nil, errors.New("OracleCardinalityCannotBeZero")
	}

	out := make([]int64, len(secondsAgos))
	for i, secondsAgo := range secondsAgos {
		tickCumulative, err := o.ObserveSingle(intermediate, time, secondsAgo, tick, index, cardinality)
		if err != nil {
			return nil, err
		}

		out[i] = tickCumulative
	}

	return out, nil
}

func (o *ObservationStorage) ObserveTriple(intermediate *Observation, time uint32, secondsAgos []uint32,
	tick int, index uint32, cardinality uint32) ([]int64, error) {
	if cardinality == 0 {
		return nil, errors.New("OracleCardinalityCannotBeZero")
	}

	out := make([]int64, 0, len(secondsAgos))
	for _, secondsAgo := range secondsAgos {
		tickCumulative, err := o.ObserveSingle(intermediate, time, secondsAgo, tick, index, cardinality)
		if err != nil {
			return nil, err
		}

		out = append(out, tickCumulative)
	}

	return out, nil
}

func (o *ObservationStorage) ObserveSingle(intermediate *Observation, time, secondsAgo uint32,
	tick int, index uint32, cardinality uint32) (int64, error) {
	if secondsAgo == 0 {
		if intermediate.BlockTimestamp != time {
			return transform(intermediate, time, tick).TickCumulative, nil
		}
	}

	target := time - secondsAgo

	beforeOrAt, atOrAfter, err := o.getSurroundingObservations(intermediate, time, target, tick, index, cardinality)
	if err != nil {
		return 0, err
	}

	switch target {
	case beforeOrAt.BlockTimestamp:
		return beforeOrAt.TickCumulative, nil
	case atOrAfter.BlockTimestamp:
		return atOrAfter.TickCumulative, nil
	}

	observationTimeDelta := atOrAfter.BlockTimestamp - beforeOrAt.BlockTimestamp
	targetDelta := target - beforeOrAt.BlockTimestamp

	return beforeOrAt.TickCumulative +
		((atOrAfter.TickCumulative-beforeOrAt.TickCumulative)/int64(observationTimeDelta))*
			int64(targetDelta), nil
}

func (o *ObservationStorage) Write(
	intermediate *Observation,
	index uint32,
	blockTimestamp uint32,
	tick int,
	cardinality uint32,
	cardinalityNext uint32,
	minInterval uint32,
) (*Observation, uint32, uint32) {
	if intermediate.BlockTimestamp == blockTimestamp {
		return intermediate, index, cardinality
	}

	intermediateUpdated := transform(intermediate, blockTimestamp, tick)

	if blockTimestamp-o.data[index].BlockTimestamp < minInterval {
		return intermediateUpdated, index, cardinality
	}

	var cardinalityUpdated uint32
	if cardinalityNext > cardinality && index == (cardinality-1) {
		cardinalityUpdated = cardinalityNext
	} else {
		cardinalityUpdated = cardinality
	}

	indexUpdated := (index + 1) % cardinalityUpdated
	o.data[indexUpdated] = intermediateUpdated

	return intermediateUpdated, indexUpdated, cardinalityUpdated
}

func transform(last *Observation, blockTimestamp uint32, tick int) *Observation {
	delta := blockTimestamp - last.BlockTimestamp

	if (tick - last.PrevTick) > MAX_ABS_TICK_MOVE {
		tick = last.PrevTick + MAX_ABS_TICK_MOVE
	} else if (tick - last.PrevTick) < -MAX_ABS_TICK_MOVE {
		tick = last.PrevTick - MAX_ABS_TICK_MOVE
	}

	return &Observation{
		BlockTimestamp: blockTimestamp,
		PrevTick:       tick,
		TickCumulative: last.TickCumulative + int64(tick)*int64(delta),
		Initialized:    true,
	}
}
