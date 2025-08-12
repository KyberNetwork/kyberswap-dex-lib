package ldf

import (
	uniformLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/uniform"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"
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
	tickLower, tickUpper, shiftMode := uniformLib.DecodeParams(u.tickSpacing, twapTick, ldfParams)
	initialized, lastTickLower := DecodeState(ldfState)

	if initialized {
		tickLength := tickUpper - tickLower
		minUsableTick := math.MinUsableTick(u.tickSpacing)
		maxUsableTick := math.MaxUsableTick(u.tickSpacing)
		tickLower = max(minUsableTick, shiftmode.EnforceShiftMode(tickLower, int(lastTickLower), shiftMode))
		tickUpper = min(maxUsableTick, tickLower+tickLength)
		shouldSurge = tickLower != int(lastTickLower)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = u.query(
		roundedTick, tickLower, tickUpper,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = EncodeState(tickLower)
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
	tickLower, tickUpper, shiftMode := uniformLib.DecodeParams(u.tickSpacing, twapTick, ldfParams)
	initialized, lastTickLower := DecodeState(ldfState)

	if initialized {
		tickLength := tickUpper - tickLower
		tickLower = shiftmode.EnforceShiftMode(tickLower, int(lastTickLower), shiftMode)
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
		success, roundedTick = uniformLib.InverseCumulativeAmount0(
			u.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// Compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = uniformLib.CumulativeAmount0(
				u.tickSpacing,
				roundedTick+u.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		} else {
			cumulativeAmount0_, err = uniformLib.CumulativeAmount0(
				u.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = uniformLib.CumulativeAmount1(
				u.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		} else {
			cumulativeAmount1_, err = uniformLib.CumulativeAmount1(
				u.tickSpacing,
				roundedTick-u.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		// Compute roundedTick by inverting the cumulative amount1
		success, roundedTick = uniformLib.InverseCumulativeAmount1(
			u.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			tickLower,
			tickUpper,
			false,
		)
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// Compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = uniformLib.CumulativeAmount1(
				u.tickSpacing,
				roundedTick-u.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		} else {
			cumulativeAmount1_, err = uniformLib.CumulativeAmount1(
				u.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				false)

		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = uniformLib.CumulativeAmount0(
				u.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		} else {
			cumulativeAmount0_, err = uniformLib.CumulativeAmount0(
				u.tickSpacing,
				roundedTick+u.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				false,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	// Compute swap liquidity
	swapLiquidity = uniformLib.LiquidityDensityX96(roundedTick, u.tickSpacing, tickLower, tickUpper)
	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}

// query computes the liquidity density and cumulative amounts using uniformLib
func (u *UniformDistribution) query(
	roundedTick, tickLower, tickUpper int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// Use the uniformLib Query function to avoid code duplication
	liquidityDensityX96 = uniformLib.LiquidityDensityX96(roundedTick, u.tickSpacing, tickLower, tickUpper)

	length := (tickUpper - tickLower) / u.tickSpacing
	if length <= 0 {
		return liquidityDensityX96, uint256.NewInt(0), uint256.NewInt(0), nil
	}

	lengthBig := uint256.NewInt(uint64(length))
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
