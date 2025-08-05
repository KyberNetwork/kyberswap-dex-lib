package ldf

import (
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
		params.minTick = EnforceShiftMode(params.minTick, int(lastMinTick), params.shiftMode)
		shouldSurge = params.minTick != int(lastMinTick)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = c.query(
		roundedTick, params,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = c.encodeState(params.minTick)
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
		params.minTick = EnforceShiftMode(params.minTick, int(lastMinTick), params.shiftMode)
	}

	return c.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		params,
	)
}

// Params represents the parameters for CarpetedDoubleGeometricDistribution
type Params struct {
	minTick, length0, length1                            int
	alpha0X96, alpha1X96, weight0, weight1, weightCarpet *uint256.Int
	shiftMode                                            ShiftMode
}

// decodeParams decodes the LDF parameters from bytes32
func (c *CarpetedDoubleGeometricDistribution) decodeParams(twapTick int, ldfParams [32]byte) Params {
	// | shiftMode - 1 byte | minTickOrOffset - 3 bytes | length0 - 2 bytes | alpha0 - 4 bytes | weight0 - 4 bytes | length1 - 2 bytes | alpha1 - 4 bytes | weight1 - 4 bytes | weightCarpet - 4 bytes |
	shiftMode := ShiftMode(ldfParams[0])
	length0 := int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))
	length1 := int(int16(uint16(ldfParams[6])<<8 | uint16(ldfParams[7])))

	alpha0 := uint32(ldfParams[8])<<24 | uint32(ldfParams[9])<<16 | uint32(ldfParams[10])<<8 | uint32(ldfParams[11])
	alpha1 := uint32(ldfParams[12])<<24 | uint32(ldfParams[13])<<16 | uint32(ldfParams[14])<<8 | uint32(ldfParams[15])
	weight0Val := uint32(ldfParams[16])<<24 | uint32(ldfParams[17])<<16 | uint32(ldfParams[18])<<8 | uint32(ldfParams[19])
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
	if shiftMode != ShiftModeStatic {
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

	return Params{
		minTick:      minTick,
		length0:      length0,
		length1:      length1,
		alpha0X96:    alpha0X96,
		alpha1X96:    alpha1X96,
		weight0:      weight0,
		weight1:      weight1,
		weightCarpet: weightCarpet,
		shiftMode:    shiftMode,
	}
}

// encodeState encodes the state into bytes32
func (c *CarpetedDoubleGeometricDistribution) encodeState(minTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((minTick >> 16) & 0xFF)
	state[2] = byte((minTick >> 8) & 0xFF)
	state[3] = byte(minTick & 0xFF)
	return state
}

// query computes the liquidity density and cumulative amounts
func (c *CarpetedDoubleGeometricDistribution) query(
	roundedTick int,
	params Params,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	// compute liquidityDensityX96
	liquidityDensityX96, err = c.liquidityDensityX96(roundedTick, params)
	if err != nil {
		return nil, nil, nil, err
	}

	// compute cumulativeAmount0DensityX96 (scaled by 2^100)
	scaledQ96 := uint256.NewInt(1)
	scaledQ96.Lsh(scaledQ96, 100) // Q96 << 4
	cumulativeAmount0DensityX96, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, scaledQ96, params)
	if err != nil {
		return nil, nil, nil, err
	}
	cumulativeAmount0DensityX96.Rsh(cumulativeAmount0DensityX96, 4) // >> 4

	// compute cumulativeAmount1DensityX96 (scaled by 2^100)
	cumulativeAmount1DensityX96, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, scaledQ96, params)
	if err != nil {
		return nil, nil, nil, err
	}
	cumulativeAmount1DensityX96.Rsh(cumulativeAmount1DensityX96, 4) // >> 4

	return
}

// computeSwap computes the swap parameters
func (c *CarpetedDoubleGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	params Params,
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
		success, roundedTick, err = c.inverseCumulativeAmount0(inverseCumulativeAmountInput, totalLiquidity, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		if !success {
			return false, 0, nil, nil, nil, nil
		}

		// compute the cumulative amount up to roundedTick
		if exactIn {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, totalLiquidity, params)
		} else {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick, totalLiquidity, params)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		// compute the cumulative amount of the complementary token
		if exactIn {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick, totalLiquidity, params)
		} else {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, totalLiquidity, params)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		// compute liquidity of the rounded tick that will handle the remainder of the swap
		liquidityDensityX96, err := c.liquidityDensityX96(roundedTick, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		swapLiquidity = uint256.NewInt(0)
		swapLiquidity.Mul(liquidityDensityX96, totalLiquidity)
		swapLiquidity.Rsh(swapLiquidity, 96)
	} else {
		// compute roundedTick by inverting the cumulative amount
		success, roundedTick, err = c.inverseCumulativeAmount1(inverseCumulativeAmountInput, totalLiquidity, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		if !success {
			return false, 0, nil, nil, nil, nil
		}

		// compute the cumulative amount up to roundedTick
		if exactIn {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick-c.tickSpacing, totalLiquidity, params)
		} else {
			cumulativeAmount1_, err = c.cumulativeAmount1(roundedTick, totalLiquidity, params)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		// compute the cumulative amount of the complementary token
		if exactIn {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick, totalLiquidity, params)
		} else {
			cumulativeAmount0_, err = c.cumulativeAmount0(roundedTick+c.tickSpacing, totalLiquidity, params)
		}
		if err != nil {
			return false, 0, nil, nil, nil, err
		}

		// compute liquidity of the rounded tick that will handle the remainder of the swap
		liquidityDensityX96, err := c.liquidityDensityX96(roundedTick, params)
		if err != nil {
			return false, 0, nil, nil, nil, err
		}
		swapLiquidity = uint256.NewInt(0)
		swapLiquidity.Mul(liquidityDensityX96, totalLiquidity)
		swapLiquidity.Rsh(swapLiquidity, 96)
	}

	return
}

// cumulativeAmount0 computes the cumulative amount of token0
func (c *CarpetedDoubleGeometricDistribution) cumulativeAmount0(
	roundedTick int,
	totalLiquidity *uint256.Int,
	params Params,
) (*uint256.Int, error) {
	length := params.length0 + params.length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(totalLiquidity, params.minTick, length, params.weightCarpet)

	// Left carpet amount0
	leftCarpetAmount0, err := c.uniformCumulativeAmount0(roundedTick, leftCarpetLiquidity, minUsableTick, params.minTick, true)
	if err != nil {
		return nil, err
	}

	// Main double geometric amount0
	mainAmount0, err := c.doubleGeometricCumulativeAmount0(roundedTick, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
	if err != nil {
		return nil, err
	}

	// Right carpet amount0
	rightCarpetAmount0, err := c.uniformCumulativeAmount0(roundedTick, rightCarpetLiquidity, params.minTick+length*c.tickSpacing, maxUsableTick, true)
	if err != nil {
		return nil, err
	}

	// Sum all amounts - reuse leftCarpetAmount0 for total
	leftCarpetAmount0.Add(leftCarpetAmount0, mainAmount0)
	leftCarpetAmount0.Add(leftCarpetAmount0, rightCarpetAmount0)

	return leftCarpetAmount0, nil
}

// cumulativeAmount1 computes the cumulative amount of token1
func (c *CarpetedDoubleGeometricDistribution) cumulativeAmount1(
	roundedTick int,
	totalLiquidity *uint256.Int,
	params Params,
) (*uint256.Int, error) {
	length := params.length0 + params.length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(totalLiquidity, params.minTick, length, params.weightCarpet)

	// Left carpet amount1
	leftCarpetAmount1, err := c.uniformCumulativeAmount1(roundedTick, leftCarpetLiquidity, minUsableTick, params.minTick, true)
	if err != nil {
		return nil, err
	}

	// Main double geometric amount1
	mainAmount1, err := c.doubleGeometricCumulativeAmount1(roundedTick, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
	if err != nil {
		return nil, err
	}

	// Right carpet amount1
	rightCarpetAmount1, err := c.uniformCumulativeAmount1(roundedTick, rightCarpetLiquidity, params.minTick+length*c.tickSpacing, maxUsableTick, true)
	if err != nil {
		return nil, err
	}

	// Sum all amounts - reuse leftCarpetAmount1 for total
	leftCarpetAmount1.Add(leftCarpetAmount1, mainAmount1)
	leftCarpetAmount1.Add(leftCarpetAmount1, rightCarpetAmount1)

	return leftCarpetAmount1, nil
}

// inverseCumulativeAmount0 computes the inverse cumulative amount0
func (c *CarpetedDoubleGeometricDistribution) inverseCumulativeAmount0(
	cumulativeAmount0_,
	totalLiquidity *uint256.Int,
	params Params,
) (bool, int, error) {
	if cumulativeAmount0_.IsZero() {
		return true, math.MaxUsableTick(c.tickSpacing), nil
	}

	// try LDFs in the order of right carpet, main, left carpet
	length := params.length0 + params.length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(totalLiquidity, params.minTick, length, params.weightCarpet)

	rightCarpetCumulativeAmount0, err := c.uniformCumulativeAmount0(params.minTick+length*c.tickSpacing, rightCarpetLiquidity, params.minTick+length*c.tickSpacing, maxUsableTick, true)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount0_.Cmp(rightCarpetCumulativeAmount0) <= 0 && !rightCarpetLiquidity.IsZero() {
		// use right carpet
		return c.uniformInverseCumulativeAmount0(cumulativeAmount0_, rightCarpetLiquidity, params.minTick+length*c.tickSpacing, maxUsableTick, true)
	} else {
		// Reuse rightCarpetCumulativeAmount0 for remainder calculation
		rightCarpetCumulativeAmount0.Sub(cumulativeAmount0_, rightCarpetCumulativeAmount0)

		mainCumulativeAmount0, err := c.doubleGeometricCumulativeAmount0(params.minTick, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
		if err != nil {
			return false, 0, err
		}

		if rightCarpetCumulativeAmount0.Cmp(mainCumulativeAmount0) <= 0 {
			// use main
			return c.doubleGeometricInverseCumulativeAmount0(rightCarpetCumulativeAmount0, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
		} else if !leftCarpetLiquidity.IsZero() {
			// use left carpet - reuse rightCarpetCumulativeAmount0 for final remainder
			rightCarpetCumulativeAmount0.Sub(rightCarpetCumulativeAmount0, mainCumulativeAmount0)
			return c.uniformInverseCumulativeAmount0(rightCarpetCumulativeAmount0, leftCarpetLiquidity, minUsableTick, params.minTick, true)
		}
	}
	return false, 0, nil
}

// inverseCumulativeAmount1 computes the inverse cumulative amount1
func (c *CarpetedDoubleGeometricDistribution) inverseCumulativeAmount1(
	cumulativeAmount1_,
	totalLiquidity *uint256.Int,
	params Params,
) (bool, int, error) {
	if cumulativeAmount1_.IsZero() {
		return true, math.MinUsableTick(c.tickSpacing) - c.tickSpacing, nil
	}

	// try LDFs in the order of left carpet, main, right carpet
	length := params.length0 + params.length1
	leftCarpetLiquidity, mainLiquidity, rightCarpetLiquidity, minUsableTick, maxUsableTick := c.getCarpetedLiquidity(totalLiquidity, params.minTick, length, params.weightCarpet)

	leftCarpetCumulativeAmount1, err := c.uniformCumulativeAmount1(params.minTick, leftCarpetLiquidity, minUsableTick, params.minTick, true)
	if err != nil {
		return false, 0, err
	}

	if cumulativeAmount1_.Cmp(leftCarpetCumulativeAmount1) <= 0 && !leftCarpetLiquidity.IsZero() {
		// use left carpet
		return c.uniformInverseCumulativeAmount1(cumulativeAmount1_, leftCarpetLiquidity, minUsableTick, params.minTick, true)
	} else {
		// Reuse leftCarpetCumulativeAmount1 for remainder calculation
		leftCarpetCumulativeAmount1.Sub(cumulativeAmount1_, leftCarpetCumulativeAmount1)

		mainCumulativeAmount1, err := c.doubleGeometricCumulativeAmount1(params.minTick+length*c.tickSpacing, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
		if err != nil {
			return false, 0, err
		}

		if leftCarpetCumulativeAmount1.Cmp(mainCumulativeAmount1) <= 0 {
			// use main
			return c.doubleGeometricInverseCumulativeAmount1(leftCarpetCumulativeAmount1, mainLiquidity, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
		} else if !rightCarpetLiquidity.IsZero() {
			// use right carpet - reuse leftCarpetCumulativeAmount1 for final remainder
			leftCarpetCumulativeAmount1.Sub(leftCarpetCumulativeAmount1, mainCumulativeAmount1)
			return c.uniformInverseCumulativeAmount1(leftCarpetCumulativeAmount1, rightCarpetLiquidity, params.minTick+length*c.tickSpacing, maxUsableTick, true)
		}
	}
	return false, 0, nil
}

// liquidityDensityX96 computes the liquidity density
func (c *CarpetedDoubleGeometricDistribution) liquidityDensityX96(
	roundedTick int,
	params Params,
) (*uint256.Int, error) {
	length := params.length0 + params.length1
	if roundedTick >= params.minTick && roundedTick < params.minTick+length*c.tickSpacing {
		// Main distribution
		mainDensity, err := c.doubleGeometricLiquidityDensityX96(roundedTick, params.minTick, params.length0, params.length1, params.alpha0X96, params.alpha1X96, params.weight0, params.weight1)
		if err != nil {
			return nil, err
		}

		// Apply carpet weight: mainDensity * (WAD - weightCarpet) / WAD
		var wadMinusWeightCarpet uint256.Int
		wadMinusWeightCarpet.Sub(math.WAD, params.weightCarpet)
		result, err := math.FullMulDiv(mainDensity, &wadMinusWeightCarpet, math.WAD)
		if err != nil {
			return nil, err
		}
		return result, nil
	} else {
		// Carpet distribution
		minUsableTick := math.MinUsableTick(c.tickSpacing)
		maxUsableTick := math.MaxUsableTick(c.tickSpacing)
		numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/c.tickSpacing - length
		if numRoundedTicksCarpeted <= 0 {
			return uint256.NewInt(0), nil
		}

		// Carpet liquidity: Q96 * weightCarpet / numRoundedTicksCarpeted
		var carpetLiquidity uint256.Int
		carpetLiquidity.Mul(math.Q96, params.weightCarpet)
		result, err := math.FullMulDiv(&carpetLiquidity, math.Q96, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

// getCarpetedLiquidity computes the carpeted liquidity distribution
func (c *CarpetedDoubleGeometricDistribution) getCarpetedLiquidity(
	totalLiquidity *uint256.Int,
	minTick, length int,
	weightCarpet *uint256.Int,
) (
	leftCarpetLiquidity,
	mainLiquidity,
	rightCarpetLiquidity *uint256.Int,
	minUsableTick,
	maxUsableTick int,
) {
	minUsableTick = math.MinUsableTick(c.tickSpacing)
	maxUsableTick = math.MaxUsableTick(c.tickSpacing)
	numRoundedTicksCarpeted := (maxUsableTick-minUsableTick)/c.tickSpacing - length
	if numRoundedTicksCarpeted <= 0 {
		return uint256.NewInt(0), totalLiquidity, uint256.NewInt(0), minUsableTick, maxUsableTick
	}

	// Main liquidity: totalLiquidity * (WAD - weightCarpet) / WAD
	var mainLiquidityVar uint256.Int
	mainLiquidityVar.Mul(totalLiquidity, math.WAD)
	var wadMinusWeightCarpet uint256.Int
	wadMinusWeightCarpet.Sub(math.WAD, weightCarpet)
	mainLiquidityVar.Mul(&mainLiquidityVar, &wadMinusWeightCarpet)
	mainLiquidityVar.Div(&mainLiquidityVar, math.WAD)
	mainLiquidity = &mainLiquidityVar

	// Carpet liquidity: totalLiquidity - mainLiquidity
	var carpetLiquidity uint256.Int
	carpetLiquidity.Sub(totalLiquidity, &mainLiquidityVar)

	// Right carpet liquidity: carpetLiquidity * rightCarpetTicks / numRoundedTicksCarpeted
	rightCarpetTicks := (maxUsableTick-minTick)/c.tickSpacing - length
	var rightCarpetLiquidityVar uint256.Int
	rightCarpetLiquidityVar.Mul(&carpetLiquidity, uint256.NewInt(uint64(rightCarpetTicks)))
	rightCarpetLiquidityVar.Div(&rightCarpetLiquidityVar, uint256.NewInt(uint64(numRoundedTicksCarpeted)))
	rightCarpetLiquidity = &rightCarpetLiquidityVar

	// Left carpet liquidity: carpetLiquidity - rightCarpetLiquidity
	carpetLiquidity.Sub(&carpetLiquidity, &rightCarpetLiquidityVar)
	leftCarpetLiquidity = &carpetLiquidity

	return
}

// Helper functions for uniform distribution
func (c *CarpetedDoubleGeometricDistribution) uniformCumulativeAmount0(roundedTick int, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (*uint256.Int, error) {
	if roundedTick >= tickUpper {
		return uint256.NewInt(0), nil
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	amount0, err := math.GetAmount0Delta(
		sqrtPriceLower,
		sqrtPriceUpper,
		liquidity,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	return amount0, nil
}

func (c *CarpetedDoubleGeometricDistribution) uniformCumulativeAmount1(roundedTick int, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (*uint256.Int, error) {
	if roundedTick <= tickLower {
		return uint256.NewInt(0), nil
	}

	sqrtPriceLower, err := math.GetSqrtPriceAtTick(tickLower)
	if err != nil {
		return nil, err
	}
	sqrtPriceUpper, err := math.GetSqrtPriceAtTick(tickUpper)
	if err != nil {
		return nil, err
	}

	amount1, err := math.GetAmount1Delta(
		sqrtPriceLower,
		sqrtPriceUpper,
		liquidity,
		roundUp,
	)
	if err != nil {
		return nil, err
	}

	return amount1, nil
}

func (c *CarpetedDoubleGeometricDistribution) uniformInverseCumulativeAmount0(cumulativeAmount0_, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (bool, int, error) {
	// Simplified binary search implementation
	left := tickLower
	right := tickUpper

	for left < right {
		mid := (left + right) / 2
		mid = (mid / c.tickSpacing) * c.tickSpacing // round to tick spacing

		amount0, err := c.uniformCumulativeAmount0(mid, liquidity, tickLower, tickUpper, roundUp)
		if err != nil {
			return false, 0, err
		}

		if amount0.Cmp(cumulativeAmount0_) >= 0 {
			right = mid
		} else {
			left = mid + c.tickSpacing
		}
	}

	return true, left, nil
}

func (c *CarpetedDoubleGeometricDistribution) uniformInverseCumulativeAmount1(cumulativeAmount1_, liquidity *uint256.Int, tickLower, tickUpper int, roundUp bool) (bool, int, error) {
	// Simplified binary search implementation
	left := tickLower
	right := tickUpper

	for left < right {
		mid := (left + right) / 2
		mid = (mid / c.tickSpacing) * c.tickSpacing // round to tick spacing

		amount1, err := c.uniformCumulativeAmount1(mid, liquidity, tickLower, tickUpper, roundUp)
		if err != nil {
			return false, 0, err
		}

		if amount1.Cmp(cumulativeAmount1_) >= 0 {
			right = mid
		} else {
			left = mid + c.tickSpacing
		}
	}

	return true, left, nil
}

// Helper functions for double geometric distribution
func (c *CarpetedDoubleGeometricDistribution) doubleGeometricCumulativeAmount0(roundedTick int, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Simplified implementation - would use the full double geometric distribution logic
	// For now, return a reasonable approximation
	if roundedTick >= minTick+(length0+length1)*c.tickSpacing {
		return uint256.NewInt(0), nil
	}

	// Use simplified calculation
	var result uint256.Int
	for i := (roundedTick - minTick) / c.tickSpacing; i < length0+length1; i++ {
		density, err := c.doubleGeometricLiquidityDensityX96(minTick+i*c.tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		amount0, err := math.GetAmount0Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount0)
	}

	return &result, nil
}

func (c *CarpetedDoubleGeometricDistribution) doubleGeometricCumulativeAmount1(roundedTick int, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Simplified implementation - would use the full double geometric distribution logic
	if roundedTick <= minTick {
		return uint256.NewInt(0), nil
	}

	// Use simplified calculation
	var result uint256.Int
	for i := 0; i < (roundedTick-minTick)/c.tickSpacing && i < length0+length1; i++ {
		density, err := c.doubleGeometricLiquidityDensityX96(minTick+i*c.tickSpacing, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return nil, err
		}

		sqrtPriceLower, err := math.GetSqrtPriceAtTick(minTick + i*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		sqrtPriceUpper, err := math.GetSqrtPriceAtTick(minTick + (i+1)*c.tickSpacing)
		if err != nil {
			return nil, err
		}

		amount1, err := math.GetAmount1Delta(
			sqrtPriceLower,
			sqrtPriceUpper,
			density,
			true, // roundUp
		)
		if err != nil {
			return nil, err
		}

		result.Add(&result, amount1)
	}

	return &result, nil
}

func (c *CarpetedDoubleGeometricDistribution) doubleGeometricLiquidityDensityX96(roundedTick, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (*uint256.Int, error) {
	// Simplified implementation - would use the full double geometric distribution logic
	// For now, return a reasonable approximation
	if roundedTick < minTick || roundedTick >= minTick+(length0+length1)*c.tickSpacing {
		return uint256.NewInt(0), nil
	}

	// Use simplified calculation based on position in distribution
	x := (roundedTick - minTick) / c.tickSpacing
	if x < length0 {
		// Right distribution
		alphaPowX, err := math.Rpow(alpha0X96, x, math.Q96)
		if err != nil {
			return nil, err
		}
		var result uint256.Int
		result.Mul(alphaPowX, weight0)
		result.Div(&result, math.Q96)
		return &result, nil
	} else {
		// Left distribution
		xLeft := x - length0
		alphaPowX, err := math.Rpow(alpha1X96, xLeft, math.Q96)
		if err != nil {
			return nil, err
		}
		var result uint256.Int
		result.Mul(alphaPowX, weight1)
		result.Div(&result, math.Q96)
		return &result, nil
	}
}

func (c *CarpetedDoubleGeometricDistribution) doubleGeometricInverseCumulativeAmount0(cumulativeAmount0_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	// Simplified binary search implementation
	left := minTick
	right := minTick + (length0+length1)*c.tickSpacing

	for left < right {
		mid := (left + right) / 2
		mid = (mid / c.tickSpacing) * c.tickSpacing // round to tick spacing

		amount0, err := c.doubleGeometricCumulativeAmount0(mid, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, err
		}

		if amount0.Cmp(cumulativeAmount0_) >= 0 {
			right = mid
		} else {
			left = mid + c.tickSpacing
		}
	}

	return true, left, nil
}

func (c *CarpetedDoubleGeometricDistribution) doubleGeometricInverseCumulativeAmount1(cumulativeAmount1_, totalLiquidity *uint256.Int, minTick, length0, length1 int, alpha0X96, alpha1X96, weight0, weight1 *uint256.Int) (bool, int, error) {
	// Simplified binary search implementation
	left := minTick
	right := minTick + (length0+length1)*c.tickSpacing

	for left < right {
		mid := (left + right) / 2
		mid = (mid / c.tickSpacing) * c.tickSpacing // round to tick spacing

		amount1, err := c.doubleGeometricCumulativeAmount1(mid, totalLiquidity, minTick, length0, length1, alpha0X96, alpha1X96, weight0, weight1)
		if err != nil {
			return false, 0, err
		}

		if amount1.Cmp(cumulativeAmount1_) >= 0 {
			right = mid
		} else {
			left = mid + c.tickSpacing
		}
	}

	return true, left, nil
}
