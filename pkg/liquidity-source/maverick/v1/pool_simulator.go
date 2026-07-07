package maverickv1

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	decimals []uint8
	state    *MaverickPoolState
}

var _ = pool.RegisterFactory0(DexTypeMaverickV1, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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

	binMap := extra.BinMap
	binMapIds := lo.Keys(binMap)

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  entityPool.Address,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
				Reserves: []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])},
			},
		},
		decimals: []uint8{entityPool.Tokens[0].Decimals, entityPool.Tokens[1].Decimals},
		state: &MaverickPoolState{
			Fee:              extra.Fee,
			ProtocolFeeRatio: extra.ProtocolFeeRatio,
			Bins:             extra.Bins,
			BinPositions:     extra.BinPositions,
			BinMap:           binMap,
			TickSpacing:      staticExtra.TickSpacing,
			ActiveTick:       extra.ActiveTick,
			minBinMapIndex:   slices.Min(binMapIds),
			maxBinMapIndex:   slices.Max(binMapIds),
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountIn, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	scaledAmountIn, err := scaleFromAmount(amountIn, p.decimals[tokenInIndex])
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	newState := p.state.Clone()
	_, amountOut, binCrossed, err := swap(newState, scaledAmountIn, tokenInIndex == 0, false, false)
	if err != nil {
		return nil, fmt.Errorf("can not get amount out, err: %v", err)
	}

	scaledAmountOut, err := ScaleToAmount(amountOut, p.decimals[tokenOutIndex])
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: scaledAmountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		// this is not really correct, because some ticks required `nextActive` while some doesn't
		// but should be good enough for now
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick: newState.ActiveTick,
			bins:       newState.Bins,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountOut, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrOverflow
	}

	scaledAmountOut, err := scaleFromAmount(amountOut, p.decimals[tokenOutIndex])
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	newState := p.state.Clone()
	amountIn, _, binCrossed, err := swap(newState, scaledAmountOut, tokenInIndex == 0, true, false)
	if err != nil {
		return nil, fmt.Errorf("swap failed, err: %v", err)
	}

	scaledAmountIn, err := ScaleToAmount(amountIn, p.decimals[tokenInIndex])
	if err != nil {
		return nil, fmt.Errorf("can not scale amount maverick, err: %v", err)
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: scaledAmountIn.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		// this is not really correct, because some tick required `nextActive` while some doesn't
		// but should be good enough for now
		Gas: GasSwap + GasCrossBin*int64(binCrossed),
		SwapInfo: maverickSwapInfo{
			activeTick: newState.ActiveTick,
			bins:       newState.Bins,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.state = p.state.Clone()
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	newState, ok := params.SwapInfo.(maverickSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalancer for Maverick pool, wrong swapInfo type")
		return
	}

	p.state.Bins = newState.bins
	p.state.ActiveTick = newState.activeTick
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return nil
}

func (state *MaverickPoolState) Clone() *MaverickPoolState {
	cloned := *state
	cloned.Bins = lo.MapValues(state.Bins, func(bin Bin, _ uint32) Bin {
		bin.ReserveA = bin.ReserveA.Clone()
		bin.ReserveB = bin.ReserveB.Clone()
		return bin
	})
	return &cloned
}
