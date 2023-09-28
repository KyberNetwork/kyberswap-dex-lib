package iziswap

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/logger"
	"github.com/izumiFinance/iZiSwap-SDK-go/swap"
	"github.com/pkg/errors"
)

type PoolSimulator struct {
	pool.Pool
	PoolInfo swap.PoolInfo
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.LimitOrders == nil {
		return nil, ErrLimitOrderNil
	}

	if extra.Liquidities == nil {
		return nil, ErrLiquidityNil
	}

	if len(entityPool.Reserves) != 2 {
		return nil, ErrInvalidReservesLength
	}
	if len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidTokensLength
	}

	reserves0, ok := new(big.Int).SetString(entityPool.Reserves[0], 10)
	if !ok {
		return nil, errors.Wrapf(ErrInvalidReserve, "fail to parse reserve[0] %s to big.Int", entityPool.Reserves[0])
	}

	reserves1, ok := new(big.Int).SetString(entityPool.Reserves[1], 10)
	if !ok {
		return nil, errors.Wrapf(ErrInvalidReserve, "fail to parse reserve[1] %s to big.Int", entityPool.Reserves[1])
	}

	// swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), boneFloat)
	// swapFee, _ := swapFeeFl.Int(nil)

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:    strings.ToLower(entityPool.Address),
				ReserveUsd: entityPool.ReserveUsd,
				// SwapFee:    swapFee,
				Exchange: entityPool.Exchange,
				Type:     entityPool.Type,
				Tokens:   []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address},
				Reserves: []*big.Int{reserves0, reserves1},
			},
		},
		PoolInfo: swap.PoolInfo(extra),
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	tokenInAddr := tokenAmountIn.Token
	tokenOutAddr := tokenOut

	tokenInIndex := p.GetTokenIndex(tokenInAddr)
	tokenOutIndex := p.GetTokenIndex(tokenOutAddr)
	if tokenInIndex < 0 || tokenOutIndex < 0 || tokenInIndex == tokenOutIndex {
		return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	// Clone tokenAmountIn.Amount, since the SDK will mutate it
	tokenAmountInAmount := new(big.Int).Set(tokenAmountIn.Amount)

	x2y := tokenInAddr < tokenOutAddr
	if x2y {
		// todo, not limit swap-range in the future
		//    or give a way to modify it
		lowPt := p.PoolInfo.CurrentPoint - SIMULATOR_PT_RANGE
		ret, err := swap.SwapX2Y(tokenAmountInAmount, lowPt, p.PoolInfo)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountY := ret.AmountY
		// // Fee can be ignored for now
		// amountX := ret.AmountX
		// fee := new(big.Int).Mul(&amountX, big.NewInt(int64(p.PoolInfo.Fee)))
		// fee.Div(fee, feeBase)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountY,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: nil,
			},
			SwapInfo: iZiSwapInfo{
				nextPoint:      ret.CurrentPoint,
				nextLiquidity:  new(big.Int).Set(ret.Liquidity),
				nextLiquidityX: new(big.Int).Set(ret.LiquidityX),
			},
		}, nil
	} else {
		// todo, not limit swap-range in the future
		//    or give a way to modify it
		highPt := p.PoolInfo.CurrentPoint + SIMULATOR_PT_RANGE
		ret, err := swap.SwapY2X(tokenAmountInAmount, highPt, p.PoolInfo)
		if err != nil {
			return &pool.CalcAmountOutResult{}, err
		}
		amountX := ret.AmountX
		// // Fee can be ignored for now
		// fee := new(big.Int).Mul(amountX, big.NewInt(int64(p.PoolInfo.Fee)))
		// fee.Div(fee, feeBase)
		return &pool.CalcAmountOutResult{
			TokenAmountOut: &pool.TokenAmount{
				Token:  tokenOut,
				Amount: amountX,
			},
			Fee: &pool.TokenAmount{
				Token:  tokenAmountIn.Token,
				Amount: nil,
			},
			SwapInfo: iZiSwapInfo{
				nextPoint:      ret.CurrentPoint,
				nextLiquidity:  new(big.Int).Set(ret.Liquidity),
				nextLiquidityX: new(big.Int).Set(ret.LiquidityX),
			},
		}, nil
	}
}

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

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	limitPoint := p.PoolInfo.CurrentPoint - SIMULATOR_PT_RANGE
	if tokenIn > tokenOut {
		limitPoint = p.PoolInfo.CurrentPoint + SIMULATOR_PT_RANGE
	}
	return Meta{LimitPoint: limitPoint}
}
