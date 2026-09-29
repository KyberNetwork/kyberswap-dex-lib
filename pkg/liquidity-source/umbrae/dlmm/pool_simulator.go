package umbraedlmm

import (
	"math/big"
	"sort"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolSimulator struct {
	pool.Pool

	binStep          uint16
	scaleX           *uint256.Int // 10^(18-decimalsX)
	scaleY           *uint256.Int // 10^(18-decimalsY)
	priceDenominator *uint256.Int // scaleY * 10^decimalsX — identical in both swap directions

	activeID uint32
	bins     []Bin          // sorted ascending by ID; normalized reserves
	binIndex map[uint32]int // bin ID -> index into bins

	fp             FeeParameters
	variableFeeCap uint16
	// Raw volatility state (V2 anchored-displacement model). The working volatility and window
	// anchor are derived per swap exactly as PairSwapLib.performSwap does, using the tracked
	// timestamp as the clock.
	volAcc       uint64
	volRef       uint32 // anchor bin id; < 2^22 means unset
	lastVolUpd   uint64
	trackedAt    uint64
	router       string // DLMM Router: the token spender / approval target
}

var _ = pool.RegisterFactory0(DexType, NewPoolSimulator)

func NewPoolSimulator(ep entity.Pool) (*PoolSimulator, error) {
	var static StaticExtra
	if err := json.Unmarshal([]byte(ep.StaticExtra), &static); err != nil {
		return nil, err
	}
	var extra Extra
	if err := json.Unmarshal([]byte(ep.Extra), &extra); err != nil {
		return nil, err
	}

	bins := append([]Bin(nil), extra.Bins...)
	sort.Slice(bins, func(i, j int) bool { return bins[i].ID < bins[j].ID })
	binIndex := make(map[uint32]int, len(bins))
	for i, b := range bins {
		binIndex[b.ID] = i
	}

	scaleY := pow10(18 - static.DecimalsY)
	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		binStep:          static.BinStep,
		scaleX:           pow10(18 - static.DecimalsX),
		scaleY:           scaleY,
		priceDenominator: new(uint256.Int).Mul(scaleY, pow10(static.DecimalsX)),
		activeID:         extra.ActiveID,
		bins:             bins,
		binIndex:         binIndex,
		fp:               extra.FeeParameters,
		variableFeeCap:   extra.VariableFeeCap,
		volAcc:           extra.VolatilityAccumulator,
		volRef:           extra.VolatilityReference,
		lastVolUpd:       extra.LastVolatilityUpdate,
		trackedAt:        extra.Timestamp,
		router:           static.Router,
	}
	return s, nil
}

func (s *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(param.TokenAmountIn.Token), s.GetTokenIndex(param.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	amountIn, overflow := uint256.FromBig(param.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInvalidAmountIn
	}

	swapForY := indexIn == 0 // tokenX (index 0) in -> tokenY out

	amountOut, fee, swapInfo, err := s.traverse(amountIn, swapForY)
	if err != nil {
		return nil, err
	}
	if amountOut.Sign() <= 0 {
		// The pair reverts LBPair__InsufficientAmountOut on a zero output even with amountOutMin=0.
		return nil, ErrInsufficientOutput
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            defaultGas + int64(len(swapInfo.binUpdates))*gasPerBin,
		SwapInfo:       swapInfo,
	}, nil
}

// traverse reproduces the deployed PairSwapLib.performSwap loop:
//   - the per-bin fee is charged on the remaining GROSS input (native decimals) BEFORE pricing,
//     with the unspent slice of a partially-consumed bin's fee returned to the gross remainder;
//   - the working volatility is the anchored displacement |bin - anchor| on top of the window
//     base, clamped at maxVolatilityAccumulator (V2 R12/M-3 — no cumulative ratchet);
//   - a bin with no output-side reserve, or whose price rounds the output to zero, is skipped
//     uncharged;
//   - it errors with ErrInsufficientLiquidity when the book runs dry and with
//     ErrSwapMovementExceeded when the walk would leave the movement bound — both are execution
//     reverts on-chain (the view-quote's partial fill would over-promise a reverting route).
func (s *PoolSimulator) traverse(amountIn *uint256.Int, swapForY bool) (*uint256.Int, *uint256.Int, SwapInfo, error) {
	scaleIn, scaleOut := s.scaleX, s.scaleY
	if !swapForY {
		scaleIn, scaleOut = s.scaleY, s.scaleX
	}

	entryBinID := s.activeID

	// Derive the working volatility and window anchor (PairSwapLib.performSwap lines 81–147).
	vol := getDecayedVolatility(s.volAcc, s.lastVolUpd, uint64(s.fp.FilterPeriod), uint64(s.fp.DecayPeriod), s.trackedAt)
	var anchor uint32
	var volBase uint64
	var sinceVolUpd uint64
	if s.trackedAt > s.lastVolUpd {
		sinceVolUpd = s.trackedAt - s.lastVolUpd
	}
	if s.volRef < minActiveBin || sinceVolUpd >= uint64(s.fp.FilterPeriod) {
		anchor, volBase = entryBinID, vol
	} else {
		anchor = s.volRef
		if disp := distanceFrom(entryBinID, anchor); vol > disp {
			volBase = vol - disp
		} else {
			volBase = 0
		}
	}

	currentBinID := entryBinID
	amountInLeft := new(uint256.Int).Set(amountIn)
	amountOutNormalized := uint256.NewInt(0)
	totalFee := uint256.NewInt(0)
	crossed := false

	updates := make(map[int]binUpdate)

	for !amountInLeft.IsZero() {
		feeRate := calculateDynamicFee(s.fp.BaseFactor, s.fp.VariableFeeControl, vol, s.binStep, s.variableFeeCap)
		binTotalFee, err := getFeeAmountFrom(amountInLeft, feeRate)
		if err != nil {
			return nil, nil, SwapInfo{}, err
		}
		amountInAfterFee := new(uint256.Int).Sub(amountInLeft, binTotalFee)
		amountInAfterFeeNormalized, over := new(uint256.Int).MulOverflow(amountInAfterFee, scaleIn)
		if over {
			return nil, nil, SwapInfo{}, ErrMathOverflow
		}

		if idx, ok := s.binIndex[currentBinID]; ok {
			b := s.workingBin(updates, idx)
			binReserveOut := b.ReserveY
			if !swapForY {
				binReserveOut = b.ReserveX
			}
			if binReserveOut != nil && !binReserveOut.IsZero() { // hasLiquidityForSwap
				price := getNormalizedPriceFromId(currentBinID, s.binStep)
				if !isNormalizedPriceRepresentable(price) {
					return nil, nil, SwapInfo{}, ErrPriceNotRepresentable
				}

				binAmountOut, consumedNorm, err := calculateSwapWithinBin(
					binReserveOut, amountInAfterFeeNormalized, price, s.priceDenominator, scaleIn, scaleOut, swapForY)
				if err != nil {
					return nil, nil, SwapInfo{}, err
				}

				if !binAmountOut.IsZero() {
					amountOutNormalized.Add(amountOutNormalized, binAmountOut)

					// Fee proration to what was actually consumed; the unspent fee slice returns to
					// the gross remainder (PairSwapLib._swapCurrentBin).
					actualConsumed := new(uint256.Int).Div(consumedNorm, scaleIn)
					actualTotalFee := uint256.NewInt(0)
					if actualConsumed.Sign() > 0 && amountInAfterFee.Sign() > 0 {
						actualTotalFee.Mul(binTotalFee, actualConsumed)
						actualTotalFee.Div(actualTotalFee, amountInAfterFee)
					}
					totalFee.Add(totalFee, actualTotalFee)

					netLeftNorm := new(uint256.Int).Sub(amountInAfterFeeNormalized, consumedNorm)
					amountInLeft = new(uint256.Int).Div(netLeftNorm, scaleIn)
					amountInLeft.Add(amountInLeft, new(uint256.Int).Sub(binTotalFee, actualTotalFee))

					s.recordBinUpdate(updates, idx, binAmountOut, consumedNorm, swapForY)
				}
			}
		}

		if amountInLeft.IsZero() {
			break
		}

		nextBin, found := s.findNextActiveBin(currentBinID, swapForY)
		if !found {
			return nil, nil, SwapInfo{}, ErrInsufficientLiquidity
		}
		// SwapMovement.isAllowed: bounded distance from the ENTRY bin of this swap. Execution
		// reverts LBPair__SwapBinDistanceExceeded past it.
		dist := distanceFrom(nextBin, entryBinID)
		if dist > maxBinIDDistance || dist*uint64(s.binStep) > maxCumulativeStepBps {
			return nil, nil, SwapInfo{}, ErrSwapMovementExceeded
		}
		currentBinID = nextBin
		crossed = true
		vol = min(volBase+distanceFrom(nextBin, anchor), uint64(s.fp.MaxVolatilityAccumulator))
	}

	amountOut := new(uint256.Int).Div(amountOutNormalized, scaleOut)

	return amountOut, totalFee, SwapInfo{
		newActiveID: currentBinID,
		binUpdates:  lo.Values(updates),
		binsCrossed: crossed,
		newVolAcc:   vol,
		newVolRef:   anchor,
	}, nil
}

// workingBin returns the current (possibly already-updated) reserves of bin at index idx.
func (s *PoolSimulator) workingBin(updates map[int]binUpdate, idx int) Bin {
	if u, ok := updates[idx]; ok {
		return Bin{ID: s.bins[idx].ID, ReserveX: u.reserveX, ReserveY: u.reserveY}
	}
	return s.bins[idx]
}

// recordBinUpdate applies a bin's post-swap reserves (output reduced, input increased by the net
// consumed) so UpdateBalance can replay them without recomputing. Fees never enter bin reserves in
// V2 — only the post-fee consumed input does.
func (s *PoolSimulator) recordBinUpdate(updates map[int]binUpdate, idx int, outDelta, inDeltaNorm *uint256.Int, swapForY bool) {
	cur := s.workingBin(updates, idx)
	newX := new(uint256.Int).Set(orZero(cur.ReserveX))
	newY := new(uint256.Int).Set(orZero(cur.ReserveY))
	if swapForY {
		newY.Sub(newY, outDelta)
		newX.Add(newX, inDeltaNorm)
	} else {
		newX.Sub(newX, outDelta)
		newY.Add(newY, inDeltaNorm)
	}
	updates[idx] = binUpdate{index: idx, reserveX: newX, reserveY: newY}
}

// findNextActiveBin returns the next bin with liquidity in the swap direction: selling X for Y
// pushes the price down (descending bin ids), selling Y for X pushes it up (ascending). Unchanged
// from V1 and re-verified against the deployed PairSwapLib.findNextActiveBin bitmap walk.
func (s *PoolSimulator) findNextActiveBin(current uint32, swapForY bool) (uint32, bool) {
	if !swapForY { // Y->X: search up
		i := sort.Search(len(s.bins), func(i int) bool { return s.bins[i].ID > current })
		if i < len(s.bins) {
			return s.bins[i].ID, true
		}
		return 0, false
	}
	// X->Y: search down for the largest ID < current
	i := sort.Search(len(s.bins), func(i int) bool { return s.bins[i].ID >= current })
	if i > 0 {
		return s.bins[i-1].ID, true
	}
	return 0, false
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	si, ok := params.SwapInfo.(SwapInfo)
	if !ok {
		return
	}
	s.activeID = si.newActiveID
	for _, u := range si.binUpdates {
		// Reassign the element (copy-on-write) so cloned states never share these pointers.
		s.bins[u.index] = Bin{ID: s.bins[u.index].ID, ReserveX: u.reserveX, ReserveY: u.reserveY}
	}
	// Volatility persistence mirrors the pair: the reference/clock are only written when at least
	// one bin was crossed (PairSwapLib.performSwap lines 153–159), so a same-bin swap leaves the
	// decay clock untouched.
	if si.binsCrossed {
		s.volAcc = si.newVolAcc
		s.volRef = si.newVolRef
		s.lastVolUpd = s.trackedAt
	}
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.bins = append([]Bin(nil), s.bins...)
	// binIndex maps by ID -> index; indices are stable under copy-on-write reassign, safe to share.
	return &cloned
}

// GetApprovalAddress returns the DLMM Router — the swap entry point KyberSwap's executor approves
// and calls (it transferFroms the input straight to the first pool and enforces the final
// amountOutMin).
func (s *PoolSimulator) GetApprovalAddress(_, _ string) string {
	return s.router
}

func (s *PoolSimulator) GetMetaInfo(_, _ string) any {
	return PoolMeta{BlockNumber: s.Info.BlockNumber, ApprovalAddress: s.router, BinStep: s.binStep}
}

func orZero(v *uint256.Int) *uint256.Int {
	if v == nil {
		return uint256.NewInt(0)
	}
	return v
}

func distanceFrom(a, b uint32) uint64 {
	if a >= b {
		return uint64(a - b)
	}
	return uint64(b - a)
}
