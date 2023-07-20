package uniswap

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolSimulator struct {
	pool.Pool
	Weights []uint
	gas     Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bOneFloat)
	swapFee, _ := swapFeeFl.Int(nil)
	tokens := make([]string, 2)
	weights := make([]uint, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		var weight0 = uint(50)
		if entityPool.Tokens[0].Weight > 0 {
			weight0 = entityPool.Tokens[0].Weight
		}
		var weight1 = uint(50)
		if entityPool.Tokens[1].Weight > 0 {
			weight1 = entityPool.Tokens[1].Weight
		}
		tokens[0] = entityPool.Tokens[0].Address
		weights[0] = weight0
		reserves[0] = NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		weights[1] = weight1
		reserves[1] = NewBig10(entityPool.Reserves[1])
	}
	info := pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:    pool.Pool{Info: info},
		Weights: weights,
		gas:     defaultGas,
	}, nil
}

func (t *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex: %v or tokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, err := getAmountOut(
		tokenAmountIn.Amount,
		t.Info.Reserves[tokenInIndex],
		t.Info.Reserves[tokenOutIndex],
		t.Weights[tokenInIndex],
		t.Weights[tokenOutIndex],
		t.Info.SwapFee,
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	var totalGas = t.gas.SwapBase
	if t.Weights[tokenInIndex] != t.Weights[tokenOutIndex] {
		totalGas = t.gas.SwapNonBase
	}

	if amountOut.Cmp(zeroBI) > 0 {
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: nil,
			},
			Gas: totalGas,
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("invalid amount out: %v", amountOut.String())
}

func (t *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = new(big.Int).Div(new(big.Int).Mul(input.Amount, new(big.Int).Sub(bOne, t.Info.SwapFee)), bOne)
	var outputAmount = output.Amount
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}

func (t *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	if t.GetInfo().SwapFee == nil {
		return Meta{
			SwapFee: defaultSwapFee,
		}
	}

	return Meta{
		SwapFee: t.GetInfo().SwapFee.String(),
	}
}
