package wombatlsd

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"math/big"
)

type PoolSimulator struct {
	pool.Pool
	haircutRate   *big.Int
	ampFactor     *big.Int
	startCovRatio *big.Int
	endCovRatio   *big.Int
	assets        map[string]wombat.Asset
	gas           wombat.Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra wombat.Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, len(entityPool.Tokens))
	for _, token := range entityPool.Tokens {
		tokens = append(tokens, token.Address)
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Type:     entityPool.Type,
				Exchange: entityPool.Exchange,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		haircutRate:   extra.HaircutRate,
		ampFactor:     extra.AmpFactor,
		startCovRatio: extra.StartCovRatio,
		endCovRatio:   extra.EndCovRatio,
		assets:        extra.AssetMap,
		gas:           DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, haircut, err := QuotePotentialSwap(
		tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount,
		p.haircutRate, p.ampFactor, p.startCovRatio, p.endCovRatio,
		p.assets,
	)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOut,
	}
	fee := &pool.TokenAmount{
		Token:  tokenAmountOut.Token,
		Amount: haircut,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            fee,
		Gas:            p.gas.Swap,
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	fromAsset, err := assetOf(params.TokenAmountIn.Token, p.assets)
	if err != nil {
		return
	}

	toAsset, err := assetOf(params.TokenAmountOut.Token, p.assets)
	if err != nil {
		return
	}

	fromAsset.Cash = addCash(fromAsset.Cash, params.TokenAmountIn.Amount)
	toAsset.Cash = removeCash(toAsset.Cash, params.TokenAmountOut.Amount)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
