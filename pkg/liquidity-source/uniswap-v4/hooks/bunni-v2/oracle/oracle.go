package oracle

import "errors"

const (
	MAX_ABS_TICK_MOVE = 9116
	MAX_CARDINALITY   = (1 << 24) - 1
)

type Observation struct {
	BlockTimestamp uint32
	PrevTick       int
	TickCumulative int64
	Initialized    bool
}

type ObservationStorage struct {
	observations [MAX_CARDINALITY]Observation
}

func lte(time, time1, time2 uint32) bool {
	if time1 <= time {
		return time1 <= time2
	}
	return time2 > time1
}

func (o *ObservationStorage) binarySearch(time, target uint32, index, cardinality uint32) (Observation, Observation) {
	l := uint64((index + 1) % cardinality)
	r := l + uint64(cardinality) - 1

	var i uint64
	var beforeOrAt, atOrAfter Observation
	for {
		i = (l + r) / 2

		beforeOrAt = o.observations[i%uint64(cardinality)]

		if !beforeOrAt.Initialized {
			l = i + 1
			continue
		}

		atOrAfter = o.observations[(i+1)%uint64(cardinality)]

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
	intermediate Observation,
	time uint32,
	target uint32,
	tick int,
	index uint32,
	cardinality uint32,
) (Observation, Observation, error) {
	beforeOrAt := intermediate

	if lte(time, beforeOrAt.BlockTimestamp, target) {
		if beforeOrAt.BlockTimestamp == target {
			return beforeOrAt, Observation{}, nil
		} else {
			return beforeOrAt, transform(beforeOrAt, target, tick), nil
		}
	}

	beforeOrAt = o.observations[index]
	atOrAfter := intermediate

	if lte(time, beforeOrAt.BlockTimestamp, target) {
		return beforeOrAt, atOrAfter, nil
	}

	beforeOrAt = o.observations[(index+1)%cardinality]
	if !beforeOrAt.Initialized {
		beforeOrAt = o.observations[0]
	}

	if !lte(time, beforeOrAt.BlockTimestamp, target) {
		return Observation{}, Observation{}, errors.New("TargetPredatesOldestObservation")
	}

	beforeOrAt, atOrAfter = o.binarySearch(time, target, index, cardinality)

	return beforeOrAt, atOrAfter, nil
}

func (o *ObservationStorage) ObserveDouble(intermediate Observation, time uint32, secondsAgos []uint32,
	tick int, index uint32, cardinality uint32) ([]int64, error) {
	if cardinality == 0 {
		return nil, errors.New("OracleCardinalityCannotBeZero")
	}

	tickCumulatives := make([]int64, 0, len(secondsAgos))
	for _, secondsAgo := range secondsAgos {
		tickCumulative, err := o.ObserveSingle(intermediate, time, secondsAgo, tick, index, cardinality)
		if err != nil {
			return nil, err
		}

		tickCumulatives = append(tickCumulatives, tickCumulative)
	}

	return tickCumulatives, nil
}

func (o *ObservationStorage) ObserveTriple(intermediate Observation, time uint32, secondsAgos []uint32,
	tick int, index uint32, cardinality uint32) ([]int64, error) {
	if cardinality == 0 {
		return nil, errors.New("OracleCardinalityCannotBeZero")
	}

	tickCumulatives := make([]int64, 0, len(secondsAgos))
	for _, secondsAgo := range secondsAgos {
		tickCumulative, err := o.ObserveSingle(intermediate, time, secondsAgo, tick, index, cardinality)
		if err != nil {
			return nil, err
		}

		tickCumulatives = append(tickCumulatives, tickCumulative)
	}

	return tickCumulatives, nil
}

func (o *ObservationStorage) ObserveSingle(intermediate Observation, time, secondsAgo uint32,
	tick int, index uint32, cardinality uint32) (int64, error) {
	if secondsAgo == 0 {
		if intermediate.BlockTimestamp != time {
			intermediate = transform(intermediate, time, tick)
			return intermediate.TickCumulative, nil
		}
		return intermediate.TickCumulative, nil
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
	intermediate Observation,
	index uint32,
	blockTimestamp uint32,
	tick int,
	cardinality uint32,
	cardinalityNext uint32,
	minInterval uint32,
) (Observation, uint32, uint32) {
	if intermediate.BlockTimestamp == blockTimestamp {
		return intermediate, index, cardinality
	}

	intermediateUpdated := transform(intermediate, blockTimestamp, tick)

	if blockTimestamp-o.observations[index].BlockTimestamp < minInterval {
		return intermediateUpdated, index, cardinality
	}

	var cardinalityUpdated uint32
	if cardinalityNext > cardinality && index == (cardinality-1) {
		cardinalityUpdated = cardinalityNext
	} else {
		cardinalityUpdated = cardinality
	}

	indexUpdated := (index + 1) % cardinalityUpdated
	o.observations[indexUpdated] = intermediateUpdated

	return intermediateUpdated, indexUpdated, cardinalityUpdated
}

func transform(last Observation, blockTimestamp uint32, tick int) Observation {
	delta := blockTimestamp - last.BlockTimestamp

	if (tick - last.PrevTick) > MAX_ABS_TICK_MOVE {
		tick = last.PrevTick + MAX_ABS_TICK_MOVE
	} else if (tick - last.PrevTick) < -MAX_ABS_TICK_MOVE {
		tick = last.PrevTick - MAX_ABS_TICK_MOVE
	}

	return Observation{
		BlockTimestamp: blockTimestamp,
		PrevTick:       tick,
		TickCumulative: last.TickCumulative + int64(tick)*int64(delta),
		Initialized:    true,
	}
}
