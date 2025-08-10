package ldf

import (
	carpetedDoubleGeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/carpeted-double-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
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
	params := c.decodeParams(twapTick, ldfParams)
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

	newLdfState = c.encodeState(params.MinTick)
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
	params := c.decodeParams(twapTick, ldfParams)
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

// decodeParams decodes the LDF parameters from bytes32
func (c *CarpetedDoubleGeometricDistribution) decodeParams(twapTick int, ldfParams [32]byte) carpetedDoubleGeoLib.Params {
	// | shiftMode - 1 byte | offset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes | weightCarpet - 4 bytes |
	shiftMode := shiftmode.ShiftMode(ldfParams[0])
	length0 := int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	alpha0 := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	weight0Val := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])
	length1 := int(int16(uint16(ldfParams[14])<<8 | uint16(ldfParams[15])))
	alpha1 := uint32(ldfParams[16])<<24 | uint32(ldfParams[17])<<16 | uint32(ldfParams[18])<<8 | uint32(ldfParams[19])
	weight1Val := uint32(ldfParams[20])<<24 | uint32(ldfParams[21])<<16 | uint32(ldfParams[22])<<8 | uint32(ldfParams[23])
	weightCarpetVal := uint32(ldfParams[24])<<24 | uint32(ldfParams[25])<<16 | uint32(ldfParams[26])<<8 | uint32(ldfParams[27])

	// Convert alphas to alphaX96
	alpha0X96 := uint256.NewInt(uint64(alpha0))
	alpha0X96.Mul(alpha0X96, math.Q96)
	alpha0X96.Div(alpha0X96, math.ALPHA_BASE)

	alpha1X96 := uint256.NewInt(uint64(alpha1))
	alpha1X96.Mul(alpha1X96, math.Q96)
	alpha1X96.Div(alpha1X96, math.ALPHA_BASE)

	// Convert weights to WAD
	weight0 := uint256.NewInt(uint64(weight0Val))
	weight1 := uint256.NewInt(uint64(weight1Val))
	weightCarpet := uint256.NewInt(uint64(weightCarpetVal))

	var minTick int
	if shiftMode != shiftmode.Static {
		// use rounded TWAP value + offset as minTick
		offset := int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
		minTick = math.RoundTickSingle(twapTick+offset, c.tickSpacing)

		// bound distribution to be within the range of usable ticks
		minUsableTick := math.MinUsableTick(c.tickSpacing)
		maxUsableTick := math.MaxUsableTick(c.tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-(length0+length1)*c.tickSpacing {
			minTick = maxUsableTick - (length0+length1)*c.tickSpacing
		}
	} else {
		// static minTick set in params
		minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
	}

	return carpetedDoubleGeoLib.Params{
		MinTick:      minTick,
		Length0:      length0,
		Length1:      length1,
		Alpha0X96:    alpha0X96,
		Alpha1X96:    alpha1X96,
		Weight0:      weight0,
		Weight1:      weight1,
		WeightCarpet: weightCarpet,
		ShiftMode:    shiftMode,
	}
}

// encodeState encodes the state into bytes32
func (c *CarpetedDoubleGeometricDistribution) encodeState(minTick int) [32]byte {
	var state [32]byte

	minTickUint24 := uint32(minTick) & 0xFFFFFF
	combined := INITIALIZED_STATE + minTickUint24

	state[0] = byte((combined >> 24) & 0xFF)
	state[1] = byte((combined >> 16) & 0xFF)
	state[2] = byte((combined >> 8) & 0xFF)
	state[3] = byte(combined & 0xFF)

	return state
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
	// compute liquidityDensityX96
	liquidityDensityX96, err = carpetedDoubleGeoLib.LiquidityDensityX96(c.tickSpacing, roundedTick, params)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96
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

	// compute cumulativeAmount1DensityX96
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
		// compute roundedTick by inverting the cumulative amount
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

		// compute the cumulative amount up to roundedTick
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

		// compute the cumulative amount of the complementary token
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

		// compute liquidity of the rounded tick that will handle the remainder of the swap
		liquidityDensityX96, err := carpetedDoubleGeoLib.LiquidityDensityX96(c.tickSpacing, roundedTick, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		swapLiquidity = uint256.NewInt(0)
		swapLiquidity.Mul(liquidityDensityX96, totalLiquidity)
		swapLiquidity.Rsh(swapLiquidity, 96)
	} else {
		// compute roundedTick by inverting the cumulative amount
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

		// compute the cumulative amount up to roundedTick
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

		// compute the cumulative amount of the complementary token
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

		// compute liquidity of the rounded tick that will handle the remainder of the swap
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
