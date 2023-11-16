package smardex

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var now = time.Now

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var pair SmardexPair
	if err := json.Unmarshal([]byte(entityPool.Extra), &pair); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		SmardexPair: SmardexPair{
			PairFee:        pair.PairFee,
			FictiveReserve: pair.FictiveReserve,
			PriceAverage:   pair.PriceAverage,
			FeeToAmount:    pair.FeeToAmount,
		},
		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param poolpkg.CalcAmountOutParams) (*poolpkg.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
	)
	if tokenAmountIn.Token == tokenOut {
		return nil, ErrSameAddress
	}

	if isZero(tokenAmountIn.Amount) {
		return nil, ErrZeroAmount
	}

	var zeroForOne bool
	if tokenAmountIn.Token == p.GetTokens()[0] {
		zeroForOne = true
	}

	var (
		fictiveReserveIn   *big.Int = p.FictiveReserve.FictiveReserve0
		fictiveReserveOut  *big.Int = p.FictiveReserve.FictiveReserve1
		priceAverageIn     *big.Int = p.PriceAverage.PriceAverage0
		priceAverageOut    *big.Int = p.PriceAverage.PriceAverage1
		balanceIn          *big.Int = p.GetReserves()[0]
		balanceOut         *big.Int = p.GetReserves()[1]
		userTradeTimestamp          = now().Unix()
	)
	if !zeroForOne {
		fictiveReserveIn = p.FictiveReserve.FictiveReserve1
		fictiveReserveOut = p.FictiveReserve.FictiveReserve0
		priceAverageIn = p.PriceAverage.PriceAverage1
		priceAverageOut = p.PriceAverage.PriceAverage0
		balanceIn = p.GetReserves()[1]
		balanceOut = p.GetReserves()[0]
	}

	var err error
	// compute new price average
	priceAverageIn, priceAverageOut, err = getUpdatedPriceAverage(
		fictiveReserveIn, fictiveReserveOut,
		p.PriceAverage.PriceAverageLastTimestamp,
		priceAverageIn,
		priceAverageOut,
		big.NewInt(userTradeTimestamp))
	if err != nil {
		return nil, err
	}
	result, err := getAmountOut(
		GetAmountParameters{
			amount:            tokenAmountIn.Amount,
			reserveIn:         balanceIn,
			reserveOut:        balanceOut,
			fictiveReserveIn:  fictiveReserveIn,
			fictiveReserveOut: fictiveReserveOut,
			priceAverageIn:    priceAverageIn,
			priceAverageOut:   priceAverageOut,
			feesLP:            p.PairFee.FeesLP,
			feesPool:          p.PairFee.FeesPool,
			feesBase:          p.PairFee.FeesBase,
		})
	if err != nil {
		return nil, err
	}

	amount0, amount1 := tokenAmountIn.Amount, result.amountOut
	feeToAmount0, feeToAmount1 := p.FeeToAmount.Fees0, p.FeeToAmount.Fees1
	newPriceAverageIn, newPriceAverageOut := priceAverageIn, priceAverageOut
	newFictiveReserveIn, newFictiveReserveOut := result.newFictiveReserveIn, result.newFictiveReserveOut
	if zeroForOne {
		feeToAmount0 = feeToAmount0.Add(
			feeToAmount0,
			new(big.Int).Div(new(big.Int).Mul(amount0, p.PairFee.FeesPool), p.PairFee.FeesBase))
	} else {
		amount0, amount1 = result.amountOut, tokenAmountIn.Amount
		feeToAmount1 = feeToAmount1.Add(
			feeToAmount1,
			new(big.Int).Div(new(big.Int).Mul(amount1, p.PairFee.FeesPool), p.PairFee.FeesBase))
		newPriceAverageIn, newPriceAverageOut = priceAverageOut, priceAverageIn
		newFictiveReserveIn, newFictiveReserveOut = result.newFictiveReserveOut, result.newFictiveReserveIn
	}

	if zeroForOne {
		return &poolpkg.CalcAmountOutResult{
			TokenAmountOut: &poolpkg.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: amount1,
			},
			Fee: &poolpkg.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: feeToAmount1,
			},
			Gas: p.gas.Swap,
			SwapInfo: SwapInfo{
				NewReserveIn:              new(big.Int).Sub(result.newReserveIn, feeToAmount0),
				NewReserveOut:             new(big.Int).Sub(result.newReserveOut, feeToAmount1),
				NewFictiveReserveIn:       newFictiveReserveIn,
				NewFictiveReserveOut:      newFictiveReserveOut,
				NewPriceAverageIn:         newPriceAverageIn,
				NewPriceAverageOut:        newPriceAverageOut,
				PriceAverageLastTimestamp: big.NewInt(userTradeTimestamp),
				FeeToAmount0:              feeToAmount0,
				FeeToAmount1:              feeToAmount1,
			},
		}, nil
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: amount0,
		},
		Fee: &poolpkg.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: feeToAmount0,
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			NewReserveIn:              new(big.Int).Sub(result.newReserveIn, feeToAmount0),
			NewReserveOut:             new(big.Int).Sub(result.newReserveOut, feeToAmount1),
			NewFictiveReserveIn:       newFictiveReserveIn,
			NewFictiveReserveOut:      newFictiveReserveOut,
			NewPriceAverageIn:         newPriceAverageIn,
			NewPriceAverageOut:        newPriceAverageOut,
			PriceAverageLastTimestamp: big.NewInt(userTradeTimestamp),
			FeeToAmount0:              feeToAmount0,
			FeeToAmount1:              feeToAmount1,
		},
	}, nil

}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Smardex %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}
	p.Info.Reserves = []*big.Int{si.NewReserveIn, si.NewReserveOut}
	p.FictiveReserve = FictiveReserve{
		si.NewFictiveReserveIn,
		si.NewFictiveReserveOut,
	}
	p.PriceAverage = PriceAverage{
		si.NewPriceAverageIn,
		si.NewPriceAverageOut,
		si.PriceAverageLastTimestamp,
	}
	p.FeeToAmount = FeeToAmount{
		si.FeeToAmount0,
		si.FeeToAmount1,
	}
}
