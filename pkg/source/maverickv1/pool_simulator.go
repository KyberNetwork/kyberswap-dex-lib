package maverickv1

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
)

type Pool struct {
	pool.Pool
	decimals []uint8
	state    *MaverickPoolState
	gas      Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*Pool, error) {
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

	var decimals = make([]uint8, 2)
	decimals[0] = entityPool.Tokens[0].Decimals
	decimals[1] = entityPool.Tokens[1].Decimals

	return &Pool{
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
		decimals: decimals,
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

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		var tokenAIn bool
		var scaleAmount *big.Int
		var err error

		// in paraswap code, side is the input of exactOutput. In our simulation, exactOutput always equals false
		// https://github.com/paraswap/paraswap-dex-lib/blob/master/src/dex/maverick-v1/maverick-v1-pool.ts#L329

		if strings.EqualFold(tokenAmountIn.Token, p.Pool.Info.Tokens[0]) {
			tokenAIn = true
			scaleAmount, err = scaleFromAmount(tokenAmountIn.Amount, p.decimals[0])
			if err != nil {
				return &pool.CalcAmountOutResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		} else {
			tokenAIn = false
			scaleAmount, err = scaleFromAmount(tokenAmountIn.Amount, p.decimals[1])
			if err != nil {
				return &pool.CalcAmountOutResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		}

		newState, err := p.deepcopyState(p.state)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not deepcopy maverick state, err: %v", err)
		}

		_, amountOut, err := GetAmountOut(newState, scaleAmount, tokenAIn, false, false)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not get amount out, err: %v", err)
		}

		var scaleAmountOut *big.Int
		if strings.EqualFold(tokenAmountIn.Token, p.Pool.Info.Tokens[0]) {
			scaleAmountOut, err = scaleToAmount(amountOut, p.decimals[1])
			if err != nil {
				return &pool.CalcAmountOutResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		} else {
			scaleAmountOut, err = scaleToAmount(amountOut, p.decimals[0])
			if err != nil {
				return &pool.CalcAmountOutResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		}

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: scaleAmountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: nil,
			},
			Gas: p.gas.Swap,
			SwapInfo: MaverickSwapInfo{
				ActiveTick: newState.ActiveTick,
				Bins:       newState.Bins,
			},
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(MaverickSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalancer for Maverick pool, wrong swapInfo type")
		return
	}

	p.state.Bins = newState.Bins
	p.state.ActiveTick = newState.ActiveTick
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *Pool) deepcopyState(state *MaverickPoolState) (*MaverickPoolState, error) {
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
