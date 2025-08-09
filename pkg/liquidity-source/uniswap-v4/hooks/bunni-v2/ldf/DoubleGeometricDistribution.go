package ldf

import (
	doubleGeo "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/double-geometric"
	shiftmode "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/shift-mode"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"
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
	minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1, shiftMode := d.decodeParams(twapTick, ldfParams)
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

	newLdfState = d.encodeState(minTick)
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
	minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1, shiftMode := d.decodeParams(twapTick, ldfParams)
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

// decodeParams decodes the LDF parameters from bytes32
func (d *DoubleGeometricDistribution) decodeParams(
	twapTick int,
	ldfParams [32]byte,
) (
	minTick,
	length0,
	length1 int,
	alpha0X96,
	alpha1X96,
	weight0,
	weight1 *uint256.Int,
	shiftMode shiftmode.ShiftMode,
) {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes |
	shiftMode = shiftmode.ShiftMode(ldfParams[0])
	length0 = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	length1 = int(int16(uint16(ldfParams[6])<<8 | uint16(ldfParams[7])))
	alpha0 := uint32(ldfParams[8])<<24 | uint32(ldfParams[9])<<16 | uint32(ldfParams[10])<<8 | uint32(ldfParams[11])
	alpha1 := uint32(ldfParams[12])<<24 | uint32(ldfParams[13])<<16 | uint32(ldfParams[14])<<8 | uint32(ldfParams[15])
	weight0Val := uint32(ldfParams[16])<<24 | uint32(ldfParams[17])<<16 | uint32(ldfParams[18])<<8 | uint32(ldfParams[19])
	weight1Val := uint32(ldfParams[20])<<24 | uint32(ldfParams[21])<<16 | uint32(ldfParams[22])<<8 | uint32(ldfParams[23])

	// Convert alphas to alphaX96
	alpha0X96 = uint256.NewInt(uint64(alpha0))
	alpha0X96.Mul(alpha0X96, math.Q96)
	alpha0X96.Div(alpha0X96, math.ALPHA_BASE)

	alpha1X96 = uint256.NewInt(uint64(alpha1))
	alpha1X96.Mul(alpha1X96, math.Q96)
	alpha1X96.Div(alpha1X96, math.ALPHA_BASE)

	// Convert weights to WAD
	weight0 = uint256.NewInt(uint64(weight0Val))
	weight1 = uint256.NewInt(uint64(weight1Val))

	if shiftMode != shiftmode.Static {
		// use rounded TWAP value + offset as minTick
		offset := int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
		minTick = math.RoundTickSingle(twapTick+offset, d.tickSpacing)

		// bound distribution to be within the range of usable ticks
		minUsableTick := math.MinUsableTick(d.tickSpacing)
		maxUsableTick := math.MaxUsableTick(d.tickSpacing)
		if minTick < minUsableTick {
			minTick = minUsableTick
		} else if minTick > maxUsableTick-(length0+length1)*d.tickSpacing {
			minTick = maxUsableTick - (length0+length1)*d.tickSpacing
		}
	} else {
		// static minTick set in params
		minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))
	}

	return
}

// encodeState encodes the state into bytes32
func (d *DoubleGeometricDistribution) encodeState(minTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((minTick >> 16) & 0xFF)
	state[2] = byte((minTick >> 8) & 0xFF)
	state[3] = byte(minTick & 0xFF)
	return state
}

// query computes the liquidity density and cumulative amounts
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
	liquidityDensityX96, err = d.liquidityDensityX96(
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

	return
}

// liquidityDensityX96 computes the liquidity density using weighted sum of two geometric distributions
func (d *DoubleGeometricDistribution) liquidityDensityX96(roundedTick, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Calculate density for distribution 0 (right distribution)
	density0, err := d.geometricLiquidityDensityX96(roundedTick, minTick+length1*d.tickSpacing, length0, alpha0X96)
	if err != nil {
		return nil, err
	}

	// Calculate density for distribution 1 (left distribution)
	density1, err := d.geometricLiquidityDensityX96(roundedTick, minTick, length1, alpha1X96)
	if err != nil {
		return nil, err
	}

	// Apply weighted sum: density0 * weight0 + density1 * weight1
	var weightedDensity0 uint256.Int
	weightedDensity0.Mul(density0, weight0)
	var weightedDensity1 uint256.Int
	weightedDensity1.Mul(density1, weight1)

	var result uint256.Int
	result.Add(&weightedDensity0, &weightedDensity1)
	var totalWeight uint256.Int
	totalWeight.Add(weight0, weight1)
	result.Div(&result, &totalWeight)

	return &result, nil
}

// geometricLiquidityDensityX96 computes the liquidity density for a single geometric distribution
func (d *DoubleGeometricDistribution) geometricLiquidityDensityX96(roundedTick, minTick, length int, alphaX96 *uint256.Int) (*uint256.Int, error) {
	if roundedTick < minTick || roundedTick >= minTick+length*d.tickSpacing {
		return uint256.NewInt(0), nil
	}

	x := (roundedTick - minTick) / d.tickSpacing

	if alphaX96.Cmp(math.Q96) > 0 {
		// alpha > 1
		var alphaInvX96 uint256.Int
		alphaInvX96.Mul(math.Q96, math.Q96)
		alphaInvX96.Div(&alphaInvX96, alphaX96)

		term1, err := math.Rpow(&alphaInvX96, length-x, math.Q96)
		if err != nil {
			return nil, err
		}
		var term2 uint256.Int
		term2.Sub(alphaX96, math.Q96)
		term3, err := math.Rpow(&alphaInvX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(term1, &term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// alpha <= 1
		var term1 uint256.Int
		term1.Sub(math.Q96, alphaX96)
		term2, err := math.Rpow(alphaX96, x, math.Q96)
		if err != nil {
			return nil, err
		}
		term3, err := math.Rpow(alphaX96, length, math.Q96)
		if err != nil {
			return nil, err
		}
		var denom uint256.Int
		denom.Sub(math.Q96, term3)

		result, err := math.FullMulDiv(&term1, term2, &denom)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
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
		// compute roundedTick by inverting the cumulative amount0
		success, roundedTick, err = doubleGeo.InverseCumulativeAmount0(d.tickSpacing, inverseCumulativeAmountInput, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
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
		// compute roundedTick by inverting the cumulative amount1
		success, roundedTick, err = doubleGeo.InverseCumulativeAmount1(d.tickSpacing, inverseCumulativeAmountInput, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
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

	// compute swap liquidity
	swapLiquidity, err = d.liquidityDensityX96(roundedTick, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
