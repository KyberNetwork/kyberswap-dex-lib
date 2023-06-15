package lido

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

type Pool struct {
	poolPkg.Pool
	// extra fields
	StEthPerToken  *big.Int
	TokensPerStEth *big.Int
	LpToken        string
	gas            Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var numTokens = len(entityPool.Tokens)
	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)

	var staticExtra StaticExtra
	var err = json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra)
	if err != nil {
		return nil, err
	}

	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
		reserves[i] = utils.NewBig10(entityPool.Reserves[i])
	}

	var extraStr Extra
	err = json.Unmarshal([]byte(entityPool.Extra), &extraStr)
	if err != nil {
		return nil, err
	}

	return &Pool{
		Pool: poolPkg.Pool{
			Info: poolPkg.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    constant.Zero,
				Exchange:   entityPool.Exchange,
				Type:       entityPool.Type,
				Tokens:     tokens,
				Reserves:   reserves,
				Checked:    false,
			},
		},
		StEthPerToken:  extraStr.StEthPerToken,
		TokensPerStEth: extraStr.TokensPerStEth,
		LpToken:        staticExtra.LpToken,
		gas:            DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn poolPkg.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var amountOut *big.Int
	var totalGas int64

	if strings.EqualFold(tokenOut, p.LpToken) {
		amountOut = new(big.Int).Div(new(big.Int).Mul(tokenAmountIn.Amount, p.TokensPerStEth), constant.BONE)
		totalGas = p.gas.Wrap
	} else {
		amountOut = new(big.Int).Div(new(big.Int).Mul(tokenAmountIn.Amount, p.StEthPerToken), constant.BONE)
		totalGas = p.gas.Unwrap
	}

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

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *Pool) GetLpToken() string {
	return p.LpToken
}

func (p *Pool) GetMidPrice(_ string, tokenOut string, _ *big.Int) *big.Int {
	if strings.EqualFold(tokenOut, p.LpToken) {
		return p.TokensPerStEth
	}

	return p.StEthPerToken
}

func (p *Pool) CalcExactQuote(_ string, tokenOut string, _ *big.Int) *big.Int {
	if strings.EqualFold(tokenOut, p.LpToken) {
		return p.TokensPerStEth
	}

	return p.StEthPerToken
}

func (p *Pool) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
