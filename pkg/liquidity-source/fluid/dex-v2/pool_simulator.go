package dexv2

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool
	v3Simulator uniswapv3.PoolSimulator

	token0Decimals int64
	token1Decimals int64
	extra          Extra
	staticExtra    StaticExtra
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

	ticks := lo.Map(extra.Ticks, func(t Tick, _ int) uniswapv3.TickU256 {
		liquidityGross := new(uint256.Int)
		liquidityGross.SetFromBig(t.LiquidityGross)
		liquidityNet := new(int256.Int)
		liquidityNet.SetFromBig(t.LiquidityNet)
		return uniswapv3.TickU256{
			Index:          t.Index,
			LiquidityGross: liquidityGross,
			LiquidityNet:   liquidityNet,
		}
	})
	extraTickU256.Ticks = ticks
	entityPool.SwapFee = float64(staticExtra.Fee)

	v3Simulator, err := uniswapv3.NewPoolSimulatorWithExtra(entityPool, chainID, &extraTickU256, false)
	if err != nil {
		return nil, err
	}
	v3Simulator.Gas.BaseGas = defaultGas.BaseGas
	v3Simulator.Gas.CrossInitTickGas = defaultGas.CrossInitTickGas

	simulator := &PoolSimulator{
		Pool:           v3Simulator.Pool,
		v3Simulator:    *v3Simulator,
		token0Decimals: int64(entityPool.Tokens[0].Decimals),
		token1Decimals: int64(entityPool.Tokens[1].Decimals),
		extra:          extra,
		staticExtra:    staticExtra,
	}

	return simulator, nil
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

	amountIn := param.TokenAmountIn.Amount
	if err := _verifyAmountLimits(amountIn); err != nil {
		return nil, err
	}

	zeroForOne := tokenInIndex == 0

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
					amountIn,
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

	if err := _verifyAdjustedAmountLimits(&amountInRawAdjusted); err != nil {
		return nil, err
	}

	adjustedParam := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  tokenIn,
			Amount: &amountInRawAdjusted,
		},
		TokenOut: tokenOut,
		Limit:    param.Limit,
	}

	amountOutResult, err := p.v3Simulator.CalcAmountOut(adjustedParam)
	if err != nil {
		return nil, err
	}

	swapInfo := amountOutResult.SwapInfo.(uniswapv3.SwapInfo)
	if (swapInfo.RemainingAmountIn != nil && swapInfo.RemainingAmountIn.Sign() > 0) ||
		swapInfo.NextStateTickCurrent < MIN_TICK ||
		swapInfo.NextStateTickCurrent > MAX_TICK {
		return nil, ErrNextTickOutOfBounds
	}

	// Adjust amountOut
	var amountOut big.Int
	amountOutRawAdjusted := amountOutResult.TokenAmountOut.Amount
	if err := _verifyAdjustedAmountLimits(amountOutRawAdjusted); err != nil {
		return nil, err
	}

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
	if err := _verifyAmountLimits(&amountOut); err != nil {
		return nil, err
	}

	amountOutResult.TokenAmountOut.Amount = &amountOut

	return amountOutResult, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.v3Simulator = *p.v3Simulator.CloneState().(*uniswapv3.PoolSimulator)
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	p.v3Simulator.UpdateBalance(params)
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
