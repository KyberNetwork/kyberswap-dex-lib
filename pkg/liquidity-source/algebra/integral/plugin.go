package integral

import (
	"cmp"
	"sync"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
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

	if target == timestampAfter || timestampAfter == timestampBefore {
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

			var (
				tmp              uint256.Int
				windowDelta      uint256.Int
				timeDeltaUint256 uint256.Int
			)
			windowDelta.SetUint64(uint64(currentTime - WINDOW - oldestTimestamp))
			timeDeltaUint256.SetUint64(uint64(timeDeltaBetweenPoints))
			cumulativeVolatilityAtStart = tmp.Add(
				cumulativeVolatilityAtStart,
				tmp.Div(
					tmp.Mul(
						tmp.Sub(
							windowStartPlus1.VolatilityCumulative,
							cumulativeVolatilityAtStart,
						),
						&windowDelta,
					),
					&timeDeltaUint256,
				),
			)
		} else {
			var err error
			if cumulativeVolatilityAtStart, err = s.getVolatilityCumulativeAt(currentTime, WINDOW, tick,
				lastIndex, oldestIndex); err != nil {
				return nil, err
			}
		}

		var window uint256.Int
		window.SetUint64(uint64(WINDOW))
		return cumulativeVolatilityAtStart.Div(
			cumulativeVolatilityAtStart.Sub(lastCumulativeVolatility, cumulativeVolatilityAtStart),
			&window,
		), nil

	} else if currentTime != oldestTimestamp {
		oldestVolatilityCumulative := oldestTimepoint.VolatilityCumulative
		unbiasedDenominator := currentTime - oldestTimestamp
		if unbiasedDenominator > 1 {
			unbiasedDenominator--
		}

		var (
			tmp                  uint256.Int
			unbiasedDenominatorU uint256.Int
		)
		unbiasedDenominatorU.SetUint64(uint64(unbiasedDenominator))
		return tmp.Div(
			tmp.Sub(lastCumulativeVolatility, oldestVolatilityCumulative),
			&unbiasedDenominatorU,
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

	var (
		ret                 uint256.Int
		timepointTimeDeltaU uint256.Int
		targetDeltaU        uint256.Int
	)
	timepointTimeDeltaU.SetUint64(uint64(timepointTimeDelta))
	targetDeltaU.SetUint64(uint64(targetDelta))
	return ret.Add(
		volatilityCumulativeBefore,
		ret.Mul(
			ret.Div(
				ret.Sub(
					volatilityCumulativeAfter,
					volatilityCumulativeBefore,
				),
				&timepointTimeDeltaU,
			),
			&targetDeltaU,
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
	k := int64((tick1 - tick0) - (avgTick1 - avgTick0))
	b := int64(tick0-avgTick0) * int64(dt)
	sumOfSequence := uint64(dt) * (uint64(dt) + 1)

	var tmp uint256.Int

	// sumOfSquares = sumOfSequence * (2 * dt + 1)
	var sumOfSquares uint256.Int
	sumOfSquares.SetUint64(sumOfSequence)
	sumOfSquares.Mul(&sumOfSquares, tmp.SetUint64(2*uint64(dt)+1))

	// k^2 * sumOfSquares
	term1 := sumOfSquares.Mul(tmp.SetUint64(uint64(k*k)), &sumOfSquares)

	// 6 * b * k * sumOfSequence
	term2, tmp2 := big256.NewI(6*k), big256.NewI(b)
	term2.Mul(term2, tmp2).Mul(term2, tmp.SetUint64(sumOfSequence))

	// 6 * dt * b^2
	term3 := tmp2.SetUint64(uint64(6 * dt))
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

	beforeOrAt = s.Get(uint16(indexBeforeOrAt)) // this downcast takes care of modulo
	atOrAfter = beforeOrAt

	firstIteration := true
	for left <= right {
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
	return beforeOrAt, beforeOrAt, indexBeforeOrAt
}

func (p *PoolSimulator) calculateFeeFactors(currentTick, lastTick int32, priceChangeFactor uint16) (*SlidingFeeConfig, error) {
	tickDelta := lo.Clamp(currentTick-lastTick, utils.MinTick, utils.MaxTick)

	var sqrtPriceDelta v3Utils.Uint160
	err := v3Utils.GetSqrtRatioAtTickV2(int(tickDelta), &sqrtPriceDelta)
	if err != nil {
		return nil, err
	}

	priceRatioSquared, err := v3Utils.MulDiv(&sqrtPriceDelta, &sqrtPriceDelta, FEE_FACTOR_MULTIPLIER)
	if err != nil {
		return nil, err
	}
	priceChangeRatio := priceRatioSquared.Sub(priceRatioSquared, FEE_FACTOR_MULTIPLIER)

	var factor uint256.Int
	factor.SetUint64(uint64(priceChangeFactor))
	feeFactorImpact := priceChangeRatio.Mul(priceChangeRatio, &factor).Div(priceChangeRatio, FACTOR_DENOMINATOR)

	feeFactors := p.slidingFee
	newZeroToOneFeeFactor := feeFactors.ZeroToOneFeeFactor.Sub(feeFactors.ZeroToOneFeeFactor, feeFactorImpact)
	if 0 < newZeroToOneFeeFactor.Sign() && newZeroToOneFeeFactor.Cmp(DOUBLE_FEE_MULTIPLIER) < 0 {
		feeFactors.ZeroToOneFeeFactor = newZeroToOneFeeFactor
		feeFactors.OneToZeroFeeFactor.Add(feeFactors.OneToZeroFeeFactor, feeFactorImpact)
	} else if newZeroToOneFeeFactor.Sign() <= 0 {
		feeFactors.ZeroToOneFeeFactor.Clear()
		feeFactors.OneToZeroFeeFactor.Set(DOUBLE_FEE_MULTIPLIER)
	} else {
		feeFactors.ZeroToOneFeeFactor.Set(DOUBLE_FEE_MULTIPLIER)
		feeFactors.OneToZeroFeeFactor.Clear()
	}
	return feeFactors, nil
}

func getInputTokenDelta01(res, to, from, liquidity *uint256.Int) error {
	return getToken0Delta(res, to, from, liquidity, true)
}

func getInputTokenDelta10(res, to, from, liquidity *uint256.Int) error {
	return getToken1Delta(res, from, to, liquidity, true)
}

func getOutputTokenDelta01(res, to, from, liquidity *uint256.Int) error {
	return getToken1Delta(res, to, from, liquidity, false)
}

func getOutputTokenDelta10(res, to, from, liquidity *uint256.Int) error {
	return getToken0Delta(res, from, to, liquidity, false)
}

// https://github.com/cryptoalgebra/Algebra/blob/357ae6b/src/core/contracts/libraries/TokenDeltaMath.sol#L20
func getToken0Delta(res, priceLower, priceUpper, liquidity *uint256.Int, roundUp bool) error {
	if priceLower.Sign() < 0 {
		return ErrInvalidPriceLower
	}
	var priceDelta, liquidityShifted uint256.Int
	priceDelta.Sub(priceUpper, priceLower)
	liquidityShifted.Lsh(liquidity, RESOLUTION)

	if roundUp {
		if err := v3Utils.MulDivRoundingUpV2(&priceDelta, &liquidityShifted, priceUpper, res); err != nil {
			return err
		}
		v3Utils.DivRoundingUp(res, priceLower, res)
		return nil
	}

	if _, overflow := res.MulDivOverflow(&priceDelta, &liquidityShifted, priceUpper); overflow {
		return ErrOverflow
	}
	res.Div(res, priceLower)
	return nil
}

// https://github.com/cryptoalgebra/Algebra/blob/357ae6b/src/core/contracts/libraries/TokenDeltaMath.sol#L39
func getToken1Delta(res, priceLower, priceUpper, liquidity *uint256.Int, roundUp bool) error {
	if priceUpper.Cmp(priceLower) < 0 {
		return ErrInvalidPriceUpperLower
	}
	// Same mulDiv by Q96 as TokenDeltaMath, as a shift.
	if err := uniswapv3.GetAmount1DeltaV2(priceLower, priceUpper, liquidity, roundUp, res); err != nil {
		return lo.Ternary(roundUp, err, ErrOverflow)
	}
	return nil
}

func getNewPriceAfterInput(res, price, liquidity, input *uint256.Int, zeroToOne bool) error {
	return getNewPrice(res, price, liquidity, input, zeroToOne, true)
}

func getNewPriceAfterOutput(res, price, liquidity, output *uint256.Int, zeroToOne bool) error {
	return getNewPrice(res, price, liquidity, output, zeroToOne, false)
}

// getNewPrice writes the price after moving amount into res, which must not alias the inputs.
func getNewPrice(res, price, liquidity, amount *uint256.Int, zeroToOne, fromInput bool) error {
	if price.IsZero() {
		return ErrZeroPrice
	} else if liquidity.IsZero() {
		return ErrZeroLiquidity
	} else if amount.IsZero() {
		res.Set(price)
		return nil
	}

	var liquidityShifted uint256.Int
	liquidityShifted.Lsh(liquidity, RESOLUTION)

	if zeroToOne == fromInput {
		var product uint256.Int
		if _, overflow := product.MulOverflow(amount, price); overflow {
			return ErrOverflow
		}

		var denominator uint256.Int
		if fromInput {
			if _, overflow := denominator.AddOverflow(&liquidityShifted, &product); overflow {
				return ErrOverflow
			}
		} else {
			if _, overflow := denominator.SubOverflow(&liquidityShifted, &product); overflow {
				return ErrUnderflow
			}
		}
		if err := v3Utils.MulDivRoundingUpV2(&liquidityShifted, price, &denominator, res); err != nil {
			return err
		} else if res.BitLen() > 160 {
			return ErrOverflow
		}
		return nil
	}

	var shiftedAmount uint256.Int
	if fromInput {
		if amount.BitLen() < 160 {
			shiftedAmount.Lsh(amount, RESOLUTION)
			shiftedAmount.Div(&shiftedAmount, liquidity)
		} else {
			shiftedAmount.Lsh(uONE, RESOLUTION)
			if _, overflow := shiftedAmount.MulDivOverflow(amount, &shiftedAmount, liquidity); overflow {
				return ErrOverflow
			}
		}

		if _, overflow := res.AddOverflow(price, &shiftedAmount); overflow || res.BitLen() > 160 {
			return ErrOverflow
		}
		return nil
	}

	if amount.BitLen() < 160 {
		shiftedAmount.Lsh(amount, RESOLUTION)
		v3Utils.DivRoundingUp(&shiftedAmount, liquidity, &shiftedAmount)
	} else {
		var oneShifted uint256.Int
		oneShifted.Lsh(uONE, RESOLUTION)
		if err := v3Utils.MulDivRoundingUpV2(amount, &oneShifted, liquidity, &shiftedAmount); err != nil {
			return err
		}
	}

	if _, underflow := res.SubOverflow(price, &shiftedAmount); underflow {
		return ErrUnderflow
	} else if res.BitLen() > 160 {
		return ErrOverflow
	}
	return nil
}

// lteConsideringOverflow returns true if a <= b with c as greatest value anchor for overflow checking.
// a <= b <= c | true
// b <= c <  a | true
// c <  a <= b | true
// a <= c <  b | false
// b <  a <= c | false
// c <  b <  a | false
func lteConsideringOverflow[T cmp.Ordered](a, b, currentTime T) bool {
	return algebra.LteConsideringOverflow(a, b, currentTime)
}
