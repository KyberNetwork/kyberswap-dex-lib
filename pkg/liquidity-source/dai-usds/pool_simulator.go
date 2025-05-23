package dai_usds

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	gas int64
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {

	numTokens := len(entityPool.Tokens)
	if numTokens != 2 {
		return nil, fmt.Errorf("invalid pool tokens %v, %v", entityPool, numTokens)
	}
	if numTokens != len(entityPool.Reserves) {
		return nil, fmt.Errorf("invalid pool reserves %v, %v", entityPool, numTokens)
	}

	var tokens = make([]string, numTokens)
	var reserves = make([]*big.Int, numTokens)
	for i := 0; i < numTokens; i += 1 {
		tokens[i] = entityPool.Tokens[i].Address
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
		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut

	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex: %v or tokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	totalGas := p.gas

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: uint256.MustFromBig(tokenAmountIn.Amount).ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: totalGas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	// can only swap from ETH to stETH
	// to convert back (withdraw) we'll need to interact with another contract
	if strings.EqualFold(p.Info.Tokens[1], address) {
		return []string{p.Info.Tokens[0]}
	}
	return []string{p.Info.Tokens[1]}
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	// can only swap from ETH to stETH
	// to convert back (withdraw) we'll need to interact with another contract
	if strings.EqualFold(p.Info.Tokens[0], address) {
		return []string{p.Info.Tokens[1]}
	}
	return []string{p.Info.Tokens[0]}
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
