package algebrav1

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
)

var (
	ErrTickNil      = errors.New("tick is nil")
	ErrV3TicksEmpty = errors.New("v3Ticks empty")
)

type PoolSimulator struct {
	pool.Pool
	globalState               GlobalState
	liquidity                 *big.Int
	volumePerLiquidityInBlock *big.Int
	// totalFeeGrowth0Token      *big.Int
	// totalFeeGrowth1Token      *big.Int
	ticks       *v3Entities.TickListDataProvider
	gas         Gas
	tickMin     int
	tickMax     int
	tickSpacing int

	timepoints TimepointStorage
	feeConf    FeeConfiguration
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.GlobalState.Tick == nil {
		return nil, ErrTickNil
	}

	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	// if the tick list is empty, the pool should be ignored
	if len(extra.Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	ticks, err := v3Entities.NewTickListDataProvider(extra.Ticks, int(extra.TickSpacing))
	if err != nil {
		return nil, err
	}
	fmt.Println("---", len(extra.Ticks), extra.Ticks[0])

	tickMin := extra.Ticks[0].Index
	tickMax := extra.Ticks[len(extra.Ticks)-1].Index

	var info = pool.PoolInfo{
		Address:    strings.ToLower(entityPool.Address),
		ReserveUsd: entityPool.ReserveUsd,
		Exchange:   entityPool.Exchange,
		Type:       entityPool.Type,
		Tokens:     tokens,
		Reserves:   reserves,
		Checked:    false,
	}

	return &PoolSimulator{
		Pool:                      pool.Pool{Info: info},
		globalState:               extra.GlobalState,
		liquidity:                 extra.Liquidity,
		volumePerLiquidityInBlock: extra.VolumePerLiquidityInBlock,
		ticks:                     ticks,
		// gas:     defaultGas,
		tickMin:     tickMin,
		tickMax:     tickMax,
		tickSpacing: int(extra.TickSpacing),
		timepoints:  TimepointStorage{data: extra.Timepoints, updates: map[uint16]Timepoint{}},
		feeConf:     extra.FeeConfig,
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

func (p *PoolSimulator) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(tokenOut, p.Info.Tokens[0]) {
			zeroForOne = false
		} else {
			zeroForOne = true
		}

		priceLimit := p.getSqrtPriceLimit(zeroForOne)
		logger.Debugf("price limit %v", priceLimit)
		err, amount0, amount1, stateUpdate := p._calculateSwapAndLock(zeroForOne, tokenAmountIn.Amount, priceLimit)
		var amountOut *big.Int
		if zeroForOne {
			amountOut = new(big.Int).Neg(amount1)
		} else {
			amountOut = new(big.Int).Neg(amount0)
		}

		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}

		// var totalGas = p.gas.Swap
		if amountOut.Cmp(bignumber.ZeroBI) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut,
				},
				Fee: &pool.TokenAmount{
					Token:  tokenAmountIn.Token,
					Amount: nil,
				},
				// Gas: totalGas,
				SwapInfo: *stateUpdate,
			}, nil
		}

		return &pool.CalcAmountOutResult{}, errors.New("amountOut is 0")
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(StateUpdate)
	if !ok {
		logger.Warnf("failed to UpdateBalance for Algebra %v %v pool, wrong swapInfo type", p.Info.Address, p.Info.Exchange)
		return
	}
	p.liquidity = new(big.Int).Set(si.Liquidity)
	p.volumePerLiquidityInBlock = new(big.Int).Set(si.VolumePerLiquidityInBlock)
	p.globalState = si.GlobalState
	p.timepoints.updates = make(map[uint16]Timepoint, len(si.NewTimepoints))
	for i, tp := range si.NewTimepoints {
		p.timepoints.updates[i] = tp
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
