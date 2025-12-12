package dexv2

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
)

type PoolSimulator struct {
	V3Pool *UniV3FluidV2Pool
	pool.Pool
	Gas     Gas
	tickMin int
	tickMax int

	poolAccountingFlag bool
	tokenReserves      [2]*big.Int

	extra       Extra
	staticExtra StaticExtra
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	extraTickU256 := uniswapv3.ExtraTickU256{
		Liquidity:    new(uint256.Int),
		SqrtPriceX96: new(uint256.Int),
		TickSpacing:  uint64(staticExtra.TickSpacing),
	}
	extraTickU256.Liquidity.SetFromBig(extra.Liquidity)
	extraTickU256.SqrtPriceX96.SetFromBig(extra.SqrtPriceX96)
	currentTick := int(extra.Tick.Int64())
	extraTickU256.Tick = &currentTick

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, "")
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, "")

	v3Ticks := make([]v3Entities.Tick, 0, len(extra.Ticks))

	// Ticks are sorted from the pool service, so we don't have to do it again here
	// Purpose: to improve the latency
	for _, t := range extra.Ticks {
		liquidityGross := new(uint256.Int)
		liquidityGross.SetFromBig(t.LiquidityGross)
		// LiquidityGross = 0 means that the tick is uninitialized
		if liquidityGross.IsZero() {
			continue
		}
		liquidityNet := new(int256.Int)
		liquidityNet.SetFromBig(t.LiquidityNet)

		v3Ticks = append(v3Ticks, v3Entities.Tick{
			Index:          t.Index,
			LiquidityGross: liquidityGross,
			LiquidityNet:   liquidityNet,
		})
	}

	if len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	entityPool.SwapFee = float64(staticExtra.Fee)

	ticks, err := v3Entities.NewTickListDataProvider(v3Ticks, int(extraTickU256.TickSpacing))
	if err != nil {
		return nil, err
	}

	v3Pool, err := NewUniV3FluidV2Pool(
		token0,
		token1,
		constants.FeeAmount(entityPool.SwapFee),
		extraTickU256.SqrtPriceX96,
		extraTickU256.Liquidity,
		*extraTickU256.Tick,
		ticks,
		staticExtra.TickSpacing,
		extra.DexVariables2,
	)
	if err != nil {
		return nil, err
	}

	tickMin, tickMax := MIN_TICK, MAX_TICK
	if len(v3Ticks) > 0 {
		tickMin = v3Ticks[0].Index
		tickMax = v3Ticks[len(v3Ticks)-1].Index
	}

	var poolAccounting big.Int
	poolAccounting.Set(extra.DexVariables2).
		Rsh(&poolAccounting, BITS_DEX_V2_VARIABLES2_POOL_ACCOUNTING_FLAG).
		And(&poolAccounting, bignumber.One)

	token0Reserves, token1Reserves := extractTokenReserves(extra.TokenReserves)

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	var info = pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  swapFee,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	simulator := &PoolSimulator{
		Pool:    pool.Pool{Info: info},
		V3Pool:  v3Pool,
		Gas:     defaultGas,
		tickMin: tickMin,
		tickMax: tickMax,

		poolAccountingFlag: poolAccounting.Cmp(bignumber.ZeroBI) == 0,
		tokenReserves:      [2]*big.Int{token0Reserves, token1Reserves},
		extra:              extra,
		staticExtra:        staticExtra,
	}

	return simulator, nil
}

func (p *PoolSimulator) GetSqrtPriceLimit(zeroForOne bool, result *v3Utils.Uint160) error {
	tickLimit := lo.Ternary(zeroForOne, p.tickMin, p.tickMax)
	if err := v3Utils.GetSqrtRatioAtTickV2(tickLimit, result); err != nil {
		return err
	}
	lo.Ternary(zeroForOne, result.AddUint64, result.SubUint64)(result, 1)
	return nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if p.staticExtra.Controller != "" {
		return nil, ErrUnsupportedController
	}

	tokenIn, tokenOut := param.TokenAmountIn.Token, param.TokenOut
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	amountInBI := param.TokenAmountIn.Amount
	if err := _verifyAmountLimits(amountInBI); err != nil {
		return nil, err
	}

	zeroForOne := tokenInIndex == 0

	// Adjust amountIn
	c, err := _calculateVars(p.extra.DexVariables2, p.extra.Token0ExchangePricesAndConfig, p.extra.Token1ExchangePricesAndConfig)
	if err != nil {
		return nil, err
	}

	var amountInRawAdjusted, tmp big.Int
	if zeroForOne {
		amountInRawAdjusted = *amountToAdjusted(amountInBI, c.Token0NumeratorPrecision, c.Token0DenominatorPrecision, c.Token0SupplyExchangePrice)
	} else {
		amountInRawAdjusted = *amountToAdjusted(amountInBI, c.Token1NumeratorPrecision, c.Token1DenominatorPrecision, c.Token1SupplyExchangePrice)
	}

	if err := _verifyAdjustedAmountLimits(&amountInRawAdjusted); err != nil {
		return nil, err
	}

	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(&amountInRawAdjusted); overflow {
		return nil, ErrOverflow
	}

	var priceLimit v3Utils.Uint160
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetSqrtPriceLimit, err: %+v", err)
	}
	amountOutResult, err := p.V3Pool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, err
	}

	err = _verifySqrtPriceX96ChangeLimits(p.extra.SqrtPriceX96, amountOutResult.SqrtRatioX96.ToBig())
	if err != nil {
		return nil, err
	}

	amountOutRawAdjusted := amountOutResult.ReturnedAmount.ToBig()

	if p.poolAccountingFlag {
		if tmp.Add(p.tokenReserves[tokenInIndex], amountInBI).Cmp(X128) > 0 {
			return nil, ErrTokenReservesOverflow
		}
		if tmp.Sub(p.tokenReserves[tokenOutIndex], amountOutRawAdjusted).Sign() < 0 {
			return nil, ErrTokenReservesUnderflow
		}
	}

	if amountOutResult.RemainingAmountIn != nil && amountOutResult.RemainingAmountIn.Sign() > 0 {
		return nil, ErrNextTickOutOfBounds
	}

	// Adjust amountOut
	var amountOut big.Int
	if err := _verifyAdjustedAmountLimits(amountOutRawAdjusted); err != nil {
		return nil, err
	}

	if zeroForOne {
		amountOut = *adjustedToAmount(amountOutRawAdjusted, c.Token1NumeratorPrecision, c.Token1DenominatorPrecision, c.Token1SupplyExchangePrice)
	} else {
		amountOut = *adjustedToAmount(amountOutRawAdjusted, c.Token0NumeratorPrecision, c.Token0DenominatorPrecision, c.Token0SupplyExchangePrice)
	}

	amountOut.Sub(&amountOut, bignumber.One)
	if err := _verifyAmountLimits(&amountOut); err != nil {
		return nil, err
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: &amountOut,
		},
		Fee: &pool.TokenAmount{
			Token: tokenIn,
		},
		Gas: p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(amountOutResult.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			RemainingAmountIn:     amountOutResult.RemainingAmountIn,
			NextStateSqrtRatioX96: amountOutResult.SqrtRatioX96,
			NextStateTickCurrent:  amountOutResult.CurrentTick,
			nextStateLiquidity:    amountOutResult.Liquidity,
			amountInRawAdjusted:   &amountInRawAdjusted,
			amountOutRawAdjusted:  amountOutRawAdjusted,
		},
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	v3Pool := *p.V3Pool
	cloned.V3Pool = &v3Pool
	cloned.tokenReserves = [2]*big.Int{
		new(big.Int).Set(p.tokenReserves[0]),
		new(big.Int).Set(p.tokenReserves[1]),
	}
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for FluidDexV2 pool, wrong swapInfo type")
		return
	}
	p.V3Pool.SqrtRatioX96 = si.NextStateSqrtRatioX96
	p.V3Pool.TickCurrent = si.NextStateTickCurrent
	p.V3Pool.Liquidity = si.nextStateLiquidity

	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)

	p.tokenReserves[tokenInIndex].Add(p.tokenReserves[tokenInIndex], si.amountInRawAdjusted)
	p.tokenReserves[tokenOutIndex].Sub(p.tokenReserves[tokenOutIndex], si.amountOutRawAdjusted)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, tokenOut string) any {
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)

	return PoolMeta{
		Dex:         p.staticExtra.Dex,
		ZeroForOne:  tokenInIndex == 0,
		DexType:     p.staticExtra.DexType,
		Fee:         p.staticExtra.Fee,
		TickSpacing: p.staticExtra.TickSpacing,
		Controller:  p.staticExtra.Controller,

		IsNativeIn:  p.staticExtra.IsNative[tokenInIndex],
		IsNativeOut: p.staticExtra.IsNative[tokenOutIndex],
	}
}
