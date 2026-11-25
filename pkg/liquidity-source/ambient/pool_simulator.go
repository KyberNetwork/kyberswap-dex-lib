package ambient

import (
	"math/big"
	"slices"
	"strings"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
	levels  map[int32]BookLevel // buildLevels(state), shared by clones; nil falls back to a per-call build
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
	if extra.State == nil || extra.State.Curve.PriceRoot.IsZero() {
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
		levels:  buildLevels(extra.State),
	}, nil
}

func (p *PoolSimulator) ambientToken(tokenIndex int) string {
	return lo.Ternary(tokenIndex == 0, p.base, p.quote)
}

// CalcAmtOut is CalcAmountOut with uint256 amounts. It shares quote with the legacy method.
func (p *PoolSimulator) CalcAmtOut(params pool.CalcAmtOutParams) (*pool.CalcAmtOutResult, error) {
	q, err := p.quoteSwap(params.TokenAmountIn.Token, params.TokenOut, &params.TokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	if limit := params.Limit; limit != nil {
		if inventoryLimit := limit.GetLimit(p.ambientToken(q.tokenOutIndex)); inventoryLimit != nil &&
			q.amountOut.Gt(inventoryLimit) {
			return nil, pool.ErrNotEnoughInventory
		}
	}

	return &pool.CalcAmtOutResult{
		TokenAmountOut: &pool.TokenAmt{Token: params.TokenOut, Amount: q.amountOut},
		RemainingTokenAmountIn: &pool.TokenAmt{
			Token:  params.TokenAmountIn.Token,
			Amount: q.remainingIn,
		},
		Gas:      q.gas,
		SwapInfo: SwapInfo{nextCurve: q.nextCurve},
	}, nil
}

type quoteResult struct {
	amountOut     uint256.Int
	remainingIn   uint256.Int
	tokenOutIndex int
	gas           int64
	nextCurve     CurveState
}

// quoteSwap runs the sweep for amountIn. It checks the reserve but not the inventory limit.
func (p *PoolSimulator) quoteSwap(tokenIn, tokenOut string, amountIn *uint256.Int) (*quoteResult, error) {
	if amountIn.IsZero() {
		return nil, ErrZeroAmount
	}

	inBaseQty, ok := p.matchDirection(tokenIn, tokenOut)
	if !ok {
		return nil, ErrPairNotFound
	}
	isBuy := inBaseQty

	curve := p.state.Curve
	var swap SwapDirective
	swap.InBaseQty = inBaseQty
	swap.IsBuy = isBuy
	swap.Qty = *amountIn
	swap.LimitPrice = defaultLimitPrice(isBuy)

	levels := p.levels
	if levels == nil {
		levels = buildLevels(p.state)
	}
	bmpView := newSnapshotBitmapView(p.state, levels)
	accum, err := SweepSwap(&curve, &swap, &p.state.PoolParams, bmpView)
	if err != nil {
		return nil, err
	}

	if bmpView.BoundaryExceeded() && !swap.Qty.IsZero() {
		return nil, ErrTickRangeExceeded
	}

	var q quoteResult
	q.amountOut = outputAmount(accum, inBaseQty)
	if q.amountOut.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	q.tokenOutIndex = 1
	if !inBaseQty {
		q.tokenOutIndex = 0
	}
	reserve, overflow := uint256.FromBig(p.Info.Reserves[q.tokenOutIndex])
	if overflow {
		return nil, ErrOverflow
	}
	if q.amountOut.Gt(reserve) {
		return nil, ErrInsufficientFund
	}

	q.remainingIn = swap.Qty
	q.gas = p.estimateGas(accum)
	q.nextCurve = curve
	return &q, nil
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
	if !ok || swapInfo.nextCurve.PriceRoot.IsZero() {
		return
	}

	indexIn := p.GetTokenIndex(strings.ToLower(params.TokenAmountIn.Token))
	indexOut := p.GetTokenIndex(strings.ToLower(params.TokenAmountOut.Token))

	p.Info.Reserves[indexIn] = new(big.Int).Add(p.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	p.Info.Reserves[indexOut] = new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	p.state.Curve = swapInfo.nextCurve

	if limit := params.SwapLimitBig; limit != nil {
		if _, _, err := limit.UpdateLimit(p.ambientToken(indexOut), p.ambientToken(indexIn),
			params.TokenAmountOut.Amount, params.TokenAmountIn.Amount); err != nil {
			logger.Errorf("unable to update ambient limit, error: %v", err)
		}
	}
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, _ string) any {
	tokenInIndex := p.GetTokenIndex(strings.ToLower(tokenIn))
	return Meta{
		SwapDex:     p.swapDex,
		Base:        lo.Ternary(tokenInIndex == 0, p.base, p.quote),
		Quote:       lo.Ternary(tokenInIndex == 0, p.quote, p.base),
		PoolIdx:     new(big.Int).SetUint64(p.poolIdx),
		BlockNumber: p.Info.BlockNumber,
	}
}

func (p *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	tokenInIndex := p.GetTokenIndex(strings.ToLower(tokenIn))
	if !valueobject.IsZero(lo.Ternary(tokenInIndex == 0, p.base, p.quote)) {
		return p.swapDex
	}
	return ""
}

func (s *PoolSimulator) SwapReceiveNativeIn(tokenIn, tokenOut string, chainId valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return valueobject.IsZero(meta.Base) && valueobject.IsWrappedNative(tokenIn, chainId)
}

func (s *PoolSimulator) SwapReturnNativeOut(tokenIn, tokenOut string, chainId valueobject.ChainID) bool {
	meta := s.GetMetaInfo(tokenIn, tokenOut).(Meta)
	return valueobject.IsZero(meta.Quote) && valueobject.IsWrappedNative(tokenOut, chainId)
}

func outputAmount(accum *SwapAccum, inBaseQty bool) uint256.Int {
	// The output token's flow is negative (user receives). Negate to get positive amount.
	flow := accum.BaseFlow
	if inBaseQty {
		flow = accum.QuoteFlow
	}
	var out uint256.Int
	return *out.Neg(&flow)
}

func defaultLimitPrice(isBuy bool) uint256.Int {
	if isBuy {
		return uMaxSqrtRatioMinus1
	}
	return uMinSqrtRatio
}
