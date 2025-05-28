package iziswap

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap/swap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	PoolInfo swap.PoolInfoU256
}

var _ = pool.RegisterFactory0(DexTypeiZiSwap, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra ExtraU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	} else if extra.LimitOrders == nil {
		return nil, ErrLimitOrderNil
	} else if extra.Liquidities == nil {
		return nil, ErrLiquidityNil
	} else if len(entityPool.Reserves) != 2 {
		return nil, ErrInvalidReservesLength
	} else if len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidTokensLength
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  strings.ToLower(entityPool.Address),
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
				Reserves: []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]),
					bignumber.NewBig10(entityPool.Reserves[1])},
			},
		},
		PoolInfo: extra,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenIn, tokenOut := tokenAmountIn.Token, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return nil, ErrInvalidToken
	}

	tokenAmountInAmount, overflow := uint256.FromBig(tokenAmountIn.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}

	x2y := tokenIn < tokenOut
	ptLimit := p.PoolInfo.CurrentPoint // TODO: not limit swap-range in the future or give a way to modify it
	swapFn := swap.SwapX2Y
	if x2y {
		ptLimit -= SIMULATOR_PT_RANGE
	} else {
		ptLimit += SIMULATOR_PT_RANGE
		swapFn = swap.SwapY2X
	}

	ret, err := swapFn(tokenAmountInAmount, ptLimit, p.PoolInfo)
	if err != nil {
		return nil, err
	}

	amountOut := ret.AmountX
	if x2y {
		amountOut = ret.AmountY
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		Gas: gasBase + gasPerCrossedLiqPt*ret.CrossedPoints,
		SwapInfo: iZiSwapInfo{
			nextPoint:      ret.CurrentPoint,
			nextLiquidity:  ret.Liquidity,
			nextLiquidityX: ret.LiquidityX,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	tokenIn, tokenOut := param.TokenIn, tokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return nil, ErrInvalidToken
	}

	tokenAmountOutAmount, overflow := uint256.FromBig(tokenAmountOut.Amount)
	if overflow {
		return nil, ErrInvalidAmount
	}

	x2y := tokenIn < tokenOut
	ptLimit := p.PoolInfo.CurrentPoint // TODO: not limit swap-range in the future or give a way to modify it
	swapFn := swap.SwapX2YDesireY
	if x2y {
		ptLimit -= SIMULATOR_PT_RANGE
	} else {
		ptLimit += SIMULATOR_PT_RANGE
		swapFn = swap.SwapY2XDesireX
	}

	ret, err := swapFn(tokenAmountOutAmount, ptLimit, p.PoolInfo)
	if err != nil {
		return nil, err
	}

	amountIn := ret.AmountY
	remainingAmountOut := new(big.Int)
	if x2y {
		amountIn = ret.AmountX
		remainingAmountOut.Sub(tokenAmountOut.Amount, ret.AmountY.ToBig())
	} else {
		remainingAmountOut.Sub(tokenAmountOut.Amount, ret.AmountX.ToBig())
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn.ToBig(),
		},
		RemainingTokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: remainingAmountOut,
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountOut.Token,
		},
		Gas: gasBase + gasPerCrossedLiqPt*ret.CrossedPoints,
		SwapInfo: iZiSwapInfo{
			nextPoint:      ret.CurrentPoint,
			nextLiquidity:  ret.Liquidity,
			nextLiquidityX: ret.LiquidityX,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.PoolInfo.Liquidity = p.PoolInfo.Liquidity.Clone()
	cloned.PoolInfo.LiquidityX = p.PoolInfo.LiquidityX.Clone()
	return &cloned
}

// UpdateBalance updates pool state
// we should notice that,
// unlike liquidity distribution,
// limit orders may change more frequently, and often changed after each exchange
// (if any limit order is dealed during that exchange)
//
// the function `UpdateBalance` which only change
// `currentPoint`, `liquidity` and `liquidityX` on currentPoint
// is not enough to trace actual limit order distribution on the pool in time,
// since then we may get inaccurate value of `amountOut` in `CalcAmountOut` if there
// exists too many limit orders (especially around current point) on that pool
//
// that means, if there exists some limit orders around the current point of a pool
// and we still want to get the value of `amountOut` as accurate as possible, we need
// to call pool_tracker of that pool more frequently to update limit order distribution in time
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(iZiSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for UniV3 pool, wrong swapInfo type")
		return
	}
	p.PoolInfo.CurrentPoint = si.nextPoint
	p.PoolInfo.Liquidity = si.nextLiquidity
	p.PoolInfo.LiquidityX = si.nextLiquidityX
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	limitPoint := p.PoolInfo.CurrentPoint
	if tokenIn < tokenOut {
		limitPoint -= SIMULATOR_PT_RANGE
	} else {
		limitPoint += SIMULATOR_PT_RANGE
	}
	return Meta{LimitPoint: limitPoint}
}
