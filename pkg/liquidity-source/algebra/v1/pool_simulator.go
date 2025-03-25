package algebrav1

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/goccy/go-json"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool
	globalState GlobalStateUint256
	liquidity   *v3Utils.Uint128
	ticks       *v3Entities.TickListDataProvider
	tickMin     int
	tickMax     int
	tickSpacing int
}

var _ = pool.RegisterFactory0(DexTypeAlgebraV1, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra ExtraUint256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	} else if extra.GlobalState.Tick < v3Utils.MinTick || extra.GlobalState.Tick > v3Utils.MaxTick {
		return nil, ErrTickInvalid
	} else if len(extra.Ticks) == 0 {
		return nil, ErrTicksEmpty
	} else if !extra.GlobalState.Unlocked {
		return nil, ErrPoolLocked
	} else if len(entityPool.Reserves) != 2 || len(entityPool.Tokens) != 2 {
		return nil, ErrInvalidToken
	}

	ticks, err := v3Entities.NewTickListDataProvider(extra.Ticks, int(extra.TickSpacing))
	if err != nil {
		return nil, err
	}

	tokens := []string{entityPool.Tokens[0].Address, entityPool.Tokens[1].Address}
	reserves := []*big.Int{bignumber.NewBig10(entityPool.Reserves[0]), bignumber.NewBig10(entityPool.Reserves[1])}
	tickMin := extra.Ticks[0].Index
	tickMax := extra.Ticks[len(extra.Ticks)-1].Index

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:    strings.ToLower(entityPool.Address),
			ReserveUsd: entityPool.ReserveUsd,
			Exchange:   entityPool.Exchange,
			Type:       entityPool.Type,
			Tokens:     tokens,
			Reserves:   reserves,
		}},
		globalState: extra.GlobalState,
		liquidity:   extra.Liquidity,
		ticks:       ticks,
		tickMin:     tickMin,
		tickMax:     tickMax,
		tickSpacing: int(extra.TickSpacing),
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	zeroForOne := strings.EqualFold(tokenIn, p.Info.Tokens[0])
	var amountOut v3Utils.Int256
	if overflow := amountOut.SetFromBig(tokenAmountOut.Amount); overflow {
		return nil, ErrOverflow
	}
	var priceLimit v3Utils.Uint160
	if err := p.getSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, errors.WithMessage(err, "CalcAmountOut")
	}

	swapResult, err := p._calculateSwapAndLock(zeroForOne, amountOut.Neg(&amountOut), &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("_calculateSwapAndLock failed: %v", err)
	} else if swapResult.amountCalculated.Sign() <= 0 {
		return nil, ErrZeroAmountIn
	}

	amountIn := swapResult.amountCalculated.ToBig()
	var remainingTokenAmountOut *pool.TokenAmount
	if swapResult.remainingAmountRequired != nil {
		remainingTokenAmountOut = &pool.TokenAmount{
			Token:  tokenAmountOut.Token,
			Amount: swapResult.remainingAmountRequired.ToBig(),
		}
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountIn,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		RemainingTokenAmountOut: remainingTokenAmountOut,
		Gas:                     BaseGas + swapResult.crossInitTickLoops*CrossInitTickGas,
		SwapInfo:                swapResult.StateUpdate,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	zeroForOne := strings.EqualFold(tokenOut, p.Info.Tokens[1])
	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	var priceLimit v3Utils.Uint160
	if err := p.getSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, errors.WithMessage(err, "CalcAmountOut")
	}

	swapResult, err := p._calculateSwapAndLock(zeroForOne, &amountIn, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("_calculateSwapAndLock failed: %v", err)
	} else if swapResult.amountCalculated.Sign() >= 0 {
		return nil, ErrZeroAmountOut
	}

	amountOut := swapResult.amountCalculated.Neg(swapResult.amountCalculated).ToBig()
	var remainingTokenAmountIn *pool.TokenAmount
	if swapResult.remainingAmountRequired != nil {
		remainingTokenAmountIn = &pool.TokenAmount{
			Token:  tokenAmountIn.Token,
			Amount: swapResult.remainingAmountRequired.ToBig(),
		}
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Gas:                    BaseGas + swapResult.crossInitTickLoops*CrossInitTickGas,
		SwapInfo:               swapResult.StateUpdate,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.liquidity = p.liquidity.Clone()
	cloned.globalState.Price = p.globalState.Price.Clone()
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(*StateUpdate)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Algebra %v %v pool, wrong swapInfo type",
			p.Info.Address, p.Info.Exchange)
		return
	}
	p.liquidity.Set(si.Liquidity)
	p.globalState = si.GlobalState
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	zeroForOne := strings.EqualFold(tokenIn, p.Info.Tokens[0])
	var priceLimit v3Utils.Uint160
	_ = p.getSqrtPriceLimit(zeroForOne, &priceLimit)
	return PoolMeta{
		PriceLimit: &priceLimit,
	}
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool, result *v3Utils.Uint160) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := v3Utils.GetSqrtRatioAtTickV2(tickLimit, result); err != nil {
		return err
	}

	if zeroForOne {
		result.AddUint64(result, 1) // = (sqrtPrice at minTick) + 1
	} else {
		result.SubUint64(result, 1) // = (sqrtPrice at maxTick) - 1
	}

	return nil
}
