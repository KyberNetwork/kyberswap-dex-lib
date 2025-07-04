package v3

import (
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
	pool.Pool

	V3Pool *v3Entities.Pool
	Gas    Gas

	unlocked bool
	tickMin  int
	tickMax  int
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

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	token0 := coreEntities.NewToken(
		uint(chainID),
		common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals),
		entityPool.Tokens[0].Symbol,
		"",
	)
	token1 := coreEntities.NewToken(
		uint(chainID),
		common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals),
		entityPool.Tokens[1].Symbol,
		"",
	)

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Tokens))
	if len(entityPool.Reserves) >= 2 && len(entityPool.Tokens) >= 2 {
		for i := range entityPool.Tokens {
			tokens[i] = entityPool.Tokens[i].Address
			reserves[i] = bignumber.NewBig(entityPool.Reserves[i])
		}
	}

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	v3Ticks := make([]v3Entities.Tick, 0, len(extra.Ticks))
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

	tickSpacing := int(staticExtra.TickSpacing)
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
		Address:     strings.ToLower(entityPool.Address),
		SwapFee:     swapFee,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Pool:     pool.Pool{Info: info},
		V3Pool:   v3Pool,
		Gas:      defaultGas,
		unlocked: extra.Unlocked,
		tickMin:  tickMin,
		tickMax:  tickMax,
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
	if !p.unlocked {
		return nil, ErrPoolLocked
	}

	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenOut := tokenAmountOut.Token

	tokenInIndex := p.GetTokenIndex(tokenIn)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 {
		return nil, ErrTokenInInvalid
	}
	if tokenOutIndex < 0 {
		return nil, ErrTokenOutInvalid
	}

	totalGas := p.Gas.BaseGas
	// Add unwrap gas cost if tokenIn is not a LP token
	if tokenInIndex < 2 {
		totalGas += WrapGasCost
	}
	// Add unwrap gas cost if tokenOut is not a LP token
	if tokenOutIndex < 2 {
		totalGas += WrapGasCost
	}

	zeroForOne := tokenInIndex%2 == 0
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
		return nil, ErrAmountInZero
	}
	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{
			Token:  tokenIn,
			Amount: amountInBI,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: totalGas,
		SwapInfo: SwapInfo{
			NextStateSqrtRatioX96: newPoolState.SqrtRatioX96,
			nextStateLiquidity:    newPoolState.Liquidity,
			nextStateTickCurrent:  newPoolState.TickCurrent,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if !p.unlocked {
		return nil, ErrPoolLocked
	}

	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex := p.GetTokenIndex(tokenIn)
	tokenOutIndex := p.GetTokenIndex(tokenOut)

	if tokenInIndex < 0 {
		return nil, ErrTokenInInvalid
	}
	if tokenOutIndex < 0 {
		return nil, ErrTokenOutInvalid
	}

	gasCost := p.Gas.BaseGas
	// Add unwrap gas cost if tokenIn is not a LP token
	if tokenInIndex < 2 {
		gasCost += UnwrapGasCost
	}
	// Add wrap gas cost if tokenOut is not a LP token
	if tokenOutIndex < 2 {
		gasCost += WrapGasCost
	}

	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}

	zeroForOne := tokenInIndex%2 == 0
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
		return nil, ErrAmountOutZero
	}

	numTokens := len(p.Info.Tokens)
	lpTokenIn, lpTokenOut := p.Info.Tokens[numTokens-2], p.Info.Tokens[numTokens-1]
	if !zeroForOne {
		lpTokenIn, lpTokenOut = p.Info.Tokens[numTokens-1], p.Info.Tokens[numTokens-2]
	}

	// Add cross tick gas cost
	gasCost += p.Gas.CrossInitTickGas * int64(amountOutResult.CrossInitTickLoops)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: amountOut.ToBig(),
		},
		RemainingTokenAmountIn: remainingTokenAmountIn,
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: gasCost,
		SwapInfo: SwapInfo{
			LpTokenIn:             lpTokenIn,
			LpTokenOut:            lpTokenOut,
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
		logger.Warn("failed to UpdateBalance for native-v3 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = si.NextStateSqrtRatioX96
	p.V3Pool.Liquidity = si.nextStateLiquidity
	p.V3Pool.TickCurrent = si.nextStateTickCurrent
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	idx := p.GetTokenIndex(address)
	if idx < 0 {
		return []string{}
	}

	result := make([]string, 0, len(p.Info.Tokens))
	for i, token := range p.Info.Tokens {
		if i != idx && i%2 != idx%2 {
			result = append(result, token)
		}
	}

	return result
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	var priceLimit v3Utils.Uint160
	zeroForOne := p.GetTokenIndex(tokenIn)%2 == 0
	_ = p.GetSqrtPriceLimit(zeroForOne, &priceLimit)

	return PoolMeta{
		SwapFee:         uint32(p.Pool.Info.SwapFee.Int64()),
		PriceLimit:      &priceLimit,
		BlockNumber:     p.Info.BlockNumber,
		ApprovalAddress: p.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if idx := p.GetTokenIndex(tokenIn); idx >= 0 && idx < 2 {
		return p.Info.Tokens[idx]
	}

	return ""
}
