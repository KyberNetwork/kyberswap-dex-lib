package ldf

import (
	doubleGeo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/double-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// DoubleGeometricDistribution represents a double geometric distribution LDF
type DoubleGeometricDistribution struct {
	tickSpacing int
}

// NewDoubleGeometricDistribution creates a new DoubleGeometricDistribution
func NewDoubleGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &DoubleGeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// Query implements the Query method for DoubleGeometricDistribution
func (d *DoubleGeometricDistribution) Query(
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
	minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1, shiftMode :=
		doubleGeo.DecodeParams(d.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
		shouldSurge = minTick != int(lastMinTick)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = d.query(
		roundedTick, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = EncodeState(minTick)
	return
}

// ComputeSwap implements the ComputeSwap method for DoubleGeometricDistribution
func (d *DoubleGeometricDistribution) ComputeSwap(
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
	minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1, shiftMode :=
		doubleGeo.DecodeParams(d.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		minTick = shiftmode.EnforceShiftMode(minTick, int(lastMinTick), shiftMode)
	}

	return d.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		minTick,
		length0,
		length1,
		alpha0X96,
		alpha1X96,
		weight0,
		weight1,
	)
}

// query computes the liquidity density and cumulative amounts using doubleGeo lib functions
func (d *DoubleGeometricDistribution) query(
	roundedTick,
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// compute liquidityDensityX96
	liquidityDensityX96, err = doubleGeo.LiquidityDensityX96(
		d.tickSpacing,
		roundedTick,
		minTick,
		length0,
		length1,
		alpha0X96,
		alpha1X96,
		weight0,
		weight1,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = doubleGeo.CumulativeAmount0(
		d.tickSpacing,
		roundedTick+d.tickSpacing,
		math.Q96,
		minTick,
		length0,
		length1,
		alpha0X96,
		alpha1X96,
		weight0,
		weight1,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = doubleGeo.CumulativeAmount1(
		d.tickSpacing,
		roundedTick-d.tickSpacing,
		math.Q96,
		minTick,
		length0,
		length1,
		alpha0X96,
		alpha1X96,
		weight0,
		weight1,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	return liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, nil
}

// computeSwap computes the swap parameters
func (d *DoubleGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	minTick, length0, length1 int,
	alpha0X96, alpha1X96, weight0, weight1 *uint256.Int,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	if exactIn == zeroForOne {
		success, roundedTick, err = doubleGeo.InverseCumulativeAmount0(d.tickSpacing, inverseCumulativeAmountInput, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		if exactIn {
			cumulativeAmount0_, err = doubleGeo.CumulativeAmount0(d.tickSpacing, roundedTick+d.tickSpacing, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		} else {
			cumulativeAmount0_, err = doubleGeo.CumulativeAmount0(d.tickSpacing, roundedTick, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = doubleGeo.CumulativeAmount1(d.tickSpacing, roundedTick, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		} else {
			cumulativeAmount1_, err = doubleGeo.CumulativeAmount1(d.tickSpacing, roundedTick-d.tickSpacing, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		success, roundedTick, err = doubleGeo.InverseCumulativeAmount1(d.tickSpacing, inverseCumulativeAmountInput, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		if exactIn {
			cumulativeAmount1_, err = doubleGeo.CumulativeAmount1(d.tickSpacing, roundedTick-d.tickSpacing, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		} else {
			cumulativeAmount1_, err = doubleGeo.CumulativeAmount1(d.tickSpacing, roundedTick, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = doubleGeo.CumulativeAmount0(d.tickSpacing, roundedTick, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		} else {
			cumulativeAmount0_, err = doubleGeo.CumulativeAmount0(d.tickSpacing, roundedTick+d.tickSpacing, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	swapLiquidity, err = doubleGeo.LiquidityDensityX96(
		d.tickSpacing,
		roundedTick,
		minTick,
		length0,
		length1,
		alpha0X96,
		alpha1X96,
		weight0,
		weight1,
	)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
