package pancakev3

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
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrOverflow       = errors.New("bigInt overflow int/uint256")
	ErrInvalidFeeTier = errors.New("invalid feeTier")
)

type PoolSimulator struct {
	V3Pool *v3Entities.Pool
	pool.Pool
	gas     Gas
	tickMin int
	tickMax int
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraTickU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address), uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, entityPool.Tokens[0].Name)
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address), uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, entityPool.Tokens[1].Name)

	swapFeeFl := big.NewFloat(entityPool.SwapFee)
	swapFee, _ := swapFeeFl.Int(nil)
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
		if tickSpacing = TickSpacings[feeTier]; tickSpacing == 0 {
			return nil, ErrInvalidFeeTier
		}
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
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		SwapFee:    swapFee,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
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
	var tickLimit int
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	err := v3Utils.GetSqrtRatioAtTickV2(tickLimit, result)

	if err != nil {
		return err
	}

	return nil
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	var tokenInIndex = p.GetTokenIndex(param.TokenIn)
	var tokenOutIndex = p.GetTokenIndex(param.TokenAmountOut.Token)
	var tokenOut *coreEntities.Token
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(param.TokenAmountOut.Token, p.V3Pool.Token0.Address.String()) {
			zeroForOne = false
			tokenOut = p.V3Pool.Token0
		} else {
			tokenOut = p.V3Pool.Token1
			zeroForOne = true
		}

		amountOut := coreEntities.FromRawAmount(tokenOut, param.TokenAmountOut.Amount)
		var priceLimit v3Utils.Uint160
		err := p.getSqrtPriceLimit(zeroForOne, &priceLimit)
		if err != nil {
			return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
		}
		amountIn, newPoolState, err := p.V3Pool.GetInputAmount(amountOut, &priceLimit)

		if err != nil {
			return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
		}

		totalGas := p.gas.BaseGas // TODO: update GetInputAmount to return crossed tick if we ever need this

		amountInBI := amountIn.Quotient()
		if amountInBI.Cmp(zeroBI) > 0 {
			return &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:  param.TokenIn,
					Amount: amountInBI,
				},
				Fee: &pool.TokenAmount{
					Token:  param.TokenIn,
					Amount: nil,
				},
				Gas: totalGas,
				SwapInfo: SwapInfo{
					nextStateSqrtRatioX96: new(uint256.Int).Set(newPoolState.SqrtRatioX96),
					nextStateLiquidity:    new(uint256.Int).Set(newPoolState.Liquidity),
					nextStateTickCurrent:  newPoolState.TickCurrent,
				},
			}, nil
		}

		return nil, errors.New("amountIn is 0")
	}

	return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(tokenOut, p.V3Pool.Token0.Address.String()) {
			zeroForOne = false
		} else {
			zeroForOne = true
		}
		var amountIn v3Utils.Int256
		overflow := amountIn.SetFromBig(tokenAmountIn.Amount)
		if overflow {
			return nil, ErrOverflow
		}
		var priceLimit v3Utils.Uint160
		err := p.getSqrtPriceLimit(zeroForOne, &priceLimit)
		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}
		amountOutResult, err := p.V3Pool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)

		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}

		amountOut := amountOutResult.ReturnedAmount

		var remainingTokenAmountIn = &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		}
		if amountOutResult.RemainingAmountIn != nil {
			remainingTokenAmountIn.Amount = amountOutResult.RemainingAmountIn.ToBig()
		} else {
			remainingTokenAmountIn.Amount = big.NewInt(0)
		}

		var totalGas = p.gas.BaseGas + p.gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops)

		//p.nextState.SqrtRatioX96 = newPoolState.SqrtRatioX96
		//p.nextState.Liquidity = newPoolState.Liquidity
		//p.nextState.TickCurrent = newPoolState.TickCurrent

		if amountOut.Sign() > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.ToBig(),
				},
				RemainingTokenAmountIn: remainingTokenAmountIn,
				Fee: &pool.TokenAmount{
					Token:  tokenAmountIn.Token,
					Amount: nil,
				},
				Gas: totalGas,
				SwapInfo: SwapInfo{
					nextStateSqrtRatioX96: new(uint256.Int).Set(amountOutResult.SqrtRatioX96),
					nextStateLiquidity:    new(uint256.Int).Set(amountOutResult.Liquidity),
					nextStateTickCurrent:  amountOutResult.CurrentTick,
				},
			}, nil
		}

		return &pool.CalcAmountOutResult{}, errors.New("amountOut is 0")
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	v3Pool := *p.V3Pool
	v3Pool.SqrtRatioX96 = v3Pool.SqrtRatioX96.Clone()
	v3Pool.Liquidity = v3Pool.Liquidity.Clone()
	return &PoolSimulator{
		V3Pool:  &v3Pool,
		Pool:    p.Pool,
		gas:     p.gas,
		tickMin: p.tickMin,
		tickMax: p.tickMax,
	}
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
	zeroForOne := strings.EqualFold(tokenIn, p.V3Pool.Token0.Address.String())
	var priceLimit v3Utils.Uint160
	_ = p.getSqrtPriceLimit(zeroForOne, &priceLimit)

	return PoolMeta{
		BlockNumber: p.Pool.Info.BlockNumber,
		PriceLimit:  bignumber.CapPriceLimit(priceLimit.ToBig()),
	}
}
