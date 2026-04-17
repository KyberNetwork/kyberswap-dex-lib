package ambient

import (
	"math/big"
	"strings"

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
	NextState *TrackerExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}
	if extra.State == nil {
		return nil, ErrNoTrackedPairs
	}

	return &PoolSimulator{
		Pool:    pool.FromEntity(ep),
		gas:     defaultGas,
		swapDex: staticExtra.SwapDex,
		base:    staticExtra.Base,
		quote:   staticExtra.Quote,
		poolIdx: staticExtra.PoolIdx,
		state:   cloneTrackerExtra(extra.State),
	}, nil
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

	state := cloneTrackerExtra(p.state)
	swap := &SwapDirective{
		Qty:        new(big.Int).Set(params.TokenAmountIn.Amount),
		InBaseQty:  inBaseQty,
		IsBuy:      isBuy,
		LimitPrice: defaultLimitPrice(isBuy),
	}
	bmpView := NewSnapshotBitmapView(state)
	accum, err := SweepSwap(&state.Curve, swap, &state.PoolParams, bmpView)
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

	tokenOutIndex := p.GetTokenIndex(strings.ToLower(params.TokenOut))
	if tokenOutIndex < 0 {
		return nil, ErrInvalidToken
	}
	if amountOut.Cmp(p.Info.Reserves[tokenOutIndex]) > 0 {
		return nil, ErrInsufficientFund
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
		SwapInfo: SwapInfo{NextState: state},
	}, nil
}

func (p *PoolSimulator) estimateGas(accum *SwapAccum) int64 {
	return p.gas.BaseGas +
		p.gas.CrossInitTickGas*int64(accum.CrossInitTickLoops) +
		p.gas.PinSpillGas*int64(accum.PinSpillLoops) +
		p.gas.KnockoutCrossGas*int64(accum.KnockoutCrossLoops)
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
	info := pool.PoolInfo{
		Address:     p.Info.Address,
		Exchange:    p.Info.Exchange,
		Type:        p.Info.Type,
		Tokens:      append([]string(nil), p.Info.Tokens...),
		Reserves:    make([]*big.Int, len(p.Info.Reserves)),
		BlockNumber: p.Info.BlockNumber,
	}
	if p.Info.SwapFee != nil {
		info.SwapFee = new(big.Int).Set(p.Info.SwapFee)
	}
	for i, reserve := range p.Info.Reserves {
		info.Reserves[i] = copyBigInt(reserve)
	}

	return &PoolSimulator{
		Pool:    pool.Pool{Info: info},
		gas:     p.gas,
		swapDex: p.swapDex,
		base:    p.base,
		quote:   p.quote,
		poolIdx: p.poolIdx,
		state:   cloneTrackerExtra(p.state),
	}
}

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	swapInfo, ok := params.SwapInfo.(SwapInfo)
	if !ok || swapInfo.NextState == nil {
		return
	}

	indexIn := p.GetTokenIndex(strings.ToLower(params.TokenAmountIn.Token))
	indexOut := p.GetTokenIndex(strings.ToLower(params.TokenAmountOut.Token))
	if indexIn >= 0 {
		p.Info.Reserves[indexIn] = new(big.Int).Add(p.Info.Reserves[indexIn], params.TokenAmountIn.Amount)
	}
	if indexOut >= 0 {
		p.Info.Reserves[indexOut] = new(big.Int).Sub(p.Info.Reserves[indexOut], params.TokenAmountOut.Amount)
	}
	p.state = cloneTrackerExtra(swapInfo.NextState)
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
		return new(big.Int).Sub(MaxSqrtRatio, bignumber.One)
	}
	return new(big.Int).Set(MinSqrtRatio)
}

func copyBigInt(v *big.Int) *big.Int {
	if v == nil {
		return nil
	}
	return new(big.Int).Set(v)
}

func cloneCurveState(state CurveState) CurveState {
	return CurveState{
		PriceRoot:    copyBigInt(state.PriceRoot),
		AmbientSeeds: copyBigInt(state.AmbientSeeds),
		ConcLiq:      copyBigInt(state.ConcLiq),
		SeedDeflator: state.SeedDeflator,
		ConcGrowth:   state.ConcGrowth,
	}
}

func cloneTrackerExtra(extra *TrackerExtra) *TrackerExtra {
	if extra == nil {
		return nil
	}
	cloned := *extra
	cloned.Curve = cloneCurveState(extra.Curve)
	return &cloned
}
