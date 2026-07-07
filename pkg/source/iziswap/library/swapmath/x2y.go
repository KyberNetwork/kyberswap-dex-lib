package swapmath

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/amountmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

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

func X2YAtPrice(amountX, sqrtPrice96, currY *uint256.Int) (costX, acquireY *uint256.Int) {

	l := calc.MulDivFloor(amountX, sqrtPrice96, utils.Pow96)
	acquireY = calc.MulDivFloor(l, sqrtPrice96, utils.Pow96)
	if acquireY.Cmp(currY) > 0 {
		acquireY.Set(currY)
	}
	l = calc.MulDivCeil(acquireY, utils.Pow96, sqrtPrice96)
	costX = calc.MulDivCeil(l, utils.Pow96, sqrtPrice96)

	return costX, acquireY
}

type X2YAtPriceLiquidityResult struct {
	CostX         *uint256.Int
	AcquireY      *uint256.Int
	NewLiquidityX *uint256.Int
}

func x2YAtPriceLiquidity(amountX, sqrtPrice96, liquidity, liquidityX *uint256.Int) X2YAtPriceLiquidityResult {
	var costX, acquireY, newLiquidityX *uint256.Int
	var maxTransformLiquidityX, transformLiquidityX *uint256.Int

	liquidityY := new(uint256.Int).Sub(liquidity, liquidityX)
	maxTransformLiquidityX = calc.MulDivFloor(amountX, sqrtPrice96, utils.Pow96)
	transformLiquidityX = calc.Min(maxTransformLiquidityX, liquidityY)

	costX = calc.MulDivCeil(transformLiquidityX, utils.Pow96, sqrtPrice96)
	acquireY = calc.MulDivFloor(transformLiquidityX, sqrtPrice96, utils.Pow96)
	newLiquidityX = new(uint256.Int).Add(liquidityX, transformLiquidityX)

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

func x2YRangeComplete(rg RangeX2Y, amountX *uint256.Int) X2YRangeCompRet {
	var ret X2YRangeCompRet
	sqrtPricePrM196 := calc.MulDivCeil(rg.SqrtPriceR96, utils.Pow96, rg.SqrtRate96)
	sqrtPricePrMl96, _ := calc.GetSqrtPrice(rg.RightPt - rg.LeftPt)
	maxX := calc.MulDivCeil(rg.Liquidity, new(uint256.Int).Sub(sqrtPricePrMl96, utils.Pow96),
		new(uint256.Int).Sub(rg.SqrtPriceR96, sqrtPricePrM196))

	if maxX.Cmp(amountX) <= 0 {
		ret.CostX = maxX
		ret.AcquireY = amountmath.GetAmountY(rg.Liquidity, rg.SqrtPriceL96, rg.SqrtPriceR96, rg.SqrtRate96, false)
		ret.CompleteLiquidity = true
	} else {
		sqrtValue96 := new(uint256.Int).Add(
			new(uint256.Int).Div(
				new(uint256.Int).Mul(
					amountX,
					new(uint256.Int).Sub(rg.SqrtPriceR96, sqrtPricePrM196),
				),
				rg.Liquidity,
			),
			utils.Pow96,
		)

		logValue, _ := calc.GetLogSqrtPriceFloor(sqrtValue96)

		ret.LocPt = rg.RightPt - logValue
		ret.LocPt = min(ret.LocPt, rg.RightPt)
		ret.LocPt = max(ret.LocPt, rg.LeftPt+1)
		ret.CompleteLiquidity = false

		if ret.LocPt == rg.RightPt {
			ret.CostX = uint256.NewInt(0)
			ret.AcquireY = uint256.NewInt(0)
			ret.LocPt = ret.LocPt - 1
			ret.SqrtLoc96, _ = calc.GetSqrtPrice(ret.LocPt)
		} else {
			sqrtPricePrMloc96, _ := calc.GetSqrtPrice(rg.RightPt - ret.LocPt)
			costX256 := calc.MulDivCeil(rg.Liquidity, new(uint256.Int).Sub(sqrtPricePrMloc96, utils.Pow96),
				new(uint256.Int).Sub(rg.SqrtPriceR96, sqrtPricePrM196))
			ret.CostX = calc.Min(costX256, amountX)
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
			ret.AcquireY = amountmath.GetAmountY(rg.Liquidity, sqrtLocA196, rg.SqrtPriceR96, rg.SqrtRate96, false)
		}
	}

	return ret
}

func X2YRange(currentState utils.State, leftPt int, sqrtRate96 *uint256.Int, amountX *uint256.Int) X2YRangeRetState {
	var retState X2YRangeRetState
	retState.CostX = uint256.NewInt(0)
	retState.AcquireY = uint256.NewInt(0)
	retState.LiquidityX = uint256.NewInt(0)
	retState.Finished = false

	currentHasY := currentState.LiquidityX.Cmp(currentState.Liquidity) < 0
	if currentHasY && (currentState.LiquidityX.Cmp(new(uint256.Int).SetUint64(0)) > 0 || leftPt == currentState.CurrentPoint) {
		ret := x2YAtPriceLiquidity(amountX, currentState.SqrtPrice96, currentState.Liquidity, currentState.LiquidityX)
		retState.CostX = ret.CostX
		retState.AcquireY = ret.AcquireY
		retState.LiquidityX = ret.NewLiquidityX
		if retState.LiquidityX.Cmp(currentState.Liquidity) < 0 || retState.CostX.Cmp(amountX) >= 0 {
			retState.Finished = true
			retState.FinalPt = currentState.CurrentPoint
			retState.SqrtFinalPrice96 = currentState.SqrtPrice96
		} else {
			amountX.Sub(amountX, retState.CostX)
		}
	} else if currentHasY { // all y
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
			amountX,
		)
		retState.CostX.Add(retState.CostX, ret.CostX)
		amountX.Sub(amountX, ret.CostX)
		retState.AcquireY.Add(retState.AcquireY, ret.AcquireY)
		if ret.CompleteLiquidity {
			retState.Finished = (amountX.Cmp(new(uint256.Int).SetUint64(0)) == 0)
			retState.FinalPt = leftPt
			retState.SqrtFinalPrice96 = sqrtPriceL96
			retState.LiquidityX = currentState.Liquidity
		} else {
			locRet := x2YAtPriceLiquidity(amountX, ret.SqrtLoc96, currentState.Liquidity, uint256.NewInt(0))
			locCostX := locRet.CostX
			locAcquireY := locRet.AcquireY
			retState.LiquidityX = locRet.NewLiquidityX
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
