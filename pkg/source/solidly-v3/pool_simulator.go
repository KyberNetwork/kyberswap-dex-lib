package solidlyv3

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/logger"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/daoleno/uniswapv3-sdk/constants"
	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrTickNil      = errors.New("tick is nil")
	ErrV3TicksEmpty = errors.New("v3Ticks empty")
)

type PoolSimulator struct {
	V3Pool *v3Entities.Pool
	pool.Pool
	gas     Gas
	tickMin int
	tickMax int
}

var _ = pool.RegisterFactory1(DexTypeSolidlyV3, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address), uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, "")
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address), uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, "")

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = NewBig10(entityPool.Reserves[1])
	}

	var v3Ticks []v3Entities.Tick

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	for _, t := range extra.Ticks {
		// LiquidityGross = 0 means that the tick is uninitialized
		if t.LiquidityGross.Cmp(zeroBI) == 0 {
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

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, int(extra.TickSpacing))
	if err != nil {
		return nil, err
	}

	v3Pool, err := v3Entities.NewPool(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
		extra.SqrtPriceX96,
		extra.Liquidity,
		int(extra.Tick.Int64()),
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
		gas:     defaultGas,
		tickMin: tickMin,
		tickMax: tickMax,
	}, nil
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *PoolSimulator) getSqrtPriceLimit(zeroForOne bool) *big.Int {
	var tickLimit int
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	sqrtPriceX96Limit, err := v3Utils.GetSqrtRatioAtTick(tickLimit)

	if err != nil {
		return nil
	}

	return sqrtPriceX96Limit
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var tokenIn *coreEntities.Token
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(tokenOut, hexutil.Encode(p.V3Pool.Token0.Address[:])) {
			zeroForOne = false
			tokenIn = p.V3Pool.Token1
		} else {
			tokenIn = p.V3Pool.Token0
			zeroForOne = true
		}
		amountIn := coreEntities.FromRawAmount(tokenIn, tokenAmountIn.Amount)
		amountOutResult, err := p.V3Pool.GetOutputAmount(amountIn, p.getSqrtPriceLimit(zeroForOne))

		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}
		amountOut := amountOutResult.ReturnedAmount
		newPoolState := amountOutResult.NewPoolState

		var remainingTokenAmountIn = &pool.TokenAmount{
			Token: tokenAmountIn.Token,
		}
		if amountOutResult.RemainingAmountIn != nil {
			remainingTokenAmountIn.Amount = amountOutResult.RemainingAmountIn.Quotient()
		} else {
			remainingTokenAmountIn.Amount = big.NewInt(0)
		}

		var totalGas = p.gas.BaseGas + p.gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops)

		if amountOut.Quotient().Cmp(zeroBI) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.Quotient(),
				},
				RemainingTokenAmountIn: remainingTokenAmountIn,
				Fee: &pool.TokenAmount{
					Token:  tokenAmountIn.Token,
					Amount: nil,
				},
				Gas: totalGas,
				SwapInfo: SolidlyV3SwapInfo{
					nextStateSqrtRatioX96: new(big.Int).Set(newPoolState.SqrtRatioX96),
					nextStateLiquidity:    new(big.Int).Set(newPoolState.Liquidity),
					nextStateTickCurrent:  newPoolState.TickCurrent,
				},
			}, nil
		}

		return nil, errors.New("amountOut is 0")
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenAmountOut := param.TokenAmountOut
	var tokenInIndex = p.GetTokenIndex(param.TokenIn)
	var tokenOutIndex = p.GetTokenIndex(param.TokenAmountOut.Token)
	var tokenOut *coreEntities.Token
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(param.TokenAmountOut.Token, hexutil.Encode(p.V3Pool.Token0.Address[:])) {
			zeroForOne = false
			tokenOut = p.V3Pool.Token0
		} else {
			tokenOut = p.V3Pool.Token1
			zeroForOne = true
		}
		amountOut := coreEntities.FromRawAmount(tokenOut, param.TokenAmountOut.Amount)
		getInputAmountResult, err := p.V3Pool.GetInputAmount(amountOut, p.getSqrtPriceLimit(zeroForOne))

		if err != nil {
			return nil, fmt.Errorf("can not GetInputAmount, err: %+v", err)
		}

		amountIn := getInputAmountResult.ReturnedAmount
		newPoolState := getInputAmountResult.NewPoolState

		var remainingTokenAmountOut = &pool.TokenAmount{
			Token: tokenAmountOut.Token,
		}
		if getInputAmountResult.RemainingAmountOut != nil {
			remainingTokenAmountOut.Amount = getInputAmountResult.RemainingAmountOut.Quotient()
		} else {
			remainingTokenAmountOut.Amount = big.NewInt(0)
		}

		var totalGas = p.gas.BaseGas + p.gas.CrossInitTickGas*int64(getInputAmountResult.CrossInitTickLoops)

		amountInBI := amountIn.Quotient()
		if amountInBI.Cmp(zeroBI) > 0 {
			return &pool.CalcAmountInResult{
				TokenAmountIn: &pool.TokenAmount{
					Token:  param.TokenIn,
					Amount: amountInBI,
				},
				RemainingTokenAmountOut: remainingTokenAmountOut,
				Fee: &pool.TokenAmount{
					Token:  param.TokenIn,
					Amount: nil,
				},
				Gas: totalGas,
				SwapInfo: SolidlyV3SwapInfo{
					nextStateSqrtRatioX96: new(big.Int).Set(newPoolState.SqrtRatioX96),
					nextStateLiquidity:    new(big.Int).Set(newPoolState.Liquidity),
					nextStateTickCurrent:  newPoolState.TickCurrent,
				},
			}, nil
		}

		return nil, errors.New("amountIn is 0")
	}

	return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SolidlyV3SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for SolidlyV3 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = si.nextStateSqrtRatioX96
	p.V3Pool.Liquidity = si.nextStateLiquidity
	p.V3Pool.TickCurrent = si.nextStateTickCurrent
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	zeroForOne := strings.EqualFold(tokenIn, hexutil.Encode(p.V3Pool.Token0.Address[:]))
	return PoolMeta{
		PriceLimit: bignumber.CapPriceLimit(p.getSqrtPriceLimit(zeroForOne)),
	}
}
