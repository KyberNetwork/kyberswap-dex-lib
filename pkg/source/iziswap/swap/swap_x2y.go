package swap

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/calc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/swapmath"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/swapmathdesire"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/library/utils"
)

func SwapX2Y(amount *uint256.Int, lowPt int, pool PoolInfoU256) (SwapResult, error) {
	if amount.Sign() <= 0 {
		return SwapResult{}, errors.New("AP")
	}

	lowPt = max(lowPt, pool.LeftMostPt)
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

	orderData := InitX2Y(
		pool.Liquidities,
		pool.LimitOrders,
		pool.CurrentPoint,
	)

	for lowPt <= currentPoint && !finished {
		if orderData.IsLimitOrder(currentPoint) {
			// amount <= uint128.max
			amountNoFee := new(uint256.Int).Mul(amount, uint256.NewInt(uint64(1e6-fee)))
			amountNoFee.Div(amountNoFee, uint256.NewInt(1e6))
			if amountNoFee.Sign() > 0 {

				currY := orderData.UnsafeGetLimitSellingY()
				costX, acquireY := swapmath.X2YAtPrice(amountNoFee, sqrtPrice96, currY)

				if acquireY.Cmp(currY) < 0 || costX.Cmp(amountNoFee) >= 0 {
					finished = true
				}

				feeAmount := new(uint256.Int)
				if costX.Cmp(amountNoFee) >= 0 {
					feeAmount.Sub(amount, costX)
				} else {
					// costX <= amountX <= uint128.max
					feeAmount.Mul(costX, uint256.NewInt(uint64(fee)))
					feeAmount.Div(feeAmount, uint256.NewInt(uint64(1e6-fee)))
					mod := new(uint256.Int).Mul(costX, uint256.NewInt(uint64(fee)))
					mod.Mod(mod, uint256.NewInt(uint64(1e6-fee)))
					if mod.Sign() > 0 {
						feeAmount.Add(feeAmount, uint256.NewInt(1))
					}
				}

				amount.Sub(amount, costX)
				amount.Sub(amount, feeAmount)
				amountX.Add(amountX, costX)
				amountX.Add(amountX, feeAmount)
				amountY.Add(amountY, acquireY)

				orderData.ConsumeLimitOrder(false)
			} else {
				finished = true
			}
		}

		if finished {
			break
		}

		searchStart := currentPoint - 1

		// step2: clear the liquidity if the currentPoint is an endpoint
		if orderData.IsLiquidity(currentPoint) {
			amountNoFee := new(uint256.Int).Mul(amount, uint256.NewInt(uint64(1e6-fee)))
			amountNoFee.Div(amountNoFee, uint256.NewInt(uint64(1e6)))
			if amountNoFee.Sign() > 0 {
				if liquidity.Sign() > 0 {
					st := utils.State{
						LiquidityX:   new(uint256.Int).Set(liquidityX),
						Liquidity:    new(uint256.Int).Set(liquidity),
						CurrentPoint: currentPoint,
						SqrtPrice96:  sqrtPrice96,
					}
					retState := swapmath.X2YRange(st, currentPoint, sqrtRate96, new(uint256.Int).Set(amountNoFee))
					crossedPoints++
					finished = retState.Finished

					feeAmount := new(uint256.Int)
					if retState.CostX.Cmp(amountNoFee) >= 0 {
						feeAmount.Sub(amount, retState.CostX)
					} else {
						feeAmount.Mul(retState.CostX, uint256.NewInt(uint64(fee)))
						feeAmount.Div(feeAmount, uint256.NewInt(uint64(1e6-fee)))
						mod := new(uint256.Int).Mul(retState.CostX, uint256.NewInt(uint64(fee)))
						mod.Mod(mod, uint256.NewInt(uint64(1e6-fee)))
						if mod.Sign() > 0 {
							feeAmount.Add(feeAmount, uint256.NewInt(1))
						}
					}

					amountX.Add(amountX, retState.CostX)
					amountX.Add(amountX, feeAmount)
					amountY.Add(amountY, retState.AcquireY)
					amount.Sub(amount, retState.CostX)
					amount.Sub(amount, feeAmount)
					currentPoint = retState.FinalPt
					sqrtPrice96 = retState.SqrtFinalPrice96
					liquidityX = retState.LiquidityX
				}
				if !finished {
					delta := orderData.UnsafeGetDeltaLiquidity()
					liquidity.Sub(liquidity, delta)
					currentPoint -= 1
					sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
					liquidityX.SetUint64(0)
				}
			} else {
				finished = true
			}
		}

		if finished || currentPoint < lowPt {
			break
		}

		nextPt := max(orderData.MoveX2Y(searchStart, pointDelta), lowPt)

		if liquidity.Sign() == 0 {
			// no liquidity in the range [nextPt, st.currentPoint]
			currentPoint = nextPt
			sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
		} else {
			amountNoFee := new(uint256.Int).Mul(amount, uint256.NewInt(uint64(1e6-fee)))
			amountNoFee.Div(amountNoFee, uint256.NewInt(uint64(1e6)))
			if amountNoFee.Sign() > 0 {
				st := utils.State{
					LiquidityX:   new(uint256.Int).Set(liquidityX),
					Liquidity:    new(uint256.Int).Set(liquidity),
					CurrentPoint: currentPoint,
					SqrtPrice96:  sqrtPrice96,
				}
				retState := swapmath.X2YRange(st, nextPt, sqrtRate96, new(uint256.Int).Set(amountNoFee))
				crossedPoints++
				finished = retState.Finished
				feeAmount := new(uint256.Int)
				if retState.CostX.Cmp(amountNoFee) >= 0 {
					feeAmount.Sub(amount, retState.CostX)
				} else {
					feeAmount.Mul(retState.CostX, uint256.NewInt(uint64(fee)))
					feeAmount.Div(feeAmount, uint256.NewInt(uint64(1e6-fee)))
					mod := new(uint256.Int).Mul(retState.CostX, uint256.NewInt(uint64(fee)))
					mod.Mod(mod, uint256.NewInt(uint64(1e6-fee)))
					if mod.Sign() > 0 {
						feeAmount.Add(feeAmount, uint256.NewInt(1))
					}
				}
				amountY.Add(amountY, retState.AcquireY)
				amountX.Add(amountX, retState.CostX)
				amountX.Add(amountX, feeAmount)
				amount.Sub(amount, retState.CostX)
				amount.Sub(amount, feeAmount)

				currentPoint = retState.FinalPt
				sqrtPrice96 = retState.SqrtFinalPrice96
				liquidityX = retState.LiquidityX
			} else {
				finished = true
			}
		}

		if currentPoint <= lowPt {
			break
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

func SwapX2YDesireY(desireY *uint256.Int, lowPt int, pool PoolInfoU256) (SwapResult, error) {
	if desireY.Sign() <= 0 {
		return SwapResult{}, errors.New("AP")
	}

	lowPt = max(lowPt, pool.LeftMostPt)
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

	orderData := InitX2Y(
		pool.Liquidities,
		pool.LimitOrders,
		pool.CurrentPoint,
	)

	for lowPt <= currentPoint && !finished {
		if orderData.IsLimitOrder(currentPoint) {
			currY := orderData.UnsafeGetLimitSellingY()
			costX, acquireY := swapmathdesire.X2YAtPrice(desireY, sqrtPrice96, currY)

			if acquireY.Cmp(desireY) >= 0 {
				finished = true
			}

			feeAmount := calc.MulDivCeil(costX, uint256.NewInt(uint64(fee)), uint256.NewInt(1e6))

			if desireY.Cmp(acquireY) <= 0 {
				desireY = zeroBI
			} else {
				desireY.Sub(desireY, acquireY)
			}

			amountX.Add(amountX, costX)
			amountX.Add(amountX, feeAmount)
			amountY.Add(amountY, acquireY)

			orderData.ConsumeLimitOrder(false)
		}

		if finished {
			break
		}

		searchStart := currentPoint - 1

		// second, clear the liquid if the currentPoint is an endpoint
		if orderData.IsLiquidity(currentPoint) {
			if liquidity.Sign() > 0 {
				st := utils.State{
					LiquidityX:   new(uint256.Int).Set(liquidityX),
					Liquidity:    new(uint256.Int).Set(liquidity),
					CurrentPoint: currentPoint,
					SqrtPrice96:  sqrtPrice96,
				}
				retState := swapmathdesire.X2YRange(st, currentPoint, sqrtRate96, new(uint256.Int).Set(desireY))
				crossedPoints++
				finished = retState.Finished

				feeAmount := calc.MulDivCeil(retState.CostX, uint256.NewInt(uint64(fee)),
					uint256.NewInt(uint64(1e6-fee)))

				amountX.Add(amountX, retState.CostX)
				amountX.Add(amountX, feeAmount)
				amountY.Add(amountY, retState.AcquireY)
				desireY.Sub(desireY, calc.Min(desireY, retState.AcquireY))
				currentPoint = retState.FinalPt
				sqrtPrice96 = retState.SqrtFinalPrice96
				liquidityX = retState.LiquidityX
			}

			if !finished {
				delta := orderData.UnsafeGetDeltaLiquidity()
				liquidity.Sub(liquidity, delta)
				currentPoint -= 1
				sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
				liquidityX.SetUint64(0)
			}
		}

		if finished || currentPoint < lowPt {
			break
		}

		nextPt := max(orderData.MoveX2Y(searchStart, pointDelta), lowPt)

		// in [nextPt, st.currentPoint)
		if liquidity.Sign() == 0 {
			// no liquidity in the range [nextPt, st.currentPoint]
			currentPoint = nextPt
			sqrtPrice96, _ = calc.GetSqrtPrice(currentPoint)
		} else {
			st := utils.State{
				LiquidityX:   new(uint256.Int).Set(liquidityX),
				Liquidity:    new(uint256.Int).Set(liquidity),
				CurrentPoint: currentPoint,
				SqrtPrice96:  sqrtPrice96,
			}
			retState := swapmathdesire.X2YRange(st, nextPt, sqrtRate96, new(uint256.Int).Set(desireY))
			crossedPoints++
			finished = retState.Finished
			feeAmount := calc.MulDivCeil(retState.CostX, uint256.NewInt(uint64(fee)), uint256.NewInt(uint64(1e6-fee)))
			amountY.Add(amountY, retState.AcquireY)
			amountX.Add(amountX, retState.CostX)
			amountX.Add(amountX, feeAmount)
			desireY.Sub(desireY, calc.Min(desireY, retState.AcquireY))

			currentPoint = retState.FinalPt
			sqrtPrice96 = retState.SqrtFinalPrice96
			liquidityX = retState.LiquidityX
		}

		if currentPoint <= lowPt {
			break
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
