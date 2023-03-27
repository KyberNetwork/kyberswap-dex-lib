package univ3

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/daoleno/uniswapv3-sdk/constants"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/utils"

	v3Entities "github.com/daoleno/uniswapv3-sdk/entities"
	v3Utils "github.com/daoleno/uniswapv3-sdk/utils"
)

var (
	ErrV3TicksEmpty                  = errors.New("v3Ticks empty")
	ErrNewTickListDataProviderFailed = errors.New("new tick list data provider failed")
)

type Pool struct {
	pool.Pool
	v3Pool    *v3Entities.Pool
	nextState NextState
	gas       Gas
	tickMin   int
	tickMax   int
}

func NewPool(entityPool entity.Pool, chainID int) (*Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
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

	var v3Ticks []v3Entities.Tick

	for _, t := range extra.Ticks {
		v3Ticks = append(v3Ticks, v3Entities.Tick{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}

	// Sort the ticks because function NewTickListDataProvider needs
	sort.SliceStable(v3Ticks, func(i, j int) bool {
		return v3Ticks[i].Index < v3Ticks[j].Index
	})

	// if the tick list is empty, the pool should be ignored
	if len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, constants.TickSpacings[constants.FeeAmount(entityPool.SwapFee)])
	if err != nil {
		return nil, ErrNewTickListDataProviderFailed
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
		Pool:      pool.Pool{Info: info},
		v3Pool:    v3Pool,
		nextState: NextState{},
		gas:       DefaultGas,
		tickMin:   tickMin,
		tickMax:   tickMax,
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

	sqrtPriceX96Limit, err := v3Utils.GetSqrtRatioAtTick(tickLimit)

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
		if strings.EqualFold(tokenOut, p.v3Pool.Token0.Address.String()) {
			zeroForOne = false
			tokenIn = p.v3Pool.Token1
		} else {
			tokenIn = p.v3Pool.Token0
			zeroForOne = true
		}
		amountIn := coreEntities.FromRawAmount(tokenIn, tokenAmountIn.Amount)
		amountOut, newPoolState, err := p.v3Pool.GetOutputAmount(amountIn, p.getSqrtPriceLimit(zeroForOne))

		if err != nil {
			return &pool.CalcAmountOutResult{}, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
		}

		var totalGas = p.gas.Swap

		p.nextState.SqrtRatioX96 = newPoolState.SqrtRatioX96
		p.nextState.Liquidity = newPoolState.Liquidity
		p.nextState.TickCurrent = newPoolState.TickCurrent

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
			}, nil
		}

		return &pool.CalcAmountOutResult{}, errors.New("amountOut is 0")
	}

	return &pool.CalcAmountOutResult{}, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
}

func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	p.v3Pool.SqrtRatioX96 = p.nextState.SqrtRatioX96
	p.v3Pool.Liquidity = p.nextState.Liquidity
	p.v3Pool.TickCurrent = p.nextState.TickCurrent
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
	if strings.EqualFold(tokenOut, p.v3Pool.Token0.Address.String()) {
		return p.v3Pool.Token0Price().Quotient()
	} else {
		return p.v3Pool.Token1Price().Quotient()
	}
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	if strings.EqualFold(tokenOut, p.v3Pool.Token0.Address.String()) {
		return p.v3Pool.Token0Price().Quotient()
	} else {
		return p.v3Pool.Token1Price().Quotient()
	}
}

func (p *Pool) GetMetaInfo(tokenIn string, tokenOut string) interface{} {
	return nil
}
