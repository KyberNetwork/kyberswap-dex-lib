package usd_ai

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	scaleFactor       *uint256.Int // 10^(18 - baseTokenDecimals)
	baseTokenDecimals uint8
	gas               int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	usdaiDecimals := entityPool.Tokens[0].Decimals
	baseTokenDecimals := entityPool.Tokens[1].Decimals

	scaleFactor := big256.TenPow(int(usdaiDecimals) - int(baseTokenDecimals))

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	for i := 0; i < 2; i++ {
		tokens[i] = strings.ToLower(entityPool.Tokens[i].Address)
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: reserves,
			},
		},
		scaleFactor:       scaleFactor,
		baseTokenDecimals: baseTokenDecimals,
		gas:               DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("usd-ai: invalid token in/out")
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow || amountIn.IsZero() {
		return nil, fmt.Errorf("usd-ai: invalid amount in")
	}

	var amountOut *uint256.Int
	if tokenInIndex == 1 {
		amountOut = new(uint256.Int).Mul(amountIn, p.scaleFactor)
	} else {
		amountOut = new(uint256.Int).Div(amountIn, p.scaleFactor)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: p.gas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(_ pool.UpdateBalanceParams) {}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	tokenInIndex := p.GetTokenIndex(tokenIn)

	isDeposit := tokenInIndex == 1

	return MetaInfo{
		IsDeposit:         isDeposit,
		BaseTokenDecimals: p.baseTokenDecimals,
	}
}
