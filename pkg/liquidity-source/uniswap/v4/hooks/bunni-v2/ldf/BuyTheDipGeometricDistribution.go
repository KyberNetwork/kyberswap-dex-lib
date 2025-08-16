package ldf

import (
	buythedipLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/buy-the-dip-geometric"

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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := buythedipLib.DecodeParams(ldfParams)
	initialized, lastTwapTick := DecodeState(ldfState)

	if initialized {
		shouldSurge = buythedipLib.ShouldUseAltAlpha(twapTick, altThreshold, altThresholdDirection) !=
			buythedipLib.ShouldUseAltAlpha(int(lastTwapTick), altThreshold, altThresholdDirection)
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = buythedipLib.Query(
		roundedTick,
		b.tickSpacing,
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

	newLdfState = EncodeState(twapTick)
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
	minTick, length, altThreshold, alphaX96, altAlphaX96, altThresholdDirection := buythedipLib.DecodeParams(ldfParams)

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
