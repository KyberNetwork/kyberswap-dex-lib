package uniswapv3

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
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *v3Entities.Pool
	pool.Pool
	Gas     Gas
	tickMin int
	tickMax int
}

var _ = pool.RegisterFactory1(DexTypeUniswapV3, NewPoolSimulator)

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

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = NewBig10(entityPool.Reserves[1])
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
	// For some pools that not yet initialized tickSpacing in their extra,
	// we will get the tickSpacing through feeTier mapping.
	if tickSpacing == 0 {
		feeTier := constants.FeeAmount(entityPool.SwapFee)
		if _, ok := constants.TickSpacings[feeTier]; !ok {
			return nil, ErrInvalidFeeTier
		}
		tickSpacing = constants.TickSpacings[feeTier]
	}
	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, tickSpacing)
	if err != nil {
		return nil, err
	}

	v3Pool, err := v3Entities.NewPoolV2(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
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
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  swapFee,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool:    pool.Pool{Info: info},
		V3Pool:  v3Pool,
		Gas:     defaultGas,
		tickMin: tickMin,
		tickMax: tickMax,
	}, nil
}

// GetSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
func (p *PoolSimulator) GetSqrtPriceLimit(zeroForOne bool, result *v3Utils.Uint160) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := v3Utils.GetSqrtRatioAtTickV2(tickLimit, result); err != nil {
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
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	zeroForOne := tokenInIndex == 0
	amountOut := coreEntities.FromRawAmount(lo.Ternary(zeroForOne, p.V3Pool.Token1, p.V3Pool.Token0),
		tokenAmountOut.Amount)
	var priceLimit v3Utils.Uint160
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
	}
	amountIn, newPoolState, err := p.V3Pool.GetInputAmount(amountOut, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
	}

	amountInBI := amountIn.Quotient()
	if amountInBI.Sign() <= 0 {
		return nil, errors.New("amountIn is 0")
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInBI,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.Gas.BaseGas, // TODO: update GetInputAmount to return crossed tick if we ever need this
		SwapInfo: SwapInfo{
			NextStateSqrtRatioX96: newPoolState.SqrtRatioX96,
			nextStateLiquidity:    newPoolState.Liquidity,
			nextStateTickCurrent:  newPoolState.TickCurrent,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	zeroForOne := tokenInIndex == 0
	var priceLimit v3Utils.Uint160
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
	}
	amountOutResult, err := p.V3Pool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
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
	amountOut := amountOutResult.ReturnedAmount
	if amountOut.Sign() <= 0 {
		return nil, errors.New("amountOut is 0")
	}
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
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
			nextStateTickCurrent:  amountOutResult.CurrentTick,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Pool := *p.V3Pool
	v3Pool.SqrtRatioX96 = v3Pool.SqrtRatioX96.Clone()
	v3Pool.Liquidity = v3Pool.Liquidity.Clone()
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
	p.V3Pool.TickCurrent = si.nextStateTickCurrent
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0], &priceLimit)
	return PoolMeta{
		SwapFee:    uint32(p.Pool.Info.SwapFee.Int64()),
		PriceLimit: &priceLimit,
	}
}
