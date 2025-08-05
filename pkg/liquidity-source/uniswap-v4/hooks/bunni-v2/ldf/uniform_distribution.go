package ldf

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// UniformDistribution represents a uniform distribution LDF
type UniformDistribution struct {
	tickSpacing int
}

// NewUniformDistribution creates a new UniformDistribution
func NewUniformDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &UniformDistribution{
		tickSpacing: tickSpacing,
	}
}

// Query implements the Query method for UniformDistribution
func (u *UniformDistribution) Query(
	roundedTick,
	twapTick,
	spotPriceTick int,
	ldfParams,
	ldfState [32]byte,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	newLdfState [32]byte,
	shouldSurge bool,
	err error,
) {
	tickLower, tickUpper, shiftMode := u.decodeParams(twapTick, ldfParams)
	initialized, lastTickLower := DecodeState(ldfState)

	if initialized {
		tickLength := tickUpper - tickLower
		minUsableTick := math.MinUsableTick(u.tickSpacing)
		maxUsableTick := math.MaxUsableTick(u.tickSpacing)
		tickLower = max(minUsableTick, EnforceShiftMode(tickLower, int(lastTickLower), shiftMode))
		tickUpper = min(maxUsableTick, tickLower+tickLength)
		shouldSurge = tickLower != int(lastTickLower)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = u.query(
		roundedTick, tickLower, tickUpper,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = u.encodeState(tickLower)
	return
}

// ComputeSwap implements the ComputeSwap method for UniformDistribution
func (u *UniformDistribution) ComputeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	twapTick,
	_ int,
	ldfParams,
	ldfState [32]byte,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	tickLower, tickUpper, shiftMode := u.decodeParams(twapTick, ldfParams)
	initialized, lastTickLower := DecodeState(ldfState)

	if initialized {
		tickLength := tickUpper - tickLower
		tickLower = EnforceShiftMode(tickLower, int(lastTickLower), shiftMode)
		tickUpper = tickLower + tickLength
	}

	return u.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		tickLower,
		tickUpper,
	)
}

// computeSwap computes the swap parameters
func (u *UniformDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	tickLower, tickUpper int,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	if exactIn == zeroForOne {
		// Compute roundedTick by inverting the cumulative amount0
		success, roundedTick = u.inverseCumulativeAmount0(
			inverseCumulativeAmountInput, totalLiquidity, tickLower, tickUpper,
		)
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// Compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = u.cumulativeAmount0(roundedTick+u.tickSpacing, totalLiquidity, tickLower, tickUpper)
		} else {
			cumulativeAmount0_, err = u.cumulativeAmount0(roundedTick, totalLiquidity, tickLower, tickUpper)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = u.cumulativeAmount1(roundedTick, totalLiquidity, tickLower, tickUpper)
		} else {
			cumulativeAmount1_, err = u.cumulativeAmount1(roundedTick-u.tickSpacing, totalLiquidity, tickLower, tickUpper)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		// Compute roundedTick by inverting the cumulative amount1
		success, roundedTick = u.inverseCumulativeAmount1(
			inverseCumulativeAmountInput, totalLiquidity, tickLower, tickUpper,
		)
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// Compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = u.cumulativeAmount1(roundedTick-u.tickSpacing, totalLiquidity, tickLower, tickUpper)
		} else {
			cumulativeAmount1_, err = u.cumulativeAmount1(roundedTick, totalLiquidity, tickLower, tickUpper)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = u.cumulativeAmount0(roundedTick, totalLiquidity, tickLower, tickUpper)
		} else {
			cumulativeAmount0_, err = u.cumulativeAmount0(roundedTick+u.tickSpacing, totalLiquidity, tickLower, tickUpper)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	// Compute swap liquidity
	swapLiquidity, err = u.liquidityDensityX96(roundedTick, tickLower, tickUpper)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}
	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}

// decodeParams decodes the LDF parameters from bytes32
func (u *UniformDistribution) decodeParams(twapTick int, ldfParams [32]byte) (tickLower, tickUpper int, shiftMode ShiftMode) {
	// | shiftMode - 1 byte | tickLowerOrOffset - 3 bytes | tickUpperOrOffset - 3 bytes |
	shiftMode = ShiftMode(ldfParams[0])
	tickLowerOrOffset := int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
	tickUpperOrOffset := int(int32(uint32(ldfParams[4])<<16 | uint32(ldfParams[5])<<8 | uint32(ldfParams[6])))

	if shiftMode != ShiftModeStatic {
		// use rounded TWAP value + offset as tickLower
		tickLower = math.RoundTickSingle(twapTick+tickLowerOrOffset, u.tickSpacing)
		tickUpper = math.RoundTickSingle(twapTick+tickUpperOrOffset, u.tickSpacing)
	} else {
		// static ticks set in params
		tickLower = tickLowerOrOffset
		tickUpper = tickUpperOrOffset
	}

	return
}

// encodeState encodes the state into bytes32
func (u *UniformDistribution) encodeState(tickLower int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((tickLower >> 16) & 0xFF)
	state[2] = byte((tickLower >> 8) & 0xFF)
	state[3] = byte(tickLower & 0xFF)
	return state
}

// query computes the liquidity density and cumulative amounts
func (u *UniformDistribution) query(
	roundedTick, tickLower, tickUpper int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// compute liquidityDensityX96
	liquidityDensityX96, err = u.liquidityDensityX96(roundedTick, tickLower, tickUpper)
	if err != nil {
		return nil, nil, nil, err
	}

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return liquidityDensityX96, uint256.NewInt(0), uint256.NewInt(0), nil
	}

	lengthBig := uint256.NewInt(uint64(length))

	// liquidity = Q96.divUp(length)
	liquidity := math.DivUp(math.Q96, lengthBig)

	sqrtRatioTickLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, nil, nil, err
	}
	sqrtRatioTickUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96 for the rounded tick to the right of the rounded current tick
	if roundedTick+u.tickSpacing >= tickUpper {
		// cumulativeAmount0DensityX96 is just 0
		cumulativeAmount0DensityX96 = uint256.NewInt(0)
	} else if roundedTick+u.tickSpacing <= tickLower {
		cumulativeAmount0DensityX96, err = math.GetAmount0Delta(
			sqrtRatioTickLower, sqrtRatioTickUpper, liquidity, true,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		sqrtPriceRoundedTickPlusSpacing, err := math.GetSqrtPriceAtTick(roundedTick + u.tickSpacing)
		if err != nil {
			return nil, nil, nil, err
		}
		cumulativeAmount0DensityX96, err = math.GetAmount0Delta(
			sqrtPriceRoundedTickPlusSpacing, sqrtRatioTickUpper, liquidity, true,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	// compute cumulativeAmount1DensityX96 for the rounded tick to the left of the rounded current tick
	if roundedTick-u.tickSpacing < tickLower {
		// cumulativeAmount1DensityX96 is just 0
		cumulativeAmount1DensityX96 = uint256.NewInt(0)
	} else if roundedTick >= tickUpper {
		cumulativeAmount1DensityX96, err = math.GetAmount1Delta(
			sqrtRatioTickLower, sqrtRatioTickUpper, liquidity, true,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	} else {
		sqrtPriceRoundedTick, err := math.GetSqrtPriceAtTick(roundedTick)
		if err != nil {
			return nil, nil, nil, err
		}
		cumulativeAmount1DensityX96, err = math.GetAmount1Delta(
			sqrtRatioTickLower, sqrtPriceRoundedTick, liquidity, true,
		)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, nil
}

// cumulativeAmount0 computes the cumulative amount0
func (u *UniformDistribution) cumulativeAmount0(roundedTick int, totalLiquidity *uint256.Int, tickLower, tickUpper int) (*uint256.Int, error) {
	if roundedTick >= tickUpper || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick < tickLower {
		roundedTick = tickLower
	}

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	sqrtPriceRoundedTick, err := math.GetSqrtPriceAtTick(roundedTick)
	if err != nil {
		return nil, err
	}
	sqrtPriceTickUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	// For uniform distribution: totalLiquidity.fullMulX96Up(getAmount0Delta(Q96.divUp(length)))
	// Using the non-carpet version (isCarpet = false)
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))

	amount0, err := math.GetAmount0Delta(
		sqrtPriceRoundedTick,
		sqrtPriceTickUpper,
		liquidityPerTick,
		true, // roundUp
	)
	if err != nil {
		return nil, err
	}

	// Apply the uniform distribution scaling: totalLiquidity.fullMulX96Up(amount0)
	result, err := math.FullMulX96Up(totalLiquidity, amount0)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// cumulativeAmount1 computes the cumulative amount1
func (u *UniformDistribution) cumulativeAmount1(roundedTick int, totalLiquidity *uint256.Int, tickLower, tickUpper int) (*uint256.Int, error) {
	if roundedTick < tickLower || tickLower >= tickUpper {
		return uint256.NewInt(0), nil
	}
	if roundedTick > tickUpper-u.tickSpacing {
		roundedTick = tickUpper - u.tickSpacing
	}

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	sqrtPriceTickLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceRoundedTickPlusSpacing, err := math.GetSqrtPriceAtTick(roundedTick + u.tickSpacing)
	if err != nil {
		return nil, err
	}

	// For uniform distribution: totalLiquidity.fullMulX96Up(getAmount1Delta(Q96.divUp(length)))
	// Using the non-carpet version (isCarpet = false)
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))

	amount1, err := math.GetAmount1Delta(
		sqrtPriceTickLower,
		sqrtPriceRoundedTickPlusSpacing,
		liquidityPerTick,
		true, // roundUp
	)
	if err != nil {
		return nil, err
	}

	// Apply the uniform distribution scaling: totalLiquidity.fullMulX96Up(amount1)
	result, err := math.FullMulX96Up(totalLiquidity, amount1)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// inverseCumulativeAmount0 computes the inverse of cumulative amount0
func (u *UniformDistribution) inverseCumulativeAmount0(cumulativeAmount0_, totalLiquidity *uint256.Int, tickLower, tickUpper int) (bool, int) {
	if cumulativeAmount0_.IsZero() {
		return true, tickUpper
	}

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return false, 0
	}

	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return false, 0
	}

	// For uniform distribution: cumulativeAmount0_.fullMulDiv(Q96, totalLiquidity)
	var scaledAmount uint256.Int
	scaledAmount.Mul(cumulativeAmount0_, math.Q96)
	scaledAmount.Div(&scaledAmount, totalLiquidity)

	// Get next sqrt price from amount0
	// Using Q96.divUp(length) for liquidity
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))
	sqrtPrice, err := math.GetNextSqrtPriceFromAmount0RoundingUp(
		sqrtPriceUpper,
		liquidityPerTick,
		&scaledAmount,
		true,
	)
	if err != nil {
		return false, 0
	}

	// Convert sqrt price to tick
	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	// Round tick to spacing
	roundedTick := math.RoundTickSingle(tick, u.tickSpacing)

	// Ensure roundedTick is within valid range
	if roundedTick < tickLower || roundedTick > tickUpper {
		return false, 0
	}

	// Ensure that roundedTick is not tickUpper when cumulativeAmount0_ is non-zero
	if roundedTick == tickUpper {
		return true, tickUpper - u.tickSpacing
	}

	return true, roundedTick
}

// inverseCumulativeAmount1 computes the inverse of cumulative amount1
func (u *UniformDistribution) inverseCumulativeAmount1(cumulativeAmount1_, totalLiquidity *uint256.Int, tickLower, tickUpper int) (bool, int) {
	if cumulativeAmount1_.IsZero() {
		return true, tickLower - u.tickSpacing
	}

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return false, 0
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return false, 0
	}

	// For uniform distribution: cumulativeAmount1_.fullMulDiv(Q96, totalLiquidity)
	var scaledAmount uint256.Int
	scaledAmount.Mul(cumulativeAmount1_, math.Q96)
	scaledAmount.Div(&scaledAmount, totalLiquidity)

	// Get next sqrt price from amount1
	// Using Q96.divUp(length) for liquidity
	liquidityPerTick := math.DivUp(math.Q96, uint256.NewInt(uint64(length)))
	sqrtPrice, err := math.GetNextSqrtPriceFromAmount1RoundingDown(
		sqrtPriceLower,
		liquidityPerTick,
		&scaledAmount,
		true,
	)
	if err != nil {
		return false, 0
	}

	// Convert sqrt price to tick
	tick, err := math.GetTickAtSqrtPrice(sqrtPrice)
	if err != nil {
		return false, 0
	}

	// Handle edge case where tick == tickUpper
	if tick == tickUpper {
		tick -= 1
	}

	// Round tick to spacing
	roundedTick := math.RoundTickSingle(tick, u.tickSpacing)

	// Ensure roundedTick is within valid range
	if roundedTick < tickLower-u.tickSpacing || roundedTick >= tickUpper {
		return false, 0
	}

	// Ensure that roundedTick is not (tickLower - tickSpacing) when cumulativeAmount1_ is non-zero
	if roundedTick == tickLower-u.tickSpacing {
		return true, tickLower
	}

	return true, roundedTick
}

// liquidityDensityX96 computes the liquidity density at a given tick
func (u *UniformDistribution) liquidityDensityX96(roundedTick, tickLower, tickUpper int) (*uint256.Int, error) {
	if roundedTick < tickLower || roundedTick >= tickUpper {
		return uint256.NewInt(0), nil
	}
	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return uint256.NewInt(0), nil
	}

	var result uint256.Int
	result.Div(math.Q96, uint256.NewInt(uint64(length)))
	return &result, nil
}
