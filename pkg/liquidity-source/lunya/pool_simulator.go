package lunya

import (
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

// nowUnix is the clock a STABLE pool's amplification ramp is read against.
var nowUnix = func() uint64 { return uint64(time.Now().Unix()) }

// PoolSimulator prices every Lunya pool type. The pool type decides which half of the state is filled
// in and which curve prices the swap: ticks and the Uniswap V3 loop for CL and CP, the curve reserves
// and StableSwap for STABLE.
type PoolSimulator struct {
	pool.Pool

	poolType uint8

	sqrtPriceX96 uint256.Int
	liquidity    uint256.Int
	fee          uint32
	feeToken     uint8
	halted       bool

	// CL and CP. ticks and tickSqrtPrices are read-only after construction; CloneState shares them.
	tick           int
	ticks          []Tick
	tickSqrtPrices []uint256.Int

	// STABLE
	curveReserve0     uint256.Int
	curveReserve1     uint256.Int
	rate0             uint256.Int
	rate1             uint256.Int
	priceScaleSqrtQ96 uint256.Int
	amplificationX100 uint32
	// ramp is read-only after construction; CloneState shares it.
	ramp *AmplificationRamp

	// sqrtPriceLimitMin/Max bound a swap: at the outermost initialized ticks for CL and CP, where the
	// liquidity ends, and at the widest legal prices for STABLE.
	sqrtPriceLimitMin uint256.Int
	sqrtPriceLimitMax uint256.Int

	gas Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if len(entityPool.StaticExtra) > 0 {
		if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
			return nil, err
		}
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.SqrtPriceX96 == nil || extra.SqrtPriceX96.IsZero() || extra.Liquidity == nil {
		return nil, ErrNotInitialized
	}

	sim := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     entityPool.Address,
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      lo.Map(entityPool.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(entityPool.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig10(r) }),
			SwapFee:     big.NewInt(int64(extra.Fee)),
			BlockNumber: entityPool.BlockNumber,
		}},
		poolType:     staticExtra.PoolType,
		sqrtPriceX96: *extra.SqrtPriceX96,
		liquidity:    *extra.Liquidity,
		fee:          extra.Fee,
		feeToken:     extra.FeeToken,
		halted:       extra.Halted,
	}
	sim.sqrtPriceLimitMin.AddUint64(minSqrtRatio, 1)
	sim.sqrtPriceLimitMax.SubUint64(maxSqrtRatio, 1)

	var err error
	switch staticExtra.PoolType {
	case poolTypeCL, poolTypeCP:
		err = sim.initCL(&extra)
	case poolTypeStable:
		err = sim.initStable(&extra)
	default:
		err = ErrUnsupportedPoolType
	}
	if err != nil {
		return nil, err
	}

	return sim, nil
}

func (p *PoolSimulator) initCL(extra *Extra) error {
	p.tick = extra.Tick
	p.gas = clGas

	p.ticks = make([]Tick, 0, len(extra.Ticks))
	for _, tick := range extra.Ticks {
		if tick.LiquidityGross == nil || tick.LiquidityGross.IsZero() || tick.LiquidityNet == nil {
			continue
		}
		p.ticks = append(p.ticks, tick)
	}

	p.tickSqrtPrices = make([]uint256.Int, len(p.ticks))
	for i, tick := range p.ticks {
		if err := uniswapv3.GetSqrtRatioAtTick(tick.Index, &p.tickSqrtPrices[i]); err != nil {
			return err
		}
	}
	if len(p.ticks) > 0 {
		p.sqrtPriceLimitMin.AddUint64(&p.tickSqrtPrices[0], 1)
		p.sqrtPriceLimitMax.SubUint64(&p.tickSqrtPrices[len(p.ticks)-1], 1)
	}

	return nil
}

func (p *PoolSimulator) initStable(extra *Extra) error {
	if extra.CurveReserve0 == nil || extra.CurveReserve1 == nil {
		return ErrNotInitialized
	}
	// every STABLE pool has non-zero rates and a price scale from construction
	for _, v := range []*uint256.Int{extra.Rate0, extra.Rate1, extra.PriceScaleSqrtQ96} {
		if v == nil || v.IsZero() {
			return ErrNotInitialized
		}
	}

	p.curveReserve0 = *extra.CurveReserve0
	p.curveReserve1 = *extra.CurveReserve1
	p.rate0 = *extra.Rate0
	p.rate1 = *extra.Rate1
	p.priceScaleSqrtQ96 = *extra.PriceScaleSqrtQ96
	p.amplificationX100 = extra.AmplificationX100
	p.ramp = extra.Ramp
	p.gas = stableGas

	return nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenIn, tokenOut := params.TokenAmountIn.Token, params.TokenOut
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	var amountIn uint256.Int
	if overflow := amountIn.SetFromBig(params.TokenAmountIn.Amount); overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}

	zeroForOne := indexIn == 0
	var (
		amountOut uint256.Int
		remaining uint256.Int
		gas       int64
		swapInfo  SwapInfo
	)

	if p.poolType == poolTypeStable {
		s, err := p.stableSwap(zeroForOne, true, &amountIn, p.priceLimit(zeroForOne))
		if err != nil {
			return nil, err
		}
		amountOut.Set(&s.amountOut)
		// what the swap did not charge - a price limit, or the rounding to raw units - stays with the caller
		if amountIn.Gt(&s.amountIn) {
			remaining.Sub(&amountIn, &s.amountIn)
		}
		gas = p.gas.BaseGas
		swapInfo = SwapInfo{
			NextStateSqrtRatioX96: &s.sqrtPriceX96,
			NextCurveReserve0:     &s.curveReserve0,
			NextCurveReserve1:     &s.curveReserve1,
		}
	} else {
		s, err := p.clSwap(zeroForOne, &amountIn, p.priceLimit(zeroForOne))
		if err != nil {
			return nil, err
		}
		amountOut.Neg(&s.amountCalculated)
		remaining.Set(&s.amountSpecifiedRemaining)
		gas = p.gas.BaseGas + p.gas.CrossInitTickGas*int64(s.crossedTicks)
		swapInfo = SwapInfo{
			NextStateSqrtRatioX96: &s.sqrtPriceX96,
			NextStateLiquidity:    &s.liquidity,
			NextStateTickCurrent:  s.tick,
		}
	}

	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}
	amountOutBI := amountOut.ToBig()
	if amountOutBI.Cmp(p.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInsufficientReserve
	}
	swapInfo.RemainingAmountIn = &remaining

	return &pool.CalcAmountOutResult{
		TokenAmountOut:         &pool.TokenAmount{Token: tokenOut, Amount: amountOutBI},
		Fee:                    &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		RemainingTokenAmountIn: &pool.TokenAmount{Token: tokenIn, Amount: remaining.ToBig()},
		Gas:                    gas,
		SwapInfo:               swapInfo,
	}, nil
}

func (p *PoolSimulator) CalcAmountIn(params pool.CalcAmountInParams) (*pool.CalcAmountInResult, error) {
	tokenIn, tokenOut := params.TokenIn, params.TokenAmountOut.Token
	indexIn, indexOut := p.GetTokenIndex(tokenIn), p.GetTokenIndex(tokenOut)
	if indexIn < 0 || indexOut < 0 || indexIn == indexOut {
		return nil, ErrInvalidToken
	}

	var amountOut uint256.Int
	if overflow := amountOut.SetFromBig(params.TokenAmountOut.Amount); overflow || amountOut.Sign() <= 0 {
		return nil, ErrInvalidAmount
	} else if params.TokenAmountOut.Amount.Cmp(p.Info.Reserves[indexOut]) > 0 {
		return nil, ErrInsufficientReserve
	}

	zeroForOne := indexIn == 0
	var (
		amountIn uint256.Int
		gas      int64
		swapInfo SwapInfo
	)

	if p.poolType == poolTypeStable {
		s, err := p.stableSwap(zeroForOne, false, &amountOut, p.priceLimit(zeroForOne))
		if err != nil {
			return nil, err
		}
		if s.amountOut.Lt(&amountOut) {
			return nil, ErrPartialFill
		}
		amountIn.Set(&s.amountIn)
		gas = p.gas.BaseGas
		swapInfo = SwapInfo{
			NextStateSqrtRatioX96: &s.sqrtPriceX96,
			NextCurveReserve0:     &s.curveReserve0,
			NextCurveReserve1:     &s.curveReserve1,
		}
	} else {
		var amountSpecified uint256.Int
		amountSpecified.Neg(&amountOut)
		s, err := p.clSwap(zeroForOne, &amountSpecified, p.priceLimit(zeroForOne))
		if err != nil {
			return nil, err
		}
		if !s.amountSpecifiedRemaining.IsZero() {
			return nil, ErrPartialFill
		}
		amountIn.Set(&s.amountCalculated)
		gas = p.gas.BaseGas + p.gas.CrossInitTickGas*int64(s.crossedTicks)
		swapInfo = SwapInfo{
			NextStateSqrtRatioX96: &s.sqrtPriceX96,
			NextStateLiquidity:    &s.liquidity,
			NextStateTickCurrent:  s.tick,
		}
	}

	if amountIn.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	return &pool.CalcAmountInResult{
		TokenAmountIn:           &pool.TokenAmount{Token: tokenIn, Amount: amountIn.ToBig()},
		RemainingTokenAmountOut: &pool.TokenAmount{Token: tokenOut, Amount: bignumber.ZeroBI},
		Fee:                     &pool.TokenAmount{Token: tokenIn, Amount: bignumber.ZeroBI},
		Gas:                     gas,
		SwapInfo:                swapInfo,
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		logger.Warn("failed to UpdateBalance for lunya pool, wrong swapInfo type")
		return
	}

	if si.NextStateSqrtRatioX96 != nil {
		p.sqrtPriceX96.Set(si.NextStateSqrtRatioX96)
	}
	if si.NextStateLiquidity != nil {
		p.liquidity.Set(si.NextStateLiquidity)
		p.tick = si.NextStateTickCurrent
	}
	if si.NextCurveReserve0 != nil && si.NextCurveReserve1 != nil {
		p.curveReserve0.Set(si.NextCurveReserve0)
		p.curveReserve1.Set(si.NextCurveReserve1)
	}

	reserves := slices.Clone(p.Info.Reserves)
	indexIn, indexOut := p.GetTokenIndex(params.TokenAmountIn.Token), p.GetTokenIndex(params.TokenAmountOut.Token)
	reserves[indexIn] = new(big.Int).Add(reserves[indexIn], params.TokenAmountIn.Amount)
	reserves[indexOut] = new(big.Int).Sub(reserves[indexOut], params.TokenAmountOut.Amount)
	p.Info.Reserves = reserves
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	return PoolMeta{
		PriceLimit:  p.priceLimit(p.GetTokenIndex(tokenIn) == 0),
		BlockNumber: p.Info.BlockNumber,
	}
}

func (p *PoolSimulator) priceLimit(zeroForOne bool) *uint256.Int {
	if zeroForOne {
		return &p.sqrtPriceLimitMin
	}
	return &p.sqrtPriceLimitMax
}
