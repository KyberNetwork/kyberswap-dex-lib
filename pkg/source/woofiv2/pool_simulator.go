package woofiv2

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
)

type PoolSimulator struct {
	pool.Pool
	state *WooFiV2State
	gas   Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var tokens = make([]string, len(entityPool.Tokens))
	for i, token := range entityPool.Tokens {
		tokens[i] = token.Address
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		state: &WooFiV2State{
			QuoteToken:    extra.QuoteToken,
			UnclaimedFee:  extra.UnclaimedFee,
			TokenInfos:    extra.TokenInfos,
			Timestamp:     extra.Timestamp,
			StaleDuration: extra.StaleDuration,
			Bound:         extra.Bound,
		},
		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokenInIndex := p.GetTokenIndex(tokenAmountIn.Token)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("TokenInIndex: %v or TokenOutIndex: %v is not correct", tokenInIndex, tokenOutIndex)
	}

	newState, err := p.deepCopyState(p.state)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
	}
	amountOut, err := GetAmountOut(
		tokenAmountIn.Token,
		tokenOut,
		tokenAmountIn.Amount,
		newState,
	)
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
		SwapInfo: wooFiV2SwapInfo{
			unclaimedFee: newState.UnclaimedFee,
			tokenInfos:   newState.TokenInfos,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(wooFiV2SwapInfo)
	if !ok {
		logger.Error("failed to UpdateBalancer for WooFiV2 pool, wrong swapInfo type")
		return
	}

	p.state.TokenInfos = newState.tokenInfos
	p.state.UnclaimedFee = newState.unclaimedFee
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *PoolSimulator) deepCopyState(state *WooFiV2State) (*WooFiV2State, error) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	var newState WooFiV2State
	if err := json.Unmarshal(stateBytes, &newState); err != nil {
		return nil, err
	}

	return &newState, nil
}
