package balancerweighted

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	balancer "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/balancer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type WeightedPool2Tokens struct {
	pool.Pool
	VaultAddress string
	PoolId       string
	Decimals     []uint
	Weights      []*big.Int
	gas          balancer.Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*WeightedPool2Tokens, error) {
	var staticExtra balancer.StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), bignumber.BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)

	numTokens := len(entityPool.Tokens)
	tokens := make([]string, numTokens)
	reserves := make([]*big.Int, numTokens)
	weights := make([]*big.Int, numTokens)
	decimals := make([]uint, numTokens)

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
		weights[i] = big.NewInt(int64(entityPool.Tokens[i].Weight))
		decimals[i] = uint(staticExtra.TokenDecimals[i])
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
		gas:          balancer.DefaultGas,
	}, nil
}

func (t *WeightedPool2Tokens) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenIndexFrom = t.GetTokenIndex(tokenAmountIn.Token)
	var tokenIndexTo = t.GetTokenIndex(tokenOut)
	if tokenIndexFrom >= 0 && tokenIndexTo >= 0 {
		var maxAmountIn = new(big.Int).Div(new(big.Int).Mul(t.Info.Reserves[tokenIndexFrom], MaxInRatio), bignumber.TenPowInt(2))

		if tokenAmountIn.Amount.Cmp(bignumber.ZeroBI) < 0 {
			return &pool.CalcAmountOutResult{}, errors.New("tokenAmountIn.Amount is less than 0")
		}

		if tokenAmountIn.Amount.Cmp(maxAmountIn) > 0 {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenAmountIn.Amount %v is larger than maxAmountIn %v", *tokenAmountIn.Amount, maxAmountIn)
		}

		// this scaling up of both nominator and denominator seems not needed
		// but they do this explicitly here https://github.com/balancer/balancer-v2-monorepo/blob/45bfdc2/pkg/pool-weighted/contracts/lbp/LiquidityBootstrappingPool.sol#L125
		// maybe needed to have rounded down result
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
		var maxAmountOut = new(big.Int).Div(new(big.Int).Mul(t.Info.Reserves[tokenIndexTo], MaxOutRatio), bignumber.TenPowInt(2))
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

func (t *WeightedPool2Tokens) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	mapTokenAddressToIndex := make(map[string]int)
	for idx, tokenAddress := range t.Pool.Info.Tokens {
		mapTokenAddressToIndex[tokenAddress] = idx
	}
	return balancer.Meta{
		VaultAddress:           t.VaultAddress,
		PoolId:                 t.PoolId,
		MapTokenAddressToIndex: mapTokenAddressToIndex,
	}
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
