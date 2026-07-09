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

	binStep    uint16
	decimalsY  uint8
	scaleX     *uint256.Int // 10^(18-decimalsX)
	scaleY     *uint256.Int // 10^(18-decimalsY)
	precisionX *uint256.Int // 10^decimalsX

	activeID uint32
	bins     []Bin          // sorted ascending by ID; normalized reserves
	binIndex map[uint32]int // bin ID -> index into bins

	fp             FeeParameters
	variableFeeCap uint16
	startVol       *uint256.Int // volatility accumulator, already decayed to the tracked block
	volRef         uint32       // volatility reference bin (== activeId when idle)
	minSwap        *uint256.Int // precomputed _getMinSwapForVolatility threshold (native units)
	router         string       // DLMM Router: the token spender / approval target
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

	s := &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:     ep.Address,
			Exchange:    ep.Exchange,
			Type:        ep.Type,
			Tokens:      lo.Map(ep.Tokens, func(t *entity.PoolToken, _ int) string { return t.Address }),
			Reserves:    lo.Map(ep.Reserves, func(r string, _ int) *big.Int { return bignumber.NewBig(r) }),
			BlockNumber: ep.BlockNumber,
		}},
		binStep:        static.BinStep,
		decimalsY:      static.DecimalsY,
		scaleX:         pow10(18 - static.DecimalsX),
		scaleY:         pow10(18 - static.DecimalsY),
		precisionX:     pow10(static.DecimalsX),
		activeID:       extra.ActiveID,
		bins:           bins,
		binIndex:       binIndex,
		fp:             extra.FeeParameters,
		variableFeeCap: extra.VariableFeeCap,
		startVol:       uint256.NewInt(extra.VolatilityAccumulator),
		volRef:         extra.VolatilityReference,
		minSwap:        minSwapForVolatility(extra.NativeReserveX, extra.NativeReserveY, extra.FeeParameters.MinSwapBps),
		router:         static.Router,
	}
	return s, nil
}

// minSwapForVolatility mirrors the DEPLOYED _getMinSwapForVolatility:
// (nativeReserveX + nativeReserveY) * minSwapBps / 10000 (no bin-step factor). Returns 0 when
// minSwapBps is 0 (matching the deployed early return).
func minSwapForVolatility(nativeX, nativeY string, minSwapBps uint16) *uint256.Int {
	if minSwapBps == 0 {
		return uint256.NewInt(0)
	}
	total := new(uint256.Int)
	if x, err := uint256.FromDecimal(nativeX); err == nil {
		total.Add(total, x)
	}
	if y, err := uint256.FromDecimal(nativeY); err == nil {
		total.Add(total, y)
	}
	total.Mul(total, uint256.NewInt(uint64(minSwapBps)))
	return total.Div(total, uBP)
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
		return nil, ErrInsufficientOutput
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{Token: param.TokenOut, Amount: amountOut.ToBig()},
		Fee:            &pool.TokenAmount{Token: param.TokenAmountIn.Token, Amount: fee.ToBig()},
		Gas:            defaultGas + int64(len(swapInfo.binUpdates))*gasPerBin,
		SwapInfo:       swapInfo,
	}, nil
}

// traverse reproduces the deployed PairViewer.quoteSwap loop: it starts from the tracked
// (already-decayed) volatility, ramps it by distance from volRef as bins are crossed, and applies
// the dynamic fee per bin. Returns the native output, native total fee, and post-swap bin updates.
func (s *PoolSimulator) traverse(amountIn *uint256.Int, swapForY bool) (*uint256.Int, *uint256.Int, SwapInfo, error) {
	scaleIn, scaleOut := s.scaleX, s.scaleY
	if !swapForY {
		scaleIn, scaleOut = s.scaleY, s.scaleX
	}

	currentBinID := s.activeID
	amountInLeft := new(uint256.Int).Set(amountIn)
	amountOutNormalized := uint256.NewInt(0)
	totalFee := uint256.NewInt(0)
	volatility := new(uint256.Int).Set(s.startVol) // already decayed to the tracked block
	var binsTraversed int

	updates := make(map[int]binUpdate)

	for !amountInLeft.IsZero() {
		feeRate := calculateDynamicFee(s.fp.BaseFactor, s.fp.VariableFeeControl, volatility, s.binStep, s.variableFeeCap)
		binTotalFee := getFeeAmountFrom(amountInLeft, feeRate)
		amountInAfterFee := new(uint256.Int).Sub(amountInLeft, binTotalFee)
		amountInAfterFeeNormalized := new(uint256.Int).Mul(amountInAfterFee, scaleIn)

		if idx, ok := s.binIndex[currentBinID]; ok {
			b := s.workingBin(updates, idx)
			binReserveOut := b.ReserveY
			if !swapForY {
				binReserveOut = b.ReserveX
			}
			if binReserveOut != nil && !binReserveOut.IsZero() { // hasLiquidityForSwap
				price, err := getPriceFromId(currentBinID, s.binStep, s.decimalsY)
				if err != nil {
					return nil, nil, SwapInfo{}, err
				}
				if price.IsZero() {
					return nil, nil, SwapInfo{}, ErrInvalidPrice
				}

				binAmountOut, amountInLeftNorm := simulateBinSwap(
					binReserveOut, amountInAfterFeeNormalized, price, s.precisionX, scaleIn, scaleOut, swapForY)

				amountOutNormalized.Add(amountOutNormalized, binAmountOut)

				actualConsumedNorm := new(uint256.Int).Sub(amountInAfterFeeNormalized, amountInLeftNorm)
				actualConsumed := new(uint256.Int).Div(actualConsumedNorm, scaleIn)

				actualTotalFee := uint256.NewInt(0)
				if actualConsumed.Sign() > 0 && amountInAfterFee.Sign() > 0 {
					actualTotalFee.Mul(binTotalFee, actualConsumed)
					actualTotalFee.Div(actualTotalFee, amountInAfterFee)
				}
				totalFee.Add(totalFee, actualTotalFee)

				amountInLeftAfterSwap := new(uint256.Int).Div(amountInLeftNorm, scaleIn)
				leftoverFee := new(uint256.Int).Sub(binTotalFee, actualTotalFee)
				amountInLeft.Add(amountInLeftAfterSwap, leftoverFee)

				s.recordBinUpdate(updates, idx, binAmountOut, actualConsumedNorm, swapForY)
			}
		}

		if amountInLeft.IsZero() {
			break
		}

		nextBin, found := s.findNextActiveBin(currentBinID, swapForY)
		if !found {
			break // out of tracked liquidity -> partial quote, as on-chain runs out of bins
		}
		currentBinID = nextBin
		binsTraversed++

		// _wouldUpdateVolatility: outside the filter period (snapshot model), the gate reduces to
		// amountIn >= minSwap && a fee was charged. Ramp volatility by distance from the reference.
		if amountIn.Cmp(s.minSwap) >= 0 && totalFee.Sign() > 0 {
			dist := distanceFrom(currentBinID, s.volRef)
			if dist > uint64(s.fp.MaxVolatilityAccumulator) {
				dist = uint64(s.fp.MaxVolatilityAccumulator)
			}
			volatility = uint256.NewInt(dist)
		}
	}

	amountOut := new(uint256.Int).Div(amountOutNormalized, scaleOut)

	return amountOut, totalFee, SwapInfo{
		newActiveID: currentBinID,
		binUpdates:  lo.Values(updates),
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
// consumed) so UpdateBalance can replay them without recomputing.
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

// findNextActiveBin returns the next non-empty bin in the swap direction. Direction is set from
// observed on-chain behaviour (verified against the pair's quoteSwap): selling X for Y pushes the
// price down (descending bin ids), selling Y for X pushes it up (ascending). Y lives in bins below
// the active bin, X above — so swapForY (X->Y, taking Y out) walks DOWN and !swapForY walks UP.
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
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	cloned.bins = append([]Bin(nil), s.bins...)
	// binIndex maps by ID -> index; indices are stable under copy-on-write reassign, safe to share.
	return &cloned
}

// GetApprovalAddress returns the DLMM Router — the swap entry point KyberSwap's executor approves
// and calls. The pair itself is a BeaconProxy that reverts on direct swap calls ("No extension"), so
// the pair address is never a valid spender.
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
