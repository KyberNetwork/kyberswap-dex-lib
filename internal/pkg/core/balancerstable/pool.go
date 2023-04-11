package balancerstable

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

type StablePool struct {
	pool.Pool
	A              *big.Int
	Precision      *big.Int
	VaultAddress   string
	PoolId         string
	Decimals       []uint
	ScalingFactors []*big.Int
	gas            Gas
}

func NewPool(entityPool entity.Pool) (*StablePool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)
	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
	}

	return &StablePool{
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
		VaultAddress:   strings.ToLower(staticExtra.VaultAddress),
		PoolId:         strings.ToLower(staticExtra.PoolId),
		A:              extra.AmplificationParameter.Value,
		Precision:      extra.AmplificationParameter.Precision,
		ScalingFactors: extra.ScalingFactors,
		Decimals:       staticExtra.TokenDecimals,
		gas:            DefaultGas,
	}, nil
}

func (t *StablePool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var feeAmount = mulUp(tokenAmountIn.Amount, t.Info.SwapFee)
		var amountIn = new(big.Int).Sub(tokenAmountIn.Amount, feeAmount)
		var scalingFactorTokenIn = t.getScalingFactor(tokenIndexFrom)
		amountIn = _upscale(amountIn, scalingFactorTokenIn)

		var balances = make([]*big.Int, len(t.Info.Tokens))
		var scalingFactorOut *big.Int
		for i := 0; i < len(t.Info.Tokens); i += 1 {
			var scalingFactor = t.getScalingFactor(i)
			balances[i] = _upscale(t.Info.Reserves[i], scalingFactor)
			if i == tokenIndexTo {
				scalingFactorOut = scalingFactor
			}
		}
		var invariant = _calculateInvariant(t.A, balances, true)
		if invariant == nil {
			return &pool.CalcAmountOutResult{}, errors.New("invariant equals nil")
		}
		var amountOut = _calcOutGivenIn(t.A, balances, tokenIndexFrom, tokenIndexTo, amountIn, invariant)
		if amountOut == nil {
			return &pool.CalcAmountOutResult{}, errors.New("amountOut equals nil")
		}
		amountOut = _downscaleDown(amountOut, scalingFactorOut)
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

func (t *StablePool) CanSwapTo(address string) []string {
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

func (t *StablePool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var ret = new(big.Int).Mul(base, reserveOut)
	ret = new(big.Int).Div(ret, reserveIn)
	return ret
}

func (t *StablePool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	var tokenInIndex = t.GetTokenIndex(tokenIn)
	var tokenOutIndex = t.GetTokenIndex(tokenOut)
	var reserveIn = t.Info.Reserves[tokenInIndex]
	var reserveOut = t.Info.Reserves[tokenOutIndex]
	var exactQuote = new(big.Int).Mul(base, reserveOut)
	exactQuote = new(big.Int).Div(exactQuote, reserveIn)
	return exactQuote
}

func (t *StablePool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return Meta{
		VaultAddress: t.VaultAddress,
		PoolId:       t.PoolId,
	}
}

func (t *StablePool) GetLpToken() string {
	return t.GetAddress()
}

func (t *StablePool) UpdateBalance(params pool.UpdateBalanceParams) {
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

func (t *StablePool) getScalingFactor(tokenIndex int) *big.Int {
	if t.GetType() == constant.PoolTypes.BalancerMetaStable {
		return t.ScalingFactors[tokenIndex]
	}

	return _computeScalingFactor(t.Decimals[tokenIndex])
}
