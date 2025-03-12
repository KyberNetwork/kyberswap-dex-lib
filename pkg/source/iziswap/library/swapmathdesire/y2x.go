package swapmathdesire

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

func Y2XAtPrice(desireX *uint256.Int, sqrtPrice96 *uint256.Int, currX *uint256.Int) (costY, acquireX *uint256.Int) {
	acquireX = calc.Min(desireX, currX)
	l := calc.MulDivCeil(acquireX, sqrtPrice96, utils.Pow96)
	costY = calc.MulDivCeil(l, sqrtPrice96, utils.Pow96)
	return costY, acquireX
}

type Y2XAtPriceLiquidityResult struct {
	CostY         *uint256.Int
	AcquireX      *uint256.Int
	NewLiquidityX *uint256.Int
}

func y2XAtPriceLiquidity(desireX *uint256.Int, sqrtPrice96 *uint256.Int,
	liquidityX *uint256.Int) Y2XAtPriceLiquidityResult {
	maxTransformLiquidityY := calc.MulDivCeil(desireX, sqrtPrice96, utils.Pow96)
	transformLiquidityY := calc.Min(maxTransformLiquidityY, liquidityX)
	costY := calc.MulDivCeil(transformLiquidityY, sqrtPrice96, utils.Pow96)
	acquireX := new(uint256.Int).Div(
		new(uint256.Int).Mul(
			transformLiquidityY,
			utils.Pow96,
		),
		sqrtPrice96,
	)
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

func y2XRangeComplete(rg RangeY2X, desireX *uint256.Int) Y2XRangeCompRet {
	ret := Y2XRangeCompRet{}

	maxX := amountmath.GetAmountX(rg.Liquidity, rg.LeftPt, rg.RightPt, rg.SqrtPriceR96, rg.SqrtRate96, false)
	if maxX.Cmp(desireX) <= 0 {
		ret.AcquireX = new(uint256.Int).Set(maxX)
		ret.CostY = amountmath.GetAmountY(rg.Liquidity, rg.SqrtPriceL96, rg.SqrtPriceR96, rg.SqrtRate96, true)
		ret.CompleteLiquidity = true

		return ret
	}

	sqrtPricePrPl96, _ := calc.GetSqrtPrice(rg.RightPt - rg.LeftPt)
	sqrtPricePrM196 := new(uint256.Int).Div(
		new(uint256.Int).Mul(rg.SqrtPriceR96, utils.Pow96),
		rg.SqrtRate96,
	)
	div := new(uint256.Int).Sub(
		sqrtPricePrPl96,
		calc.MulDivFloor(desireX, new(uint256.Int).Sub(rg.SqrtPriceR96, sqrtPricePrM196), rg.Liquidity),
	)
	sqrtPriceLoc96 := new(uint256.Int).Div(
		new(uint256.Int).Mul(rg.SqrtPriceR96, utils.Pow96),
		div,
	)

	ret.CompleteLiquidity = false
	ret.LocPt, _ = calc.GetLogSqrtPriceFloor(sqrtPriceLoc96)
	ret.LocPt = max(rg.LeftPt, ret.LocPt)
	ret.LocPt = min(rg.RightPt-1, ret.LocPt)
	ret.SqrtLoc96, _ = calc.GetSqrtPrice(ret.LocPt)

	if ret.LocPt == rg.LeftPt {
		ret.AcquireX = uint256.NewInt(0)
		ret.CostY = uint256.NewInt(0)
		return ret
	}

	ret.CompleteLiquidity = false
	ret.AcquireX = calc.Min(
		amountmath.GetAmountX(
			rg.Liquidity,
			rg.LeftPt,
			ret.LocPt,
			ret.SqrtLoc96,
			rg.SqrtRate96,
			false,
		),
		desireX,
	)

	ret.CostY = amountmath.GetAmountY(
		rg.Liquidity,
		rg.SqrtPriceL96,
		ret.SqrtLoc96,
		rg.SqrtRate96,
		true,
	)

	return ret
}

func Y2XRange(currentState utils.State, rightPt int, sqrtRate96 *uint256.Int, desireX *uint256.Int) Y2XRangeRetState {
	retState := Y2XRangeRetState{
		CostY:      uint256.NewInt(0),
		AcquireX:   uint256.NewInt(0),
		Finished:   false,
		LiquidityX: uint256.NewInt(0),
	}

	startHasY := currentState.LiquidityX.Cmp(currentState.Liquidity) < 0
	if startHasY {
		ret := y2XAtPriceLiquidity(desireX, currentState.SqrtPrice96, currentState.LiquidityX)
		retState.LiquidityX = ret.NewLiquidityX
		retState.CostY = ret.CostY
		retState.AcquireX = ret.AcquireX
		if retState.LiquidityX.Sign() > 0 || retState.AcquireX.Cmp(desireX) >= 0 {
			retState.Finished = true
			retState.FinalPt = currentState.CurrentPoint
			retState.SqrtFinalPrice96 = currentState.SqrtPrice96
			return retState
		} else {
			desireX.Sub(desireX, retState.AcquireX)
			currentState.CurrentPoint += 1
			if currentState.CurrentPoint == rightPt {
				retState.FinalPt = currentState.CurrentPoint
				retState.SqrtFinalPrice96, _ = calc.GetSqrtPrice(rightPt)
				return retState
			}
			mulDelta := new(uint256.Int).Mul(currentState.SqrtPrice96, new(uint256.Int).Sub(sqrtRate96, utils.Pow96))
			mulDeltaDiv := new(uint256.Int).Div(mulDelta, utils.Pow96)
			currentState.SqrtPrice96 = new(uint256.Int).Add(currentState.SqrtPrice96, mulDeltaDiv)
		}
	}

	sqrtPriceR96, _ := calc.GetSqrtPrice(rightPt)
	ret := y2XRangeComplete(
		RangeY2X{
			Liquidity:    currentState.Liquidity,
			SqrtPriceL96: currentState.SqrtPrice96,
			LeftPt:       currentState.CurrentPoint,
			SqrtPriceR96: sqrtPriceR96,
			RightPt:      rightPt,
			SqrtRate96:   sqrtRate96,
		},
		desireX,
	)
	retState.CostY.Add(retState.CostY, ret.CostY)
	retState.AcquireX.Add(retState.AcquireX, ret.AcquireX)
	desireX.Sub(desireX, ret.AcquireX)

	if ret.CompleteLiquidity {
		retState.Finished = desireX.Sign() == 0
		retState.FinalPt = rightPt
		retState.SqrtFinalPrice96 = sqrtPriceR96
	} else {
		locRet := y2XAtPriceLiquidity(desireX, ret.SqrtLoc96, currentState.Liquidity)
		locCostY := locRet.CostY
		locAcquireX := locRet.AcquireX
		retState.LiquidityX = locRet.NewLiquidityX
		retState.CostY.Add(retState.CostY, locCostY)
		retState.AcquireX.Add(retState.AcquireX, locAcquireX)
		retState.Finished = true
		retState.FinalPt = ret.LocPt
		retState.SqrtFinalPrice96 = ret.SqrtLoc96
	}

	return retState
}
