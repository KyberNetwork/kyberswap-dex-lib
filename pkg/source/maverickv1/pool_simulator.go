package maverickv1

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

	if len(extra.Bins) == 0 {
		return nil, ErrEmptyBins
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

	var minBinMapIndex, maxBinMapIndex *big.Int
	if extra.MinBinMapIndex != nil && extra.MaxBinMapIndex != nil {
		minBinMapIndex = extra.MinBinMapIndex
		maxBinMapIndex = extra.MaxBinMapIndex
	} else {
		// after new pool-service deployed we'll have extra.MinBinMapIndex and extra.MaxBinMapIndex
		// if missing then calculate here
		binMap := extra.BinMap
		binIndexBase := 10
		if len(extra.BinMapHex) > 0 {
			binMap = extra.BinMapHex
			binIndexBase = 16
		}
		var binIndex big.Int
		for k := range binMap {
			binIndex.SetString(k, binIndexBase)
			if minBinMapIndex == nil {
				minBinMapIndex = new(big.Int).Set(&binIndex)
			} else if minBinMapIndex.Cmp(&binIndex) > 0 {
				minBinMapIndex.Set(&binIndex)
			}
			if maxBinMapIndex == nil {
				maxBinMapIndex = new(big.Int).Set(&binIndex)
			} else if maxBinMapIndex.Cmp(&binIndex) < 0 {
				maxBinMapIndex.Set(&binIndex)
			}
		}
		if minBinMapIndex == nil || maxBinMapIndex == nil {
			return nil, ErrEmptyBinMap
		}
	}

	return &Pool{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				SwapFee:  extra.Fee,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   tokens,
				Reserves: lo.Map(entityPool.Reserves, func(item string, index int) *big.Int { return utils.NewBig(item) }),
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
			BinMapHex:        extra.BinMapHex,
			minBinMapIndex:   minBinMapIndex,
			maxBinMapIndex:   maxBinMapIndex,
		},
		gas: DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		var tokenAIn bool
		var scaleAmount *big.Int
		var err error

		// in paraswap code, side is the input of exactOutput. In our CalcAmountOut simulation, exactOutput always equals false
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

		newState, err := DeepcopyState(p.state)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not deepcopy maverick state, err: %v", err)
		}

		_, amountOut, binCrossed, err := swap(newState, scaleAmount, tokenAIn, false, false)
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

		// this is not really correct, because some tick required `nextActive` while some doesn't
		// but should be good enough for now
		crossBinGas := p.gas.CrossBin * int64(binCrossed)

		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: scaleAmountOut,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: nil,
			},
			Gas: p.gas.Swap + crossBinGas,
			SwapInfo: maverickSwapInfo{
				activeTick: newState.ActiveTick,
				bins:       newState.Bins,
			},
		}, nil
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *Pool) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn := param.TokenIn
	tokenAmountOut := param.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(tokenIn)
	var tokenOutIndex = p.GetTokenIndex(tokenAmountOut.Token)

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		var tokenAIn bool
		var scaleAmount *big.Int
		var err error

		// in paraswap code, side is the input of exactOutput. In our CalcAmountIn simulation, exactOutput always equals true
		// https://github.com/paraswap/paraswap-dex-lib/blob/master/src/dex/maverick-v1/maverick-v1-pool.ts#L329

		if strings.EqualFold(tokenIn, p.Pool.Info.Tokens[0]) {
			tokenAIn = true
			scaleAmount, err = scaleFromAmount(tokenAmountOut.Amount, p.decimals[1])
			if err != nil {
				return &pool.CalcAmountInResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		} else {
			tokenAIn = false
			scaleAmount, err = scaleFromAmount(tokenAmountOut.Amount, p.decimals[0])
			if err != nil {
				return &pool.CalcAmountInResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		}

		newState, err := DeepcopyState(p.state)
		if err != nil {
			return &pool.CalcAmountInResult{}, fmt.Errorf("can not deepcopy maverick state, err: %v", err)
		}

		amountIn, _, binCrossed, err := swap(newState, scaleAmount, tokenAIn, true, false)
		if err != nil {
			return &pool.CalcAmountInResult{}, fmt.Errorf("swap failed, err: %v", err)
		}

		var scaleAmountIn *big.Int
		if strings.EqualFold(tokenIn, p.Pool.Info.Tokens[0]) {
			scaleAmountIn, err = ScaleToAmount(amountIn, p.decimals[0])
			if err != nil {
				return &pool.CalcAmountInResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		} else {
			scaleAmountIn, err = ScaleToAmount(amountIn, p.decimals[1])
			if err != nil {
				return &pool.CalcAmountInResult{}, fmt.Errorf("can not scale amount maverick, err: %v", err)
			}
		}

		// this is not really correct, because some tick required `nextActive` while some doesn't
		// but should be good enough for now
		crossBinGas := p.gas.CrossBin * int64(binCrossed)

		return &pool.CalcAmountInResult{
			TokenAmountIn: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: scaleAmountIn,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenIn,
				Amount: nil,
			},
			Gas: p.gas.Swap + crossBinGas,
			SwapInfo: maverickSwapInfo{
				activeTick: newState.ActiveTick,
				bins:       newState.Bins,
			},
		}, nil
	}

	return &pool.CalcAmountInResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
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

func DeepcopyState(state *MaverickPoolState) (*MaverickPoolState, error) {
	newState := &MaverickPoolState{
		TickSpacing:      new(big.Int).Set(state.TickSpacing),
		Fee:              new(big.Int).Set(state.Fee),
		ProtocolFeeRatio: new(big.Int).Set(state.ProtocolFeeRatio),
		ActiveTick:       new(big.Int).Set(state.ActiveTick),
		BinCounter:       new(big.Int).Set(state.BinCounter),
		minBinMapIndex:   new(big.Int).Set(state.minBinMapIndex),
		maxBinMapIndex:   new(big.Int).Set(state.maxBinMapIndex),
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
	binMapHexLen := len(state.BinMapHex)
	if binMapHexLen > 0 {
		newState.BinMapHex = make(map[string]*big.Int, binMapHexLen)
		for k, v := range state.BinMapHex {
			newState.BinMapHex[k] = new(big.Int).Set(v)
		}
	} else {
		newState.BinMap = make(map[string]*big.Int, len(state.BinMap))
		for k, v := range state.BinMap {
			newState.BinMap[k] = new(big.Int).Set(v)
		}
	}

	return newState, nil
}
