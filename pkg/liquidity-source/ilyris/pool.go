package ilyris

import (
	"fmt"
	"math/big"
	"sort"
)

// BinReserves is one bin's settled reserves.
type BinReserves struct {
	ReserveX *big.Int
	ReserveY *big.Int
}

// PoolParams is the static configuration plus the two values a swap moves.
//
// VolatilityAccumulator is the ALREADY-DECAYED value the NEXT swap prices from, not the raw
// stored one. Reading the stored value without applying decay overstates the fee, which
// makes this pool look worse than it is and quietly loses routes.
type PoolParams struct {
	BinStepBps            int
	SwapFeeBps            int
	ActiveID              int
	DecimalsX             int
	DecimalsY             int
	VariableFeeControl    int
	VolatilityAccumulator int
}

// PoolSimulator prices swaps against a snapshot of the book.
//
// Quoting does not mutate it, so one instance may be reused across the array of amounts
// KyberSwap asks about. The traversal is stateful within a single quote only.
type binSimulator struct {
	params PoolParams
	bins   map[int]BinReserves
	// initializedIDs is kept sorted ascending. Bin traversal is a walk outward from the
	// active bin, so an ordered index is what makes it O(log n) to enter and O(1) to step
	// rather than a scan of the whole book per bin.
	initializedIDs []int
}

// NewPoolSimulator builds a simulator over the given bins. Bins with no reserves on either
// side are dropped: they cannot be a source of output and carrying them would make every
// traversal step over dead entries.
func newBinSimulator(params PoolParams, bins map[int]BinReserves) (*binSimulator, error) {
	if params.BinStepBps <= 0 || params.BinStepBps > 1000 {
		return nil, fmt.Errorf("ilyris: invalid bin step %d", params.BinStepBps)
	}
	if params.DecimalsX < 0 || params.DecimalsX > 18 || params.DecimalsY < 0 || params.DecimalsY > 18 {
		return nil, fmt.Errorf("ilyris: invalid decimals %d/%d", params.DecimalsX, params.DecimalsY)
	}
	if params.ActiveID < MinBinID || params.ActiveID > MaxBinID {
		return nil, fmt.Errorf("ilyris: activeId %d out of range", params.ActiveID)
	}

	s := &binSimulator{
		params:         params,
		bins:           make(map[int]BinReserves, len(bins)),
		initializedIDs: make([]int, 0, len(bins)),
	}
	for id, r := range bins {
		if id < MinBinID || id > MaxBinID {
			return nil, fmt.Errorf("ilyris: bin id %d out of range", id)
		}
		x, y := r.ReserveX, r.ReserveY
		if x == nil {
			x = new(big.Int)
		}
		if y == nil {
			y = new(big.Int)
		}
		if x.Sign() < 0 || y.Sign() < 0 {
			return nil, fmt.Errorf("ilyris: bin %d has negative reserves", id)
		}
		if x.Sign() == 0 && y.Sign() == 0 {
			continue
		}
		s.bins[id] = BinReserves{ReserveX: new(big.Int).Set(x), ReserveY: new(big.Int).Set(y)}
		s.initializedIDs = append(s.initializedIDs, id)
	}
	sort.Ints(s.initializedIDs)
	return s, nil
}

// Params exposes the configuration, notably ActiveID for computing bins crossed.
func (s *binSimulator) Params() PoolParams { return s.params }

// BaseFeeRate is the flat component, at 1e9 precision.
func (s *binSimulator) BaseFeeRate() *big.Int {
	r := new(big.Int).Mul(big.NewInt(int64(s.params.SwapFeeBps)), FeePrecision)
	return r.Div(r, BPS)
}

// VariableFeeRate is the volatility surcharge, at 1e9 precision.
//
// Ceiling division, matching the contract: the surcharge must never round away to zero,
// because a fee that vanishes under small volatility is a fee an arbitrageur can plan around.
func (s *binSimulator) VariableFeeRate(volatilityAccumulator *big.Int) *big.Int {
	if s.params.VariableFeeControl == 0 {
		return new(big.Int)
	}
	term := new(big.Int).Mul(volatilityAccumulator, big.NewInt(int64(s.params.BinStepBps)))
	num := new(big.Int).Mul(big.NewInt(int64(s.params.VariableFeeControl)), term)
	num.Mul(num, term)
	num.Add(num, variableFeeScale)
	num.Sub(num, one)
	return num.Div(num, variableFeeScale)
}

// TotalFeeRate is base + variable, capped, at 1e9 precision.
func (s *binSimulator) TotalFeeRate() *big.Int {
	rate := new(big.Int).Add(
		s.BaseFeeRate(),
		s.VariableFeeRate(big.NewInt(int64(s.params.VolatilityAccumulator))),
	)
	if rate.Cmp(MaxFeeRate) > 0 {
		return new(big.Int).Set(MaxFeeRate)
	}
	return rate
}

// findNextWithOutput returns the next bin from fromId (inclusive) holding output-side
// reserves, walking DOWN for xForY and UP otherwise. Returns false when the book is exhausted.
//
// The direction matters and is easy to invert: selling X consumes Y, and Y sits at and BELOW
// the active bin, so an X-for-Y swap walks down. Reversing this quotes against liquidity that
// is not on the side being bought.
func (s *binSimulator) findNextWithOutput(fromID int, xForY bool) (int, bool) {
	if fromID < MinBinID || fromID > MaxBinID {
		return 0, false
	}
	if xForY {
		// Largest initialized id <= fromID, walking down.
		i := sort.SearchInts(s.initializedIDs, fromID+1) - 1
		for ; i >= 0; i-- {
			id := s.initializedIDs[i]
			if s.bins[id].ReserveY.Sign() > 0 {
				return id, true
			}
		}
		return 0, false
	}
	// Smallest initialized id >= fromID, walking up.
	i := sort.SearchInts(s.initializedIDs, fromID)
	for ; i < len(s.initializedIDs); i++ {
		id := s.initializedIDs[i]
		if s.bins[id].ReserveX.Sign() > 0 {
			return id, true
		}
	}
	return 0, false
}

// ExactInQuote mirrors BinPool.quoteExactIn.
type ExactInQuote struct {
	AmountOut *big.Int
	FeeAmount *big.Int
	FinalID   int
}

// ExactOutQuote mirrors BinPool.quoteExactOut.
type ExactOutQuote struct {
	AmountIn    *big.Int
	FeeAmount   *big.Int
	NetAmountIn *big.Int
	FinalID     int
}

// QuoteExactIn prices a sell of amountIn. Mirror of BinPool.quoteExactIn / _quoteOnly.
func (s *binSimulator) QuoteExactIn(xForY bool, amountIn *big.Int) (*ExactInQuote, error) {
	if err := requireUint256(amountIn, "amountIn"); err != nil {
		return nil, err
	}
	if amountIn.Sign() == 0 {
		return &ExactInQuote{AmountOut: new(big.Int), FeeAmount: new(big.Int), FinalID: s.params.ActiveID}, nil
	}

	feeDenominator := new(big.Int).Sub(FeePrecision, s.TotalFeeRate())
	netIn := new(big.Int).Mul(amountIn, feeDenominator)
	if err := requireUint256(netIn, "exact-in fee multiplication"); err != nil {
		return nil, err
	}
	netIn.Div(netIn, FeePrecision)
	feeAmount := new(big.Int).Sub(amountIn, netIn)

	// A gross input whose fee rounds the net to zero settles nowhere, so the trade stays in
	// the active bin. Both the contract and the TypeScript port return activeId here; a port
	// that returned 0 would report a nonsensical final bin for dust.
	if netIn.Sign() == 0 {
		return &ExactInQuote{AmountOut: new(big.Int), FeeAmount: feeAmount, FinalID: s.params.ActiveID}, nil
	}

	remaining := new(big.Int).Set(netIn)
	amountOut := new(big.Int)
	cursor := s.params.ActiveID
	finalID := s.params.ActiveID

	for remaining.Sign() != 0 {
		id, ok := s.findNextWithOutput(cursor, xForY)
		if !ok {
			return nil, ErrInsufficientLiquidity
		}
		bin := s.bins[id]
		price, err := PriceFromID(s.params.BinStepBps, id)
		if err != nil {
			return nil, err
		}

		availableOut := bin.ReserveY
		if !xForY {
			availableOut = bin.ReserveX
		}

		var maxIn *big.Int
		if xForY {
			maxIn, err = XFromQuoteUp(availableOut, price, s.params.DecimalsX, s.params.DecimalsY)
		} else {
			maxIn, err = QuoteFromXUp(availableOut, price, s.params.DecimalsX, s.params.DecimalsY)
		}
		if err != nil {
			return nil, err
		}

		// A bin whose output is worth less than one raw input unit cannot be entered at
		// all. Skipping it rather than dividing by it is what stops a zero-consumption
		// infinite loop.
		if maxIn.Sign() == 0 {
			if xForY {
				cursor = id - 1
			} else {
				cursor = id + 1
			}
			continue
		}

		consumed := remaining
		if remaining.Cmp(maxIn) >= 0 {
			consumed = maxIn
		}

		var out *big.Int
		if consumed.Cmp(maxIn) == 0 {
			// Exact-fill uses the bin's whole reserve rather than recomputing, so the
			// rounding that produced maxIn cannot pay out a unit more than the bin holds.
			out = new(big.Int).Set(availableOut)
		} else if xForY {
			out, err = QuoteFromX(consumed, price, s.params.DecimalsX, s.params.DecimalsY)
		} else {
			out, err = XFromQuote(consumed, price, s.params.DecimalsX, s.params.DecimalsY)
		}
		if err != nil {
			return nil, err
		}

		amountOut = new(big.Int).Add(amountOut, out)
		if err := requireUint256(amountOut, "exact-in amountOut"); err != nil {
			return nil, err
		}
		remaining = new(big.Int).Sub(remaining, consumed)

		if remaining.Sign() == 0 {
			cursor = id
		} else if xForY {
			cursor = id - 1
		} else {
			cursor = id + 1
		}
		finalID = id
	}

	return &ExactInQuote{AmountOut: amountOut, FeeAmount: feeAmount, FinalID: finalID}, nil
}

// QuoteExactOut prices a buy of amountOut. Mirror of BinPool.quoteExactOut.
func (s *binSimulator) QuoteExactOut(xForY bool, amountOut *big.Int) (*ExactOutQuote, error) {
	if err := requireUint256(amountOut, "amountOut"); err != nil {
		return nil, err
	}
	if amountOut.Sign() == 0 {
		return &ExactOutQuote{
			AmountIn: new(big.Int), FeeAmount: new(big.Int),
			NetAmountIn: new(big.Int), FinalID: s.params.ActiveID,
		}, nil
	}

	remainingOut := new(big.Int).Set(amountOut)
	netAmountIn := new(big.Int)
	cursor := s.params.ActiveID
	finalID := s.params.ActiveID

	for remainingOut.Sign() != 0 {
		id, ok := s.findNextWithOutput(cursor, xForY)
		if !ok {
			return nil, ErrInsufficientLiquidity
		}
		bin := s.bins[id]

		availableOut := bin.ReserveY
		if !xForY {
			availableOut = bin.ReserveX
		}
		takenOut := remainingOut
		if remainingOut.Cmp(availableOut) >= 0 {
			takenOut = availableOut
		}

		price, err := PriceFromID(s.params.BinStepBps, id)
		if err != nil {
			return nil, err
		}

		// Rounding UP on the required input: the pool must never be short-changed on a
		// buy, and a floor here would let a caller extract a unit for free per bin.
		var requiredIn *big.Int
		if xForY {
			requiredIn, err = XFromQuoteUp(takenOut, price, s.params.DecimalsX, s.params.DecimalsY)
		} else {
			requiredIn, err = QuoteFromXUp(takenOut, price, s.params.DecimalsX, s.params.DecimalsY)
		}
		if err != nil {
			return nil, err
		}

		netAmountIn = new(big.Int).Add(netAmountIn, requiredIn)
		if err := requireUint256(netAmountIn, "exact-out netAmountIn"); err != nil {
			return nil, err
		}
		remainingOut = new(big.Int).Sub(remainingOut, takenOut)

		if remainingOut.Sign() == 0 {
			cursor = id
		} else if xForY {
			cursor = id - 1
		} else {
			cursor = id + 1
		}
		finalID = id
	}

	feeDenominator := new(big.Int).Sub(FeePrecision, s.TotalFeeRate())
	grossed := new(big.Int).Mul(netAmountIn, FeePrecision)
	if err := requireUint256(grossed, "exact-out fee inversion"); err != nil {
		return nil, err
	}
	amountIn := ceilDiv(grossed, feeDenominator)

	// Fee is derived by re-applying the forward formula rather than as (amountIn -
	// netAmountIn). The ceil above means those two differ by a unit at some sizes, and the
	// contract reports this one.
	back := new(big.Int).Mul(amountIn, feeDenominator)
	if err := requireUint256(back, "exact-out fee multiplication"); err != nil {
		return nil, err
	}
	back.Div(back, FeePrecision)
	feeAmount := new(big.Int).Sub(amountIn, back)

	return &ExactOutQuote{
		AmountIn: amountIn, FeeAmount: feeAmount,
		NetAmountIn: netAmountIn, FinalID: finalID,
	}, nil
}
