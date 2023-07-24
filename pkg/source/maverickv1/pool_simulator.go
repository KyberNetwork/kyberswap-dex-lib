package maverickv1

import (
	"encoding/json"
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"strings"
)

type PoolSimulator struct {
	pool.Pool
	state *MaverickPoolState
	gas   Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var tokens = make([]string, 2)
	tokens[0] = entityPool.Tokens[0].Address
	tokens[1] = entityPool.Tokens[1].Address

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				SwapFee:  extra.Fee,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Checked:  false,
			},
		},
		state: &MaverickPoolState{
			TickSpacing:      staticExtra.TickSpacing,
			Fee:              extra.Fee,
			ProtocolFeeRatio: extra.ProtocolFeeRatio,
			ActiveTick:       extra.ActiveTick,
			BinCounter:       extra.BinCounter,
			Bins:             extra.Bins,
			BinPositions:     extra.BinPositions,
			BinMap:           extra.BinMap,
		},
		gas: DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		var tokenAIn = false
		if strings.EqualFold(tokenAmountIn.Token, p.Pool.Info.Tokens[0]) {
			tokenAIn = true
		}

		newState, err := p.deepcopyState(p.state)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not deepcopy maverick state, err: %v", err)
		}

		_, amountOut, err := GetAmountOut(newState, tokenAmountIn.Amount, tokenAIn, false, false)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not get amount out, err: %v", err)
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
			SwapInfo: MaverickPoolState{
				ActiveTick: newState.ActiveTick,
				Bins:       newState.Bins,
			},
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(MaverickPoolState)
	if !ok {
		logger.Warn("failed to UpdateBalancer for Maverick pool, wrong swapInfo type")
		return
	}

	p.state.Bins = newState.Bins
	p.state.ActiveTick = newState.ActiveTick
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *PoolSimulator) deepcopyState(state *MaverickPoolState) (*MaverickPoolState, error) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	var newState MaverickPoolState
	if err := json.Unmarshal(stateBytes, &newState); err != nil {
		return nil, err
	}

	return &newState, nil
}
