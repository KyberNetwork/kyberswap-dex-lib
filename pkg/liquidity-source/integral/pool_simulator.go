package integral

import (
	"math/big"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/logger"
)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var pair IntegralPair
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
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		IntegralPair: IntegralPair{
			SwapFee:      pair.SwapFee,
			X_Decimals:   pair.X_Decimals,
			Y_Decimals:   pair.Y_Decimals,
			AveragePrice: pair.AveragePrice,
			SpotPrice:    pair.SpotPrice,
		},
		gas: defaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokens := p.GetTokens()
	if len(tokens) < 2 {
		return nil, ErrTokenNotFound
	}

	tokenIn := param.TokenAmountIn.Token
	tokenOut := param.TokenOut
	amountIn := ToUint256(param.TokenAmountIn.Amount)

	reserve0 := ToUint256(p.GetReserves()[0])
	reserve1 := ToUint256(p.GetReserves()[1])

	_amountIn, amountOut, fee := p.swapExactIn(tokenIn, tokenOut, amountIn)

	var newReserve0, newReserve1 *uint256.Int

	switch tokenIn {
	case tokens[0]:
		newReserve0 = AddUint256(reserve0, _amountIn)
		newReserve1 = SubUint256(reserve1, amountOut)
	case tokens[1]:
		newReserve0 = SubUint256(reserve0, _amountIn)
		newReserve1 = AddUint256(reserve1, amountOut)
	default:
		return nil, ErrInvalidTokenIn
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: ToInt256(amountOut),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: ToInt256(fee),
		},
		Gas: p.gas.Swap,
		SwapInfo: SwapInfo{
			newReserve0: ToInt256(newReserve0),
			newReserve1: ToInt256(newReserve1),
		},
	}, nil
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Smardex %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}

	p.Info.Reserves = []*big.Int{si.newReserve0, si.newReserve1}
}

// https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapPair.sol#L184
// func (p *PoolSimulator) swap(amountIn, amount0Out, amount1Out *uint256.Int) (*pool.CalcAmountOutResult, error) {
// 	if !(amount0Out.Cmp(uZERO) > 0 && amount1Out.Cmp(uZERO) == 0) && !(amount1Out.Cmp(uZERO) > 0 && amount0Out.Cmp(uZERO) == 0) {
// 		return nil, ErrTP31
// 	}

// 	reserves := p.GetReserves()

// 	reserve0 := ToUint256(reserves[0])
// 	reserve1 := ToUint256(reserves[1])

// 	if amount0Out.Cmp(reserve0) >= 0 || amount1Out.Cmp(reserve1) >= 0 {
// 		return nil, ErrTP07
// 	}

// 	swapFee := p.IntegralPair.SwapFee

// 	var balance0, balance1 *uint256.Int
// 	var balance0After, balance1After *uint256.Int

// 	// trading token1 for token0
// 	if amount0Out.Cmp(uZERO) > 0 {
// 		balance0 = reserve0
// 		balance1 = AddUint256(reserve1, amountIn)

// 		if balance1.Cmp(reserve1) <= 0 {
// 			return nil, ErrTP08
// 		}

// 		fee1 := DivUint256(SubUint256(amountIn, swapFee), precison)
// 		balance1After = SubUint256(balance1, fee1)

// 		var err error
// 		balance0After, err = p.tradeY(balance1After, reserve0, reserve1)
// 		if err != nil {
// 			return nil, err
// 		}

// 		if balance0.Cmp(balance0After) < 0 {
// 			return nil, ErrTP2E
// 		}

// 		fee0 := SubUint256(balance0, balance0After)

// 		return &pool.CalcAmountOutResult{
// 			TokenAmountOut: &pool.TokenAmount{
// 				Token:  p.GetTokens()[0],
// 				Amount: ToInt256(amount0Out),
// 			},
// 			Fee: &pool.TokenAmount{
// 				Token:  p.GetTokens()[0],
// 				Amount: ToInt256(fee0),
// 			},
// 			Gas: p.gas.Swap,
// 			SwapInfo: SwapInfo{
// 				newReserve0: ToInt256(balance0After),
// 				newReserve1: ToInt256(balance1After),
// 			},
// 		}, nil
// 	}

// 	// trading token0 for token1
// 	balance0 = AddUint256(reserve0, amountIn)
// 	balance1 = reserve1

// 	if balance0.Cmp(reserve0) <= 0 {
// 		return nil, ErrTP08
// 	}

// 	fee0 := DivUint256(MulUint256(amountIn, swapFee), precison)
// 	balance0After = SubUint256(balance0, fee0)

// 	var err error
// 	balance1After, err = p.tradeX(balance0After, reserve0, reserve1)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if balance1.Cmp(balance1After) < 0 {
// 		return nil, ErrTP2E
// 	}

// 	fee1 := SubUint256(balance1, balance1After)

// 	return &pool.CalcAmountOutResult{
// 		TokenAmountOut: &pool.TokenAmount{
// 			Token:  p.GetTokens()[1],
// 			Amount: ToInt256(amount1Out),
// 		},
// 		Fee: &pool.TokenAmount{
// 			Token:  p.GetTokens()[1],
// 			Amount: ToInt256(fee1),
// 		},
// 		Gas: p.gas.Swap,
// 		SwapInfo: SwapInfo{
// 			newReserve0: ToInt256(balance0After),
// 			newReserve1: ToInt256(balance1After),
// 		},
// 	}, nil
// }

// // https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapPair.sol#L266
// func (p *PoolSimulator) getSwapAmount0Out(amount1In *uint256.Int) (*uint256.Int, error) {
// 	reserves := p.GetReserves()

// 	reserve0 := ToUint256(reserves[0])
// 	reserve1 := ToUint256(reserves[1])

// 	swapFee := p.IntegralPair.SwapFee

// 	fee := DivUint256(MulUint256(amount1In, swapFee), precison)

// 	balance0After, err := p.tradeY(
// 		SubUint256(AddUint256(reserve1, amount1In), fee),
// 		reserve0,
// 		reserve1,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return SubUint256(reserve0, balance0After), nil
// }

// // https://github.com/IntegralHQ/Integral-SIZE-Smart-Contracts/blob/main/contracts/TwapPair.sol#L281
// func (p *PoolSimulator) getSwapAmount1Out(amount0In *uint256.Int) (*uint256.Int, error) {
// 	reserves := p.GetReserves()

// 	reserve0 := ToUint256(reserves[0])
// 	reserve1 := ToUint256(reserves[1])

// 	swapFee := p.IntegralPair.SwapFee

// 	fee := DivUint256(MulUint256(amount0In, swapFee), precison)

// 	balance1After, err := p.tradeX(
// 		SubUint256(AddUint256(reserve0, amount0In), fee),
// 		reserve0,
// 		reserve1,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return SubUint256(reserve1, balance1After), nil
// }

// func (p *PoolSimulator) getSwapAmount0In(amount1Out *uint256.Int) (*uint256.Int, error) {
// 	reserves := p.GetReserves()

// 	reserve0 := ToUint256(reserves[0])
// 	reserve1 := ToUint256(reserves[1])

// 	balance1After := SubUint256(reserve1, amount1Out)
// 	balance0After, err := p.tradeY(balance1After, reserve0, reserve1)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return CeilDivUint256(MulUint256(SubUint256(balance0After, reserve0), precison), SubUint256(precison, p.IntegralPair.SwapFee)), nil
// }

// func (p *PoolSimulator) getSwapAmount1In(amount0Out *uint256.Int) (*uint256.Int, error) {
// 	reserves := p.GetReserves()

// 	reserve0 := ToUint256(reserves[0])
// 	reserve1 := ToUint256(reserves[1])

// 	balance0After := SubUint256(reserve0, amount0Out)
// 	balance1After, err := p.tradeY(balance0After, reserve0, reserve1)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return CeilDivUint256(MulUint256(SubUint256(AddUint256(balance1After, uint256.NewInt(1)), reserve0), precison), SubUint256(precison, p.IntegralPair.SwapFee)), nil
// }

func (p *PoolSimulator) swapExactIn(tokenIn, _ string, amountIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int) {
	fee := DivUint256(MulUint256(amountIn, p.IntegralPair.SwapFee), precison)

	inverted := p.GetTokens()[0] == tokenIn

	amountOut := p.calculateAmountOut(inverted, amountIn)

	return amountIn, amountOut, fee
}

func (p *PoolSimulator) calculateAmountOut(inverted bool, amountIn *uint256.Int) *uint256.Int {
	decimalsConverter := getDecimalsConverter(p.IntegralPair.X_Decimals, p.IntegralPair.Y_Decimals, inverted)

	price := p.getPrice(inverted)

	return CeilDivUint256(MulUint256(amountIn, decimalsConverter), price)
}

func getDecimalsConverter(xDecimals, yDecimals uint64, inverted bool) (decimalsConverter *uint256.Int) {
	var exponent uint64
	if inverted {
		exponent = 18 + (yDecimals - xDecimals)
	} else {
		exponent = 18 + (xDecimals - yDecimals)
	}

	decimalsConverter = new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(exponent))

	return
}

func (p *PoolSimulator) getPrice(inverted bool) (price *uint256.Int) {
	spotPrice := p.IntegralPair.SpotPrice
	averagePrice := p.IntegralPair.AveragePrice

	if inverted {
		tenPower36 := new(uint256.Int).Exp(uint256.NewInt(10), uint256.NewInt(36))
		if spotPrice.Gt(averagePrice) {
			price = DivUint256(tenPower36, spotPrice)
		} else {
			price = DivUint256(tenPower36, averagePrice)
		}
	} else {
		if spotPrice.Lt(averagePrice) {
			price = spotPrice
		} else {
			price = averagePrice
		}
	}
	return
}
