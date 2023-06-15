package curvecompound

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type CompoundPool struct {
	pool.Pool
	A           *big.Int
	Multipliers []*big.Int
	AdminFee    *big.Int
	Rates       []*big.Int
	gas         Gas
}

func NewPool(entityPool entity.Pool) (*CompoundPool, error) {
	var staticExtra PoolStaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	multipliers := make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = staticExtra.UnderlyingTokens[i]
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		multipliers[i] = utils.NewBig10(staticExtra.PrecisionMultipliers[i])
	}

	rates := make([]*big.Int, 0, len(extra.Rates))
	for _, rateStr := range extra.Rates {
		rate, _ := new(big.Int).SetString(rateStr, 10)
		rates = append(rates, rate)
	}

	return &CompoundPool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    utils.NewBig10(extra.SwapFee),
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		Multipliers: multipliers,
		A:           utils.NewBig10(extra.A),
		AdminFee:    utils.NewBig10(extra.AdminFee),
		Rates:       rates,
		gas:         DefaultGas,
	}, nil
}

func (t *CompoundPool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)

	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		amountOut, fee, err := GetDyUnderlying(
			t.Info.Reserves,
			t.Rates,
			t.Multipliers,
			t.A,
			t.Info.SwapFee,
			tokenIndexFrom,
			tokenIndexTo,
			tokenAmountIn.Amount,
		)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		if err == nil && amountOut.Cmp(constant.Zero) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: fee,
				},
				Gas: t.gas.ExchangeUnderlying,
			}, nil

		}
	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom or tokenIndexTo is not correct: tokenIndexFrom: %v, tokenIndexTo: %v", tokenIndexFrom, tokenIndexTo)
}
func (t *CompoundPool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var inputAmount = input.Amount
	var outputAmount = output.Amount
	// swap fee
	// output = output + output * swapFee * adminFee
	outputAmount = new(big.Int).Add(
		outputAmount,
		new(big.Int).Div(
			new(big.Int).Mul(
				new(big.Int).Div(new(big.Int).Mul(outputAmount, t.Info.SwapFee), FeeDenominator),
				t.AdminFee,
			),
			FeeDenominator,
		),
	)
	for i := range t.Info.Tokens {
		if t.Info.Tokens[i] == input.Token {
			t.Info.Reserves[i] = new(big.Int).Add(t.Info.Reserves[i], inputAmount)
		}
		if t.Info.Tokens[i] == output.Token {
			t.Info.Reserves[i] = new(big.Int).Sub(t.Info.Reserves[i], outputAmount)
		}
	}
}
func (t *CompoundPool) GetLpToken() string {
	return ""
}

func (t *CompoundPool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}
func (t *CompoundPool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}
func (t *CompoundPool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	var fromId = t.GetTokenIndex(tokenIn)
	var toId = t.GetTokenIndex(tokenOut)
	return Meta{
		TokenInIndex:  fromId,
		TokenOutIndex: toId,
		Underlying:    true,
	}
}
