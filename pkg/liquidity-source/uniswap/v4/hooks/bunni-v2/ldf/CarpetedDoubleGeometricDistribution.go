package ldf

import (
	carpetedDoubleGeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/carpeted-double-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	"github.com/holiman/uint256"
)

// CarpetedDoubleGeometricDistribution represents a carpeted double geometric distribution LDF
type CarpetedDoubleGeometricDistribution struct {
	tickSpacing int
}

// NewCarpetedDoubleGeometricDistribution creates a new CarpetedDoubleGeometricDistribution
func NewCarpetedDoubleGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &CarpetedDoubleGeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// Query implements the Query method for CarpetedDoubleGeometricDistribution
func (c *CarpetedDoubleGeometricDistribution) Query(
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
	params := carpetedDoubleGeoLib.DecodeParams(c.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		params.MinTick = shiftmode.EnforceShiftMode(params.MinTick, int(lastMinTick), params.ShiftMode)
		shouldSurge = params.MinTick != int(lastMinTick)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = c.query(
		roundedTick, params,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = EncodeState(params.MinTick)
	return
}

// ComputeSwap implements the ComputeSwap method for CarpetedDoubleGeometricDistribution
func (c *CarpetedDoubleGeometricDistribution) ComputeSwap(
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
	params := carpetedDoubleGeoLib.DecodeParams(c.tickSpacing, twapTick, ldfParams)
	initialized, lastMinTick := DecodeState(ldfState)

	if initialized {
		params.MinTick = shiftmode.EnforceShiftMode(params.MinTick, int(lastMinTick), params.ShiftMode)
	}

	return c.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		params,
	)
}

// query computes the liquidity density and cumulative amounts
func (c *CarpetedDoubleGeometricDistribution) query(
	roundedTick int,
	params carpetedDoubleGeoLib.Params,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	liquidityDensityX96, err = carpetedDoubleGeoLib.LiquidityDensityX96(c.tickSpacing, roundedTick, params)
	if err != nil {
		return nil, nil, nil, err
	}

	cumulativeAmount0DensityX96, err = carpetedDoubleGeoLib.CumulativeAmount0(
		c.tickSpacing,
		roundedTick+c.tickSpacing,
		math.SCALED_Q96,
		params,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	cumulativeAmount0DensityX96.Rsh(cumulativeAmount0DensityX96, QUERY_SCALE_SHIFT)

	cumulativeAmount1DensityX96, err = carpetedDoubleGeoLib.CumulativeAmount1(
		c.tickSpacing,
		roundedTick-c.tickSpacing,
		math.SCALED_Q96,
		params,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	cumulativeAmount1DensityX96.Rsh(cumulativeAmount1DensityX96, QUERY_SCALE_SHIFT)

	return
}

// computeSwap computes the swap parameters
func (c *CarpetedDoubleGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	params carpetedDoubleGeoLib.Params,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	if exactIn == zeroForOne {
		success, roundedTick, err = carpetedDoubleGeoLib.InverseCumulativeAmount0(
			c.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			params,
		)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		if !success {
			return false, 0, nil, nil, nil, nil
		}

		if exactIn {
			cumulativeAmount0_, err = carpetedDoubleGeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
				totalLiquidity,
				params,
			)
		} else {
			cumulativeAmount0_, err = carpetedDoubleGeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				params,
			)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		if exactIn {
			cumulativeAmount1_, err = carpetedDoubleGeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				params,
			)
		} else {
			cumulativeAmount1_, err = carpetedDoubleGeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
				totalLiquidity,
				params,
			)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		liquidityDensityX96, err := carpetedDoubleGeoLib.LiquidityDensityX96(c.tickSpacing, roundedTick, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		swapLiquidity = uint256.NewInt(0)
		swapLiquidity.Mul(liquidityDensityX96, totalLiquidity)
		swapLiquidity.Rsh(swapLiquidity, 96)
	} else {
		success, roundedTick, err = carpetedDoubleGeoLib.InverseCumulativeAmount1(
			c.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			params,
		)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		if !success {
			return false, 0, nil, nil, nil, nil
		}

		if exactIn {
			cumulativeAmount1_, err = carpetedDoubleGeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick-c.tickSpacing,
				totalLiquidity,
				params,
			)
		} else {
			cumulativeAmount1_, err = carpetedDoubleGeoLib.CumulativeAmount1(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				params,
			)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		if exactIn {
			cumulativeAmount0_, err = carpetedDoubleGeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick,
				totalLiquidity,
				params,
			)
		} else {
			cumulativeAmount0_, err = carpetedDoubleGeoLib.CumulativeAmount0(
				c.tickSpacing,
				roundedTick+c.tickSpacing,
				totalLiquidity,
				params,
			)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		liquidityDensityX96, err := carpetedDoubleGeoLib.LiquidityDensityX96(c.tickSpacing, roundedTick, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		swapLiquidity = uint256.NewInt(0)
		swapLiquidity.Mul(liquidityDensityX96, totalLiquidity)
		swapLiquidity.Rsh(swapLiquidity, 96)
	}

	return
}
