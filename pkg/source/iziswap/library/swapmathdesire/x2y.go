package swapmathdesire

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/amountmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

var zeroBI = uint256.NewInt(0)

type X2YRangeRetState struct {
	// whether user run out of amountX
	Finished bool
	// actual cost of tokenX to buy tokenY
	CostX *uint256.Int
	// amount of acquired tokenY
	AcquireY *uint256.Int
	// final point after this swap
	FinalPt int
	// sqrt price on final point
	SqrtFinalPrice96 *uint256.Int
	// liquidity of tokenX at finalPt
	LiquidityX *uint256.Int
}

func X2YAtPrice(desireY, sqrtPrice96, currY *uint256.Int) (costX, acquireY *uint256.Int) {
	acquireY = new(uint256.Int).Set(desireY)
	if acquireY.Cmp(currY) > 0 {
		acquireY.Set(currY)
	}

	l := calc.MulDivCeil(acquireY, utils.Pow96, sqrtPrice96)
	costX = calc.MulDivCeil(l, utils.Pow96, sqrtPrice96)

	return costX, acquireY
}

type X2YAtPriceLiquidityResult struct {
	CostX         *uint256.Int
	AcquireY      *uint256.Int
	NewLiquidityX *uint256.Int
}

func x2YAtPriceLiquidity(desireY, sqrtPrice96, liquidity, liquidityX *uint256.Int) X2YAtPriceLiquidityResult {
	liquidityY := new(uint256.Int).Sub(liquidity, liquidityX)
	// desireY * 2^96 <= 2^128 * 2^96 <= 2^224 < 2^256
	maxTransformLiquidityX := calc.MulDivCeil(desireY, utils.Pow96, sqrtPrice96)
	// transformLiquidityX <= liquidityY <= uint128.max
	transformLiquidityX := calc.Min(maxTransformLiquidityX, liquidityY)
	// transformLiquidityX * 2^96 <= 2^128 * 2^96 <= 2^224 < 2^256
	costX := calc.MulDivCeil(transformLiquidityX, utils.Pow96, sqrtPrice96)
	// acquireY should not > uint128.max
	acquireY := calc.MulDivFloor(transformLiquidityX, sqrtPrice96, utils.Pow96)
	newLiquidityX := new(uint256.Int).Add(liquidityX, transformLiquidityX)

	return X2YAtPriceLiquidityResult{CostX: costX, AcquireY: acquireY, NewLiquidityX: newLiquidityX}
}

type X2YRangeCompRet struct {
	// cost of tokenX to buy tokenY
	CostX *uint256.Int
	// amount of acquired tokenY
	AcquireY *uint256.Int
	// whether all liquidity is used
	CompleteLiquidity bool
	// location point after this swap
	LocPt int
	// sqrt location after this swap
	SqrtLoc96 *uint256.Int
}

type RangeX2Y struct {
	// total liquidity in this range
	Liquidity *uint256.Int
	// sqrt price on left point
	SqrtPriceL96 *uint256.Int
	// left point of this range
	LeftPt int
	// sqrt price on right point
	SqrtPriceR96 *uint256.Int
	// right point of this range
	RightPt int
	// sqrt rate of this range
	SqrtRate96 *uint256.Int
}

func x2YRangeComplete(rg RangeX2Y, desireY *uint256.Int) X2YRangeCompRet {
	var ret X2YRangeCompRet

	maxY := amountmath.GetAmountY(rg.Liquidity, rg.SqrtPriceL96, rg.SqrtPriceR96, rg.SqrtRate96, false)
	if maxY.Cmp(desireY) <= 0 {
		ret.AcquireY = maxY
		ret.CostX = amountmath.GetAmountX(rg.Liquidity, rg.LeftPt, rg.RightPt, rg.SqrtPriceR96, rg.SqrtRate96, true)
		ret.CompleteLiquidity = true
		return ret
	}

	// 1. desireY * (rg.sqrtRate96 - 2^96)
	//    < 2^128 * 2^96
	//    = 2 ^ 224 < 2 ^ 256
	// 2. desireY < maxY = rg.liquidity * (rg.sqrtPriceR96 - rg.sqrtPriceL96) / (rg.sqrtRate96 - 2^96)
	// here, '/' means div of int
	// desireY < rg.liquidity * (rg.sqrtPriceR96 - rg.sqrtPriceL96) / (rg.sqrtRate96 - 2^96)
	// => desireY * (rg.sqrtRate96 - TwoPower.Pow96) / rg.liquidity < rg.sqrtPriceR96 - rg.sqrtPriceL96
	// => rg.sqrtPriceR96 - desireY * (rg.sqrtRate96 - TwoPower.Pow96) / rg.liquidity > rg.sqrtPriceL96
	cl := new(uint256.Int).Sub(
		rg.SqrtPriceR96,
		new(uint256.Int).Div(
			new(uint256.Int).Mul(
				desireY,
				new(uint256.Int).Sub(rg.SqrtRate96, utils.Pow96)),
			rg.Liquidity))
	ret.LocPt, _ = calc.GetLogSqrtPriceFloor(cl)
	ret.LocPt = ret.LocPt + 1
	ret.LocPt = min(ret.LocPt, rg.RightPt)
	ret.LocPt = max(ret.LocPt, rg.LeftPt+1)
	ret.CompleteLiquidity = false

	if ret.LocPt == rg.RightPt {
		ret.CostX = uint256.NewInt(0)
		ret.AcquireY = uint256.NewInt(0)
		ret.LocPt = ret.LocPt - 1
		ret.SqrtLoc96, _ = calc.GetSqrtPrice(ret.LocPt)
	} else {
		// rg.rightPt - ret.locPt <= 256 * 100
		// sqrtPricePrMloc96 <= 1.0001 ** 25600 * 2 ^ 96 = 13 * 2^96 < 2^100
		sqrtPricePrMloc96, _ := calc.GetSqrtPrice(rg.RightPt - ret.LocPt)
		// rg.sqrtPriceR96 * TwoPower.Pow96 < 2^160 * 2^96 = 2^256
		sqrtPricePrM196 := calc.MulDivCeil(rg.SqrtPriceR96, utils.Pow96, rg.SqrtRate96)
		// rg.liquidity * (sqrtPricePrMloc96 - TwoPower.Pow96) < 2^128 * 2^100 = 2^228 < 2^256
		ret.CostX = calc.MulDivCeil(rg.Liquidity, new(uint256.Int).Sub(sqrtPricePrMloc96, utils.Pow96),
			new(uint256.Int).Sub(rg.SqrtPriceR96, sqrtPricePrM196))

		ret.LocPt = ret.LocPt - 1
		ret.SqrtLoc96, _ = calc.GetSqrtPrice(ret.LocPt)

		sqrtLocA196 := new(uint256.Int).Add(
			ret.SqrtLoc96,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					ret.SqrtLoc96,
					new(uint256.Int).Sub(rg.SqrtRate96, utils.Pow96),
				),
				utils.Pow96,
			),
		)
		acquireT256 := amountmath.GetAmountY(rg.Liquidity, sqrtLocA196, rg.SqrtPriceR96, rg.SqrtRate96, false)
		// ret.acquireY <= desireY <= uint128.max
		ret.AcquireY = calc.Min(acquireT256, desireY)
	}

	return ret
}

func X2YRange(currentState utils.State, leftPt int, sqrtRate96 *uint256.Int, desireY *uint256.Int) X2YRangeRetState {
	var retState X2YRangeRetState

	retState.CostX = uint256.NewInt(0)
	retState.AcquireY = uint256.NewInt(0)
	retState.Finished = false

	currentHasY := currentState.LiquidityX.Cmp(currentState.Liquidity) < 0
	if currentHasY && (currentState.LiquidityX.Cmp(zeroBI) > 0 || leftPt == currentState.CurrentPoint) {
		ret := x2YAtPriceLiquidity(desireY, currentState.SqrtPrice96, currentState.Liquidity, currentState.LiquidityX)
		retState.CostX = ret.CostX
		retState.AcquireY = ret.AcquireY
		retState.LiquidityX = ret.NewLiquidityX
		if retState.LiquidityX.Cmp(currentState.Liquidity) < 0 || retState.AcquireY.Cmp(desireY) >= 0 {
			retState.Finished = true
			retState.FinalPt = currentState.CurrentPoint
			retState.SqrtFinalPrice96 = currentState.SqrtPrice96
		} else {
			desireY.Sub(desireY, retState.AcquireY)
		}
	} else if currentHasY {
		currentState.CurrentPoint = currentState.CurrentPoint + 1
		currentState.SqrtPrice96 = new(uint256.Int).Add(
			currentState.SqrtPrice96,
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					currentState.SqrtPrice96,
					new(uint256.Int).Sub(sqrtRate96, utils.Pow96),
				),
				utils.Pow96,
			),
		)
	} else {
		retState.LiquidityX = currentState.LiquidityX
	}

	if retState.Finished {
		return retState
	}

	if leftPt < currentState.CurrentPoint {
		sqrtPriceL96, _ := calc.GetSqrtPrice(leftPt)
		ret := x2YRangeComplete(
			RangeX2Y{
				Liquidity:    currentState.Liquidity,
				SqrtPriceL96: sqrtPriceL96,
				LeftPt:       leftPt,
				SqrtPriceR96: currentState.SqrtPrice96,
				RightPt:      currentState.CurrentPoint,
				SqrtRate96:   sqrtRate96,
			},
			desireY,
		)
		retState.CostX.Add(retState.CostX, ret.CostX)
		desireY.Sub(desireY, ret.AcquireY)
		retState.AcquireY.Add(retState.AcquireY, ret.AcquireY)
		if ret.CompleteLiquidity {
			retState.Finished = (desireY.Cmp(zeroBI) == 0)
			retState.FinalPt = leftPt
			retState.SqrtFinalPrice96 = sqrtPriceL96
			retState.LiquidityX = currentState.Liquidity
		} else {
			locRet := x2YAtPriceLiquidity(desireY, ret.SqrtLoc96, currentState.Liquidity, zeroBI)
			locCostX := locRet.CostX
			locAcquireY := locRet.AcquireY
			retState.CostX.Add(retState.CostX, locCostX)
			retState.AcquireY.Add(retState.AcquireY, locAcquireY)
			retState.Finished = true
			retState.SqrtFinalPrice96 = ret.SqrtLoc96
			retState.FinalPt = ret.LocPt
		}
	} else {
		retState.FinalPt = currentState.CurrentPoint
		retState.SqrtFinalPrice96 = currentState.SqrtPrice96
	}

	return retState
}
