package swapmath

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/amountmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

type Y2XRangeRetState struct {
	// whether user has run out of tokenY
	Finished bool
	// actual cost of tokenY to buy tokenX
	CostY *uint256.Int
	// actual amount of tokenX acquired
	AcquireX *uint256.Int
	// final point after this swap
	FinalPt int
	// sqrt price on final point
	SqrtFinalPrice96 *uint256.Int
	// liquidity of tokenX at finalPt
	// if finalPt is not rightPt, liquidityX is meaningless
	LiquidityX *uint256.Int
}

func Y2XAtPrice(amountY *uint256.Int, sqrtPrice96 *uint256.Int, currX *uint256.Int) (costY, acquireX *uint256.Int) {
	l := calc.MulDivFloor(amountY, utils.Pow96, sqrtPrice96)
	// acquireX <= currX <= uint128.max
	acquireX = calc.Min(calc.MulDivFloor(l, utils.Pow96, sqrtPrice96), currX)
	l = calc.MulDivCeil(acquireX, sqrtPrice96, utils.Pow96)
	costY = calc.MulDivCeil(l, sqrtPrice96, utils.Pow96)
	return costY, acquireX
}

type Y2XAtPriceLiquidityResult struct {
	CostY         *uint256.Int
	AcquireX      *uint256.Int
	NewLiquidityX *uint256.Int
}

func y2XAtPriceLiquidity(amountY *uint256.Int, sqrtPrice96 *uint256.Int,
	liquidityX *uint256.Int) Y2XAtPriceLiquidityResult {
	// amountY * TwoPower.Pow96 < 2^128 * 2^96 = 2^224 < 2^256
	maxTransformLiquidityY := new(uint256.Int).Mul(amountY, utils.Pow96)
	maxTransformLiquidityY.Div(maxTransformLiquidityY, sqrtPrice96)
	// transformLiquidityY <= liquidityX
	transformLiquidityY := calc.Min(maxTransformLiquidityY, liquidityX)
	// costY <= amountY
	costY := calc.MulDivCeil(transformLiquidityY, sqrtPrice96, utils.Pow96)
	// transformLiquidityY * 2^96 < 2^224 < 2^256
	acquireX := new(uint256.Int).Mul(transformLiquidityY, utils.Pow96)
	acquireX.Div(acquireX, sqrtPrice96)
	newLiquidityX := new(uint256.Int).Sub(liquidityX, transformLiquidityY)
	return Y2XAtPriceLiquidityResult{CostY: costY, AcquireX: acquireX, NewLiquidityX: newLiquidityX}
}

type RangeY2X struct {
	Liquidity    *uint256.Int
	SqrtPriceL96 *uint256.Int
	LeftPt       int
	SqrtPriceR96 *uint256.Int
	RightPt      int
	SqrtRate96   *uint256.Int
}

type Y2XRangeCompRet struct {
	CostY             *uint256.Int
	AcquireX          *uint256.Int
	CompleteLiquidity bool
	LocPt             int
	SqrtLoc96         *uint256.Int
}

func y2XRangeComplete(rg RangeY2X, amountY *uint256.Int) Y2XRangeCompRet {
	ret := Y2XRangeCompRet{}
	maxY := amountmath.GetAmountY(rg.Liquidity, rg.SqrtPriceL96, rg.SqrtPriceR96, rg.SqrtRate96, true)
	if maxY.Cmp(amountY) <= 0 {
		// ret.costY <= maxY <= uint128.max
		ret.CostY = maxY
		ret.AcquireX = amountmath.GetAmountX(rg.Liquidity, rg.LeftPt, rg.RightPt, rg.SqrtPriceR96, rg.SqrtRate96, false)
		// we complete this liquidity segment
		ret.CompleteLiquidity = true
	} else {
		// we should locate highest price
		// uint160 is enough for muldiv and adding, because amountY < maxY
		sqrtLoc96 := calc.MulDivFloor(amountY, new(uint256.Int).Sub(rg.SqrtRate96, utils.Pow96), rg.Liquidity)
		sqrtLoc96.Add(sqrtLoc96, rg.SqrtPriceL96)
		ret.LocPt, _ = calc.GetLogSqrtPriceFloor(sqrtLoc96)

		ret.LocPt = max(rg.LeftPt, ret.LocPt)
		ret.LocPt = min(rg.RightPt-1, ret.LocPt)

		ret.CompleteLiquidity = false
		ret.SqrtLoc96, _ = calc.GetSqrtPrice(ret.LocPt)
		if ret.LocPt == rg.LeftPt {
			ret.CostY = uint256.NewInt(0)
			ret.AcquireX = uint256.NewInt(0)
			return ret
		}

		costY256 := amountmath.GetAmountY(rg.Liquidity, rg.SqrtPriceL96, ret.SqrtLoc96, rg.SqrtRate96, true)
		// ret.costY <= amountY <= uint128.max
		ret.CostY = calc.Min(costY256, amountY)

		// costY <= amountY even if the costY is the upperbound of the result
		// because amountY is not a real and sqrtLoc96 <= sqrtLoc25696
		ret.AcquireX = amountmath.GetAmountX(rg.Liquidity, rg.LeftPt, ret.LocPt, ret.SqrtLoc96, rg.SqrtRate96, false)

	}
	return ret
}

func Y2XRange(currentState utils.State, rightPt int, sqrtRate96 *uint256.Int, amountY *uint256.Int) Y2XRangeRetState {
	retState := Y2XRangeRetState{
		CostY:      uint256.NewInt(0),
		AcquireX:   uint256.NewInt(0),
		Finished:   false,
		LiquidityX: uint256.NewInt(0),
	}

	// first, if current point is not all x, we can not move right directly
	startHasY := currentState.LiquidityX.Cmp(currentState.Liquidity) < 0
	if startHasY {
		ret := y2XAtPriceLiquidity(amountY, currentState.SqrtPrice96, currentState.LiquidityX)
		retState.LiquidityX = ret.NewLiquidityX
		retState.CostY = ret.CostY
		retState.AcquireX = ret.AcquireX
		if retState.LiquidityX.Sign() > 0 || retState.CostY.Cmp(amountY) >= 0 {
			retState.Finished = true
			retState.FinalPt = currentState.CurrentPoint
			retState.SqrtFinalPrice96 = currentState.SqrtPrice96
			return retState
		} else {
			amountY.Sub(amountY, retState.CostY)
			currentState.CurrentPoint += 1
			if currentState.CurrentPoint == rightPt {
				retState.FinalPt = currentState.CurrentPoint
				retState.SqrtFinalPrice96, _ = calc.GetSqrtPrice(rightPt)
				return retState
			}
			// sqrt(price) + sqrt(price) * (1.0001 - 1) == sqrt(price) * 1.0001
			mulDelta := new(uint256.Int).Mul(currentState.SqrtPrice96, new(uint256.Int).Sub(sqrtRate96, utils.Pow96))
			mulDeltaDiv := new(uint256.Int).Div(mulDelta, utils.Pow96)
			currentState.SqrtPrice96 = new(uint256.Int).Add(currentState.SqrtPrice96, mulDeltaDiv)
		}
	}

	sqrtPriceR96, _ := calc.GetSqrtPrice(rightPt)

	// (uint128 liquidCostY, uint256 liquidAcquireX, bool liquidComplete, int24 locPt, uint160 sqrtLoc96)
	ret := y2XRangeComplete(
		RangeY2X{
			Liquidity:    currentState.Liquidity,
			SqrtPriceL96: currentState.SqrtPrice96,
			LeftPt:       currentState.CurrentPoint,
			SqrtPriceR96: sqrtPriceR96,
			RightPt:      rightPt,
			SqrtRate96:   sqrtRate96,
		},
		amountY,
	)

	retState.CostY.Add(retState.CostY, ret.CostY)
	amountY.Sub(amountY, ret.CostY)
	retState.AcquireX.Add(retState.AcquireX, ret.AcquireX)
	if ret.CompleteLiquidity {
		retState.Finished = amountY.Sign() == 0
		retState.FinalPt = rightPt
		retState.SqrtFinalPrice96 = sqrtPriceR96
	} else {

		// locCostY, locAcquireX, retState.LiquidityX =
		locRet := y2XAtPriceLiquidity(amountY, ret.SqrtLoc96, currentState.Liquidity)
		locCostY := locRet.CostY
		locAcquireX := locRet.AcquireX
		retState.LiquidityX = locRet.NewLiquidityX

		retState.CostY.Add(retState.CostY, locCostY)
		retState.AcquireX.Add(retState.AcquireX, locAcquireX)
		retState.Finished = true
		retState.SqrtFinalPrice96 = ret.SqrtLoc96
		retState.FinalPt = ret.LocPt
	}
	return retState
}
