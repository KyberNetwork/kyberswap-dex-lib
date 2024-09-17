package integral

import (
	"fmt"
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
		return nil, fmt.Errorf("failed to unmarshal Extra: %v", err)
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i++ {
		tokenAddr := entityPool.Tokens[i].Address
		tokens[i] = tokenAddr
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
			RelayerAddress: pair.RelayerAddress,
			IsEnabled:      pair.IsEnabled,
			SwapFee:        pair.SwapFee,
			X_Decimals:     pair.X_Decimals,
			Y_Decimals:     pair.Y_Decimals,
			AveragePrice:   pair.AveragePrice,
			SpotPrice:      pair.SpotPrice,
			Token0LimitMin: pair.Token0LimitMin,
			Token1LimitMin: pair.Token1LimitMin,
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

	_amountIn, amountOut, fee, err := p.swapExactIn(tokenIn, tokenOut, amountIn)
	if err != nil {
		return nil, err
	}

	var newReserve0, newReserve1 *uint256.Int

	switch tokenIn {
	case tokens[0]:
		if reserve1.Lt(amountOut) {
			return nil, fmt.Errorf("insufficient liquidity for tokenOut")
		}
		newReserve1 = SubUint256(reserve1, amountOut)
		newReserve0 = AddUint256(reserve0, _amountIn)
	case tokens[1]:
		if reserve0.Lt(amountOut) {
			return nil, fmt.Errorf("insufficient liquidity for tokenOut")
		}
		newReserve0 = SubUint256(reserve0, amountOut)
		newReserve1 = AddUint256(reserve1, _amountIn)
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
			RelayerAddress: p.RelayerAddress,
			NewReserve0:    ToInt256(newReserve0),
			NewReserve1:    ToInt256(newReserve1),
		},
	}, nil
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Integral %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}

	p.Info.Reserves = []*big.Int{si.NewReserve0, si.NewReserve1}
}

func (p *PoolSimulator) swapExactIn(tokenIn, tokenOut string, amountIn *uint256.Int) (*uint256.Int, *uint256.Int, *uint256.Int, error) {
	if !p.IntegralPair.IsEnabled {
		return nil, nil, nil, ErrTR05
	}

	tokens := p.GetTokens()
	fee := DivUint256(MulUint256(amountIn, p.IntegralPair.SwapFee), precision)

	inverted := tokens[1] == tokenIn

	amountOut := p.calculateAmountOut(inverted, SubUint256(amountIn, fee))

	if err := p.checkLimits(tokenOut, amountOut); err != nil {
		return nil, nil, nil, err
	}

	return amountIn, amountOut, fee, nil
}

func (p *PoolSimulator) checkLimits(token string, amount *uint256.Int) error {
	if token == p.GetTokens()[0] {
		if amount.Lt(p.IntegralPair.Token0LimitMin) {
			return ErrTR03
		}
	} else if token == p.GetTokens()[1] {
		if amount.Lt(p.IntegralPair.Token1LimitMin) {
			return ErrTR03
		}
	}

	return nil
}

func (p *PoolSimulator) calculateAmountOut(inverted bool, amountIn *uint256.Int) *uint256.Int {
	decimalsConverter := getDecimalsConverter(p.IntegralPair.X_Decimals, p.IntegralPair.Y_Decimals, inverted)

	price := p.getPrice(inverted)

	return DivUint256(MulUint256(amountIn, price), decimalsConverter)
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
