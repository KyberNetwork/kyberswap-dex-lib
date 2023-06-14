package elastic

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/elastic-go-sdk/v2/constants"
	elasticEntities "github.com/KyberNetwork/elastic-go-sdk/v2/entities"
	elasticUtils "github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

var (
	ErrTickNil           = errors.New("tick is nil")
	ErrElasticTicksEmpty = errors.New("elastic ticks empty")
)

type Pool struct {
	pool.Pool
	elasticPool *elasticEntities.Pool
	gas         Gas
	tickMin     int
	tickMax     int
}

func NewPool(entityPool entity.Pool, chainID valueobject.ChainID) (*Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address), uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, entityPool.Tokens[0].Name)
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address), uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, entityPool.Tokens[1].Name)

	swapFeeFl := new(big.Float).Mul(big.NewFloat(entityPool.SwapFee), BoneFloat)
	swapFee, _ := swapFeeFl.Int(nil)
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = utils.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = utils.NewBig10(entityPool.Reserves[1])
	}

	var elasticTicks []elasticEntities.Tick

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	for _, t := range extra.Ticks {
		// LiquidityGross = 0 means that the tick is uninitialized
		if t.LiquidityGross.Cmp(zeroBI) == 0 {
			continue
		}

		elasticTicks = append(elasticTicks, elasticEntities.Tick{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}

	// if the tick list is empty, the pool should be ignored
	if len(elasticTicks) == 0 {
		return nil, ErrElasticTicksEmpty
	}

	ticks, err := elasticEntities.NewTickListDataProvider(elasticTicks, constants.TickSpacings[constants.FeeAmount(entityPool.SwapFee)])
	if err != nil {
		return nil, err
	}

	elasticPool, err := elasticEntities.NewPool(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
		extra.SqrtPriceX96,
		extra.Liquidity,
		extra.ReinvestL,
		int(extra.Tick.Int64()),
		ticks,
	)
	if err != nil {
		return nil, err
	}

	var tickMin, tickMax int
	if len(elasticTicks) == 0 {
		tickMin = elasticPool.CurrentTick
		tickMax = elasticPool.CurrentTick
	} else {
		tickMin = elasticTicks[0].Index
		tickMax = elasticTicks[len(elasticTicks)-1].Index
	}

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

	return &Pool{
		Pool:        pool.Pool{Info: info},
		elasticPool: elasticPool,
		gas:         DefaultGas,
		tickMin:     tickMin,
		tickMax:     tickMax,
	}, nil
}

/**
 * getSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
 */
func (p *Pool) getSqrtPriceLimit(zeroForOne bool) *big.Int {
	var tickLimit int
	if zeroForOne {
		tickLimit = p.tickMin
	} else {
		tickLimit = p.tickMax
	}

	sqrtPriceX96Limit, err := elasticUtils.GetSqrtRatioAtTick(tickLimit)

	if err != nil {
		return nil
	}

	return sqrtPriceX96Limit
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	var tokenInIndex = p.GetTokenIndex(tokenAmountIn.Token)
	var tokenOutIndex = p.GetTokenIndex(tokenOut)
	var tokenIn *coreEntities.Token
	var zeroForOne bool

	if tokenInIndex >= 0 && tokenOutIndex >= 0 {
		if strings.EqualFold(tokenOut, p.elasticPool.Token0.Address.String()) {
			zeroForOne = false
			tokenIn = p.elasticPool.Token1
		} else {
			tokenIn = p.elasticPool.Token0
			zeroForOne = true
		}
		amountIn := coreEntities.FromRawAmount(tokenIn, tokenAmountIn.Amount)
		amountOut, newPoolState, err := p.elasticPool.GetOutputAmount(amountIn, p.getSqrtPriceLimit(zeroForOne))

		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}

		var totalGas = p.gas.SwapBase

		if amountOut.Quotient().Cmp(constant.Zero) > 0 {
			return &pool.CalcAmountOutResult{
				TokenAmountOut: &pool.TokenAmount{
					Token:  tokenOut,
					Amount: amountOut.Quotient(),
				},
				Fee: &pool.TokenAmount{
					Token:  tokenAmountIn.Token,
					Amount: nil,
				},
				Gas: totalGas,
				SwapInfo: KSElasticSwapInfo{
					nextStateSqrtP:              new(big.Int).Set(newPoolState.SqrtP),
					nextStateBaseL:              new(big.Int).Set(newPoolState.BaseL),
					nextStateReinvestL:          new(big.Int).Set(newPoolState.ReinvestL),
					nextStateCurrentTick:        newPoolState.CurrentTick,
					nextStateNearestCurrentTick: newPoolState.NearestCurrentTick,
				},
			}, nil
		}

		return &pool.CalcAmountOutResult{}, errors.New("amountOut is 0")
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(KSElasticSwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for ProMM pool, wrong swapInfo type")
		return
	}

	p.elasticPool.SqrtP = si.nextStateSqrtP
	p.elasticPool.BaseL = si.nextStateBaseL
	p.elasticPool.ReinvestL = si.nextStateReinvestL
	p.elasticPool.CurrentTick = si.nextStateCurrentTick
	p.elasticPool.NearestCurrentTick = si.nextStateNearestCurrentTick
}

func (p *Pool) GetLpToken() string {
	return ""
}

func (p *Pool) CanSwapTo(address string) []string {
	var ret = make([]string, 0)
	var tokenIndex = p.GetTokenIndex(address)
	if tokenIndex < 0 {
		return ret
	}
	for i := 0; i < len(p.Info.Tokens); i += 1 {
		if i != tokenIndex {
			ret = append(ret, p.Info.Tokens[i])
		}
	}
	return ret
}

// GetMidPrice This function is not used
func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	if strings.EqualFold(tokenOut, p.elasticPool.Token0.Address.String()) {
		return p.elasticPool.Token0Price().Quotient()
	} else {
		return p.elasticPool.Token1Price().Quotient()
	}
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	if strings.EqualFold(tokenOut, p.elasticPool.Token0.Address.String()) {
		return p.elasticPool.Token0Price().Quotient()
	} else {
		return p.elasticPool.Token1Price().Quotient()
	}
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
