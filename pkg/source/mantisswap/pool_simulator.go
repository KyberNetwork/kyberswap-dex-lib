package mantisswap

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
)

type PoolSimulator struct {
	pool.Pool
	state *PoolState
	gas   Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
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
				Exchange: entityPool.Type,
				Type:     entityPool.Exchange,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		state: &PoolState{
			Paused:      extra.Paused,
			SwapAllowed: extra.SwapAllowed,
			BaseFee:     extra.BaseFee,
			LpRatio:     extra.LpRatio,
			SlippageA:   extra.SlippageA,
			SlippageN:   extra.SlippageN,
			SlippageK:   extra.SlippageK,
			LPs:         extra.LPs,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(tokenAmountIn pool.TokenAmount, tokenOut string) (*pool.CalcAmountOutResult, error) {
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	newState, err := p.deepCopy(p.state)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}
	amountOut, err := GetAmountOut(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount, newState)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
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
		Gas: p.gas.Swap,
		SwapInfo: swapInfo{
			lps: newState.LPs,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(swapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for MantisSwap pool, wrong swapInfo type")
	}
	p.state.LPs = newState.lps
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *PoolSimulator) deepCopy(state *PoolState) (*PoolState, error) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	var newState PoolState
	if err := json.Unmarshal(stateBytes, &newState); err != nil {
		return nil, err
	}

	return &newState, nil
}
