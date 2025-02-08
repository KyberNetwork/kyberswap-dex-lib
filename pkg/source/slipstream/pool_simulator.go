package slipstream

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *v3Entities.Pool
	pool.Pool
	gas     Gas
	tickMin int
	tickMax int
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraTickU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, entityPool.Tokens[0].Name)
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, entityPool.Tokens[1].Name)

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	v3Ticks := make([]v3Entities.Tick, 0, len(extra.Ticks))

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	for _, t := range extra.Ticks {
		// LiquidityGross = 0 means that the tick is uninitialized
		if t.LiquidityGross.IsZero() {
			continue
		}

		v3Ticks = append(v3Ticks, v3Entities.Tick{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}

	// if the tick list is empty, the pool should be ignored
	if len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	tickSpacing := int(extra.TickSpacing)
	if tickSpacing == 0 {
		return nil, ErrInvalidTickSpacing
	}

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, tickSpacing)
	if err != nil {
		return nil, err
	}

	v3Pool, err := v3Entities.NewPoolV2(
		token0,
		token1,
		constants.FeeAmount(extra.FeeTier),
		extra.SqrtPriceX96,
		extra.Liquidity,
		*extra.Tick,
		ticks,
	)
	if err != nil {
		return nil, err
	}

	tickMin := v3Ticks[0].Index
	tickMax := v3Ticks[len(v3Ticks)-1].Index

	var info = pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
	}

	return &PoolSimulator{
		Pool:    pool.Pool{Info: info},
		V3Pool:  v3Pool,
		gas:     defaultGas,
		tickMin: tickMin,
		tickMax: tickMax,
	}, nil
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool, result *v3Utils.Uint160) error {
	tickLimit := p.tickMax
	if zeroForOne {
		tickLimit = p.tickMin
	}
	return v3Utils.GetSqrtRatioAtTickV2(tickLimit, result)
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenAmountOut.Token)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	zeroForOne := strings.EqualFold(tokenIn, hexutil.Encode(p.V3Pool.Token0.Address[:]))
	var amountOut v3Utils.Int256
	if overflow := amountOut.SetFromBig(tokenAmountOut.Amount); overflow {
		return nil, ErrOverflow
	}
	var priceLimit v3Utils.Uint160
	if err := p.getSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
	}

	amountInResult, err := p.V3Pool.GetOutputAmountV2(amountOut.Neg(&amountOut), zeroForOne, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
	} else if amountInResult.ReturnedAmount.Neg(amountInResult.ReturnedAmount).Sign() <= 0 {
		return nil, errors.New("amountIn is 0")
	}

	remainingTokenAmountOut := &pool.TokenAmount{Token: tokenAmountOut.Token}
	if amountInResult.RemainingAmountIn != nil {
		remainingTokenAmountOut.Amount = amountInResult.RemainingAmountIn.Neg(amountInResult.RemainingAmountIn).ToBig()
	} else {
		remainingTokenAmountOut.Amount = bignumber.ZeroBI
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInResult.ReturnedAmount.ToBig(),
		},
		RemainingTokenAmountOut: remainingTokenAmountOut,
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.gas.BaseGas + p.gas.CrossInitTickGas*int64(amountInResult.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			nextStateSqrtRatioX96: amountInResult.SqrtRatioX96,
			nextStateLiquidity:    amountInResult.Liquidity,
			nextStateTickCurrent:  amountInResult.CurrentTick,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenAmountIn.Token), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	zeroForOne := strings.EqualFold(tokenOut, hexutil.Encode(p.V3Pool.Token1.Address[:]))
	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	var priceLimit v3Utils.Uint160
	if err := p.getSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
	}

	amountOutResult, err := p.V3Pool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
	} else if amountOutResult.ReturnedAmount.Sign() <= 0 {
		return nil, errors.New("amountOut is 0")
	}

	remainingTokenAmountIn := &pool.TokenAmount{Token: tokenAmountIn.Token}
	if amountOutResult.RemainingAmountIn != nil {
		remainingTokenAmountIn.Amount = amountOutResult.RemainingAmountIn.ToBig()
	} else {
		remainingTokenAmountIn.Amount = bignumber.ZeroBI
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOutResult.ReturnedAmount.ToBig(),
		},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Fee: &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		},
		Gas: p.gas.BaseGas + p.gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			nextStateSqrtRatioX96: amountOutResult.SqrtRatioX96,
			nextStateLiquidity:    amountOutResult.Liquidity,
			nextStateTickCurrent:  amountOutResult.CurrentTick,
		},
	}, nil
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for UniV3 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = si.nextStateSqrtRatioX96
	p.V3Pool.Liquidity = si.nextStateLiquidity
	p.V3Pool.TickCurrent = si.nextStateTickCurrent
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) interface{} {
	zeroForOne := strings.EqualFold(tokenIn, hexutil.Encode(p.V3Pool.Token0.Address[:]))
	var priceLimit v3Utils.Uint160
	_ = p.getSqrtPriceLimit(zeroForOne, &priceLimit)
	return PoolMeta{
		BlockNumber: p.Pool.Info.BlockNumber,
		PriceLimit:  bignumber.CapPriceLimit(priceLimit.ToBig()),
	}
}
