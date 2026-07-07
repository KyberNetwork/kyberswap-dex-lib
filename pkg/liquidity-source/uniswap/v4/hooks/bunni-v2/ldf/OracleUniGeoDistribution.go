package ldf

import (
	oracleUniGeoLib "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/ldf/libs/oracle-uni-geometric"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/math"
	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/holiman/uint256"
)

type OracleUniGeoParams struct {
	BondLtStablecoin bool
	FloorPrice       *uint256.Int
	LdfParamOverride LdfParamsOverride
}

type LdfParamsOverride struct {
	Overridden bool
	LdfParams  [32]byte
}

type OracleUniGeoDistribution struct {
	*OracleUniGeoParams
	tickSpacing int
}

// NewOracleUniGeoDistribution creates a new OracleUniGeoDistribution
func NewOracleUniGeoDistribution(tickSpacing int, params *OracleUniGeoParams) ILiquidityDensityFunction {
	return &OracleUniGeoDistribution{
		tickSpacing:        tickSpacing,
		OracleUniGeoParams: params,
	}
}

// ComputeSwap implements the ComputeSwap method for OracleUniGeoDistribution
func (o *OracleUniGeoDistribution) ComputeSwap(
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
	if o.LdfParamOverride.Overridden {
		ldfParams = o.LdfParamOverride.LdfParams
	}

	oracleTick, err := o.floorPriceToTick(o.FloorPrice)
	if err != nil {
		return
	}

	tickLower, tickUpper, alphaX96, distributionType := oracleUniGeoLib.DecodeParams(o.tickSpacing, oracleTick, ldfParams)

	return o.computeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity,
		zeroForOne,
		exactIn,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)
}

// computeSwap computes the swap parameters
func (o *OracleUniGeoDistribution) computeSwap(
	inverseCumulativeAmountInput,
	totalLiquidity *uint256.Int,
	zeroForOne,
	exactIn bool,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType oracleUniGeoLib.DistributionType,
) (
	success bool,
	roundedTick int,
	cumulativeAmount0_,
	cumulativeAmount1_,
	swapLiquidity *uint256.Int,
	err error,
) {
	if exactIn == zeroForOne {
		success, roundedTick, err = oracleUniGeoLib.InverseCumulativeAmount0(
			o.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			tickLower,
			tickUpper,
			alphaX96,
			distributionType,
		)
		if !success || err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}

		if exactIn {
			cumulativeAmount0_, err = oracleUniGeoLib.CumulativeAmount0(
				o.tickSpacing,
				roundedTick+o.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		} else {
			cumulativeAmount0_, err = oracleUniGeoLib.CumulativeAmount0(
				o.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		}
		if err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}

		if exactIn {
			cumulativeAmount1_, err = oracleUniGeoLib.CumulativeAmount1(
				o.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		} else {
			cumulativeAmount1_, err = oracleUniGeoLib.CumulativeAmount1(
				o.tickSpacing,
				roundedTick-o.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		}
		if err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}
	} else {
		success, roundedTick, err = oracleUniGeoLib.InverseCumulativeAmount1(
			o.tickSpacing,
			inverseCumulativeAmountInput,
			totalLiquidity,
			tickLower,
			tickUpper,
			alphaX96,
			distributionType,
		)
		if !success || err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, nil
		}

		if exactIn {
			cumulativeAmount1_, err = oracleUniGeoLib.CumulativeAmount1(
				o.tickSpacing,
				roundedTick-o.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		} else {
			cumulativeAmount1_, err = oracleUniGeoLib.CumulativeAmount1(
				o.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		}
		if err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}

		if exactIn {
			cumulativeAmount0_, err = oracleUniGeoLib.CumulativeAmount0(
				o.tickSpacing,
				roundedTick,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		} else {
			cumulativeAmount0_, err = oracleUniGeoLib.CumulativeAmount0(
				o.tickSpacing,
				roundedTick+o.tickSpacing,
				totalLiquidity,
				tickLower,
				tickUpper,
				alphaX96,
				distributionType,
			)
		}
		if err != nil {
			return false, 0, u256.U0, u256.U0, u256.U0, err
		}
	}

	swapLiquidity, err = oracleUniGeoLib.LiquidityDensityX96(
		o.tickSpacing,
		roundedTick,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)
	if err != nil {
		return false, 0, u256.U0, u256.U0, u256.U0, err
	}

	return true, roundedTick, cumulativeAmount0_, cumulativeAmount1_, swapLiquidity, nil
}

// Query implements the Query method for OracleUniGeoDistribution
func (o *OracleUniGeoDistribution) Query(
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
	if o.LdfParamOverride.Overridden {
		ldfParams = o.LdfParamOverride.LdfParams
	}

	oracleTick, err := o.floorPriceToTick(o.FloorPrice)
	if err != nil {
		return
	}

	tickLower, tickUpper, alphaX96, distributionType := oracleUniGeoLib.DecodeParams(o.tickSpacing, oracleTick, ldfParams)

	initialized, lastoracleTick, lastLdfParams := o.DecodeState(ldfState)

	if initialized {
		shouldSurge = lastLdfParams != ldfParams || oracleTick != lastoracleTick
	}

	liquidityDensityX96, cumulativeAmount0DensityX96, cumulativeAmount1DensityX96, err = o.query(
		roundedTick,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)
	if err != nil {
		return
	}

	newLdfState = o.EncodeState(oracleTick, ldfParams)

	return
}

func (o *OracleUniGeoDistribution) query(
	roundedTick,
	tickLower,
	tickUpper int,
	alphaX96 *uint256.Int,
	distributionType oracleUniGeoLib.DistributionType,
) (
	liquidityDensityX96 *uint256.Int,
	cumulativeAmount0DensityX96 *uint256.Int,
	cumulativeAmount1DensityX96 *uint256.Int,
	err error,
) {
	liquidityDensityX96, err = oracleUniGeoLib.LiquidityDensityX96(
		o.tickSpacing,
		roundedTick,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)

	if err != nil {
		return
	}

	cumulativeAmount0DensityX96, err = oracleUniGeoLib.CumulativeAmount0(
		o.tickSpacing,
		roundedTick+o.tickSpacing,
		math.Q96,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)
	if err != nil {
		return
	}

	cumulativeAmount1DensityX96, err = oracleUniGeoLib.CumulativeAmount1(
		o.tickSpacing,
		roundedTick-o.tickSpacing,
		math.Q96,
		tickLower,
		tickUpper,
		alphaX96,
		distributionType,
	)
	if err != nil {
		return
	}

	return
}

func (o *OracleUniGeoDistribution) floorPriceToTick(floorPriceWad *uint256.Int) (int, error) {
	var sqrtPriceX96 uint256.Int
	sqrtPriceX96.Lsh(floorPriceWad, 192)
	sqrtPriceX96.Div(&sqrtPriceX96, math.WAD)
	sqrtPriceX96.Sqrt(&sqrtPriceX96)

	rick, err := math.GetTickAtSqrtPrice(&sqrtPriceX96)
	if err != nil {
		return 0, err
	}

	if !o.BondLtStablecoin {
		rick = -rick
	}

	return math.RoundTickSingle(rick, o.tickSpacing), nil
}

func (o *OracleUniGeoDistribution) DecodeState(ldfState [32]byte) (
	initialized bool,
	lastOracleTick int,
	lastLdfParams [32]byte,
) {
	// | initialized - 1 byte | lastOracleTick - 3 bytes | lastLdfParams - 12 bytes |

	initialized = ldfState[0] != 0

	lastOracleTickRaw := uint32(ldfState[1])<<16 | uint32(ldfState[2])<<8 | uint32(ldfState[3])
	if lastOracleTickRaw&0x800000 != 0 {
		lastOracleTickRaw |= 0xFF000000
	}
	lastOracleTick = int(int32(lastOracleTickRaw))

	copy(lastLdfParams[:12], ldfState[4:16])

	return
}

func (o *OracleUniGeoDistribution) EncodeState(lastOracleTick int, lastLdfParams [32]byte) [32]byte {
	// | initialized - 1 byte | lastOracleTick - 3 bytes | lastLdfParams - 12 bytes |

	var ldfState [32]byte

	ldfState[0] = 1

	tickUint := uint32(lastOracleTick) & 0x00FFFFFF
	ldfState[1] = byte(tickUint >> 16)
	ldfState[2] = byte(tickUint >> 8)
	ldfState[3] = byte(tickUint)

	copy(ldfState[4:16], lastLdfParams[:12])

	return ldfState
}
