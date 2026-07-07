package smardex

import (
	"math/big"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var now = time.Now

type PoolSimulator struct {
	pool.Pool
	SmardexPair
	gas Gas
}

var _ = pool.RegisterFactory0(DexTypeSmardex, NewPoolSimulator)

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
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	var (
		tokenAmountIn = param.TokenAmountIn
		tokenOut      = param.TokenOut
		amountIn      = uint256.MustFromBig(tokenAmountIn.Amount)
	)
	if tokenAmountIn.Token == tokenOut {
		return nil, ErrSameAddress
	}

	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	var zeroForOne bool
	if tokenAmountIn.Token == p.GetTokens()[0] {
		zeroForOne = true
	}

	var (
		fictiveReserveIn   = p.FictiveReserve.FictiveReserve0
		fictiveReserveOut  = p.FictiveReserve.FictiveReserve1
		priceAverageIn     = p.PriceAverage.PriceAverage0
		priceAverageOut    = p.PriceAverage.PriceAverage1
		balanceIn          = p.GetReserves()[0]
		balanceOut         = p.GetReserves()[1]
		userTradeTimestamp = uint256.NewInt(uint64(now().Unix()))
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
	priceAverageIn, priceAverageOut, err = getUpdatedPriceAverage(fictiveReserveIn, fictiveReserveOut,
		p.PriceAverage.PriceAverageLastTimestamp, priceAverageIn, priceAverageOut, userTradeTimestamp)
	if err != nil {
		return nil, err
	}

	result, err := getAmountOut(
		GetAmountParameters{
			amount:            amountIn,
			reserveIn:         uint256.MustFromBig(balanceIn),
			reserveOut:        uint256.MustFromBig(balanceOut),
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

	amount0, amount1 := amountIn, result.amountOut
	feeToAmount0, feeToAmount1 := new(uint256.Int).Set(p.FeeToAmount.Fees0), new(uint256.Int).Set(p.FeeToAmount.Fees1)
	newPriceAverageIn, newPriceAverageOut := priceAverageIn, priceAverageOut
	newFictiveReserveIn, newFictiveReserveOut := result.newFictiveReserveIn, result.newFictiveReserveOut
	if zeroForOne {
		feeToAmount0 = feeToAmount0.Add(
			feeToAmount0,
			new(uint256.Int).Div(new(uint256.Int).Mul(amount0, p.PairFee.FeesPool), p.PairFee.FeesBase))
	} else {
		amount0, amount1 = result.amountOut, amountIn
		feeToAmount1 = feeToAmount1.Add(
			feeToAmount1,
			new(uint256.Int).Div(new(uint256.Int).Mul(amount1, p.PairFee.FeesPool), p.PairFee.FeesBase))
		newPriceAverageIn, newPriceAverageOut = priceAverageOut, priceAverageIn
		newFictiveReserveIn, newFictiveReserveOut = result.newFictiveReserveOut, result.newFictiveReserveIn
	}

	if zeroForOne {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: amount1.ToBig(),
			},
			Fee: &pool.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: feeToAmount1.ToBig(),
			},
			Gas: p.gas.Swap,
			SwapInfo: SwapInfo{
				newReserveIn:              new(uint256.Int).Sub(result.newReserveIn, feeToAmount0),
				newReserveOut:             new(uint256.Int).Sub(result.newReserveOut, feeToAmount1),
				newFictiveReserveIn:       newFictiveReserveIn,
				newFictiveReserveOut:      newFictiveReserveOut,
				newPriceAverageIn:         newPriceAverageIn,
				newPriceAverageOut:        newPriceAverageOut,
				priceAverageLastTimestamp: userTradeTimestamp,
				feeToAmount0:              feeToAmount0,
				feeToAmount1:              feeToAmount1,
			},
		}, nil
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: amount0.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: feeToAmount0.ToBig(),
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			newReserveIn:              new(uint256.Int).Sub(result.newReserveIn, feeToAmount0),
			newReserveOut:             new(uint256.Int).Sub(result.newReserveOut, feeToAmount1),
			newFictiveReserveIn:       newFictiveReserveIn,
			newFictiveReserveOut:      newFictiveReserveOut,
			newPriceAverageIn:         newPriceAverageIn,
			newPriceAverageOut:        newPriceAverageOut,
			priceAverageLastTimestamp: userTradeTimestamp,
			feeToAmount0:              feeToAmount0,
			feeToAmount1:              feeToAmount1,
		},
	}, nil

}

func (p *PoolSimulator) GetMetaInfo(_, _ string) any {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Smardex %v %v pool, wrong swapInfo type",
			p.Info.Address, p.Info.Exchange)
		return
	}
	p.Info.Reserves = []*big.Int{si.newReserveIn.ToBig(), si.newReserveOut.ToBig()}
	p.FictiveReserve = FictiveReserve{
		si.newFictiveReserveIn,
		si.newFictiveReserveOut,
	}
	p.PriceAverage = PriceAverage{
		si.newPriceAverageIn,
		si.newPriceAverageOut,
		si.priceAverageLastTimestamp,
	}
	p.FeeToAmount = FeeToAmount{
		si.feeToAmount0,
		si.feeToAmount1,
	}
}
