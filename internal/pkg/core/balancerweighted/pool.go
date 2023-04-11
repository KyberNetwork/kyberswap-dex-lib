package balancerweighted

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type WeightedPool2Tokens struct {
	pool.Pool
	VaultAddress string
	PoolId       string
	Decimals     []uint
	Weights      []*big.Int
	gas          Gas
}

func NewPool(entityPool entity.Pool) (*WeightedPool2Tokens, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	weights := make([]*big.Int, numTokens)
	decimals := make([]uint, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
		weights[i] = big.NewInt(int64(entityPool.Tokens[i].Weight))
		decimals[i] = staticExtra.TokenDecimals[i]
	}

	return &WeightedPool2Tokens{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    entityPool.Address,
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    swapFee,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		VaultAddress: strings.ToLower(staticExtra.VaultAddress),
		PoolId:       strings.ToLower(staticExtra.PoolId),
		Decimals:     decimals,
		Weights:      weights,
		gas:          DefaultGas,
	}, nil
}

func (t *WeightedPool2Tokens) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var maxAmountIn = new(big.Int).Div(new(big.Int).Mul(t.Info.Reserves[tokenIndexFrom], MaxInRatio), constant.TenPowInt(2))

		if tokenAmountIn.Amount.Cmp(constant.Zero) < 0 {
			return &pool.CalcAmountOutResult{}, errors.New("tokenAmountIn.Amount is less than 0")
		}

		if tokenAmountIn.Amount.Cmp(maxAmountIn) > 0 {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenAmountIn.Amount %v is larger than maxAmountIn %v", *tokenAmountIn.Amount, maxAmountIn)
		}
		var scalingFactorTokenIn = _computeScalingFactor(t.Decimals[tokenIndexFrom])
		var scalingFactorTokenOut = _computeScalingFactor(t.Decimals[tokenIndexTo])
		var balanceTokenIn = _upscale(t.Info.Reserves[tokenIndexFrom], scalingFactorTokenIn)
		var balanceTokenOut = _upscale(t.Info.Reserves[tokenIndexTo], scalingFactorTokenOut)
		var feeAmount = mulUp(tokenAmountIn.Amount, t.Info.SwapFee)
		var amount = _upscale(new(big.Int).Sub(tokenAmountIn.Amount, feeAmount), scalingFactorTokenIn)
		var amountOut = calcOutGivenIn(
			balanceTokenIn,
			t.Weights[tokenIndexFrom],
			balanceTokenOut,
			t.Weights[tokenIndexTo],
			amount,
		)
		amountOut = _downscaleDown(amountOut, scalingFactorTokenOut)
		var maxAmountOut = new(big.Int).Div(new(big.Int).Mul(t.Info.Reserves[tokenIndexTo], MaxOutRatio), constant.TenPowInt(2))
		if amountOut.Cmp(maxAmountOut) > 0 {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("amountOut %v is larger than maxAmountOut %v", amountOut, maxAmountOut)
		}
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: feeAmount,
			},
			Gas: t.gas.Swap,
		}, nil

	}
	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenIndexFrom %v or tokenIndexTo %v is not correct", tokenIndexFrom, tokenIndexTo)
}

func (t *WeightedPool2Tokens) CanSwapTo(address string) []string {
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

func (t *WeightedPool2Tokens) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}

func (t *WeightedPool2Tokens) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}

func (t *WeightedPool2Tokens) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return Meta{
		VaultAddress: t.VaultAddress,
		PoolId:       t.PoolId,
	}
}

func (t *WeightedPool2Tokens) GetLpToken() string {
	return t.GetAddress()
}

func (t *WeightedPool2Tokens) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output := params.TokenAmountIn, params.TokenAmountOut
	var tokenInIndex = t.GetTokenIndex(input.Token)
	var tokenOutIndex = t.GetTokenIndex(output.Token)
	if tokenInIndex >= 0 {
		t.Info.Reserves[tokenInIndex] = new(big.Int).Add(t.Info.Reserves[tokenInIndex], input.Amount)
	}
	if tokenOutIndex >= 0 {
		t.Info.Reserves[tokenOutIndex] = new(big.Int).Sub(t.Info.Reserves[tokenOutIndex], output.Amount)
	}
}
