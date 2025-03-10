package limitorder

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	constant "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	return p.calcAmountIn(param.TokenAmountOut, param.TokenIn, param.Limit)
}

func (p *PoolSimulator) calcAmountIn(
	tokenAmountOut pool.TokenAmount,
	tokenIn string,
	limit pool.SwapLimit,
) (*pool.CalcAmountInResult, error) {
	swapSide := p.getSwapSide(tokenIn, tokenAmountOut.Token)
	amountIn, swapInfo, feeAmount, err := p.calcAmountInWithSwapInfo(swapSide, tokenAmountOut, limit)
	if err != nil {
		return nil, err
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: feeAmount,
		},
		Gas:      p.estimateGas(len(swapInfo.FilledOrders)),
		SwapInfo: swapInfo,
	}, nil
}

func (p *PoolSimulator) calcAmountInWithSwapInfo(swapSide SwapSide, tokenAmountOut pool.TokenAmount, limit pool.SwapLimit) (*big.Int, SwapInfo, *big.Int, error) {
	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if limit != nil {
		orderIDs = p.filterOrdersByAllowedSenders(orderIDs, limit.GetAllowedSenders())
	}
	if len(orderIDs) == 0 {
		return big.NewInt(0), SwapInfo{}, nil, nil
	}

	totalAmountInWei := big.NewInt(0)
	totalAmountOut := new(big.Int).Set(tokenAmountOut.Amount)

	swapInfo := SwapInfo{
		FilledOrders: make([]*FilledOrderInfo, 0, len(orderIDs)),
		SwapSide:     swapSide,
	}
	totalFilledTakingAmountWei := big.NewInt(0)
	isFulfillAmountOut := false
	totalFeeAmountWei := big.NewInt(0)

	// we need to update maker's remaining balance in 2 places:
	// - in UpdateBalance: mainly to deal with case where maker has orders with same makerAsset but different takerAsset
	// - when simulating filling each order here: we cannot do the same as in kyber-pmm (simulating first then check inventory limit at the end)
	//				because in LO we have multiple makers, and also because we still need to allow orders that have part of the balance available
	//		the problem is that in this func we cannot update the limit,
	//		so we'll use this map to track filled amount for each maker, then subtract from the original balance, to have the remaining balance available
	filledMakingAmountByMaker := make(map[string]*big.Int, len(p.allMakersBalanceAllowance))

	totalMakingAmountWei := new(big.Int)
	for i, orderID := range orderIDs {
		order, ok := p.ordersMapping[orderID]
		if !ok {
			return nil, SwapInfo{}, nil, fmt.Errorf("order %d is not existed in pool", orderID)
		}

		// Get remaining making amount, taking amount
		remainingMakingAmountWei, remainingTakingAmountWei := order.RemainingAmount(limit, filledMakingAmountByMaker)
		if remainingMakingAmountWei.Sign() <= 0 || remainingTakingAmountWei.Sign() <= 0 {
			continue
		}

		totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)

		totalAmountOutBeforeFee, _ := p.calcMakerAssetAmountBeforeFee(order, totalAmountOut)

		if remainingMakingAmountWei.Cmp(totalAmountOutBeforeFee) >= 0 {
			filledMakingAmountWei := totalAmountOutBeforeFee
			filledTakingAmountWei := divCeil(
				new(big.Int).Mul(totalAmountOutBeforeFee, order.TakingAmount),
				order.MakingAmount,
			) // filledTakingAmountWei =  ceil(takingAmount * totalAmountOutBeforeFee / makingAmount)

			// order too small
			if filledTakingAmountWei.Sign() == 0 {
				continue
			}

			actualAmountIn, feeAmountWeiByOrder := p.calcTakerAssetFeeAmountExactOut(order, filledTakingAmountWei)
			totalFeeAmountWei.Add(totalFeeAmountWei, feeAmountWeiByOrder)
			totalAmountInWei.Add(totalAmountInWei, actualAmountIn)
			filledOrderInfo := newFilledOrderInfo(order, filledTakingAmountWei.String(), filledMakingAmountWei.String(), feeAmountWeiByOrder.String())
			swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			isFulfillAmountOut = true
			addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, filledMakingAmountWei)
			totalFilledTakingAmountWei.Add(totalFilledTakingAmountWei, filledTakingAmountWei)

			// threshold = totalAmountOutBeforeFee * FallbackPercentageOfTotalMakingAmount
			threshold := new(big.Float).SetInt(totalAmountOutBeforeFee)
			threshold.Mul(threshold, FallbackPercentageOfTotalMakingAmount)

			for j := i + 1; j < len(orderIDs); j++ {
				if new(big.Float).SetInt(totalMakingAmountWei).Cmp(threshold) >= 0 {
					break
				}
				order, ok := p.ordersMapping[orderIDs[j]]
				if !ok {
					continue
				}

				remainingMakingAmountWei, remainingTakingAmountWei := order.RemainingAmount(limit, filledMakingAmountByMaker)
				if remainingMakingAmountWei.Sign() <= 0 || remainingTakingAmountWei.Sign() <= 0 {
					continue
				}

				totalMakingAmountWei = new(big.Int).Add(totalMakingAmountWei, remainingMakingAmountWei)
				filledOrderInfo := newFallbackOrderInfo(order)
				swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
			}
			break
		}
		totalAmountOut.Sub(totalAmountOut, remainingMakingAmountWei)
		_, takerAssetFee := p.calcTakerAssetFeeAmountExactOut(order, remainingTakingAmountWei)
		actualAmountIn := new(big.Int).Add(remainingTakingAmountWei, takerAssetFee)
		totalAmountInWei.Add(totalAmountInWei, actualAmountIn)
		totalFeeAmountWei.Add(totalFeeAmountWei, takerAssetFee)
		filledOrderInfo := newFilledOrderInfo(order, remainingTakingAmountWei.String(), remainingMakingAmountWei.String(), takerAssetFee.String())
		swapInfo.FilledOrders = append(swapInfo.FilledOrders, filledOrderInfo)
		addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, remainingMakingAmountWei)
		totalFilledTakingAmountWei.Add(totalFilledTakingAmountWei, remainingTakingAmountWei)
	}
	if !isFulfillAmountOut {
		return nil, SwapInfo{}, nil, ErrCannotFulfillAmountOut
	}
	swapInfo.AmountIn = totalFilledTakingAmountWei.String()
	return totalAmountInWei, swapInfo, totalFeeAmountWei, nil
}

// calcMakerAssetAmountBeforeFee calculates the maker asset amount before fee.
// input is the received amount after fee.
func (p *PoolSimulator) calcMakerAssetAmountBeforeFee(order *order, makingAmount *big.Int) (makingAmountBeforeFee *big.Int, fee *big.Int) {
	if order.IsTakerAssetFee {
		return new(big.Int).Set(makingAmount), big.NewInt(0)
	}

	feePct := order.MakerTokenFeePercent
	if feePct == 0 {
		return new(big.Int).Set(makingAmount), big.NewInt(0)
	}

	// makingAmountBeforeFee = makingAmount * BasisPoint / (BasisPoint - feePct)
	makingAmountBeforeFee = divCeil(
		new(big.Int).Mul(makingAmount, constant.BasisPoint),
		new(big.Int).Sub(constant.BasisPoint, big.NewInt(int64(feePct))),
	)

	// fee = makingAmount - makingAmountBeforeFee
	fee = new(big.Int).Sub(makingAmount, makingAmountBeforeFee)

	return makingAmountBeforeFee, fee
}

func divCeil(a, b *big.Int) *big.Int {
	// (a + b - 1) / b
	a = new(big.Int).Add(a, b)
	a.Sub(a, big.NewInt(1))
	return a.Div(a, b)
}
