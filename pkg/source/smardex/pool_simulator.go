package smardex

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var pair SmardexPair
	if err := json.Unmarshal([]byte(entityPool.Extra), &pair); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	// totalSupply, _ := new(big.Int).SetString(entityPool.TotalSupply, 10)
	mapTokenAddressToIndex := make(map[string]int)
	for i := 0; i < numTokens; i++ {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		mapTokenAddressToIndex[entityPool.Tokens[i].Address] = i
	}

	var chainID uint
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &chainID); err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: poolpkg.Pool{
			Info: poolpkg.PoolInfo{
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				// SwapFee:    swapFee,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
				Checked:  false,
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

func (p *PoolSimulator) CalcAmountOut(tokenAmountIn poolpkg.TokenAmount, tokenOut string) (*poolpkg.CalcAmountOutResult, error) {
	if tokenAmountIn.Token == tokenOut {
		return nil, ErrSameAddress
	}

	if tokenAmountIn.Amount.Cmp(integer.Zero()) <= 0 {
		return nil, ErrZeroAmount
	}

	var zeroForOne bool
	if tokenAmountIn.Token == p.GetTokens()[1] {
		zeroForOne = true
	}

	var (
		amountCalculated  *big.Int = integer.Zero()
		fictiveReserveIn  *big.Int = p.FictiveReserve.FictiveReserve0
		fictiveReserveOut *big.Int = p.FictiveReserve.FictiveReserve1
		priceAverageIn    *big.Int = p.PriceAverage.PriceAverage0
		priceAverageOut   *big.Int = p.PriceAverage.PriceAverage1
		balanceIn         *big.Int = new(big.Int).Sub(p.GetReserves()[0], p.FeeToAmount.FeeToAmount0)
		balanceOut        *big.Int = new(big.Int).Sub(p.GetReserves()[1], p.FeeToAmount.FeeToAmount1)
	)
	if !zeroForOne {
		fictiveReserveIn = p.FictiveReserve.FictiveReserve1
		fictiveReserveOut = p.FictiveReserve.FictiveReserve0
		priceAverageIn = p.PriceAverage.PriceAverage1
		priceAverageOut = p.PriceAverage.PriceAverage0
		balanceIn = new(big.Int).Sub(p.GetReserves()[1], p.FeeToAmount.FeeToAmount1)
		balanceOut = new(big.Int).Sub(p.GetReserves()[0], p.FeeToAmount.FeeToAmount0)
	}

	var err error
	// compute new price average
	priceAverageIn, priceAverageOut, err = getUpdatedPriceAverage(
		fictiveReserveIn, fictiveReserveOut,
		p.PriceAverage.PriceAverageLastTimestamp,
		priceAverageIn,
		priceAverageOut,
		big.NewInt(time.Now().Unix()))
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
		})
	if err != nil {
		return nil, err
	}
	amountCalculated = result.amountOut

	amount0, amount1 := tokenAmountIn.Amount, new(big.Int).Neg(amountCalculated)
	feeToAmount0 := new(big.Int).Add(
		p.FeeToAmount.FeeToAmount0,
		new(big.Int).Div(new(big.Int).Mul(amount0, p.PairFee.FeesPool), FEES_BASE))
	feeToAmount1 := new(big.Int).Add(
		p.FeeToAmount.FeeToAmount1,
		new(big.Int).Div(new(big.Int).Mul(amount1, p.PairFee.FeesPool), FEES_BASE))
	if !zeroForOne {
		amount0, amount1 = new(big.Int).Neg(amountCalculated), tokenAmountIn.Amount
	}

	if zeroForOne {
		return &poolpkg.CalcAmountOutResult{
			TokenAmountOut: &poolpkg.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: new(big.Int).Neg(amount1),
			},
			Fee: &poolpkg.TokenAmount{
				Token:  p.GetTokens()[1],
				Amount: feeToAmount1,
			},
			Gas: p.gas.Swap,
		}, nil
	}

	return &poolpkg.CalcAmountOutResult{
		TokenAmountOut: &poolpkg.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: new(big.Int).Neg(amount0),
		},
		Fee: &poolpkg.TokenAmount{
			Token:  p.GetTokens()[0],
			Amount: feeToAmount0,
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			NewReserveIn:         result.newReserveIn,
			NewReserveOut:        result.newReserveOut,
			NewFictiveReserveIn:  result.newFictiveReserveIn,
			NewFictiveReserveOut: result.newFictiveReserveOut,
		},
	}, nil

}

// func (p *PoolSimulator) UpdateBalance(params poolpkg.UpdateBalanceParams) {
// 	zeroForOne := params.TokenAmountIn.Token < params.TokenAmountOut.Token
// 	token0, token1 := params.TokenAmountIn.Token, params.TokenAmountOut.Token
// 	if !zeroForOne {
// 		token0, token1 = params.TokenAmountOut.Token, params.TokenAmountIn.Token
// 	}
// 	pair := p.getPair[token0][token1]

// 	amount0, amount1 := params.TokenAmountIn.Amount, new(big.Int).Neg(params.TokenAmountOut.Amount)
// 	if !zeroForOne {
// 		amount0, amount1 = new(big.Int).Neg(amountCalculated), tokenAmountIn.Amount
// 	}
// 	if zeroForOne {
// 		pair.feeToAmount0 += ((uint256(amount0_) * _feesPool) / SmardexLibrary.FEES_BASE).toUint104();
// 	}
// }
