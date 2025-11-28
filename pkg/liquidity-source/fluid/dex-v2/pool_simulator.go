package dexv2

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/logger"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	v3Entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	v3Utils "github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *v3Entities.Pool
	pool.Pool
	Gas             Gas
	tickMin         int
	tickMax         int
	allowEmptyTicks bool

	Extra Extra
}

var _ = pool.RegisterFactory1(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	extraTickU256 := ExtraTickU256{
		Liquidity:    new(uint256.Int),
		SqrtPriceX96: new(uint256.Int),
		TickSpacing:  uint64(extra.TickSpacing),
	}
	extraTickU256.Liquidity.SetFromBig(extra.DexVariables2.ActiveLiquidity)
	extraTickU256.SqrtPriceX96.SetFromBig(extra.DexVariables.CurrentSqrtPriceX96)
	currentTick := int(extra.DexVariables.CurrentTick.Int64())
	extraTickU256.Tick = &currentTick

	ticks := lo.Map(extra.Ticks, func(t Tick, _ int) TickU256 {
		liquidityGross := new(uint256.Int)
		liquidityGross.SetFromBig(t.LiquidityGross)
		liquidityNet := new(int256.Int)
		liquidityNet.SetFromBig(t.LiquidityNet)
		return TickU256{
			Index:          t.Index,
			LiquidityGross: liquidityGross,
			LiquidityNet:   liquidityNet,
		}
	})
	extraTickU256.Ticks = ticks
	entityPool.SwapFee = float64(extra.Fee)

	simulator, err := NewPoolSimulatorWithExtra(entityPool, chainID, &extraTickU256, false)
	if err != nil {
		return nil, err
	}

	simulator.Extra = extra
	return simulator, nil
}

func NewPoolSimulatorWithExtra(entityPool entity.Pool, chainID valueobject.ChainID,
	extra *ExtraTickU256, allowEmptyTicks bool) (*PoolSimulator, error) {
	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	token0 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[0].Address),
		uint(entityPool.Tokens[0].Decimals), entityPool.Tokens[0].Symbol, "")
	token1 := coreEntities.NewToken(uint(chainID), common.HexToAddress(entityPool.Tokens[1].Address),
		uint(entityPool.Tokens[1].Decimals), entityPool.Tokens[1].Symbol, "")

	swapFee := big.NewInt(int64(entityPool.SwapFee))
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
	// for some uniswap-v4 hooks, we want to bypass this check due to some hooks has no ticks
	if !allowEmptyTicks && len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	tickSpacing := int(extra.TickSpacing)
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

	tickMin, tickMax := v3Utils.MinTick, v3Utils.MaxTick
	if len(v3Ticks) > 0 {
		tickMin = v3Ticks[0].Index
		tickMax = v3Ticks[len(v3Ticks)-1].Index
	}

	var info = pool.PoolInfo{
		Address:  strings.ToLower(entityPool.Address),
		SwapFee:  swapFee,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
		Reserves: reserves,
	}

	return &PoolSimulator{
		Pool:            pool.Pool{Info: info},
		V3Pool:          v3Pool,
		Gas:             defaultGas,
		tickMin:         tickMin,
		tickMax:         tickMax,
		allowEmptyTicks: allowEmptyTicks,
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

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	// TODO: Support _verifyAmountLimits check
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, fmt.Errorf("tokenInIndex %v or tokenOutIndex %v is not correct", tokenInIndex, tokenOutIndex)
	}

	zeroForOne := tokenInIndex == 0
	var priceLimit v3Utils.Uint160
	if err := p.GetSqrtPriceLimit(zeroForOne, &priceLimit); err != nil {
		return nil, fmt.Errorf("can not GetSqrtPriceLimit, err: %+v", err)
	}

	// Adjust amountIn
	c, err := p._calculateVars()
	if err != nil {
		return nil, err
	}

	var amountInRawAdjusted big.Int
	if zeroForOne {
		amountInRawAdjusted.Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					param.TokenAmountIn.Amount,
					LC_EXCHANGE_PRICES_PRECISION,
				),
				c.Token0NumeratorPrecision,
			),
			new(big.Int).Mul(
				c.Token0SupplyExchangePrice,
				c.Token0DenominatorPrecision,
			),
		)
	} else {
		amountInRawAdjusted.Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					param.TokenAmountIn.Amount,
					LC_EXCHANGE_PRICES_PRECISION,
				),
				c.Token1NumeratorPrecision,
			),
			new(big.Int).Mul(
				c.Token1SupplyExchangePrice,
				c.Token1DenominatorPrecision,
			),
		)
	}
	// TODO: _verifyAdjustedAmountLimits

	var amountIn v3Utils.Int256
	if overflow := amountIn.SetFromBig(&amountInRawAdjusted); overflow {
		return nil, ErrOverflow
	}

	amountOutResult, err := p.V3Pool.GetOutputAmountV2(&amountIn, zeroForOne, &priceLimit)
	if err != nil {
		return nil, fmt.Errorf("can not GetOutputAmount, err: %+v", err)
	}

	// Adjust amountOut
	var amountOut big.Int
	amountOutRawAdjusted := amountOutResult.ReturnedAmount.ToBig()
	if zeroForOne {
		amountOut.Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					amountOutRawAdjusted,
					c.Token1DenominatorPrecision,
				),
				c.Token1SupplyExchangePrice,
			),
			new(big.Int).Mul(
				c.Token1NumeratorPrecision,
				LC_EXCHANGE_PRICES_PRECISION,
			),
		)
	} else {
		amountOut.Div(
			new(big.Int).Mul(
				new(big.Int).Mul(
					amountOutRawAdjusted,
					c.Token0DenominatorPrecision,
				),
				c.Token0SupplyExchangePrice,
			),
			new(big.Int).Mul(
				c.Token0NumeratorPrecision,
				LC_EXCHANGE_PRICES_PRECISION,
			),
		)
	}

	amountOut.Sub(&amountOut, bignumber.One)
	// TODO: _verifyAmountLimits

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

	if !p.allowEmptyTicks {
		if amountOut.Sign() <= 0 {
			return nil, errors.New("amountOut is 0")
		}
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  tokenOut,
			Amount: &amountOut,
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
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	var priceLimit v3Utils.Uint160
	_ = p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0], &priceLimit)
	return PoolMeta{
		SwapFee:    uint32(p.Info.SwapFee.Int64()),
		PriceLimit: &priceLimit,
	}
}
