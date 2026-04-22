package ambient

import (
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool

	gas     Gas
	swapDex string
	base    string
	quote   string
	poolIdx uint64
	state   *TrackerExtra
}

type SwapInfo struct {
	nextCurve CurveState
}

var (
	_ = pool.RegisterFactory0(DexType, NewPoolSimulator)
	_ = pool.RegisterUseSwapLimit(DexType)
)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.State == nil ||
		extra.State.Curve.PriceRoot == nil ||
		extra.State.Curve.PriceRoot.Sign() == 0 {
		return nil, ErrNoTrackedPairs
	}

	return &PoolSimulator{
		Pool:    pool.FromEntity(ep),
		gas:     defaultGas,
		swapDex: staticExtra.SwapDex,
		base:    staticExtra.Base,
		quote:   staticExtra.Quote,
		poolIdx: staticExtra.PoolIdx,
		state:   extra.State,
	}, nil
}

func (p *PoolSimulator) ambientToken(tokenIndex int) string {
	return lo.Ternary(tokenIndex == 0, p.base, p.quote)
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	inBaseQty, ok := p.matchDirection(params.TokenAmountIn.Token, params.TokenOut)
	if !ok {
		return nil, ErrPairNotFound
	}
	isBuy := inBaseQty

	curve := p.state.Curve
	swap := &SwapDirective{
		Qty:        new(big.Int).Set(params.TokenAmountIn.Amount),
		InBaseQty:  inBaseQty,
		IsBuy:      isBuy,
		LimitPrice: defaultLimitPrice(isBuy),
	}
	bmpView := NewSnapshotBitmapView(p.state)
	accum, err := SweepSwap(&curve, swap, &p.state.PoolParams, bmpView)
	if err != nil {
		return nil, err
	}

	if bmpView.BoundaryExceeded() && swap.Qty.Sign() > 0 {
		return nil, ErrTickRangeExceeded
	}

	amountOut := outputAmount(accum, inBaseQty)
	if amountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	tokenOutIndex := 1
	if !inBaseQty {
		tokenOutIndex = 0
	}
	if amountOut.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientFund
	}

	if limit := params.Limit; limit != nil {
		if inventoryLimit := limit.GetLimit(p.ambientToken(tokenOutIndex)); inventoryLimit != nil &&
			amountOut.Cmp(inventoryLimit) > 0 {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	remainingAmountIn := new(big.Int).Set(swap.Qty)
	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: bignumber.ZeroBI,
		},
		RemainingTokenAmountIn: &pool.TokenAmount{
			Token:  params.TokenAmountIn.Token,
			Amount: remainingAmountIn,
		},
		Gas:      p.estimateGas(accum),
		SwapInfo: SwapInfo{nextCurve: curve},
	}, nil
}

func (p *PoolSimulator) estimateGas(accum *SwapAccum) int64 {
	return p.gas.BaseGas +
		p.gas.CrossInitTickGas*int64(accum.CrossInitTickLoops) +
		p.gas.PinSpillGas*int64(accum.PinSpillLoops) +
		p.gas.KnockoutCrossGas*int64(accum.KnockoutCrossLoops)
}

func (p *PoolSimulator) CalculateLimit() map[string]*big.Int {
	reserves := p.GetReserves()
	return map[string]*big.Int{
		p.ambientToken(0): new(big.Int).Set(reserves[0]),
		p.ambientToken(1): new(big.Int).Set(reserves[1]),
	}
}

func (p *PoolSimulator) matchDirection(tokenIn, tokenOut string) (inBaseQty, ok bool) {
	iIn := p.GetTokenIndex(strings.ToLower(tokenIn))
	iOut := p.GetTokenIndex(strings.ToLower(tokenOut))
	if iIn < 0 || iOut < 0 || iIn == iOut {
		return false, false
	}
	return iIn == 0, true
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	cloned.Info.Reserves = slices.Clone(p.Info.Reserves)
	clonedState := *p.state
	clonedState.Curve = p.state.Curve.Clone()
	cloned.state = &clonedState
	return &cloned
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok || swapInfo.nextCurve.PriceRoot == nil {
		return
	}

	indexIn := p.GetTokenIndex(strings.ToLower(params.TokenAmountIn.Token))
	indexOut := p.GetTokenIndex(strings.ToLower(params.TokenAmountOut.Token))

	p.Info.Reserves[indexIn] = new(big.Int).Add(p.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	p.Info.Reserves[indexOut] = new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	p.state.Curve = swapInfo.nextCurve

	if limit := params.SwapLimit; limit != nil {
		if _, _, err := limit.UpdateLimit(p.ambientToken(indexOut), p.ambientToken(indexIn),
			params.TokenAmountOut.Amount, params.TokenAmountIn.Amount); err != nil {
			logger.Errorf("unable to update ambient limit, error: %v", err)
		}
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	tokenInIndex := p.GetTokenIndex(strings.ToLower(tokenIn))
	return Meta{
		SwapDex: p.swapDex,
		Base:    lo.Ternary(tokenInIndex == 0, p.base, p.quote),
		Quote:   lo.Ternary(tokenInIndex == 0, p.quote, p.base),
		PoolIdx: new(big.Int).SetUint64(p.poolIdx),
	}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	tokenInIndex := p.GetTokenIndex(strings.ToLower(tokenIn))
	if !valueobject.IsZero(lo.Ternary(tokenInIndex == 0, p.base, p.quote)) {
		return p.swapDex
	}

	return ""
}

func outputAmount(accum *SwapAccum, inBaseQty bool) *big.Int {
	if inBaseQty {
		return new(big.Int).Neg(accum.QuoteFlow)
	}
	return new(big.Int).Neg(accum.BaseFlow)
}

func defaultLimitPrice(isBuy bool) *big.Int {
	if isBuy {
		return new(big.Int).Set(MaxSqrtRatioMinus1)
	}
	return new(big.Int).Set(MinSqrtRatio)
}
