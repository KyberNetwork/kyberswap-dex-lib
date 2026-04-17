package ambient

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	*NTokenPool

	gas       Gas
	swapDex   common.Address
	pairInfos map[TokenPair]*TokenPairInfo
}

type SwapInfo struct {
	Pair      TokenPair
	NextState *TrackerExtra
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, fmt.Errorf("unmarshal static extra: %w", err)
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, fmt.Errorf("unmarshal extra: %w", err)
	}

	pairInfos := make(map[TokenPair]*TokenPairInfo, len(extra.TokenPairs))
	pairs := make([]TokenPair, 0, len(extra.TokenPairs))
	for pair, info := range extra.TokenPairs {
		if info == nil || info.State == nil {
			continue
		}
		pairInfos[pair] = &TokenPairInfo{
			PoolIdx: info.PoolIdx,
			State:   cloneTrackerExtra(info.State),
		}
		pairs = append(pairs, pair)
	}
	if len(pairInfos) == 0 {
		return nil, ErrNoTrackedPairs
	}

	tokens := make([]string, len(entityPool.Tokens))
	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, token := range entityPool.Tokens {
		tokens[i] = strings.ToLower(token.Address)
		reserves[i] = bignumber.NewBig10(entityPool.Reserves[i])
	}

	basePool := pool.Pool{
		Info: pool.PoolInfo{
			Address:     strings.ToLower(entityPool.Address),
			Exchange:    entityPool.Exchange,
			Type:        entityPool.Type,
			Tokens:      tokens,
			Reserves:    reserves,
			BlockNumber: entityPool.BlockNumber,
		},
	}

	return &PoolSimulator{
		NTokenPool: NewNTokenPool(basePool, pairs, staticExtra.NativeTokenAddress),
		gas:        defaultGas,
		swapDex:    staticExtra.SwapDex,
		pairInfos:  pairInfos,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	if params.TokenAmountIn.Amount == nil || params.TokenAmountIn.Amount.Sign() <= 0 {
		return nil, ErrZeroAmount
	}

	tokenInAddr := common.HexToAddress(params.TokenAmountIn.Token)
	tokenOutAddr := common.HexToAddress(params.TokenOut)

	pair, ok := p.GetPair(tokenInAddr, tokenOutAddr)
	if !ok {
		return nil, ErrPairNotFound
	}

	pairInfo, ok := p.pairInfos[pair]
	if !ok || pairInfo == nil || pairInfo.State == nil {
		return nil, ErrPairNotFound
	}

	state := cloneTrackerExtra(pairInfo.State)
	inBaseQty := pair.Base == tokenInAddr ||
		(pair.Base == NativeTokenPlaceholderAddress && tokenInAddr == p.nativeTokenAddress)
	isBuy := inBaseQty

	swap := &SwapDirective{
		Qty:        new(big.Int).Set(params.TokenAmountIn.Amount),
		InBaseQty:  inBaseQty,
		IsBuy:      isBuy,
		LimitPrice: defaultLimitPrice(isBuy),
	}
	bmpView := NewSnapshotBitmapView(state)
	accum := SweepSwap(&state.Curve, swap, &state.PoolParams, bmpView)

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
	swapInfo := SwapInfo{
		Pair:      pair,
		NextState: state,
	}

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
		Gas:      p.gas.BaseGas,
		SwapInfo: swapInfo,
	}, nil
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

	pairs := append([]TokenPair(nil), p.pairs...)
	cloned := &PoolSimulator{
		NTokenPool: NewNTokenPool(pool.Pool{Info: info}, pairs, p.nativeTokenAddress),
		gas:        p.gas,
		swapDex:    p.swapDex,
		pairInfos:  make(map[TokenPair]*TokenPairInfo, len(p.pairInfos)),
	}
	for pair, info := range p.pairInfos {
		cloned.pairInfos[pair] = &TokenPairInfo{
			PoolIdx: info.PoolIdx,
			State:   cloneTrackerExtra(info.State),
		}
	}

	return cloned
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

	info, ok := p.pairInfos[swapInfo.Pair]
	if !ok || info == nil {
		return
	}
	info.State = cloneTrackerExtra(swapInfo.NextState)
}

func (p *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	pair, ok := p.GetPair(common.HexToAddress(tokenIn), common.HexToAddress(tokenOut))
	if !ok {
		return nil
	}
	info, ok := p.pairInfos[pair]
	if !ok || info == nil {
		return nil
	}

	return Meta{
		SwapDex: p.swapDex,
		Base:    pair.Base,
		Quote:   pair.Quote,
		PoolIdx: info.PoolIdx,
	}
}

func (p *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return p.swapDex.Hex()
}

func outputAmount(accum *SwapAccum, inBaseQty bool) *big.Int {
	if inBaseQty {
		return new(big.Int).Neg(accum.QuoteFlow)
	}
	return new(big.Int).Neg(accum.BaseFlow)
}

func defaultLimitPrice(isBuy bool) *big.Int {
	if isBuy {
		return new(big.Int).Sub(MaxSqrtRatio, big.NewInt(1))
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

func cloneBookLevel(level BookLevel) BookLevel {
	return BookLevel{
		BidLots:     copyBigInt(level.BidLots),
		AskLots:     copyBigInt(level.AskLots),
		FeeOdometer: level.FeeOdometer,
	}
}

func cloneKnockoutPivot(pivot KnockoutPivot) KnockoutPivot {
	return KnockoutPivot{
		Lots:       copyBigInt(pivot.Lots),
		PivotTime:  pivot.PivotTime,
		RangeTicks: pivot.RangeTicks,
	}
}

func cloneKnockoutMerkle(merkle KnockoutMerkle) KnockoutMerkle {
	return KnockoutMerkle{
		MerkleRoot: copyBigInt(merkle.MerkleRoot),
		PivotTime:  merkle.PivotTime,
		FeeMileage: merkle.FeeMileage,
	}
}

func cloneTrackerExtra(extra *TrackerExtra) *TrackerExtra {
	if extra == nil {
		return nil
	}

	cloned := &TrackerExtra{
		Base:           extra.Base,
		Quote:          extra.Quote,
		PoolIdx:        extra.PoolIdx,
		PoolHash:       extra.PoolHash,
		Curve:          cloneCurveState(extra.Curve),
		PoolSpec:       extra.PoolSpec,
		TemplateSpec:   extra.TemplateSpec,
		PoolParams:     extra.PoolParams,
		TemplateParams: extra.TemplateParams,
		ActiveTicks:    append([]int32(nil), extra.ActiveTicks...),
		Levels:         make([]TrackedLevel, len(extra.Levels)),
		Knockouts:      make([]TrackedKnockout, len(extra.Knockouts)),
		MinTick:        extra.MinTick,
		MaxTick:        extra.MaxTick,
	}

	for i, level := range extra.Levels {
		cloned.Levels[i] = TrackedLevel{
			Tick:  level.Tick,
			Level: cloneBookLevel(level.Level),
		}
	}
	for i, knockout := range extra.Knockouts {
		cloned.Knockouts[i] = TrackedKnockout{
			Tick:      knockout.Tick,
			BidPivot:  cloneKnockoutPivot(knockout.BidPivot),
			BidMerkle: cloneKnockoutMerkle(knockout.BidMerkle),
			AskPivot:  cloneKnockoutPivot(knockout.AskPivot),
			AskMerkle: cloneKnockoutMerkle(knockout.AskMerkle),
		}
	}

	return cloned
}
