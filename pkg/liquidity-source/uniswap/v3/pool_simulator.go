package uniswapv3

import (
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	V3Pool *Pool
	pool.Pool
	Gas               Gas
	tickMin           int
	tickMax           int
	sqrtPriceLimitMin uint256.Int // GetSqrtRatioAtTick(tickMin) + 1 for zeroForOne swaps
	sqrtPriceLimitMax uint256.Int // GetSqrtRatioAtTick(tickMax) - 1 for oneForZero swaps
	allowEmptyTicks   bool

	buyRestrictedToken string // pons-fun
}

var _ = pool.RegisterFactory1(DexTypeUniswapV3, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool, _ valueobject.ChainID) (*PoolSimulator, error) {
	var extra ExtraTickU256
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	return NewPoolSimulatorWithExtra(entityPool, &extra, SimulatorConfig{})
}

func NewPoolSimulatorWithExtra(entityPool entity.Pool,
	extra *ExtraTickU256, cfg SimulatorConfig) (*PoolSimulator, error) {
	if extra.Tick == nil {
		return nil, ErrTickNil
	}

	swapFee := big.NewInt(int64(entityPool.SwapFee))
	tokens := make([]string, 2)
	reserves := make([]*big.Int, 2)
	if len(entityPool.Reserves) == 2 && len(entityPool.Tokens) == 2 {
		tokens[0] = entityPool.Tokens[0].Address
		reserves[0] = bignumber.NewBig10(entityPool.Reserves[0])
		tokens[1] = entityPool.Tokens[1].Address
		reserves[1] = bignumber.NewBig10(entityPool.Reserves[1])
	}

	v3Ticks := make([]TickU256, 0, len(extra.Ticks))

	// Ticks are sorted from the pool service, so we don't have to do it again here.
	for _, t := range extra.Ticks {
		// LiquidityGross = 0 means the tick is uninitialized.
		if t.LiquidityGross.IsZero() {
			continue
		}
		v3Ticks = append(v3Ticks, TickU256{
			Index:          t.Index,
			LiquidityGross: t.LiquidityGross,
			LiquidityNet:   t.LiquidityNet,
		})
	}

	// if the tick list is empty, the pool should be ignored
	// for some uniswap-v4 hooks, we want to bypass this check due to some hooks has no ticks
	if !cfg.AllowEmptyTicks && len(v3Ticks) == 0 {
		return nil, ErrV3TicksEmpty
	}

	tickSpacing := int(extra.TickSpacing)
	// For some pools that not yet initialized tickSpacing in their extra,
	// we will get the tickSpacing through feeTier mapping.
	if tickSpacing == 0 {
		fallback := cfg.TickSpacingFallback
		if fallback == nil {
			fallback = TickSpacings
		}
		feeTier := FeeAmount(entityPool.SwapFee)
		if _, ok := fallback[feeTier]; !ok {
			return nil, ErrInvalidFeeTier
		}
		tickSpacing = fallback[feeTier]
	}
	v3Pool, err := NewPool(
		FeeAmount(entityPool.SwapFee),
		*extra.SqrtPriceX96,
		*extra.Liquidity,
		*extra.Tick,
		v3Ticks,
		tickSpacing,
	)
	if err != nil {
		return nil, err
	}

	tickMin, tickMax := MinTick, MaxTick
	if len(v3Ticks) > 0 {
		tickMin = v3Ticks[0].Index
		tickMax = v3Ticks[len(v3Ticks)-1].Index
	}

	sim := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  entityPool.Address,
			SwapFee:  swapFee,
			Exchange: entityPool.Exchange,
			Type:     entityPool.Type,
			Tokens:   tokens,
			Reserves: reserves,
		}},
		V3Pool:             v3Pool,
		Gas:                defaultGas,
		tickMin:            tickMin,
		tickMax:            tickMax,
		allowEmptyTicks:    cfg.AllowEmptyTicks,
		buyRestrictedToken: extra.BuyRestrictedToken,
	}
	if err := GetSqrtRatioAtTick(tickMin, &sim.sqrtPriceLimitMin); err != nil {
		return nil, err
	}
	sim.sqrtPriceLimitMin.AddUint64(&sim.sqrtPriceLimitMin, 1)
	if err := GetSqrtRatioAtTick(tickMax, &sim.sqrtPriceLimitMax); err != nil {
		return nil, err
	}
	sim.sqrtPriceLimitMax.SubUint64(&sim.sqrtPriceLimitMax, 1)
	return sim, nil
}

// GetSqrtPriceLimit get the price limit of pool based on the initialized ticks that this pool has
func (p *PoolSimulator) GetSqrtPriceLimit(zeroForOne bool) *uint256.Int {
	if zeroForOne {
		return &p.sqrtPriceLimitMin
	} else {
		return &p.sqrtPriceLimitMax
	}
}

func (p *PoolSimulator) CalcAmountIn(param pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	return p.CalcAmountInWithPriceLimit(param, uint256.Int{})
}

// CalcAmountInWithPriceLimit is CalcAmountIn with an explicit sqrtPriceLimitX96. A zero limit means
// "no limit" and reproduces plain CalcAmountIn. Forks that pin a price floor/ceiling (e.g. machima's
// XMA sell floor) call this so they do not have to rebuild the result and SwapInfo by hand.
func (p *PoolSimulator) CalcAmountInWithPriceLimit(param pool.CalcAmountInParams,
	sqrtPriceLimitX96 uint256.Int) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenAmountOut := param.TokenIn, param.TokenAmountOut
	tokenOut := tokenAmountOut.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	} else if p.isBuyRestricted(tokenOut) {
		return nil, ErrBuyRestricted
	} else if tokenAmountOut.Amount.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientBalance
	}

	zeroForOne := tokenInIndex == 0

	var amountOut uint256.Int
	if overflow := amountOut.SetFromBig(tokenAmountOut.Amount); overflow {
		return nil, ErrOverflow
	}

	result, err := p.V3Pool.GetInputAmountV2(zeroForOne, amountOut, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	}

	amountInBI := result.AmountCalculated.ToBig()
	if !p.allowEmptyTicks {
		if amountInBI.Sign() <= 0 {
			return nil, ErrZeroAmount
		}
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: amountInBI},
		Fee:           &pool.TokenAmount{Token: tokenIn},
		Gas:           p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(result.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			NextStateSqrtRatioX96: &result.SqrtRatioX96,
			NextStateLiquidity:    result.Liquidity,
			NextStateTickCurrent:  result.CurrentTick,
		},
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	return p.CalcAmountOutWithPriceLimit(param, uint256.Int{})
}

// CalcAmountOutWithPriceLimit is CalcAmountOut with an explicit sqrtPriceLimitX96. A zero limit means
// "no limit" and reproduces plain CalcAmountOut. When the limit is hit the swap only partially fills
// and the unspent input is reported in RemainingTokenAmountIn.
func (p *PoolSimulator) CalcAmountOutWithPriceLimit(param pool.CalcAmountOutParams,
	sqrtPriceLimitX96 uint256.Int) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn, tokenOut := param.TokenAmountIn, param.TokenOut
	tokenIn := tokenAmountIn.Token
	tokenInIndex, tokenOutIndex := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if tokenInIndex < 0 || tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	} else if p.isBuyRestricted(tokenOut) {
		return nil, ErrBuyRestricted
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(tokenAmountIn.Amount); overflow {
		return nil, ErrOverflow
	}
	zeroForOne := tokenInIndex == 0
	result, err := p.V3Pool.GetOutputAmountV2(zeroForOne, amountIn, sqrtPriceLimitX96)
	if err != nil {
		return nil, err
	} else if !p.allowEmptyTicks && result.AmountCalculated.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	amountOutBI := result.AmountCalculated.ToBig()
	if amountOutBI.Cmp(p.GetReserves()[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientBalance
	}

	remainingTokenAmountIn := &pool.TokenAmount{
		Token:  tokenIn,
		Amount: bignumber.ZeroBI,
	}
	if !result.RemainingAmountIn.IsZero() {
		remainingTokenAmountIn.Amount = result.RemainingAmountIn.ToBig()
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOutBI},
		RemainingTokenAmountIn: remainingTokenAmountIn, Fee: &pool.TokenAmount{Token: tokenIn},
		Gas: p.Gas.BaseGas + p.Gas.CrossInitTickGas*int64(result.CrossInitTickLoops),
		SwapInfo: SwapInfo{
			RemainingAmountIn:     &result.RemainingAmountIn,
			NextStateSqrtRatioX96: &result.SqrtRatioX96,
			NextStateLiquidity:    result.Liquidity,
			NextStateTickCurrent:  result.CurrentTick,
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
	p.V3Pool.SqrtRatioX96.Set(si.NextStateSqrtRatioX96)
	p.V3Pool.Liquidity.Set(&si.NextStateLiquidity)
	p.V3Pool.TickCurrent = si.NextStateTickCurrent
	tokenAmtIn, tokenAmtOut := params.TokenAmountIn, params.TokenAmountOut
	if p.GetTokenIndex(tokenAmtIn.Token) == 0 {
		p.Info.Reserves = []*big.Int{new(big.Int).Add(p.Info.Reserves[0], tokenAmtIn.Amount),
			new(big.Int).Sub(p.Info.Reserves[1], tokenAmtOut.Amount)}
	} else {
		p.Info.Reserves = []*big.Int{new(big.Int).Sub(p.Info.Reserves[0], tokenAmtOut.Amount),
			new(big.Int).Add(p.Info.Reserves[1], tokenAmtIn.Amount)}
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn string, _ string) any {
	return PoolMeta{
		SwapFee:    uint32(p.Info.SwapFee.Int64()),
		PriceLimit: p.GetSqrtPriceLimit(tokenIn == p.Info.Tokens[0]),
	}
}

func (p *PoolSimulator) isBuyRestricted(tokenOut string) bool {
	return p.buyRestrictedToken != "" && strings.EqualFold(tokenOut, p.buyRestrictedToken)
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	out := p.Pool.CanSwapTo(address)
	if p.buyRestrictedToken == "" {
		return out
	}

	return slices.DeleteFunc(out, p.isBuyRestricted)
}
