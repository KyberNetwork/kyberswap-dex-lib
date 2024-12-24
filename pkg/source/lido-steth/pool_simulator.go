package lido_steth

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	gas     int64
	chainID valueobject.ChainID
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	numTokens := len(entityPool.Tokens)
	if numTokens != 2 || !valueobject.IsWrappedNative(entityPool.Tokens[0].Address, chainID) {
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
		gas:     DefaultGas,
		chainID: chainID,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	stEth := p.Info.Tokens[1]
	// can only swap from ETH to stETH
	if !valueobject.IsWrappedNative(tokenAmountIn.Token, p.chainID) || !strings.EqualFold(tokenOut, stEth) {
		return nil, fmt.Errorf("Invalid tokenIn/Out %v %v", tokenAmountIn.Token, tokenOut)
	}

	/*
		function getSharesByPooledEth(uint256 _ethAmount) public view returns (uint256) {
					return _ethAmount
							.mul(_getTotalShares())
							.div(_getTotalPooledEther());
		}
		function getPooledEthByShares(uint256 _sharesAmount) public view returns (uint256) {
			return _sharesAmount
					.mul(_getTotalPooledEther())
					.div(_getTotalShares());
		}
	*/

	// convert input ETH to number of shares, then back to stETH
	totalPooledEther := p.Info.Reserves[0]
	totalShares := p.Info.Reserves[1]
	shares := new(big.Int).Div(new(big.Int).Mul(tokenAmountIn.Amount, totalShares), totalPooledEther)
	amountOut := new(big.Int).Div(new(big.Int).Mul(shares, totalPooledEther), totalShares)
	totalGas := p.gas

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: bignumber.ZeroBI,
		},
		Gas: totalGas,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	if strings.EqualFold(params.TokenAmountIn.Token, p.Info.Tokens[0]) {
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountIn.Amount)
		p.Info.Reserves[1] = new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountOut.Amount)
	} else {
		p.Info.Reserves[0] = new(big.Int).Add(p.Info.Reserves[0], params.TokenAmountOut.Amount)
		p.Info.Reserves[1] = new(big.Int).Sub(new(big.Int).Add(p.Info.Reserves[1], params.TokenAmountIn.Amount), params.Fee.Amount)
	}
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	// can only swap from ETH to stETH
	// to convert back (withdraw) we'll need to interact with another contract
	if strings.EqualFold(p.Info.Tokens[1], address) {
		return []string{p.Info.Tokens[0]}
	}
	return nil
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	// can only swap from ETH to stETH
	// to convert back (withdraw) we'll need to interact with another contract
	if strings.EqualFold(p.Info.Tokens[0], address) {
		return []string{p.Info.Tokens[1]}
	}
	return nil
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} {
	return nil
}
