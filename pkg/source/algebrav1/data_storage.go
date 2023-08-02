package algebrav1

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

// port https://github.com/cryptoalgebra/AlgebraV1/blob/dfebf532a27803dafcbf2ba49724740bd6220505/src/core/contracts/libraries/DataStorage.sol

// we won't need full 65536 points as in SC, so use a custom struct
type TimepointStorage struct {
	data    map[uint16]Timepoint // original data from SC
	updates map[uint16]Timepoint // new data written during simulation (or precalculation), will be cleared
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
		Initialized:                   false,
		BlockTimestamp:                0,
		TickCumulative:                0,
		SecondsPerLiquidityCumulative: new(big.Int),
		VolatilityCumulative:          new(big.Int),
		AverageTick:                   0,
		VolumePerLiquidityCumulative:  new(big.Int),
	}
}
func (s *TimepointStorage) Set(index uint16, v Timepoint) {
	s.updates[index] = v
}

// / @notice Calculates volatility between two sequential timepoints with resampling to 1 sec frequency
// / @param dt Timedelta between timepoints, must be within uint32 range
// / @param tick0 The tick at the left timepoint, must be within int24 range
// / @param tick1 The tick at the right timepoint, must be within int24 range
// / @param avgTick0 The average tick at the left timepoint, must be within int24 range
// / @param avgTick1 The average tick at the right timepoint, must be within int24 range
// / @return volatility The volatility between two sequential timepoints
// / If the requirements for the parameters are met, it always fits 88 bits
func _volatilityOnRange(
	dt *big.Int,
	tick0 *big.Int,
	tick1 *big.Int,
	avgTick0 *big.Int,
	avgTick1 *big.Int,
) (volatility *big.Int) {
	// On the time interval from the previous timepoint to the current
	// we can represent tick and average tick change as two straight lines:
	// tick = k*t + b, where k and b are some constants
	// avgTick = p*t + q, where p and q are some constants
	// we want to get sum of (tick(t) - avgTick(t))^2 for every t in the interval (0; dt]
	// so: (tick(t) - avgTick(t))^2 = ((k*t + b) - (p*t + q))^2 = (k-p)^2 * t^2 + 2(k-p)(b-q)t + (b-q)^2
	// since everything except t is a constant, we need to use progressions for t and t^2:
	// sum(t) for t from 1 to dt = dt*(dt + 1)/2 = sumOfSequence
	// sum(t^2) for t from 1 to dt = dt*(dt+1)*(2dt + 1)/6 = sumOfSquares
	// so result will be: (k-p)^2 * sumOfSquares + 2(k-p)(b-q)*sumOfSequence + dt*(b-q)^2
	K := new(big.Int).Sub(new(big.Int).Sub(tick1, tick0), new(big.Int).Sub(avgTick1, avgTick0)) // (k - p)*dt

	B := new(big.Int).Mul(new(big.Int).Sub(tick0, avgTick0), dt) // (b - q)*dt
	sumOfSquares := new(big.Int).Mul(
		dt,
		new(big.Int).Mul(
			new(big.Int).Add(dt, bignumber.One),
			new(big.Int).Add(new(big.Int).Mul(bignumber.Two, dt), bignumber.One),
		),
	) // sumOfSquares * 6
	sumOfSequence := new(big.Int).Mul(dt, new(big.Int).Add(dt, bignumber.One)) // sumOfSequence * 2
	volatility = new(big.Int).Div(
		new(big.Int).Add(
			new(big.Int).Mul(new(big.Int).Mul(K, K), sumOfSquares),
			new(big.Int).Add(
				new(big.Int).Mul(new(big.Int).Mul(bignumber.Six, B), new(big.Int).Mul(K, sumOfSequence)),
				new(big.Int).Mul(new(big.Int).Mul(bignumber.Six, dt), new(big.Int).Mul(B, B)),
			),
		),
		new(big.Int).Mul(bignumber.Six, new(big.Int).Mul(dt, dt)),
	)
	return
}

// / @notice Transforms a previous timepoint into a new timepoint, given the passage of time and the current tick and liquidity values
// / @dev blockTimestamp _must_ be chronologically equal to or greater than last.blockTimestamp, safe for 0 or 1 overflows
// / @param last The specified timepoint to be used in creation of new timepoint
// / @param blockTimestamp The timestamp of the new timepoint
// / @param tick The active tick at the time of the new timepoint
// / @param prevTick The active tick at the time of the last timepoint
// / @param liquidity The total in-range liquidity at the time of the new timepoint
// / @param averageTick The average tick at the time of the new timepoint
// / @param volumePerLiquidity The gmean(volumes)/liquidity at the time of the new timepoint
// / @return Timepoint The newly populated timepoint
func createNewTimepoint(
	last Timepoint,
	blockTimestamp uint32,
	tick int24,
	prevTick int24,
	liquidity *big.Int,
	averageTick int24,
	volumePerLiquidity *big.Int,
) Timepoint {
	delta := blockTimestamp - last.BlockTimestamp

	last.Initialized = true
	last.BlockTimestamp = blockTimestamp
	last.TickCumulative += int56(tick) * int64(delta)

	if liquidity.Cmp(bignumber.ZeroBI) <= 0 {
		liquidity = new(big.Int).Set(bignumber.One)
	}
	// just timedelta if liquidity == 0
	last.SecondsPerLiquidityCumulative = new(big.Int).Add(
		last.SecondsPerLiquidityCumulative,
		new(big.Int).Div(new(big.Int).Lsh(big.NewInt(int64(delta)), 128), liquidity),
	)
	// always fits 88 bits
	last.VolatilityCumulative = new(big.Int).Add(
		last.VolatilityCumulative,
		_volatilityOnRange(big.NewInt(int64(delta)), big.NewInt(int64(prevTick)), big.NewInt(int64(tick)), big.NewInt(int64(last.AverageTick)), big.NewInt(int64(averageTick))),
	)
	last.AverageTick = averageTick
	last.VolumePerLiquidityCumulative = new(big.Int).Add(last.VolumePerLiquidityCumulative, volumePerLiquidity)

	return last
}

// / @notice comparator for 32-bit timestamps
// / @dev safe for 0 or 1 overflows, a and b _must_ be chronologically before or equal to currentTime
// / @param a A comparison timestamp from which to determine the relative position of `currentTime`
// / @param b From which to determine the relative position of `currentTime`
// / @param currentTime A timestamp truncated to 32 bits
// / @return res Whether `a` is chronologically <= `b`
func lteConsideringOverflow(
	a uint32,
	b uint32,
	currentTime uint32,
) bool {
	res := a > currentTime
	if res == (b > currentTime) {
		res = a <= b
	} // if both are on the same side
	return res
}

// / @dev guaranteed that the result is within the bounds of int24
// / returns int256 for fuzzy tests
func (self *TimepointStorage) _getAverageTick(
	time uint32,
	tick int24,
	index uint16,
	oldestIndex uint16,
	lastTimestamp uint32,
	lastTickCumulative int56,
) (*big.Int, error) {
	oldest := self.Get(oldestIndex)
	oldestTimestamp := oldest.BlockTimestamp
	oldestTickCumulative := oldest.TickCumulative

	var avgTick int64

	if lteConsideringOverflow(oldestTimestamp, time-WINDOW, time) {
		if lteConsideringOverflow(lastTimestamp, time-WINDOW, time) {
			index -= 1 // considering underflow
			startTimepoint := self.Get(index)
			if startTimepoint.Initialized {
				avgTick = (lastTickCumulative - startTimepoint.TickCumulative) / int64(lastTimestamp-startTimepoint.BlockTimestamp)
			} else {
				avgTick = int64(tick)
			}
		} else {
			startOfWindow, err := self.getSingleTimepoint(time, WINDOW, tick, index, oldestIndex, bignumber.ZeroBI)
			if err != nil {
				return nil, err
			}

			//    current-WINDOW  last   current
			// _________*____________*_______*_
			//           ||||||||||||
			avgTick = (lastTickCumulative - startOfWindow.TickCumulative) / int64(lastTimestamp-time+WINDOW)
		}
	} else {
		if lastTimestamp == oldestTimestamp {
			avgTick = int64(tick)
		} else {
			avgTick = (lastTickCumulative - oldestTickCumulative) / int64(lastTimestamp-oldestTimestamp)
		}
	}

	return big.NewInt(avgTick), nil
}

// / @notice Fetches the timepoints beforeOrAt and atOrAfter a target, i.e. where [beforeOrAt, atOrAfter] is satisfied.
// / The result may be the same timepoint, or adjacent timepoints.
// / @dev The answer must be contained in the array, used when the target is located within the stored timepoint
// / boundaries: older than the most recent timepoint and younger, or the same age as, the oldest timepoint
// / @param self The stored dataStorage array
// / @param time The current block.timestamp
// / @param target The timestamp at which the reserved timepoint should be for
// / @param lastIndex The index of the timepoint that was most recently written to the timepoints array
// / @param oldestIndex The index of the oldest timepoint in the timepoints array
// / @return beforeOrAt The timepoint recorded before, or at, the target
// / @return atOrAfter The timepoint recorded at, or after, the target
func (self *TimepointStorage) binarySearch(
	time uint32,
	target uint32,
	lastIndex uint16,
	oldestIndex uint16,
) (err error, beforeOrAt, atOrAfter Timepoint) {
	left := int64(oldestIndex) // oldest timepoint
	var right int64
	if lastIndex >= oldestIndex {
		right = int64(lastIndex)
	} else {
		right = int64(lastIndex) + UINT16_MODULO
	} // newest timepoint considering one index overflow
	current := (left + right) >> 1 // "middle" point between the boundaries

	// limit number of loop to make sure we won't loop forever because of a bug somewhere
	for i := 0; i < maxBinarySearchLoop; i += 1 {
		beforeOrAt := self.Get(uint16(current)) // checking the "middle" point between the boundaries
		initializedBefore, timestampBefore := beforeOrAt.Initialized, beforeOrAt.BlockTimestamp
		if initializedBefore {
			if lteConsideringOverflow(timestampBefore, target, time) {
				// is current point before or at `target`?
				atOrAfter = self.Get(uint16(current + 1)) // checking the next point after "middle"
				initializedAfter, timestampAfter := atOrAfter.Initialized, atOrAfter.BlockTimestamp
				if initializedAfter {
					if lteConsideringOverflow(target, timestampAfter, time) {
						// is the "next" point after or at `target`?
						return nil, beforeOrAt, atOrAfter // the only fully correct way to finish
					}
					left = current + 1 // "next" point is before the `target`, so looking in the right half
				} else {
					// beforeOrAt is initialized and <= target, and next timepoint is uninitialized
					// should be impossible if initial boundaries and `target` are correct
					return nil, beforeOrAt, beforeOrAt
				}
			} else {
				right = current - 1 // current point is after the `target`, so looking in the left half
			}
		} else {
			// we've landed on an uninitialized timepoint, keep searching higher
			// should be impossible if initial boundaries and `target` are correct
			left = current + 1
		}
		current = (left + right) >> 1 // calculating the new "middle" point index after updating the bounds
	}
	return ErrMaxBinarySearchLoop, Timepoint{}, Timepoint{}
}

// / @dev Reverts if an timepoint at or before the desired timepoint timestamp does not exist.
// / 0 may be passed as `secondsAgo' to return the current cumulative values.
// / If called with a timestamp falling between two timepoints, returns the counterfactual accumulator values
// / at exactly the timestamp between the two timepoints.
// / @param self The stored dataStorage array
// / @param time The current block timestamp
// / @param secondsAgo The amount of time to look back, in seconds, at which point to return an timepoint
// / @param tick The current tick
// / @param index The index of the timepoint that was most recently written to the timepoints array
// / @param oldestIndex The index of the oldest timepoint
// / @param liquidity The current in-range pool liquidity
// / @return targetTimepoint desired timepoint or it's approximation
func (self *TimepointStorage) getSingleTimepoint(
	time uint32,
	secondsAgo uint32,
	tick int24,
	index uint16,
	oldestIndex uint16,
	liquidity *big.Int,
) (Timepoint, error) {
	target := time - secondsAgo

	// if target is newer than last timepoint
	if secondsAgo == 0 || lteConsideringOverflow(self.Get(index).BlockTimestamp, target, time) {
		// lteConsideringOverflow(self[index].blockTimestamp, target, time) -> case3
		last := self.Get(index)
		if last.BlockTimestamp == target {
			return last, nil
		} else {
			// otherwise, we need to add new timepoint
			avgTickBI, err := self._getAverageTick(time, tick, index, oldestIndex, last.BlockTimestamp, last.TickCumulative)
			if err != nil {
				return Timepoint{}, err
			}
			avgTick := int24(avgTickBI.Int64())
			prevTick := tick
			{
				if index != oldestIndex {
					var prevLast Timepoint
					_prevLast := self.Get(index - 1) // considering index underflow
					prevLast.BlockTimestamp = _prevLast.BlockTimestamp
					prevLast.TickCumulative = _prevLast.TickCumulative
					prevTick = int24((last.TickCumulative - prevLast.TickCumulative) / int64(last.BlockTimestamp-prevLast.BlockTimestamp))
				}
			}
			return createNewTimepoint(last, target, tick, prevTick, liquidity, avgTick, bignumber.ZeroBI), nil
		}
	}

	if !lteConsideringOverflow(self.Get(oldestIndex).BlockTimestamp, target, time) {
		return Timepoint{}, errors.New("OLD")
	}
	err, beforeOrAt, atOrAfter := self.binarySearch(time, target, index, oldestIndex)
	if err != nil {
		return Timepoint{}, err
	}

	if target == atOrAfter.BlockTimestamp {
		return atOrAfter, nil // we're at the right boundary
	}

	if target != beforeOrAt.BlockTimestamp {
		// we're in the middle
		timepointTimeDelta := atOrAfter.BlockTimestamp - beforeOrAt.BlockTimestamp
		targetDelta := target - beforeOrAt.BlockTimestamp

		// For gas savings the resulting point is written to beforeAt
		beforeOrAt.TickCumulative += ((atOrAfter.TickCumulative - beforeOrAt.TickCumulative) / int64(timepointTimeDelta)) * int64(targetDelta)
		beforeOrAt.SecondsPerLiquidityCumulative = new(big.Int).Add(beforeOrAt.SecondsPerLiquidityCumulative,
			new(big.Int).Div(
				new(big.Int).Mul(
					new(big.Int).Sub(atOrAfter.SecondsPerLiquidityCumulative, beforeOrAt.SecondsPerLiquidityCumulative),
					big.NewInt(int64(targetDelta)),
				),
				big.NewInt(int64(timepointTimeDelta)),
			),
		)
		beforeOrAt.VolatilityCumulative = new(big.Int).Add(beforeOrAt.VolatilityCumulative,
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Sub(atOrAfter.VolatilityCumulative, beforeOrAt.VolatilityCumulative),
					big.NewInt(int64(timepointTimeDelta)),
				),
				big.NewInt(int64(targetDelta)),
			),
		)
		beforeOrAt.VolumePerLiquidityCumulative = new(big.Int).Add(beforeOrAt.VolumePerLiquidityCumulative,
			new(big.Int).Mul(
				new(big.Int).Div(
					new(big.Int).Sub(atOrAfter.VolumePerLiquidityCumulative, beforeOrAt.VolumePerLiquidityCumulative),
					big.NewInt(int64(timepointTimeDelta)),
				),
				big.NewInt(int64(targetDelta)),
			),
		)
	}

	// we're at the left boundary or at the middle
	return beforeOrAt, nil
}

// / @notice Returns average volatility in the range from time-WINDOW to time
// / @param self The stored dataStorage array
// / @param time The current block.timestamp
// / @param tick The current tick
// / @param index The index of the timepoint that was most recently written to the timepoints array
// / @param liquidity The current in-range pool liquidity
// / @return volatilityAverage The average volatility in the recent range
// / @return volumePerLiqAverage The average volume per liquidity in the recent range
func (self *TimepointStorage) getAverages(
	time uint32,
	tick int24,
	index uint16,
	liquidity *big.Int,
) (error, *big.Int, *big.Int) {
	var oldestIndex uint16
	oldest := self.Get(0)
	nextIndex := index + 1 // considering overflow
	nxt := self.Get(nextIndex)
	if nxt.Initialized {
		oldest = nxt
		oldestIndex = nextIndex
	}

	endOfWindow, err := self.getSingleTimepoint(time, 0, tick, index, oldestIndex, liquidity)
	if err != nil {
		return err, nil, nil
	}
	logger.Debugf("endW %v", endOfWindow.BlockTimestamp)

	oldestTimestamp := oldest.BlockTimestamp
	if lteConsideringOverflow(oldestTimestamp, time-WINDOW, time) {
		startOfWindow, err := self.getSingleTimepoint(time, WINDOW, tick, index, oldestIndex, liquidity)
		if err != nil {
			return err, nil, nil
		}
		logger.Debugf("startW %v", startOfWindow.BlockTimestamp)
		return nil,
			new(big.Int).Div(new(big.Int).Sub(endOfWindow.VolatilityCumulative, startOfWindow.VolatilityCumulative), big.NewInt(WINDOW)),
			new(big.Int).Rsh(new(big.Int).Sub(endOfWindow.VolumePerLiquidityCumulative, startOfWindow.VolumePerLiquidityCumulative), 57)

	} else if time != oldestTimestamp {
		_oldestVolatilityCumulative := oldest.VolatilityCumulative
		_oldestVolumePerLiquidityCumulative := oldest.VolumePerLiquidityCumulative
		logger.Debugf("startW oldestTimestamp %v", oldestTimestamp)
		return nil,
			new(big.Int).Div(
				new(big.Int).Sub(endOfWindow.VolatilityCumulative, _oldestVolatilityCumulative),
				big.NewInt(int64(time-oldestTimestamp)),
			),
			new(big.Int).Rsh(new(big.Int).Sub(endOfWindow.VolumePerLiquidityCumulative, _oldestVolumePerLiquidityCumulative), 57)

	}
	return nil, bignumber.ZeroBI, bignumber.ZeroBI
}

// / @notice Writes an dataStorage timepoint to the array
// / @dev Writable at most once per block. Index represents the most recently written element. index must be tracked externally.
// / @param self The stored dataStorage array
// / @param index The index of the timepoint that was most recently written to the timepoints array
// / @param blockTimestamp The timestamp of the new timepoint
// / @param tick The active tick at the time of the new timepoint
// / @param liquidity The total in-range liquidity at the time of the new timepoint
// / @param volumePerLiquidity The gmean(volumes)/liquidity at the time of the new timepoint
// / @return indexUpdated The new index of the most recently written element in the dataStorage array
func (self *TimepointStorage) write(
	index uint16,
	blockTimestamp uint32,
	tick int24,
	liquidity *big.Int,
	volumePerLiquidity *big.Int,
) (indexUpdated uint16, err error) {
	_last := self.Get(index)
	// early return if we've already written an timepoint this block
	if _last.BlockTimestamp == blockTimestamp {
		return index, nil
	}
	last := _last

	// get next index considering overflow
	indexUpdated = index + 1

	var oldestIndex uint16
	// check if we have overflow in the past
	if self.Get(indexUpdated).Initialized {
		oldestIndex = indexUpdated
	}

	avgTickBI, err := self._getAverageTick(blockTimestamp, tick, index, oldestIndex, last.BlockTimestamp, last.TickCumulative)
	if err != nil {
		return 0, err
	}
	avgTick := int24(avgTickBI.Int64())
	prevTick := tick
	if index != oldestIndex {
		_prevLast := self.Get(index - 1) // considering index underflow
		_prevLastBlockTimestamp := _prevLast.BlockTimestamp
		_prevLastTickCumulative := _prevLast.TickCumulative
		prevTick = int24((last.TickCumulative - _prevLastTickCumulative) / int64(last.BlockTimestamp-_prevLastBlockTimestamp))
	}

	self.Set(indexUpdated, createNewTimepoint(last, blockTimestamp, tick, prevTick, liquidity, avgTick, volumePerLiquidity))
	return
}

// https://github.com/cryptoalgebra/AlgebraV1/blob/dfebf532a27803dafcbf2ba49724740bd6220505/src/core/contracts/DataStorageOperator.sol#L150
func (ts *TimepointStorage) _getNewFee(
	_time uint32,
	_tick int24,
	_index uint16,
	_liquidity *big.Int,
	_feeConf *FeeConfiguration,
) (uint16, error) {
	err, volatilityAverage, volumePerLiqAverage := ts.getAverages(_time, _tick, _index, _liquidity)
	if err != nil {
		return 0, err
	}
	return getFee(
		new(big.Int).Div(volatilityAverage, big.NewInt(15)),
		volumePerLiqAverage,
		_feeConf,
	), nil
}
