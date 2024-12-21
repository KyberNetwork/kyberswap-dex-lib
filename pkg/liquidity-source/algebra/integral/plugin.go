package integral

import (
	"errors"
	"fmt"
	"sync"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	"github.com/samber/lo"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	ErrTargetIsTooOld = errors.New("target is too old")
)

type TimepointStorage struct {
	mu   sync.RWMutex
	data map[uint16]Timepoint
}

func NewTimepointStorage(data map[uint16]Timepoint) *TimepointStorage {
	return &TimepointStorage{
		data: data,
	}
}

func (s *TimepointStorage) Get(index uint16) Timepoint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.data[index]; ok {
		return v
	}

	return Timepoint{
		Initialized:          false,
		BlockTimestamp:       0,
		TickCumulative:       0,
		VolatilityCumulative: uint256.NewInt(0),
		Tick:                 0,
		AverageTick:          0,
		WindowStartIndex:     0,
	}
}
func (s *TimepointStorage) set(index uint16, v Timepoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[index] = v
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

	avgTick, windowStartIndex, err := s.getAverageTick(blockTimestamp, tick, lastIndex, oldestIndex,
		last.BlockTimestamp, last.TickCumulative)
	if err != nil {
		return 0, 0, err
	}

	if windowStartIndex == indexUpdated {
		windowStartIndex++
	}
	s.set(indexUpdated, createNewTimepoint(last, blockTimestamp, tick, avgTick, windowStartIndex))
	if oldestIndex == indexUpdated {
		oldestIndex++
	}

	return indexUpdated, oldestIndex, nil
}

func (s *TimepointStorage) getAverageTick(currentTime uint32, tick int32, lastIndex, oldestIndex uint16,
	lastTimestamp uint32, lastTickCumulative int64) (int32, uint16, error) {
	self := s.Get(oldestIndex)
	oldestTimestamp, oldestTickCumulative := self.BlockTimestamp, self.TickCumulative

	currentTickCumulative := lastTickCumulative + int64(tick)*int64(currentTime-lastTimestamp)
	if !lteConsideringOverflow(oldestTimestamp, currentTime-WINDOW, currentTime) {
		if currentTime == oldestTimestamp {
			return tick, oldestIndex, nil
		}

		avgTick := (currentTickCumulative - oldestTickCumulative) / int64(currentTime-oldestTimestamp)
		return int32(avgTick), oldestIndex, nil
	}

	if lteConsideringOverflow(lastTimestamp, currentTime-WINDOW, currentTime) {
		return tick, lastIndex, nil
	} else {
		tickCumulativeAtStart, windowStartIndex, err := s.getTickCumulativeAt(currentTime, WINDOW, tick,
			lastIndex, oldestIndex)
		if err != nil {
			return 0, 0, err
		}

		avgTick := (currentTickCumulative - tickCumulativeAtStart) / int64(WINDOW)
		return int32(avgTick), windowStartIndex, nil
	}
}

func (s *TimepointStorage) getTickCumulativeAt(time, secondsAgo uint32, tick int32,
	lastIndex, oldestIndex uint16) (int64, uint16, error) {
	target := time - secondsAgo
	beforeOrAt, atOrAfter, samePoint, indexBeforeOrAt, err := s.getTimepointsAt(time, target, lastIndex, oldestIndex)
	if err != nil {
		return 0, 0, err
	}

	timestampBefore, tickCumulativeBefore := beforeOrAt.BlockTimestamp, beforeOrAt.TickCumulative
	if target == timestampBefore {
		return tickCumulativeBefore, indexBeforeOrAt, nil
	}

	if samePoint {
		return tickCumulativeBefore + int64(tick)*int64(target-timestampBefore), indexBeforeOrAt, nil
	}

	timestampAfter, tickCumulativeAfter := atOrAfter.BlockTimestamp, atOrAfter.TickCumulative

	if target == timestampAfter {
		return tickCumulativeAfter, indexBeforeOrAt + 1, nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	return tickCumulativeBefore + ((tickCumulativeAfter-tickCumulativeBefore)/int64(timepointTimeDelta))*int64(targetDelta),
		indexBeforeOrAt, nil
}

func (s *TimepointStorage) getTimepointsAt(currentTime, target uint32,
	lastIndex, oldestIndex uint16) (beforeOrAt, atOrAfter Timepoint, samePoint bool, indexBeforeOrAt uint16,
	err error) {
	lastTimepoint := s.Get(lastIndex)

	lastTimepointTimestamp := lastTimepoint.BlockTimestamp
	windowStartIndex := lastTimepoint.WindowStartIndex

	if target == currentTime || lteConsideringOverflow(lastTimepointTimestamp, target, currentTime) {
		return lastTimepoint, lastTimepoint, true, lastIndex, nil
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
		return oldestTimepoint, oldestTimepoint, true, oldestIndex, nil
	}

	if lastIndex == oldestIndex+1 {
		return oldestTimepoint, lastTimepoint, false, oldestIndex, nil
	}

	beforeOrAt, atOrAfter, indexBeforeOrAt = s.binarySearch(currentTime, target, lastIndex, oldestIndex, useHeuristic)

	return
}

// getAverageVolatility returns average volatility in the range from currentTime-WINDOW to currentTime
func (s *TimepointStorage) getAverageVolatility(currentTime uint32, tick int32,
	lastIndex, oldestIndex uint16) (*uint256.Int, error) {
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

	oldestTimepoint := s.Get(oldestIndex)
	oldestTimestamp := oldestTimepoint.BlockTimestamp
	if lteConsideringOverflow(oldestTimestamp, currentTime-WINDOW, currentTime) {
		var cumulativeVolatilityAtStart *uint256.Int
		if timeAtLastTimepoint {
			windowStart, windowStartPlus1 := s.Get(windowStartIndex), s.Get(windowStartIndex+1)
			oldestTimestamp, cumulativeVolatilityAtStart = windowStart.BlockTimestamp, windowStart.VolatilityCumulative
			timeDeltaBetweenPoints := windowStartPlus1.BlockTimestamp - oldestTimestamp

			var tmp uint256.Int
			cumulativeVolatilityAtStart = tmp.Add(
				cumulativeVolatilityAtStart,
				tmp.Div(
					tmp.Mul(
						tmp.Sub(
							windowStartPlus1.VolatilityCumulative,
							cumulativeVolatilityAtStart,
						),
						uint256.NewInt(uint64(currentTime-WINDOW-oldestTimestamp)),
					),
					uint256.NewInt(uint64(timeDeltaBetweenPoints)),
				),
			)
		} else {
			var err error
			if cumulativeVolatilityAtStart, err = s.getVolatilityCumulativeAt(currentTime, WINDOW, tick,
				lastIndex, oldestIndex); err != nil {
				return nil, err
			}
		}

		return cumulativeVolatilityAtStart.Div(
			cumulativeVolatilityAtStart.Sub(lastCumulativeVolatility, cumulativeVolatilityAtStart),
			uint256.NewInt(uint64(WINDOW)),
		), nil

	} else if currentTime != oldestTimestamp {
		oldestVolatilityCumulative := oldestTimepoint.VolatilityCumulative
		unbiasedDenominator := currentTime - oldestTimestamp
		if unbiasedDenominator > 1 {
			unbiasedDenominator--
		}

		var tmp uint256.Int
		return tmp.Div(
			tmp.Sub(lastCumulativeVolatility, oldestVolatilityCumulative),
			uint256.NewInt(uint64(unbiasedDenominator)),
		), nil
	}

	return uZERO, nil
}

func (s *TimepointStorage) getVolatilityCumulativeAt(time, secondsAgo uint32, tick int32,
	lastIndex, oldestIndex uint16) (*uint256.Int, error) {
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
		avgTick, _, err := s.getAverageTick(target, tick, lastIndex, oldestIndex, timestampBefore,
			beforeOrAt.TickCumulative)
		if err != nil {
			return nil, err
		}

		return new(uint256.Int).Add(
			volatilityCumulativeBefore,
			volatilityOnRange(target-timestampBefore, tick, tick, beforeOrAt.AverageTick, avgTick),
		), nil
	}

	timestampAfter, volatilityCumulativeAfter := atOrAfter.BlockTimestamp, atOrAfter.VolatilityCumulative
	if target == timestampAfter {
		return volatilityCumulativeAfter, nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	var ret uint256.Int
	return ret.Add(
		volatilityCumulativeBefore,
		ret.Mul(
			ret.Div(
				ret.Sub(
					volatilityCumulativeAfter,
					volatilityCumulativeBefore,
				),
				uint256.NewInt(uint64(timepointTimeDelta)),
			),
			uint256.NewInt(uint64(targetDelta)),
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

func createNewTimepoint(last Timepoint, blockTimestamp uint32, tick, averageTick int32,
	windowStartIndex uint16) Timepoint {
	delta := blockTimestamp - last.BlockTimestamp

	volatility := volatilityOnRange(delta, tick, tick, last.AverageTick, averageTick)

	return Timepoint{
		Initialized:          true,
		BlockTimestamp:       blockTimestamp,
		TickCumulative:       last.TickCumulative + int64(tick)*int64(delta),
		VolatilityCumulative: new(uint256.Int).Add(last.VolatilityCumulative, volatility),
		Tick:                 tick,
		AverageTick:          averageTick,
		WindowStartIndex:     windowStartIndex,
	}
}

func volatilityOnRange(dt uint32, tick0, tick1, avgTick0, avgTick1 int32) *uint256.Int {
	k := uint64((tick1 - tick0) - (avgTick1 - avgTick0))
	b := int64(tick0-avgTick0) * int64(dt)
	sumOfSequence := uint64(dt) * (uint64(dt) + 1)

	var tmp uint256.Int

	// sumOfSquares = sumOfSequence * (2 * dt + 1)
	sumOfSquares := uint256.NewInt(sumOfSequence)
	sumOfSquares.Mul(sumOfSquares, tmp.SetUint64(2*uint64(dt)+1))

	// k^2 * sumOfSquares
	term1 := sumOfSquares.Mul(tmp.SetUint64(k*k), sumOfSquares)

	// 6 * b * k * sumOfSequence
	term2 := uint256.NewInt(6 * k)
	term2.Mul(term2, tmp.SetUint64(uint64(b))).Mul(term2, tmp.SetUint64(sumOfSequence))

	// 6 * dt * b^2
	term3 := uint256.NewInt(uint64(6 * dt))
	term3.Mul(term3, tmp.SetUint64(uint64(b*b)))

	// Calculate volatility = (term1 + term2 + term3) / (6 * dt^2)
	numerator := term1.Add(term1, term2).Add(term1, term3)
	denominator := term2.Mul(uSIX, term2.SetUint64(uint64(dt)*uint64(dt)))

	volatility := numerator.Div(numerator, denominator)
	return volatility
}

func (s *TimepointStorage) binarySearch(
	currentTime uint32,
	target uint32,
	upperIndex uint16,
	lowerIndex uint16,
	withHeuristic bool,
) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt uint16) {
	left := uint32(lowerIndex)
	right := uint32(upperIndex)

	if upperIndex < lowerIndex {
		right += UINT16_MODULO
	}

	beforeOrAt, atOrAfter, left = s.binarySearchInternal(currentTime, target, left, right, withHeuristic)
	return beforeOrAt, atOrAfter, uint16(left)
}

func (s *TimepointStorage) binarySearchInternal(currentTime, target, left, right uint32,
	withHeuristic bool) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt uint32) {
	if withHeuristic && right-left > 2 {
		indexBeforeOrAt = left + 1
	} else {
		indexBeforeOrAt = (left + right) >> 1
	}

	beforeOrAt = s.Get(uint16(indexBeforeOrAt))
	atOrAfter = beforeOrAt

	firstIteration := true
	for {
		initializedBefore, timestampBefore := beforeOrAt.Initialized, beforeOrAt.BlockTimestamp
		if initializedBefore {
			if lteConsideringOverflow(timestampBefore, target, currentTime) {
				atOrAfter = s.Get(uint16(indexBeforeOrAt + 1))
				initializedAfter, timestampAfter := atOrAfter.Initialized, atOrAfter.BlockTimestamp
				if initializedAfter {
					if lteConsideringOverflow(target, timestampAfter, currentTime) {
						return beforeOrAt, atOrAfter, indexBeforeOrAt
					}
					left = indexBeforeOrAt + 1
				} else {
					return beforeOrAt, beforeOrAt, indexBeforeOrAt
				}
			} else {
				right = indexBeforeOrAt - 1
			}
		} else {
			left = indexBeforeOrAt + 1
		}

		useHeuristic := firstIteration && withHeuristic && left == indexBeforeOrAt+1
		if useHeuristic && right-left > 16 {
			indexBeforeOrAt = left + 8
		} else {
			indexBeforeOrAt = (left + right) >> 1
		}

		beforeOrAt = s.Get(uint16(indexBeforeOrAt))
		firstIteration = false
	}
}

func calculateFeeFactors(currentTick, lastTick int32, priceChangeFactor uint16) (*SlidingFeeConfig, error) {
	tickDelta := lo.Clamp(currentTick-lastTick, utils.MinTick, utils.MaxTick)

	var sqrtPriceDelta v3Utils.Uint160
	err := v3Utils.GetSqrtRatioAtTickV2(int(tickDelta), &sqrtPriceDelta)
	if err != nil {
		return nil, err
	}

	priceRatioSquared, err := v3Utils.MulDiv(&sqrtPriceDelta, &sqrtPriceDelta, DOUBLE_FEE_MULTIPLIER)
	if err != nil {
		return nil, err
	}

	priceChangeRatio := priceRatioSquared.Sub(priceRatioSquared, BASE_FEE_MULTIPLIER)

	factor := uint256.NewInt(uint64(priceChangeFactor))
	feeFactorImpact := priceChangeRatio.Div(priceChangeRatio.Mul(priceChangeRatio, factor),
		uint256.NewInt(FACTOR_DENOMINATOR))

	feeFactors := &SlidingFeeConfig{
		ZeroToOneFeeFactor: BASE_FEE_MULTIPLIER,
		OneToZeroFeeFactor: BASE_FEE_MULTIPLIER,
	}

	newZeroToOneFeeFactor := new(uint256.Int).Sub(feeFactors.ZeroToOneFeeFactor, feeFactorImpact)

	twoShift := DOUBLE_FEE_MULTIPLIER

	if newZeroToOneFeeFactor.Sign() > 0 && newZeroToOneFeeFactor.Cmp(twoShift) < 0 {
		feeFactors.ZeroToOneFeeFactor = newZeroToOneFeeFactor
		feeFactors.OneToZeroFeeFactor = new(uint256.Int).Add(feeFactors.OneToZeroFeeFactor, feeFactorImpact)
	} else if newZeroToOneFeeFactor.Sign() <= 0 {
		feeFactors.ZeroToOneFeeFactor = uZERO
		feeFactors.OneToZeroFeeFactor = twoShift
	} else {
		feeFactors.ZeroToOneFeeFactor = twoShift
		feeFactors.OneToZeroFeeFactor = uZERO
	}

	return feeFactors, nil
}

func getInputTokenDelta01(to, from, liquidity *uint256.Int) (*uint256.Int, error) {
	return getToken0Delta(to, from, liquidity, true)
}

func getInputTokenDelta10(to, from, liquidity *uint256.Int) (*uint256.Int, error) {
	return getToken1Delta(from, to, liquidity, true)
}

func getOutputTokenDelta01(to, from, liquidity *uint256.Int) (*uint256.Int, error) {
	return getToken1Delta(to, from, liquidity, false)
}

func getOutputTokenDelta10(to, from, liquidity *uint256.Int) (*uint256.Int, error) {
	return getToken0Delta(from, to, liquidity, false)
}

func getToken0Delta(priceLower, priceUpper, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if priceUpper.Cmp(priceLower) < 0 {
		return nil, errors.New("price upper must not be less than price lower")
	}
	priceDelta := new(uint256.Int).Sub(priceUpper, priceLower)

	liquidityShifted := new(uint256.Int).Lsh(liquidity, RESOLUTION)

	if roundUp {
		division, err := v3Utils.MulDivRoundingUp(priceDelta, liquidityShifted, priceUpper)
		if err != nil {
			return nil, err
		}

		return unsafeDivRoundingUp(division, priceLower), nil
	}

	mulDivResult, overflow := priceDelta.MulDivOverflow(priceDelta, liquidityShifted, priceUpper)
	if overflow {
		return nil, ErrOverflow
	}
	return mulDivResult.Div(mulDivResult, priceLower), nil
}

func getToken1Delta(priceLower, priceUpper, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if priceUpper.Cmp(priceLower) < 0 {
		return nil, errors.New("price upper must not be less than price lower")
	}
	priceDelta := new(uint256.Int).Sub(priceUpper, priceLower)

	if roundUp {
		return v3Utils.MulDivRoundingUp(priceDelta, liquidity, Q96)
	}

	token1Delta, overflow := priceDelta.MulDivOverflow(priceDelta, liquidity, Q96)
	if overflow {
		return nil, ErrOverflow
	}
	return token1Delta, nil
}

func getNewPriceAfterInput(price, liquidity, input *uint256.Int, zeroToOne bool) (*uint256.Int, error) {
	return getNewPrice(price, liquidity, input, zeroToOne, true)
}

func getNewPriceAfterOutput(price, liquidity, output *uint256.Int, zeroToOne bool) (*uint256.Int, error) {
	return getNewPrice(price, liquidity, output, zeroToOne, false)
}

func getNewPrice(
	price, liquidity *uint256.Int,
	amount *uint256.Int,
	zeroToOne, fromInput bool,
) (*uint256.Int, error) {
	if price.Sign() == 0 {
		return nil, fmt.Errorf("price cannot be zero")
	}
	if liquidity.Sign() == 0 {
		return nil, fmt.Errorf("liquidity cannot be zero")
	}
	if amount.Sign() == 0 {
		return new(uint256.Int).Set(price), nil
	}

	liquidityShifted := new(uint256.Int).Lsh(liquidity, RESOLUTION)

	if zeroToOne == fromInput {
		product, overflow := new(uint256.Int).MulOverflow(amount, price)
		if overflow {
			return nil, ErrOverflow
		}

		denominator := new(uint256.Int)
		if fromInput {
			if denominator, overflow = denominator.AddOverflow(liquidityShifted, product); overflow {
				return nil, ErrOverflow
			}
		} else {
			if denominator, overflow = denominator.SubOverflow(liquidityShifted, product); overflow {
				return nil, ErrUnderflow
			}
		}
		resultPrice, err := v3Utils.MulDivRoundingUp(liquidityShifted, price, denominator)
		if err != nil {
			return nil, err
		} else if resultPrice.BitLen() > 160 {
			return nil, ErrOverflow
		}

		return resultPrice, nil
	} else {
		if fromInput {
			var (
				shiftedAmount *uint256.Int
				overflow      bool
			)
			if amount.BitLen() < 160 {
				shiftedAmount = new(uint256.Int).Lsh(amount, RESOLUTION)
				shiftedAmount.Div(shiftedAmount, liquidity)
			} else {
				shiftedAmount, overflow = new(uint256.Int).MulDivOverflow(amount,
					new(uint256.Int).Lsh(uONE, RESOLUTION), liquidity)
				if overflow {
					return nil, ErrOverflow
				}
			}

			resultPrice, overflow := shiftedAmount.AddOverflow(price, shiftedAmount)
			if overflow || resultPrice.BitLen() > 160 {
				return nil, ErrOverflow
			}
			return resultPrice, nil
		} else {
			var (
				shiftedAmount *uint256.Int
				err           error
			)
			if amount.BitLen() < 160 {
				shiftedAmount = new(uint256.Int).Lsh(amount, RESOLUTION)
				shiftedAmount = unsafeDivRoundingUp(shiftedAmount, liquidity)
			} else {
				shiftedAmount, err = v3Utils.MulDivRoundingUp(amount, new(uint256.Int).Lsh(uONE, RESOLUTION), liquidity)
				if err != nil {
					return nil, err
				}
			}

			resultPrice, overflow := shiftedAmount.SubOverflow(price, shiftedAmount)
			if overflow {
				return nil, ErrUnderflow
			} else if resultPrice.BitLen() > 160 {
				return nil, ErrOverflow
			}
			return resultPrice, nil
		}
	}
}

func lteConsideringOverflow(a, b, currentTime uint32) bool {
	res := a > currentTime

	if res == (b > currentTime) {
		res = a <= b
	}

	return res
}
