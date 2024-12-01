package integral

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"
)

var (
	ErrTargetIsTooOld = errors.New("target is too old")
)

type TimepointStorage struct {
	data    map[uint16]Timepoint
	updates map[uint16]Timepoint
}

func (s *TimepointStorage) Get(index uint16) Timepoint {
	if v, ok := s.updates[index]; ok {
		logger.Debugf("access new %v %v", index, v)
		return v
	}

	if v, ok := s.data[index]; ok {
		logger.Debugf("access exists %v %v", index, v)
		return v
	}

	logger.Debugf("access none %v", index)

	return Timepoint{
		Initialized:          false,
		BlockTimestamp:       0,
		TickCumulative:       0,
		VolatilityCumulative: new(big.Int),
		Tick:                 0,
		AverageTick:          0,
		WindowStartIndex:     0,
	}
}
func (s *TimepointStorage) Set(index uint16, v Timepoint) {
	s.updates[index] = v
}

func (s *TimepointStorage) write(lastIndex uint16, blockTimestamp uint32, tick int32) (uint16, uint16, error) {
	last := s.Get(lastIndex)

	if last.BlockTimestamp == blockTimestamp {
		return lastIndex, 0, nil
	}

	var indexUpdated = lastIndex + 1

	var oldestIndex uint16
	if s.Get(indexUpdated).Initialized {
		oldestIndex = indexUpdated
	}

	avgTick, windowStartIndex, err := s.getAverageTickCasted(blockTimestamp, tick, lastIndex, oldestIndex, last.BlockTimestamp, last.TickCumulative)
	if err != nil {
		return 0, 0, err
	}

	if windowStartIndex == indexUpdated {
		windowStartIndex++
	}

	s.Set(indexUpdated, createNewTimepoint(last, blockTimestamp, tick, avgTick, windowStartIndex))

	if oldestIndex == indexUpdated {
		oldestIndex++
	}

	return indexUpdated, oldestIndex, nil
}

func (s *TimepointStorage) getAverageTickCasted(
	time uint32,
	tick int32,
	lastIndex, oldestIndex uint16,
	lastTimestamp uint32,
	lastTickCumulative int64,
) (int32, uint16, error) {
	avgTick, windowStartIndex, err := s.getAverageTick(time, tick, lastIndex, oldestIndex, lastTimestamp, lastTickCumulative)
	if err != nil {
		return 0, 0, err
	}

	return int32(avgTick.Int64()), uint16(windowStartIndex.Uint64()), nil
}

func (s *TimepointStorage) getAverageTick(currentTime uint32, tick int32, lastIndex, oldestIndex uint16, lastTimestamp uint32, lastTickCumulative int64) (*big.Int, *big.Int, error) {
	self := s.Get(oldestIndex)

	oldestTimestamp, oldestTickCumulative := self.BlockTimestamp, self.TickCumulative

	currentTickCumulative := lastTickCumulative + int64(tick)*int64(uint64(currentTime-lastTimestamp))
	if !lteConsideringOverflow(oldestTimestamp, currentTime-WINDOW, currentTime) {
		if currentTime == oldestTimestamp {
			return big.NewInt(int64(tick)), big.NewInt(int64(oldestIndex)), nil
		}

		return big.NewInt((currentTickCumulative - oldestTickCumulative) / int64(uint64(currentTime-oldestTimestamp))), big.NewInt(int64(oldestIndex)), nil
	}

	if lteConsideringOverflow(lastTimestamp, currentTime-WINDOW, currentTime) {
		return big.NewInt(int64(tick)), big.NewInt(int64(lastIndex)), nil
	} else {
		tickCumulativeAtStart, windowStartIndex, err := s.getTickCumulativeAt(currentTime, WINDOW, tick, lastIndex, oldestIndex)
		if err != nil {
			return nil, nil, err
		}

		avgTick := (currentTickCumulative - tickCumulativeAtStart) / int64(WINDOW)

		return big.NewInt(avgTick), windowStartIndex, nil
	}
}

func (s *TimepointStorage) getTickCumulativeAt(time, secondsAgo uint32, tick int32, lastIndex, oldestIndex uint16) (int64, *big.Int, error) {
	target := time - secondsAgo
	beforeOrAt, atOrAfter, samePoint, indexBeforeOrAt, err := s.getTimepointsAt(time, target, lastIndex, oldestIndex)
	if err != nil {
		return 0, nil, err
	}

	timestampBefore, tickCumulativeBefore := beforeOrAt.BlockTimestamp, beforeOrAt.TickCumulative
	if target == timestampBefore {
		return tickCumulativeBefore, indexBeforeOrAt, nil
	}

	if samePoint {
		return tickCumulativeBefore + int64(tick)*int64(uint64(target-timestampBefore)), indexBeforeOrAt, nil
	}

	timestampAfter, tickCumulativeAfter := atOrAfter.BlockTimestamp, atOrAfter.TickCumulative

	if target == timestampAfter {
		return tickCumulativeAfter, new(big.Int).Add(indexBeforeOrAt, bignumber.One), nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	return tickCumulativeBefore + ((tickCumulativeAfter-tickCumulativeBefore)/int64(uint64(timepointTimeDelta)))*int64(uint64(targetDelta)),
		indexBeforeOrAt, nil
}

func (s *TimepointStorage) getTimepointsAt(currentTime, target uint32, lastIndex, oldestIndex uint16) (
	beforeOrAt, atOrAfter Timepoint,
	samePoint bool,
	indexBeforeOrAt *big.Int,
	err error,
) {
	lastTimepoint := s.Get(lastIndex)

	lastTimepointTimestamp := lastTimepoint.BlockTimestamp
	windowStartIndex := lastTimepoint.WindowStartIndex

	if target == currentTime || lteConsideringOverflow(lastTimepointTimestamp, target, currentTime) {
		return lastTimepoint, lastTimepoint, true, big.NewInt(int64(lastIndex)), nil
	}

	var useHeuristic bool

	if lastTimepointTimestamp-target <= WINDOW {
		oldestIndex = windowStartIndex
		useHeuristic = target == currentTime-WINDOW
	}

	oldestTimepoint := s.Get(oldestIndex)

	oldestTimestamp := oldestTimepoint.BlockTimestamp

	if !lteConsideringOverflow(oldestTimestamp, target, currentTime) {
		err = ErrTargetIsTooOld
		return
	}

	if oldestTimestamp == target {
		return oldestTimepoint, oldestTimepoint, true, big.NewInt(int64(oldestIndex)), nil
	}

	if lastIndex == oldestIndex+1 {
		return oldestTimepoint, lastTimepoint, false, big.NewInt(int64(oldestIndex)), nil
	}

	beforeOrAt, atOrAfter, indexBeforeOrAt = s.binarySearch(currentTime, target, lastIndex, oldestIndex, useHeuristic)

	return
}

func (s *TimepointStorage) getAverageVolatility(currentTime uint32, tick int32, lastIndex, oldestIndex uint16) (*big.Int, error) {
	lastTimepoint := s.Get(lastIndex)

	timeAtLastTimepoint := lastTimepoint.BlockTimestamp == currentTime

	lastCumulativeVolatility := lastTimepoint.VolatilityCumulative
	windowStartIndex := lastTimepoint.WindowStartIndex

	if !timeAtLastTimepoint {
		var err error
		lastCumulativeVolatility, err = s.getVolatilityCumulativeAt(currentTime, 0, tick, lastIndex, oldestIndex)
		if err != nil {
			return nil, err
		}
	}

	oldestTimestamp := s.Get(oldestIndex).BlockTimestamp
	if lteConsideringOverflow(oldestTimestamp, currentTime-WINDOW, currentTime) {
		var cumulativeVolatilityAtStart *big.Int
		if timeAtLastTimepoint {
			oldestTimestamp, cumulativeVolatilityAtStart = s.Get(windowStartIndex).BlockTimestamp, s.Get(windowStartIndex).VolatilityCumulative

			timeDeltaBetweenPoints := s.Get(windowStartIndex+1).BlockTimestamp - oldestTimestamp

			cumulativeVolatilityAtStart = cumulativeVolatilityAtStart.Add(
				cumulativeVolatilityAtStart,
				new(big.Int).Div(
					new(big.Int).Mul(
						new(big.Int).Sub(
							s.Get(windowStartIndex+1).VolatilityCumulative,
							cumulativeVolatilityAtStart,
						),
						big.NewInt(int64(currentTime-WINDOW-oldestTimestamp)),
					),
					big.NewInt(int64(timeDeltaBetweenPoints)),
				),
			)
		} else {
			var err error
			cumulativeVolatilityAtStart, err = s.getVolatilityCumulativeAt(currentTime, WINDOW, tick, lastIndex, oldestIndex)
			if err != nil {
				return nil, err
			}
		}

		return new(big.Int).Div(
			new(big.Int).Sub(lastCumulativeVolatility, cumulativeVolatilityAtStart),
			big.NewInt(int64(WINDOW)),
		), nil

	} else if currentTime != oldestTimestamp {
		oldestVolatilityCumulative := s.Get(oldestIndex).VolatilityCumulative
		unbiasedDenominator := currentTime - oldestTimestamp
		if unbiasedDenominator > 1 {
			unbiasedDenominator--
		}

		return new(big.Int).Div(
			new(big.Int).Sub(lastCumulativeVolatility, oldestVolatilityCumulative),
			big.NewInt(int64(unbiasedDenominator)),
		), nil
	}

	return nil, nil
}

func (s *TimepointStorage) getVolatilityCumulativeAt(time, secondsAgo uint32, tick int32, lastIndex, oldestIndex uint16) (*big.Int, error) {
	target := time - secondsAgo

	beforeOrAt, atOrAfter, samePoint, _, err := s.getTimepointsAt(time, target, lastIndex, oldestIndex)
	if err != nil {
		return nil, err
	}

	timestampBefore, volatilityCumulativeBefore := beforeOrAt.BlockTimestamp, beforeOrAt.VolatilityCumulative
	if target == timestampBefore {
		return volatilityCumulativeBefore, nil
	}

	if samePoint {
		avgTick, _, err := s.getAverageTickCasted(target, tick, lastIndex, oldestIndex, timestampBefore, beforeOrAt.TickCumulative)
		if err != nil {
			return nil, err
		}

		return new(big.Int).Add(
			volatilityCumulativeBefore,
			volatilityOnRange(big.NewInt(int64(target-timestampBefore)),
				big.NewInt(int64(tick)),
				big.NewInt(int64(tick)),
				big.NewInt(int64(beforeOrAt.AverageTick)),
				big.NewInt(int64(avgTick)),
			),
		), nil
	}

	timestampAfter, volatilityCumulativeAfter := atOrAfter.BlockTimestamp, atOrAfter.VolatilityCumulative
	if target == timestampAfter {
		return volatilityCumulativeAfter, nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	return new(big.Int).Add(
		volatilityCumulativeBefore,
		new(big.Int).Mul(
			new(big.Int).Div(
				new(big.Int).Sub(
					volatilityCumulativeAfter,
					volatilityCumulativeBefore,
				),
				big.NewInt(int64(timepointTimeDelta)),
			),
			big.NewInt(int64(targetDelta)),
		),
	), nil
}

func (s *TimepointStorage) getOldestIndex(lastIndex uint16) uint16 {
	oldestIndex := lastIndex + 1
	if s.Get(oldestIndex).Initialized {
		return oldestIndex
	}

	return 0
}

func lteConsideringOverflow(a, b, currentTime uint32) bool {
	res := a > currentTime

	if res == (b > currentTime) {
		res = a <= b
	}

	return res
}

func createNewTimepoint(last Timepoint, blockTimestamp uint32, tick, averageTick int32, windowStartIndex uint16) Timepoint {
	delta := blockTimestamp - last.BlockTimestamp

	volatility := volatilityOnRange(big.NewInt(int64(delta)), big.NewInt(int64(tick)),
		big.NewInt(int64(tick)), big.NewInt(int64(last.AverageTick)), big.NewInt(int64(averageTick)))

	last.Initialized = true
	last.BlockTimestamp = blockTimestamp
	last.TickCumulative += int64(tick) * int64(delta)
	last.VolatilityCumulative = new(big.Int).Add(last.VolatilityCumulative, volatility)
	last.Tick = tick
	last.AverageTick = averageTick
	last.WindowStartIndex = windowStartIndex

	return last
}

func volatilityOnRange(dt, tick0, tick1, avgTick0, avgTick1 *big.Int) *big.Int {
	// (k = (tick1 - tick0) - (avgTick1 - avgTick0))
	k := new(big.Int).Sub(tick1, tick0)
	k.Sub(k, new(big.Int).Sub(avgTick1, avgTick0))

	// (b = (tick0 - avgTick0) * dt)
	b := new(big.Int).Sub(tick0, avgTick0)
	b.Mul(b, dt)

	// sumOfSequence = dt * (dt + 1)
	sumOfSequence := new(big.Int).Add(dt, big.NewInt(1))
	sumOfSequence.Mul(sumOfSequence, dt)

	// sumOfSquares = sumOfSequence * (2 * dt + 1)
	sumOfSquares := new(big.Int).Mul(big.NewInt(2), dt)
	sumOfSquares.Add(sumOfSquares, big.NewInt(1))
	sumOfSquares.Mul(sumOfSquares, sumOfSequence)

	// k^2
	kSquared := new(big.Int).Mul(k, k)

	// b^2
	bSquared := new(big.Int).Mul(b, b)

	// k^2 * sumOfSquares
	term1 := new(big.Int).Mul(kSquared, sumOfSquares)

	// 6 * b * k * sumOfSequence
	term2 := new(big.Int).Mul(b, k)
	term2.Mul(term2, sumOfSequence)
	term2.Mul(term2, big.NewInt(6))

	// 6 * dt * b^2
	term3 := new(big.Int).Mul(bSquared, dt)
	term3.Mul(term3, big.NewInt(6))

	// dt^2
	dtSquared := new(big.Int).Mul(dt, dt)

	// Calculate volatility = (term1 + term2 + term3) / (6 * dt^2)
	denominator := new(big.Int).Mul(dtSquared, big.NewInt(6))
	numerator := new(big.Int).Add(term1, term2)
	numerator.Add(numerator, term3)

	volatility := new(big.Int).Div(numerator, denominator)

	return volatility
}

func (s *TimepointStorage) binarySearch(
	currentTime uint32,
	target uint32,
	upperIndex uint16,
	lowerIndex uint16,
	withHeuristic bool,
) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt *big.Int) {
	var right *big.Int

	left := big.NewInt(int64(lowerIndex)) // oldest timepoint

	if upperIndex < lowerIndex {
		right = big.NewInt(int64(upperIndex) + UINT16_MODULO)
	} else {
		right = big.NewInt(int64(upperIndex))
	}

	return s.binarySearchInternal(currentTime, target, left, right, withHeuristic)
}

func (s *TimepointStorage) binarySearchInternal(
	currentTime uint32,
	target uint32,
	left,
	right *big.Int,
	withHeuristic bool,
) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt *big.Int) {
	if withHeuristic && new(big.Int).Sub(right, left).Cmp(bignumber.Two) > 0 {
		indexBeforeOrAt = new(big.Int).Add(left, bignumber.One)
	} else {
		indexBeforeOrAt = new(big.Int).Rsh(
			new(big.Int).Add(left, right),
			1,
		)
	}

	beforeOrAt = s.Get(uint16(indexBeforeOrAt.Uint64()))
	atOrAfter = beforeOrAt

	var firstIteration bool = true
	for {
		initializedBefore, timestampBefore := beforeOrAt.Initialized, beforeOrAt.BlockTimestamp
		if initializedBefore {
			if lteConsideringOverflow(timestampBefore, target, currentTime) {
				atOrAfter = s.Get(uint16(indexBeforeOrAt.Uint64() + 1))
				initializedAfter, timestampAfter := atOrAfter.Initialized, atOrAfter.BlockTimestamp
				if initializedAfter {
					if lteConsideringOverflow(target, timestampAfter, currentTime) {
						return beforeOrAt, atOrAfter, indexBeforeOrAt
					}
					left = new(big.Int).Add(indexBeforeOrAt, bignumber.One)
				} else {
					return beforeOrAt, beforeOrAt, indexBeforeOrAt
				}
			} else {
				right = new(big.Int).Sub(indexBeforeOrAt, bignumber.One)
			}
		} else {
			left = new(big.Int).Add(indexBeforeOrAt, bignumber.One)
		}

		useHeuristic := firstIteration && withHeuristic && left.Cmp(new(big.Int).Add(indexBeforeOrAt, bignumber.One)) == 0
		if useHeuristic && new(big.Int).Sub(right, left).Cmp(SIXTEEN) > 0 {
			indexBeforeOrAt = indexBeforeOrAt.Add(left, EIGHT)
		} else {
			indexBeforeOrAt = indexBeforeOrAt.Rsh(
				new(big.Int).Add(left, right),
				1,
			)
		}

		beforeOrAt = s.Get(uint16(indexBeforeOrAt.Uint64()))
		firstIteration = false
	}
}

func calculateFeeFactors(currentTick, lastTick int32, priceChangeFactor uint16) (FeeFactors, error) {
	tickDelta := new(big.Int).Sub(big.NewInt(int64(currentTick)), big.NewInt(int64(lastTick)))
	if tickDelta.Int64() > int64(utils.MaxTick) {
		tickDelta = big.NewInt(int64(utils.MaxTick))
	} else if tickDelta.Int64() < int64(utils.MinTick) {
		tickDelta = big.NewInt(int64(utils.MinTick))
	}

	sqrtPriceDelta, err := v3Utils.GetSqrtRatioAtTick(int(tickDelta.Int64()))
	if err != nil {
		return FeeFactors{}, err
	}

	priceRatioSquared, err := mulDiv(sqrtPriceDelta, sqrtPriceDelta, DOUBLE_FEE_MULTIPLIER)
	if err != nil {
		return FeeFactors{}, err
	}

	priceChangeRatio := new(big.Int).Sub(priceRatioSquared, BASE_FEE_MULTIPLIER)

	factor := new(big.Int).SetInt64(int64(priceChangeFactor))
	feeFactorImpact := new(big.Int).Div(new(big.Int).Mul(priceChangeRatio, factor), big.NewInt(FACTOR_DENOMINATOR))

	feeFactors := FeeFactors{
		zeroToOneFeeFactor: BASE_FEE_MULTIPLIER,
		oneToZeroFeeFactor: BASE_FEE_MULTIPLIER,
	}

	newZeroToOneFeeFactor := new(big.Int).Sub(feeFactors.zeroToOneFeeFactor, feeFactorImpact)

	twoShift := DOUBLE_FEE_MULTIPLIER

	if newZeroToOneFeeFactor.Cmp(bignumber.ZeroBI) > 0 && newZeroToOneFeeFactor.Cmp(twoShift) < 0 {
		feeFactors.zeroToOneFeeFactor = newZeroToOneFeeFactor
		feeFactors.oneToZeroFeeFactor = new(big.Int).Add(feeFactors.oneToZeroFeeFactor, feeFactorImpact)
	} else if newZeroToOneFeeFactor.Cmp(bignumber.ZeroBI) <= 0 {
		feeFactors.zeroToOneFeeFactor = bignumber.ZeroBI
		feeFactors.oneToZeroFeeFactor = twoShift
	} else {
		feeFactors.zeroToOneFeeFactor = twoShift
		feeFactors.oneToZeroFeeFactor = bignumber.ZeroBI
	}

	return feeFactors, nil
}
