package wombatmain

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
)

type PoolSimulator struct {
	pool.Pool
	paused        bool
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

	var tokens = make([]string, 0, len(entityPool.Tokens))
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
		paused:        extra.Paused,
		haircutRate:   extra.HaircutRate,
		ampFactor:     extra.AmpFactor,
		startCovRatio: extra.StartCovRatio,
		endCovRatio:   extra.EndCovRatio,
		assets:        extra.AssetMap,
		gas:           DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, haircut, err := Swap(
		tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount,
		p.paused, p.haircutRate, p.ampFactor, p.startCovRatio, p.endCovRatio,
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
	fromAsset, toAsset := p.assets[params.TokenAmountIn.Token], p.assets[params.TokenAmountOut.Token]

	fromAsset.Cash = new(big.Int).Add(fromAsset.Cash, params.TokenAmountIn.Amount)
	toAsset.Cash = new(big.Int).Sub(toAsset.Cash, new(big.Int).Add(params.TokenAmountOut.Amount, params.Fee.Amount))

	p.assets[params.TokenAmountIn.Token] = fromAsset
	p.assets[params.TokenAmountOut.Token] = toAsset
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
