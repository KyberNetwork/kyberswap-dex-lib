package limitorder

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const basisPoint = 10000

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	return p.calcAmountIn(param.TokenAmountOut, param.TokenIn, param.Limit)
}

func (p *PoolSimulator) calcAmountIn(
	tokenAmountOut pool.TokenAmount,
	tokenIn string,
	limit pool.SwapLimit,
) (*pool.CalcAmountInResult, error) {
	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow || tokenAmountOut.Amount.Sign() <= 0 {
		return nil, ErrCannotFulfillAmountOut
	}
	swapSide := p.getSwapSide(tokenIn, tokenAmountOut.Token)
	amountIn, err := p.calcMinAmountIn(swapSide, amountOut, limit)
	if err != nil {
		return nil, err
	}

	// Run the exact-in path so SwapInfo, Fee and fallback orders match CalcAmountOut(amountIn).
	res, err := p.calcAmountOut(pool.TokenAmount{Token: tokenIn, Amount: amountIn}, tokenAmountOut.Token, limit)
	if err != nil {
		return nil, err
	} else if res.TokenAmountOut.Amount.Cmp(tokenAmountOut.Amount) < 0 {
		return nil, ErrCannotFulfillAmountOut
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		Fee:      res.Fee,
		Gas:      res.Gas,
		SwapInfo: res.SwapInfo,
	}, nil
}

// calcMinAmountIn inverts calcAmountOutWithSwapInfo: it returns the minimum amountIn that yields at least amountOut.
// Exact-in fully fills an order only when takingAmountAfterFee(remaining amountIn) > remainingTakingAmount,
// then partially fills a later order. So amountIn at a fully filled order i is
// max(fullCost_i + amountIn_{i+1}, minFullFillAmountIn_i).
func (p *PoolSimulator) calcMinAmountIn(swapSide SwapSide, amountOut *uint256.Int,
	limit pool.SwapLimit) (*big.Int, error) {
	orderIDs := p.getOrderIDsBySwapSide(swapSide)
	if limit != nil {
		orderIDs = p.filterOrdersByAllowedSenders(orderIDs, limit.GetAllowedSenders())
	}

	type fullFill struct {
		cost, minAmountIn uint256.Int
	}
	var fullFills []fullFill
	filledMakingAmountByMaker := make(map[string]*big.Int, len(p.allMakersBalanceAllowance))

	var need, making, taking, remMaking, remTaking, filledMaking, filledTaking, makerFeeDenom, takerFeeNum,
		amountIn uint256.Int
	need.Set(amountOut)
	found := false
	for _, orderID := range orderIDs {
		order, ok := p.ordersMapping[orderID]
		if !ok {
			return nil, fmt.Errorf("order %d is not existed in pool", orderID)
		}

		remMakingBig, remTakingBig := order.RemainingAmount(limit, filledMakingAmountByMaker)
		if remMakingBig.Sign() <= 0 || remTakingBig.Sign() <= 0 {
			continue
		}
		remMaking.SetFromBig(remMakingBig)
		remTaking.SetFromBig(remTakingBig)
		making.SetFromBig(order.MakingAmount)
		taking.SetFromBig(order.TakingAmount)

		makerFeePct, takerFeePct := uint64(order.MakerTokenFeePercent), uint64(0)
		if order.IsTakerAssetFee {
			makerFeePct, takerFeePct = 0, makerFeePct
		}
		makerFeeDenom.SetUint64(basisPoint - makerFeePct)
		takerFeeNum.SetUint64(basisPoint + takerFeePct)

		// min filledMaking with filledMaking - ceil(filledMaking*makerFee/BPS) >= need; exact-in skips 0-making fills
		big256.MulDivUp(&filledMaking, &need, big256.UBasisPoint, &makerFeeDenom)
		if filledMaking.IsZero() {
			filledMaking.SetOne()
		}
		// min filledTaking with floor(filledTaking*making/taking) >= filledMaking
		big256.MulDivUp(&filledTaking, &filledMaking, &taking, &making)
		if !filledTaking.Gt(&remTaking) {
			// min amountIn with floor(amountIn*BPS/(BPS+takerFee)) >= filledTaking
			big256.MulDivUp(&amountIn, &filledTaking, &takerFeeNum, big256.UBasisPoint)
			found = true
			break
		}

		// fully fill this order: cost = remTaking + ceil(remTaking*takerFee/BPS)
		var f fullFill
		f.cost.SetUint64(takerFeePct)
		big256.MulDivUp(&f.cost, &remTaking, &f.cost, big256.UBasisPoint)
		f.cost.Add(&f.cost, &remTaking)
		// min amountIn with floor(amountIn*BPS/(BPS+takerFee)) > remTaking
		f.minAmountIn.AddUint64(&remTaking, 1)
		big256.MulDivUp(&f.minAmountIn, &f.minAmountIn, &takerFeeNum, big256.UBasisPoint)
		fullFills = append(fullFills, f)

		// out = remMaking - ceil(remMaking*makerFee/BPS)
		filledMaking.SetUint64(makerFeePct)
		big256.MulDivUp(&filledMaking, &remMaking, &filledMaking, big256.UBasisPoint)
		filledMaking.Sub(&remMaking, &filledMaking)
		if need.Gt(&filledMaking) {
			need.Sub(&need, &filledMaking)
		} else {
			need.Clear()
		}
		addFilledMakingAmount(filledMakingAmountByMaker, order.Maker, remMakingBig)
	}
	if !found {
		return nil, ErrCannotFulfillAmountOut
	}

	for i := len(fullFills) - 1; i >= 0; i-- {
		if amountIn.Add(&amountIn, &fullFills[i].cost); amountIn.Lt(&fullFills[i].minAmountIn) {
			amountIn.Set(&fullFills[i].minAmountIn)
		}
	}
	return amountIn.ToBig(), nil
}
