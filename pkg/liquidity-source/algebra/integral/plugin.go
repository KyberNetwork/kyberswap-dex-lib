package integral

import (
	"errors"
	"fmt"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"

	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
)

var (
	ErrTargetIsTooOld = errors.New("target is too old")
)

type TimepointStorage struct {
	data map[uint16]Timepoint
}

func NewTimepointStorage(data map[uint16]Timepoint) *TimepointStorage {
	return &TimepointStorage{
		data: data,
	}
}

func (s *TimepointStorage) Get(index uint16) Timepoint {
	if v, ok := s.data[index]; ok {
		logger.Debugf("access exists %v %v", index, v)
		return v
	}

	logger.Debugf("access none %v", index)

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

	avgTick, windowStartIndex, err := s.getAverageTickCasted(blockTimestamp, tick, lastIndex, oldestIndex, last.BlockTimestamp, last.TickCumulative)
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

func (s *TimepointStorage) getAverageTick(currentTime uint32, tick int32, lastIndex, oldestIndex uint16, lastTimestamp uint32, lastTickCumulative int64) (*int256.Int, *uint256.Int, error) {
	self := s.Get(oldestIndex)

	oldestTimestamp, oldestTickCumulative := self.BlockTimestamp, self.TickCumulative

	currentTickCumulative := lastTickCumulative + int64(tick)*int64(uint64(currentTime-lastTimestamp))
	if !lteConsideringOverflow(oldestTimestamp, currentTime-WINDOW, currentTime) {
		if currentTime == oldestTimestamp {
			return int256.NewInt(int64(tick)), uint256.NewInt(uint64(oldestIndex)), nil
		}

		return int256.NewInt((currentTickCumulative - oldestTickCumulative) / int64(uint64(currentTime-oldestTimestamp))), uint256.NewInt(uint64(oldestIndex)), nil
	}

	if lteConsideringOverflow(lastTimestamp, currentTime-WINDOW, currentTime) {
		return int256.NewInt(int64(tick)), uint256.NewInt(uint64(lastIndex)), nil
	} else {
		tickCumulativeAtStart, windowStartIndex, err := s.getTickCumulativeAt(currentTime, WINDOW, tick, lastIndex, oldestIndex)
		if err != nil {
			return nil, nil, err
		}

		avgTick := (currentTickCumulative - tickCumulativeAtStart) / int64(WINDOW)

		return int256.NewInt(avgTick), windowStartIndex, nil
	}
}

func (s *TimepointStorage) getTickCumulativeAt(time, secondsAgo uint32, tick int32, lastIndex, oldestIndex uint16) (int64, *uint256.Int, error) {
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
		return tickCumulativeAfter, new(uint256.Int).Add(indexBeforeOrAt, uONE), nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	return tickCumulativeBefore + ((tickCumulativeAfter-tickCumulativeBefore)/int64(uint64(timepointTimeDelta)))*int64(uint64(targetDelta)),
		indexBeforeOrAt, nil
}

func (s *TimepointStorage) getTimepointsAt(currentTime, target uint32, lastIndex, oldestIndex uint16) (
	beforeOrAt, atOrAfter Timepoint,
	samePoint bool,
	indexBeforeOrAt *uint256.Int,
	err error,
) {
	lastTimepoint := s.Get(lastIndex)

	lastTimepointTimestamp := lastTimepoint.BlockTimestamp
	windowStartIndex := lastTimepoint.WindowStartIndex

	if target == currentTime || lteConsideringOverflow(lastTimepointTimestamp, target, currentTime) {
		return lastTimepoint, lastTimepoint, true, uint256.NewInt(uint64(lastIndex)), nil
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
		return oldestTimepoint, oldestTimepoint, true, uint256.NewInt(uint64(oldestIndex)), nil
	}

	if lastIndex == oldestIndex+1 {
		return oldestTimepoint, lastTimepoint, false, uint256.NewInt(uint64(oldestIndex)), nil
	}

	beforeOrAt, atOrAfter, indexBeforeOrAt = s.binarySearch(currentTime, target, lastIndex, oldestIndex, useHeuristic)

	return
}

func (s *TimepointStorage) getAverageVolatility(currentTime uint32, tick int32, lastIndex, oldestIndex uint16) (*uint256.Int, error) {
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
		var cumulativeVolatilityAtStart *uint256.Int
		if timeAtLastTimepoint {
			oldestTimestamp, cumulativeVolatilityAtStart = s.Get(windowStartIndex).BlockTimestamp, s.Get(windowStartIndex).VolatilityCumulative

			timeDeltaBetweenPoints := s.Get(windowStartIndex+1).BlockTimestamp - oldestTimestamp

			cumulativeVolatilityAtStart = cumulativeVolatilityAtStart.Add(
				cumulativeVolatilityAtStart,
				new(uint256.Int).Div(
					new(uint256.Int).Mul(
						new(uint256.Int).Sub(
							s.Get(windowStartIndex+1).VolatilityCumulative,
							cumulativeVolatilityAtStart,
						),
						uint256.NewInt(uint64(currentTime-WINDOW-oldestTimestamp)),
					),
					uint256.NewInt(uint64(timeDeltaBetweenPoints)),
				),
			)
		} else {
			var err error
			cumulativeVolatilityAtStart, err = s.getVolatilityCumulativeAt(currentTime, WINDOW, tick, lastIndex, oldestIndex)
			if err != nil {
				return nil, err
			}
		}

		return new(uint256.Int).Div(
			new(uint256.Int).Sub(lastCumulativeVolatility, cumulativeVolatilityAtStart),
			uint256.NewInt(uint64(WINDOW)),
		), nil

	} else if currentTime != oldestTimestamp {
		oldestVolatilityCumulative := s.Get(oldestIndex).VolatilityCumulative
		unbiasedDenominator := currentTime - oldestTimestamp
		if unbiasedDenominator > 1 {
			unbiasedDenominator--
		}

		return new(uint256.Int).Div(
			new(uint256.Int).Sub(lastCumulativeVolatility, oldestVolatilityCumulative),
			uint256.NewInt(uint64(unbiasedDenominator)),
		), nil
	}

	return nil, nil
}

func (s *TimepointStorage) getVolatilityCumulativeAt(time, secondsAgo uint32, tick int32, lastIndex, oldestIndex uint16) (*uint256.Int, error) {
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

		return new(uint256.Int).Add(
			volatilityCumulativeBefore,
			volatilityOnRange(uint256.NewInt(uint64(target-timestampBefore)),
				uint256.NewInt(uint64(tick)),
				uint256.NewInt(uint64(tick)),
				uint256.NewInt(uint64(beforeOrAt.AverageTick)),
				uint256.NewInt(uint64(avgTick)),
			),
		), nil
	}

	timestampAfter, volatilityCumulativeAfter := atOrAfter.BlockTimestamp, atOrAfter.VolatilityCumulative
	if target == timestampAfter {
		return volatilityCumulativeAfter, nil
	}

	timepointTimeDelta, targetDelta := timestampAfter-timestampBefore, target-timestampBefore

	return new(uint256.Int).Add(
		volatilityCumulativeBefore,
		new(uint256.Int).Mul(
			new(uint256.Int).Div(
				new(uint256.Int).Sub(
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

func createNewTimepoint(last Timepoint, blockTimestamp uint32, tick, averageTick int32, windowStartIndex uint16) Timepoint {
	delta := blockTimestamp - last.BlockTimestamp

	volatility := volatilityOnRange(uint256.NewInt(uint64(delta)), uint256.NewInt(uint64(tick)),
		uint256.NewInt(uint64(tick)), uint256.NewInt(uint64(last.AverageTick)), uint256.NewInt(uint64(averageTick)))

	last.Initialized = true
	last.BlockTimestamp = blockTimestamp
	last.TickCumulative += int64(tick) * int64(delta)
	last.VolatilityCumulative = new(uint256.Int).Add(last.VolatilityCumulative, volatility)
	last.Tick = tick
	last.AverageTick = averageTick
	last.WindowStartIndex = windowStartIndex

	return last
}

func volatilityOnRange(dt, tick0, tick1, avgTick0, avgTick1 *uint256.Int) *uint256.Int {
	// (k = (tick1 - tick0) - (avgTick1 - avgTick0))
	k := new(uint256.Int).Sub(tick1, tick0)
	k.Sub(k, new(uint256.Int).Sub(avgTick1, avgTick0))

	// (b = (tick0 - avgTick0) * dt)
	b := new(uint256.Int).Sub(tick0, avgTick0)
	b.Mul(b, dt)

	// sumOfSequence = dt * (dt + 1)
	sumOfSequence := new(uint256.Int).Add(dt, uONE)
	sumOfSequence.Mul(sumOfSequence, dt)

	// sumOfSquares = sumOfSequence * (2 * dt + 1)
	sumOfSquares := new(uint256.Int).Mul(uTWO, dt)
	sumOfSquares.Add(sumOfSquares, uONE)
	sumOfSquares.Mul(sumOfSquares, sumOfSequence)

	// k^2
	kSquared := new(uint256.Int).Mul(k, k)

	// b^2
	bSquared := new(uint256.Int).Mul(b, b)

	// k^2 * sumOfSquares
	term1 := new(uint256.Int).Mul(kSquared, sumOfSquares)

	// 6 * b * k * sumOfSequence
	term2 := new(uint256.Int).Mul(b, k)
	term2.Mul(term2, sumOfSequence)
	term2.Mul(term2, uSIX)

	// 6 * dt * b^2
	term3 := new(uint256.Int).Mul(bSquared, dt)
	term3.Mul(term3, uSIX)

	// dt^2
	dtSquared := new(uint256.Int).Mul(dt, dt)

	// Calculate volatility = (term1 + term2 + term3) / (6 * dt^2)
	denominator := new(uint256.Int).Mul(dtSquared, uSIX)
	numerator := new(uint256.Int).Add(term1, term2)
	numerator.Add(numerator, term3)

	volatility := new(uint256.Int).Div(numerator, denominator)

	return volatility
}

func (s *TimepointStorage) binarySearch(
	currentTime uint32,
	target uint32,
	upperIndex uint16,
	lowerIndex uint16,
	withHeuristic bool,
) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt *uint256.Int) {
	var right *uint256.Int

	left := uint256.NewInt(uint64(lowerIndex))

	if upperIndex < lowerIndex {
		right = uint256.NewInt(uint64(upperIndex) + UINT16_MODULO)
	} else {
		right = uint256.NewInt(uint64(upperIndex))
	}

	return s.binarySearchInternal(currentTime, target, left, right, withHeuristic)
}

func (s *TimepointStorage) binarySearchInternal(
	currentTime uint32,
	target uint32,
	left,
	right *uint256.Int,
	withHeuristic bool,
) (beforeOrAt, atOrAfter Timepoint, indexBeforeOrAt *uint256.Int) {
	if withHeuristic && new(uint256.Int).Sub(right, left).Cmp(uTWO) > 0 {
		indexBeforeOrAt = new(uint256.Int).Add(left, uONE)
	} else {
		indexBeforeOrAt = new(uint256.Int).Rsh(
			new(uint256.Int).Add(left, right),
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
					left = new(uint256.Int).Add(indexBeforeOrAt, uONE)
				} else {
					return beforeOrAt, beforeOrAt, indexBeforeOrAt
				}
			} else {
				right = new(uint256.Int).Sub(indexBeforeOrAt, uONE)
			}
		} else {
			left = new(uint256.Int).Add(indexBeforeOrAt, uONE)
		}

		useHeuristic := firstIteration && withHeuristic && left.Cmp(new(uint256.Int).Add(indexBeforeOrAt, uONE)) == 0
		if useHeuristic && new(uint256.Int).Sub(right, left).Cmp(uSIXTEEN) > 0 {
			indexBeforeOrAt = indexBeforeOrAt.Add(left, uEIGHT)
		} else {
			indexBeforeOrAt = indexBeforeOrAt.Rsh(
				new(uint256.Int).Add(left, right),
				1,
			)
		}

		beforeOrAt = s.Get(uint16(indexBeforeOrAt.Uint64()))
		firstIteration = false
	}
}

func calculateFeeFactors(currentTick, lastTick int32, priceChangeFactor uint16) (FeeFactors, error) {
	tickDelta := new(int256.Int).Sub(int256.NewInt(int64(currentTick)), int256.NewInt(int64(lastTick)))
	if tickDelta.Int64() > int64(utils.MaxTick) {
		tickDelta = int256.NewInt(int64(utils.MaxTick))
	} else if tickDelta.Int64() < int64(utils.MinTick) {
		tickDelta = int256.NewInt(int64(utils.MinTick))
	}

	sqrtPriceDelta, err := v3Utils.GetSqrtRatioAtTick(int(tickDelta.Uint64()))
	if err != nil {
		return FeeFactors{}, err
	}

	sqrtPriceDelta256 := uint256.MustFromBig(sqrtPriceDelta)

	priceRatioSquared, err := v3Utils.MulDiv(sqrtPriceDelta256, sqrtPriceDelta256, DOUBLE_FEE_MULTIPLIER)
	if err != nil {
		return FeeFactors{}, err
	}

	priceChangeRatio := new(uint256.Int).Sub(priceRatioSquared, BASE_FEE_MULTIPLIER)

	factor := new(uint256.Int).SetUint64(uint64(priceChangeFactor))
	feeFactorImpact := new(uint256.Int).Div(new(uint256.Int).Mul(priceChangeRatio, factor), uint256.NewInt(FACTOR_DENOMINATOR))

	feeFactors := FeeFactors{
		ZeroToOneFeeFactor: BASE_FEE_MULTIPLIER,
		OneToZeroFeeFactor: BASE_FEE_MULTIPLIER,
	}

	newZeroToOneFeeFactor := new(uint256.Int).Sub(feeFactors.ZeroToOneFeeFactor, feeFactorImpact)

	twoShift := DOUBLE_FEE_MULTIPLIER

	if newZeroToOneFeeFactor.Cmp(uZERO) > 0 && newZeroToOneFeeFactor.Cmp(twoShift) < 0 {
		feeFactors.ZeroToOneFeeFactor = newZeroToOneFeeFactor
		feeFactors.OneToZeroFeeFactor = new(uint256.Int).Add(feeFactors.OneToZeroFeeFactor, feeFactorImpact)
	} else if newZeroToOneFeeFactor.Cmp(uZERO) <= 0 {
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
	priceDelta := new(uint256.Int).Sub(priceUpper, priceLower)
	if priceDelta.Cmp(priceUpper) >= 0 {
		return nil, errors.New("price delta must be greater than price upper")
	}

	liquidityShifted := new(uint256.Int).Lsh(liquidity, RESOLUTION)

	if roundUp {
		division, err := v3Utils.MulDivRoundingUp(priceDelta, liquidityShifted, priceUpper)
		if err != nil {
			return nil, err
		}

		token0Delta, err := unsafeDivRoundingUp(division, priceLower)
		if err != nil {
			return nil, err
		}

		return token0Delta, nil
	} else {
		mulDivResult, err := v3Utils.MulDiv(priceDelta, liquidityShifted, priceUpper)
		if err != nil {
			return nil, err
		}

		token0Delta := new(uint256.Int).Div(mulDivResult, priceLower)

		return token0Delta, nil
	}
}

func getToken1Delta(priceLower, priceUpper, liquidity *uint256.Int, roundUp bool) (*uint256.Int, error) {
	if priceUpper.Cmp(priceLower) < 0 {
		return nil, errors.New("price upper must be greater than price lower")
	}

	priceDelta := new(uint256.Int).Sub(priceUpper, priceLower)

	if roundUp {
		return v3Utils.MulDivRoundingUp(priceDelta, liquidity, Q96)
	} else {
		token1Delta, err := v3Utils.MulDiv(priceDelta, liquidity, Q96)
		if err != nil {
			return nil, err
		}

		return token1Delta, nil
	}
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
		if fromInput {
			product := new(uint256.Int).Mul(amount, price)
			if new(uint256.Int).Div(product, amount).Cmp(price) != 0 {
				return nil, fmt.Errorf("product overflow")
			}

			denominator := new(uint256.Int).Add(liquidityShifted, product)
			if denominator.Cmp(liquidityShifted) < 0 {
				return nil, fmt.Errorf("denominator underflow")
			}

			resultPrice, err := v3Utils.MulDivRoundingUp(liquidityShifted, price, denominator)
			if err != nil {
				return nil, err
			}

			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}

			return resultPrice, nil
		} else {
			product := new(uint256.Int).Mul(amount, price)
			if new(uint256.Int).Div(product, amount).Cmp(price) != 0 {
				return nil, fmt.Errorf("product overflow")
			}
			if liquidityShifted.Cmp(product) <= 0 {
				return nil, fmt.Errorf("denominator underflow")
			}

			denominator := new(uint256.Int).Sub(liquidityShifted, product)

			resultPrice, err := v3Utils.MulDivRoundingUp(liquidityShifted, price, denominator)
			if err != nil {
				return nil, err
			}
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}

			return resultPrice, nil
		}
	} else {
		if fromInput {
			var (
				shiftedAmount = new(uint256.Int)
				err           error
			)
			if amount.Cmp(new(uint256.Int).Sub(new(uint256.Int).Lsh(uONE, 160), uONE)) <= 0 {
				shiftedAmount = new(uint256.Int).Lsh(amount, RESOLUTION)
				shiftedAmount.Div(shiftedAmount, liquidity)
			} else {
				shiftedAmount, err = v3Utils.MulDiv(amount, new(uint256.Int).Lsh(uONE, RESOLUTION), liquidity)
				if err != nil {
					return nil, err
				}
			}

			resultPrice := new(uint256.Int).Add(price, shiftedAmount)
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
			}
			return resultPrice, nil
		} else {
			var (
				quotient *uint256.Int
				err      error
			)
			if amount.Cmp(new(uint256.Int).Sub(new(uint256.Int).Lsh(uONE, 160), uONE)) <= 0 {
				shiftedAmount := new(uint256.Int).Lsh(amount, RESOLUTION)
				quotient, err = unsafeDivRoundingUp(shiftedAmount, liquidity)
				if err != nil {
					return nil, err
				}
			} else {
				quotient, err = v3Utils.MulDivRoundingUp(amount, new(uint256.Int).Lsh(uONE, RESOLUTION), liquidity)
				if err != nil {
					return nil, err
				}
			}

			if price.Cmp(quotient) <= 0 {
				return nil, fmt.Errorf("price must be greater than quotient")
			}

			resultPrice := new(uint256.Int).Sub(price, quotient)
			if resultPrice.BitLen() > 160 {
				return nil, fmt.Errorf("resulting price exceeds 160 bits")
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
