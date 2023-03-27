package uni

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"
)

type Pool struct {
	pool.Pool
	Weights []uint
	gas     Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), BoneFloat)
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
		reserves[0] = utils.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		weights[1] = weight1
		reserves[1] = utils.NewBig10(entityPool.Reserves[1])
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

	return &Pool{
		Pool:    pool.Pool{Info: info},
		Weights: weights,
		gas:     DefaultGas,
	}, nil
}

func (t *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
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
		if amountOut.Cmp(constant.Zero) > 0 {
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
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (t *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = new(big.Int).Div(new(big.Int).Mul(input.Amount, new(big.Int).Sub(Bone, t.Info.SwapFee)), Bone)
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

func (t *Pool) GetLpToken() string {
	return ""
}

func (t *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = t.GetTokenIndex(address)
	if tokenIndex < 0 {
		return ret
	}
	for i := 0; i < len(t.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, t.Info.Tokens[i])
		}
	}
	return ret
}

func (t *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var ret = new(big.Int).Mul(new(big.Int).Mul(base, t.Info.Reserves[tokenOutIndex]), big.NewInt(int64(t.Weights[tokenInIndex])))
	ret = new(big.Int).Div(new(big.Int).Div(ret, t.Info.Reserves[tokenInIndex]), big.NewInt(int64(t.Weights[tokenOutIndex])))
	return ret
}

func (t *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var exactQuote = new(big.Int).Mul(new(big.Int).Mul(base, t.Info.Reserves[tokenOutIndex]), big.NewInt(int64(t.Weights[tokenInIndex])))
	exactQuote = new(big.Int).Div(new(big.Int).Div(exactQuote, t.Info.Reserves[tokenInIndex]), big.NewInt(int64(t.Weights[tokenOutIndex])))
	return exactQuote
}

func (t *Pool) GetMetaInfo(_ string, _ string) interface{} {
	if t.GetInfo().SwapFee == nil {
		return Meta{
			SwapFee: DefaultSwapFee,
		}
	}

	return Meta{
		SwapFee: t.GetInfo().SwapFee.String(),
	}
}
