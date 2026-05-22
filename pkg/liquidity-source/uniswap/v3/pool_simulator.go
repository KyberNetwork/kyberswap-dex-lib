package uniswapv3

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *Pool
	pool.Pool
	Gas             Gas
	tickMin         int
	tickMax         int
	allowEmptyTicks bool
}

var _ = pool.RegisterFactory1(DexTypeUniswapV3, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraTickU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return NewPoolSimulatorWithExtra(entityPool, chainID, &extra, false)
}

func NewPoolSimulatorWithExtra(entityPool entity.Pool, chainID valueobject.ChainID,
	extra *ExtraTickU256, allowEmptyTicks bool) (*PoolSimulator, error) {
	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	_ = chainID // chain ID not needed after removing coreEntities.Token

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	v3Ticks := make([]TickU256, 0, len(extra.Ticks))

	// Ticks are sorted from the pool service, so we don't have to do it again here.
	for _, t := range extra.Ticks {
		// LiquidityGross = 0 means the tick is uninitialized.
		if t.LiquidityGross.IsZero() {
			continue
		}
		v3Ticks = append(v3Ticks, TickU256{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}

	// if the tick list is empty, the pool should be ignored
	// for some uniswap-v4 hooks, we want to bypass this check due to some hooks has no ticks
	if !allowEmptyTicks && len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	tickSpacing := int(extra.TickSpacing)
	// For some pools that not yet initialized tickSpacing in their extra,
	// we will get the tickSpacing through feeTier mapping.
	if tickSpacing == 0 {
		feeTier := FeeAmount(entityPool.SwapFee)
		if _, ok := TickSpacings[feeTier]; !ok {
			return nil, ErrInvalidFeeTier
		}
		tickSpacing = TickSpacings[feeTier]
	}
	v3Pool, err := newPool(
		FeeAmount(entityPool.SwapFee),
		extra.SqrtPriceX96,
		extra.Liquidity,
		*extra.Tick,
		v3Ticks,
		tickSpacing,
	)
	if err != nil {
		return nil, err
	}

	tickMin, tickMax := MinTick, MaxTick
	if len(v3Ticks) > 0 {
		tickMin = v3Ticks[0].Index
		tickMax = v3Ticks[len(v3Ticks)-1].Index
	}

	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			SwapFee:  swapFee,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens:   tokens,
			Reserves: reserves,
		}},
		V3Pool:          v3Pool,
		Gas:             defaultGas,
		tickMin:         tickMin,
		tickMax:         tickMax,
		allowEmptyTicks: allowEmptyTicks,
	}, nil
}

// GetSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
func (p *PoolSimulator) GetSqrtPriceLimit(zeroForOne bool, result *uint256.Int) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := GetSqrtRatioAtTick(tickLimit, result); err != nil {
		return err
	}
	lo.Ternary(zeroForOne, result.AddUint64, result.SubUint64)(result, 1)
	return nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenOut := tokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	} else if tokenAmountOut.Amount.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientBalance
	}

	zeroForOne := tokenInIndex == 0

	var amountOutI256 int256.Int
	if overflow := amountOutI256.SetFromBig(tokenAmountOut.Amount); overflow {
		return nil, ErrOverflow
	}

	var priceLimit uint256.Int
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetSqrtPriceLimit, err: %+v", err)
	}
	amountIn, newPoolState, err := p.V3Pool.getInputAmountV2(&amountOutI256, zeroForOne, &priceLimit)
	if err != nil {
		return nil, err
	}

	amountInBI := amountIn.ToBig()
	if !p.allowEmptyTicks {
		if amountInBI.Sign() <= 0 {
			return nil, ErrZeroAmount
		}
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInBI,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.Gas.BaseGas, // TODO: update getInputAmountV2 to return crossed tick if we ever need this
		SwapInfo: SwapInfo{
			NextStateSqrtRatioX96: newPoolState.SqrtRatioX96,
			nextStateLiquidity:    newPoolState.Liquidity,
			NextStateTickCurrent:  newPoolState.TickCurrent,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}

	var amountIn int256.Int
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	zeroForOne := tokenInIndex == 0
	var priceLimit uint256.Int
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetSqrtPriceLimit, err: %+v", err)
	}
	amountOutResult, err := p.V3Pool.getOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, err
	}

	amountOut := amountOutResult.ReturnedAmount
	if !p.allowEmptyTicks && amountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountOutBI := amountOut.ToBig()
	if amountOutBI.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientBalance
	}

	remainingTokenAmountIn := &pool.TokenAmount{
		Token:  tokenIn,
		Amount: bignumber.ZeroBI,
	}
	if amountOutResult.RemainingAmountIn != nil {
		if amountOutResult.RemainingAmountIn.Sign() == 0 {
			amountOutResult.RemainingAmountIn = nil
		} else {
			remainingTokenAmountIn.Amount = amountOutResult.RemainingAmountIn.ToBig()
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOutBI,
		},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			RemainingAmountIn:     amountOutResult.RemainingAmountIn,
			NextStateSqrtRatioX96: amountOutResult.SqrtRatioX96,
			nextStateLiquidity:    amountOutResult.Liquidity,
			NextStateTickCurrent:  amountOutResult.CurrentTick,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Pool := *p.V3Pool
	cloned.V3Pool = &v3Pool
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for UniV3 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = si.NextStateSqrtRatioX96
	p.V3Pool.Liquidity = si.nextStateLiquidity
	p.V3Pool.TickCurrent = si.NextStateTickCurrent
	tokenAmtIn, tokenAmtOut := params.TokenAmountIn, params.TokenAmountOut
	if p.GetTokenIndex(tokenAmtIn.Token) == 0 {
		p.Info.Reserves = []*big.Int{new(big.Int).Add(p.Info.Reserves[0], tokenAmtIn.Amount),
			new(big.Int).Sub(p.Info.Reserves[1], tokenAmtOut.Amount)}
	} else {
		p.Info.Reserves = []*big.Int{new(big.Int).Sub(p.Info.Reserves[0], tokenAmtOut.Amount),
			new(big.Int).Add(p.Info.Reserves[1], tokenAmtIn.Amount)}
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	var priceLimit uint256.Int
	_ = p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0], &priceLimit)
	return PoolMeta{
		SwapFee:    uint32(p.Info.SwapFee.Int64()),
		PriceLimit: &priceLimit,
	}
}
