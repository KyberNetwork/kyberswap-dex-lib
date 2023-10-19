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
			scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[1])
			if err != nil {
				return &pool.CalcAmountOutResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		} else {
			scaleAmountOut, err = ScaleToAmount(amountOut, p.decimals[0])
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
			SwapInfo: maverickSwapInfo{
				activeTick: newState.ActiveTick,
				bins:       newState.Bins,
			},
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(maverickSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalancer for Maverick pool, wrong swapInfo type")
		return
	}

	p.state.Bins = newState.bins
	p.state.ActiveTick = newState.activeTick
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}

func (p *Pool) deepcopyState(state *MaverickPoolState) (*MaverickPoolState, error) {
	newState := &MaverickPoolState{
		TickSpacing:      new(big.Int).Set(state.TickSpacing),
		Fee:              new(big.Int).Set(state.Fee),
		ProtocolFeeRatio: new(big.Int).Set(state.ProtocolFeeRatio),
		ActiveTick:       new(big.Int).Set(state.ActiveTick),
		BinCounter:       new(big.Int).Set(state.BinCounter),
	}

	// Clone state.Bins
	newState.Bins = make(map[string]Bin, len(state.Bins))
	for k, v := range state.Bins {
		newState.Bins[k] = Bin{
			ReserveA:  new(big.Int).Set(v.ReserveA),
			ReserveB:  new(big.Int).Set(v.ReserveB),
			LowerTick: new(big.Int).Set(v.LowerTick),
			Kind:      new(big.Int).Set(v.Kind),
			MergeID:   new(big.Int).Set(v.MergeID),
		}
	}

	// Clone state.BinPositions
	newState.BinPositions = make(map[string]map[string]*big.Int, len(state.BinPositions))
	for k, v := range state.BinPositions {
		newState.BinPositions[k] = make(map[string]*big.Int, len(v))
		for k1, v1 := range v {
			newState.BinPositions[k][k1] = new(big.Int).Set(v1)
		}
	}

	// Clone state.BinMap
	newState.BinMap = make(map[string]*big.Int, len(state.BinMap))
	for k, v := range state.BinMap {
		newState.BinMap[k] = new(big.Int).Set(v)
	}

	return newState, nil
}
