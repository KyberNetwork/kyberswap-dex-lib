//go:generate go run github.com/tinylib/msgp -unexported -tests=false -v
//msgp:tuple PoolSimulator Gas
//msgp:shim *big.Int as:[]byte using:msgpencode.EncodeInt/msgpencode.DecodeInt

package lido

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	// extra fields
	StEthPerToken  *big.Int
	TokensPerStEth *big.Int
	LpToken        string
	gas            Gas
}

type Gas struct {
	Wrap   int64
	Unwrap int64
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	var extraStr Extra
	err = json.Unmarshal([]byte(entityPool.Extra), &extraStr)
	if err != nil {
		return nil, err
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				SwapFee:    bignumber.ZeroBI,
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var amountOut *big.Int
	var totalGas int64

	if strings.EqualFold(tokenOut, p.LpToken) {
		amountOut = new(big.Int).Div(new(big.Int).Mul(tokenAmountIn.Amount, p.TokensPerStEth), bignumber.BONE)
		totalGas = p.gas.Wrap
	} else {
		amountOut = new(big.Int).Div(new(big.Int).Mul(tokenAmountIn.Amount, p.StEthPerToken), bignumber.BONE)
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

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
