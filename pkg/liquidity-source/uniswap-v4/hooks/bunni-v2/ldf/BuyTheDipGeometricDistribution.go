package ldf

import (
	buythedipLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/ldf/libs/buy-the-dip-geometric"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/hooks/bunni-v2/math"

	"github.com/holiman/uint256"
)

// BuyTheDipGeometricDistribution represents a buy the dip geometric distribution LDF
type BuyTheDipGeometricDistribution struct {
	tickSpacing int
}

// NewBuyTheDipGeometricDistribution creates a new BuyTheDipGeometricDistribution
func NewBuyTheDipGeometricDistribution(tickSpacing int) ILiquidityDensityFunction {
	return &BuyTheDipGeometricDistribution{
		tickSpacing: tickSpacing,
	}
}

// decodeParams decodes the LDF parameters from bytes32
func (b *BuyTheDipGeometricDistribution) decodeParams(ldfParams [32]byte) (
	minTick, length, altThreshold int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThresholdDirection bool,
) {
	// | shiftMode - 1 byte | minTick - 3 bytes | length - 2 bytes | alpha - 4 bytes | altAlpha - 4 bytes | altThreshold - 3 bytes | altThresholdDirection - 1 byte |
	// minTick = int24(uint24(bytes3(ldfParams << 8)))
	minTick = int(int32(uint32(ldfParams[1])<<16 | uint32(ldfParams[2])<<8 | uint32(ldfParams[3])))

	// length = int24(int16(uint16(bytes2(ldfParams << 32))))
	length = int(int16(uint16(ldfParams[4])<<8 | uint16(ldfParams[5])))

	// uint256 alpha = uint32(bytes4(ldfParams << 48))
	alpha := uint32(ldfParams[6])<<24 | uint32(ldfParams[7])<<16 | uint32(ldfParams[8])<<8 | uint32(ldfParams[9])
	// alphaX96 = alpha.mulDiv(Q96, ALPHA_BASE)
	alphaX96 = uint256.NewInt(uint64(alpha))
	alphaX96.Mul(alphaX96, math.Q96)
	alphaX96.Div(alphaX96, math.ALPHA_BASE)

	// uint256 altAlpha = uint32(bytes4(ldfParams << 80))
	altAlpha := uint32(ldfParams[10])<<24 | uint32(ldfParams[11])<<16 | uint32(ldfParams[12])<<8 | uint32(ldfParams[13])
	// altAlphaX96 = altAlpha.mulDiv(Q96, ALPHA_BASE)
	altAlphaX96 = uint256.NewInt(uint64(altAlpha))
	altAlphaX96.Mul(altAlphaX96, math.Q96)
	altAlphaX96.Div(altAlphaX96, math.ALPHA_BASE)

	// altThreshold = int24(uint24(bytes3(ldfParams << 112)))
	altThreshold = int(int32(uint32(ldfParams[14])<<16 | uint32(ldfParams[15])<<8 | uint32(ldfParams[16])))

	// altThresholdDirection = uint8(bytes1(ldfParams << 136)) != 0
	altThresholdDirection = ldfParams[17] != 0

	return
}

// encodeState encodes the state into bytes32
func (b *BuyTheDipGeometricDistribution) encodeState(twapTick int) [32]byte {
	var state [32]byte
	state[0] = 1 // initialized = true
	state[1] = byte((twapTick >> 16) & 0xFF)
	state[2] = byte((twapTick >> 8) & 0xFF)
	state[3] = byte(twapTick & 0xFF)
	return state
}

// decodeBuyTheDipState decodes the LDF state from bytes32 for BuyTheDipGeometricDistribution
func decodeBuyTheDipState(ldfState [32]byte) (initialized bool, lastTwapTick int32) {
	// | initialized - 1 byte | lastTwapTick - 3 bytes |
	initialized = ldfState[0] == 1
	lastTwapTick = int32(uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3]))
	return
}

// Query implements the Query method for BuyTheDipGeometricDistribution
func (b *BuyTheDipGeometricDistribution) Query(
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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := b.decodeParams(ldfParams)
	initialized, lastTwapTick := decodeBuyTheDipState(ldfState)

	if initialized {
		// should surge if switched from one alpha to another
		shouldSurge = buythedipLib.ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) !=
			buythedipLib.ShouldUseAltAlpha(int(lastTwapTick), altThreshold, altThresholdDirection)
	}

	// compute liquidityDensityX96
	liquidityDensityX96, err = buythedipLib.LiquidityDensityX96(
		b.tickSpacing,
		roundedTick,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	// compute cumulativeAmount0DensityX96
	cumulativeAmount0DensityX96, err = buythedipLib.CumulativeAmount0(
		b.tickSpacing,
		roundedTick+b.tickSpacing,
		math.Q96,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	// compute cumulativeAmount1DensityX96
	cumulativeAmount1DensityX96, err = buythedipLib.CumulativeAmount1(
		b.tickSpacing,
		roundedTick-b.tickSpacing,
		math.Q96,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return nil, nil, nil, [32]byte{}, false, err
	}

	newLdfState = b.encodeState(twapTick)
	return
}

func (b *BuyTheDipGeometricDistribution) ComputeSwap(
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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := b.decodeParams(ldfParams)

	return b.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
}

// computeSwap computes the swap parameters
func (b *BuyTheDipGeometricDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	twapTick, minTick, length int,
	alphaX96, altAlphaX96 *uint256.Int,
	altThreshold int,
	altThresholdDirection bool,
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
		success, roundedTick, err = buythedipLib.InverseCumulativeAmount0(
			b.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			twapTick, minTick, length,
			alphaX96, altAlphaX96,
			altThreshold, altThresholdDirection,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount0_, err = buythedipLib.CumulativeAmount0(
				b.tickSpacing,
				roundedTick+b.tickSpacing,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		} else {
			cumulativeAmount0_, err = buythedipLib.CumulativeAmount0(
				b.tickSpacing,
				roundedTick,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount1_, err = buythedipLib.CumulativeAmount1(
				b.tickSpacing,
				roundedTick,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		} else {
			cumulativeAmount1_, err = buythedipLib.CumulativeAmount1(
				b.tickSpacing,
				roundedTick-b.tickSpacing,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	} else {
		// compute roundedTick by inverting the cumulative amount1
		success, roundedTick, err = buythedipLib.InverseCumulativeAmount1(
			b.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			twapTick, minTick, length,
			alphaX96, altAlphaX96,
			altThreshold, altThresholdDirection,
		)
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
		if !success {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), nil
		}

		// compute cumulative amounts
		if exactIn {
			cumulativeAmount1_, err = buythedipLib.CumulativeAmount1(
				b.tickSpacing,
				roundedTick-b.tickSpacing,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		} else {
			cumulativeAmount1_, err = buythedipLib.CumulativeAmount1(
				b.tickSpacing,
				roundedTick,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}

		if exactIn {
			cumulativeAmount0_, err = buythedipLib.CumulativeAmount0(
				b.tickSpacing,
				roundedTick,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		} else {
			cumulativeAmount0_, err = buythedipLib.CumulativeAmount0(
				b.tickSpacing,
				roundedTick+b.tickSpacing,
				totalLiquidity,
				twapTick,
				minTick,
				length,
				alphaX96,
				altAlphaX96,
				altThreshold,
				altThresholdDirection,
			)
		}
		if err != nil {
			return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
		}
	}

	// compute swap liquidity
	swapLiquidity, err = buythedipLib.LiquidityDensityX96(
		b.tickSpacing,
		roundedTick,
		twapTick,
		minTick,
		length,
		alphaX96,
		altAlphaX96,
		altThreshold,
		altThresholdDirection,
	)
	if err != nil {
		return false, 0, uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0), err
	}

	swapLiquidity.Mul(swapLiquidity, totalLiquidity)
	swapLiquidity.Rsh(swapLiquidity, 96)

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}
