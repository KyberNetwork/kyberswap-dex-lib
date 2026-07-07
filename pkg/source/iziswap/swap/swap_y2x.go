package swap

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/swapmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/swapmathdesire"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

func SwapY2X(amount *uint256.Int, highPt int, pool PoolInfoU256) (SwapResult, error) {
	if amount.Sign() <= 0 {
		return SwapResult{}, errors.New("AP")
	}

	highPt = min(highPt, pool.RightMostPt)

	amountX := uint256.NewInt(0)
	amountY := uint256.NewInt(0)

	sqrtPrice96, _ := calc.GetSqrtPrice(pool.CurrentPoint)

	liquidityX := new(uint256.Int).Set(pool.LiquidityX)
	liquidity := new(uint256.Int).Set(pool.Liquidity)

	finished := false
	sqrtRate96, _ := calc.GetSqrtPrice(1)
	pointDelta := pool.PointDelta
	currentPoint := pool.CurrentPoint
	fee := pool.Fee
	var crossedPoints int64

	orderData := InitY2X(
		pool.Liquidities,
		pool.LimitOrders,
		pool.CurrentPoint,
	)

	for currentPoint < highPt && !finished {
		if orderData.IsLimitOrder(currentPoint) {
			// amount <= uint128.max
			amountNoFee := new(uint256.Int).Mul(amount, uint256.NewInt(uint64(1e6-fee)))
			amountNoFee.Div(amountNoFee, uint256.NewInt(1e6))
			if amountNoFee.Sign() > 0 {
				// clear limit order first
				currX := orderData.UnsafeGetLimitSellingX()
				costY, acquireX := swapmath.Y2XAtPrice(amountNoFee, sqrtPrice96, currX)
				if acquireX.Cmp(currX) < 0 || costY.Cmp(amountNoFee) >= 0 {
					finished = true
				}
				var feeAmount *uint256.Int
				if costY.Cmp(amountNoFee) >= 0 {
					feeAmount = new(uint256.Int).Sub(amount, costY)
				} else {
					// amount <= uint128.max
					feeAmount = new(uint256.Int).Mul(costY, uint256.NewInt(uint64(fee)))
					feeAmount.Div(feeAmount, uint256.NewInt(uint64(1e6-fee)))
					mod := new(uint256.Int).Mod(new(uint256.Int).Mul(costY, uint256.NewInt(uint64(fee))),
						uint256.NewInt(uint64(1e6-fee)))
					if mod.Sign() > 0 {
						feeAmount.Add(feeAmount, uint256.NewInt(1))
					}
				}
				amount.Sub(amount, new(uint256.Int).Add(costY, feeAmount))
				amountY.Add(amountY, new(uint256.Int).Add(costY, feeAmount))
				amountX.Add(amountX, acquireX)
				orderData.ConsumeLimitOrder(true)
			} else {
				finished = true
			}
		}

		if finished {
			break
		}

		nextPoint := min(orderData.MoveY2X(currentPoint, pointDelta), highPt)

		// in [st.currentPoint, nextPoint)
		if liquidity.Sign() == 0 {
			// no liquidity in the range [st.currentPoint, nextPoint)
			currentPoint = nextPoint
			sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
			if orderData.IsLiquidity(currentPoint) {
				delta := orderData.UnsafeGetDeltaLiquidity()
				liquidity.Add(liquidity, delta)
				liquidityX = liquidity
			}
		} else {
			// amount <= uint128.max
			amountNoFee := new(uint256.Int).Mul(amount, uint256.NewInt(uint64(1e6-fee)))
			amountNoFee.Div(amountNoFee, uint256.NewInt(1e6))
			if amountNoFee.Sign() > 0 {
				st := utils.State{
					LiquidityX:   new(uint256.Int).Set(liquidityX),
					Liquidity:    new(uint256.Int).Set(liquidity),
					CurrentPoint: currentPoint,
					SqrtPrice96:  sqrtPrice96,
				}
				retState := swapmath.Y2XRange(st, nextPoint, sqrtRate96, new(uint256.Int).Set(amountNoFee))
				crossedPoints++

				finished = retState.Finished
				var feeAmount *uint256.Int
				if retState.CostY.Cmp(amountNoFee) >= 0 {
					feeAmount = new(uint256.Int).Sub(amount, retState.CostY)
				} else {
					// retState.costY <= uint128.max
					feeAmount = new(uint256.Int).Mul(retState.CostY, uint256.NewInt(uint64(fee)))
					feeAmount.Div(feeAmount, uint256.NewInt(uint64(1e6-fee)))
					mod := new(uint256.Int).Mod(new(uint256.Int).Mul(retState.CostY, uint256.NewInt(uint64(fee))),
						uint256.NewInt(uint64(1e6-fee)))
					if mod.Sign() > 0 {
						feeAmount.Add(feeAmount, uint256.NewInt(1))
					}
				}

				amountX.Add(amountX, retState.AcquireX)
				amountY.Add(amountY, new(uint256.Int).Add(retState.CostY, feeAmount))
				amount.Sub(amount, new(uint256.Int).Add(retState.CostY, feeAmount))

				currentPoint = retState.FinalPt
				sqrtPrice96 = retState.SqrtFinalPrice96
				liquidityX = retState.LiquidityX
			} else {
				finished = true
			}

			if currentPoint == nextPoint {
				if orderData.IsLiquidity(nextPoint) {
					delta := orderData.UnsafeGetDeltaLiquidity()
					liquidity.Add(liquidity, delta)
				}
				liquidityX = liquidity
			}
		}
	}

	swapResult := SwapResult{
		CurrentPoint:  currentPoint,
		Liquidity:     liquidity,
		LiquidityX:    liquidityX,
		AmountX:       amountX,
		AmountY:       amountY,
		CrossedPoints: crossedPoints,
	}
	return swapResult, nil
}

func SwapY2XDesireX(desireX *uint256.Int, highPt int, pool PoolInfoU256) (SwapResult, error) {
	if desireX.Sign() <= 0 {
		return SwapResult{}, errors.New("XP")
	}

	highPt = min(highPt, pool.RightMostPt)

	amountX := uint256.NewInt(0)
	amountY := uint256.NewInt(0)

	sqrtPrice96, _ := calc.GetSqrtPrice(pool.CurrentPoint)

	liquidityX := new(uint256.Int).Set(pool.LiquidityX)
	liquidity := new(uint256.Int).Set(pool.Liquidity)

	finished := false
	sqrtRate96, _ := calc.GetSqrtPrice(1)
	pointDelta := pool.PointDelta
	currentPoint := pool.CurrentPoint
	fee := pool.Fee
	var crossedPoints int64

	orderData := InitY2X(
		pool.Liquidities,
		pool.LimitOrders,
		pool.CurrentPoint,
	)

	for currentPoint < highPt && !finished {
		if orderData.IsLimitOrder(currentPoint) {
			// clear limit order first
			currX := orderData.UnsafeGetLimitSellingX()
			costY, acquireX := swapmathdesire.Y2XAtPrice(desireX, sqrtPrice96, currX)
			if acquireX.Cmp(desireX) >= 0 {
				finished = true
			}
			feeAmount := calc.MulDivCeil(costY, uint256.NewInt(uint64(fee)), uint256.NewInt(uint64(1e6-fee)))
			if desireX.Cmp(acquireX) <= 0 {
				desireX = uint256.NewInt(0)
			} else {
				desireX.Sub(desireX, acquireX)
			}
			amountY.Add(amountY, new(uint256.Int).Add(costY, feeAmount))
			amountX.Add(amountX, acquireX)
			orderData.ConsumeLimitOrder(true)
		}

		if finished {
			break
		}

		nextPoint := min(orderData.MoveY2X(currentPoint, pointDelta), highPt)

		// in [st.currentPoint, nextPoint)
		if liquidity.Sign() == 0 {
			// no liquidity in the range [st.currentPoint, nextPoint)
			currentPoint = nextPoint
			sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
			if orderData.IsLiquidity(currentPoint) {
				delta := orderData.UnsafeGetDeltaLiquidity()
				liquidity.Add(liquidity, delta)
				liquidityX = liquidity
			}
		} else {
			// desireX > 0
			if desireX.Sign() > 0 {
				st := utils.State{
					LiquidityX:   new(uint256.Int).Set(liquidityX),
					Liquidity:    new(uint256.Int).Set(liquidity),
					CurrentPoint: currentPoint,
					SqrtPrice96:  sqrtPrice96,
				}
				retState := swapmathdesire.Y2XRange(st, nextPoint, sqrtRate96, new(uint256.Int).Set(desireX))
				crossedPoints++

				finished = retState.Finished
				feeAmount := calc.MulDivCeil(retState.CostY, uint256.NewInt(uint64(fee)),
					uint256.NewInt(uint64(1e6-fee)))

				amountX.Add(amountX, retState.AcquireX)
				amountY.Add(amountY, new(uint256.Int).Add(retState.CostY, feeAmount))
				desireX.Sub(desireX, calc.Min(desireX, retState.AcquireX))

				currentPoint = retState.FinalPt
				sqrtPrice96 = retState.SqrtFinalPrice96
				liquidityX = retState.LiquidityX
			} else {
				finished = true
			}

			if currentPoint == nextPoint {
				if orderData.IsLiquidity(nextPoint) {
					delta := orderData.UnsafeGetDeltaLiquidity()
					liquidity.Add(liquidity, delta)
				}
				liquidityX = liquidity
			}
		}
	}

	swapResult := SwapResult{
		CurrentPoint:  currentPoint,
		Liquidity:     liquidity,
		LiquidityX:    liquidityX,
		AmountX:       amountX,
		AmountY:       amountY,
		CrossedPoints: crossedPoints,
	}

	return swapResult, nil
}
