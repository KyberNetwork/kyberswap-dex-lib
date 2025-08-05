package ldf

import "github.com/holiman/uint256"

type ILiquidityDensityFunction interface {
	ComputeSwap(
		inverseCumulativeAmountInput,
		totalLiquidity *uint256.Int,
		zeroForOne,
		exactIn bool,
		twapTick,
		spotPriceTick int,
		ldfParams,
		ldfState [32]byte,
	) (
		success bool,
		roundedTick int,
		cumulativeAmount0,
		cumulativeAmount1,
		swapLiquidity *uint256.Int,
		err error,
	)

	Query(
		roundedTick,
		twapTick,
		spotPriceTick int,
		ldfParams,
		ldfState [32]byte,
	) (
		liquidityDensityX96,
		cumulativeAmount0DensityX96,
		cumulativeAmount1DensityX96 *uint256.Int,
		newLdfState [32]byte,
		shouldSurge bool,
		err error,
	)
}
