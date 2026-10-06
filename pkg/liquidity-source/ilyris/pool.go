package ilyris

import (
	"fmt"
	"sort"

	"github.com/holiman/uint256"
)

// BinReserves is one bin's settled reserves.
type BinReserves struct {
	ReserveX *uint256.Int
	ReserveY *uint256.Int
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
		// A nil side is an absent side, not an error. Negative is impossible now that the
		// type is unsigned, which is one whole class of malformed input the caller can no
		// longer construct.
		x, y := r.ReserveX, r.ReserveY
		if x == nil {
			x = new(uint256.Int)
		}
		if y == nil {
			y = new(uint256.Int)
		}
		if x.IsZero() && y.IsZero() {
			continue
		}
		s.bins[id] = BinReserves{
			ReserveX: new(uint256.Int).Set(x),
			ReserveY: new(uint256.Int).Set(y),
		}
		s.initializedIDs = append(s.initializedIDs, id)
	}
	sort.Ints(s.initializedIDs)
	return s, nil
}

// Params exposes the configuration, notably ActiveID for computing bins crossed.
func (s *binSimulator) Params() PoolParams { return s.params }

// BaseFeeRate is the flat component, at 1e9 precision.
func (s *binSimulator) BaseFeeRate() *uint256.Int {
	var r uint256.Int
	r.SetUint64(uint64(s.params.SwapFeeBps))
	r.Mul(&r, FeePrecision)
	return r.Div(&r, BPS)
}

// VariableFeeRate is the volatility surcharge, at 1e9 precision.
//
// Ceiling division, matching the contract: the surcharge must never round away to zero,
// because a fee that vanishes under small volatility is a fee an arbitrageur can plan around.
//
// No overflow check, and the bound is why: on chain both variableFeeControl and
// volatilityAccumulator are uint24 and binStep is capped at 1000, so the widest possible
// numerator is 2^24 * (2^24 * 1000)^2, under 2^93. Unsigned wraparound is unreachable here.
func (s *binSimulator) VariableFeeRate(volatilityAccumulator *uint256.Int) *uint256.Int {
	if s.params.VariableFeeControl == 0 {
		return new(uint256.Int)
	}
	var term, num uint256.Int
	term.SetUint64(uint64(s.params.BinStepBps))
	term.Mul(volatilityAccumulator, &term)

	num.SetUint64(uint64(s.params.VariableFeeControl))
	num.Mul(&num, &term)
	num.Mul(&num, &term)
	num.Add(&num, variableFeeScale)
	num.SubUint64(&num, 1)
	return num.Div(&num, variableFeeScale)
}

// TotalFeeRate is base + variable, capped, at 1e9 precision.
func (s *binSimulator) TotalFeeRate() *uint256.Int {
	var volAcc uint256.Int
	volAcc.SetUint64(uint64(s.params.VolatilityAccumulator))

	rate := new(uint256.Int).Add(s.BaseFeeRate(), s.VariableFeeRate(&volAcc))
	if rate.Gt(MaxFeeRate) {
		return new(uint256.Int).Set(MaxFeeRate)
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
			if !s.bins[id].ReserveY.IsZero() {
				return id, true
			}
		}
		return 0, false
	}
	// Smallest initialized id >= fromID, walking up.
	i := sort.SearchInts(s.initializedIDs, fromID)
	for ; i < len(s.initializedIDs); i++ {
		id := s.initializedIDs[i]
		if !s.bins[id].ReserveX.IsZero() {
			return id, true
		}
	}
	return 0, false
}

// BinFill is the net reserve movement of one bin during a quote. UpdateBalance
// applies these so split/multi-hop re-quotes see the remaining book, not the
// pre-swap one. Amounts are NET of fee: the fee is taken once on input before
// any bin is touched, matching BinPool.
type BinFill struct {
	ID         int
	AmountXIn  *uint256.Int
	AmountXOut *uint256.Int
	AmountYIn  *uint256.Int
	AmountYOut *uint256.Int
}

// ExactInQuote mirrors BinPool.quoteExactIn.
type ExactInQuote struct {
	AmountOut *uint256.Int
	FeeAmount *uint256.Int
	FinalID   int
	Fills     []BinFill
}

// ExactOutQuote mirrors BinPool.quoteExactOut.
type ExactOutQuote struct {
	AmountIn    *uint256.Int
	FeeAmount   *uint256.Int
	NetAmountIn *uint256.Int
	FinalID     int
}

// netOfFee applies the contract's `amount * (FEE_PRECISION - rate) / FEE_PRECISION`.
//
// The multiplication is checked rather than folded into a mulDiv, and the distinction is
// deliberate: BinPool multiplies and then divides in uint256, so an intermediate product that
// does not fit is a revert on chain. A 512-bit mulDiv would succeed where the contract fails
// and hand the router a quote it cannot execute.
func netOfFee(amount, feeDenominator *uint256.Int) (*uint256.Int, error) {
	var product uint256.Int
	if _, overflow := product.MulOverflow(amount, feeDenominator); overflow {
		return nil, fmt.Errorf("%w: fee multiplication", ErrOverflow)
	}
	return product.Div(&product, FeePrecision), nil
}

// QuoteExactIn prices a sell of amountIn. Mirror of BinPool.quoteExactIn / _quoteOnly.
func (s *binSimulator) QuoteExactIn(xForY bool, amountIn *uint256.Int) (*ExactInQuote, error) {
	if amountIn.IsZero() {
		return &ExactInQuote{
			AmountOut: new(uint256.Int), FeeAmount: new(uint256.Int), FinalID: s.params.ActiveID,
		}, nil
	}

	feeDenominator := new(uint256.Int).Sub(FeePrecision, s.TotalFeeRate())
	netIn, err := netOfFee(amountIn, feeDenominator)
	if err != nil {
		return nil, err
	}
	feeAmount := new(uint256.Int).Sub(amountIn, netIn)

	// A gross input whose fee rounds the net to zero settles nowhere, so the trade stays in
	// the active bin. Both the contract and the TypeScript port return activeId here; a port
	// that returned 0 would report a nonsensical final bin for dust.
	if netIn.IsZero() {
		return &ExactInQuote{AmountOut: new(uint256.Int), FeeAmount: feeAmount, FinalID: s.params.ActiveID}, nil
	}

	remaining := new(uint256.Int).Set(netIn)
	amountOut := new(uint256.Int)
	cursor := s.params.ActiveID
	finalID := s.params.ActiveID
	var fills []BinFill

	for !remaining.IsZero() {
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

		var maxIn *uint256.Int
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
		if maxIn.IsZero() {
			if xForY {
				cursor = id - 1
			} else {
				cursor = id + 1
			}
			continue
		}

		consumed := remaining
		if !remaining.Lt(maxIn) {
			consumed = maxIn
		}

		var out *uint256.Int
		if consumed.Eq(maxIn) {
			// Exact-fill uses the bin's whole reserve rather than recomputing, so the
			// rounding that produced maxIn cannot pay out a unit more than the bin holds.
			out = new(uint256.Int).Set(availableOut)
		} else if xForY {
			out, err = QuoteFromX(consumed, price, s.params.DecimalsX, s.params.DecimalsY)
		} else {
			out, err = XFromQuote(consumed, price, s.params.DecimalsX, s.params.DecimalsY)
		}
		if err != nil {
			return nil, err
		}

		// Checked: the running total is the one place a long traversal could accumulate past
		// uint256, and unsigned addition wraps silently where big.Int simply grew.
		var summed uint256.Int
		if _, overflow := summed.AddOverflow(amountOut, out); overflow {
			return nil, fmt.Errorf("%w: exact-in amountOut", ErrOverflow)
		}
		amountOut = &summed

		fill := BinFill{ID: id}
		if xForY {
			fill.AmountXIn = new(uint256.Int).Set(consumed)
			fill.AmountYOut = new(uint256.Int).Set(out)
		} else {
			fill.AmountYIn = new(uint256.Int).Set(consumed)
			fill.AmountXOut = new(uint256.Int).Set(out)
		}
		fills = append(fills, fill)

		remaining = new(uint256.Int).Sub(remaining, consumed)

		if remaining.IsZero() {
			cursor = id
		} else if xForY {
			cursor = id - 1
		} else {
			cursor = id + 1
		}
		finalID = id
	}

	return &ExactInQuote{AmountOut: amountOut, FeeAmount: feeAmount, FinalID: finalID, Fills: fills}, nil
}

// QuoteExactOut prices a buy of amountOut. Mirror of BinPool.quoteExactOut.
func (s *binSimulator) QuoteExactOut(xForY bool, amountOut *uint256.Int) (*ExactOutQuote, error) {
	if amountOut.IsZero() {
		return &ExactOutQuote{
			AmountIn: new(uint256.Int), FeeAmount: new(uint256.Int),
			NetAmountIn: new(uint256.Int), FinalID: s.params.ActiveID,
		}, nil
	}

	remainingOut := new(uint256.Int).Set(amountOut)
	netAmountIn := new(uint256.Int)
	cursor := s.params.ActiveID
	finalID := s.params.ActiveID

	for !remainingOut.IsZero() {
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
		if !remainingOut.Lt(availableOut) {
			takenOut = availableOut
		}

		price, err := PriceFromID(s.params.BinStepBps, id)
		if err != nil {
			return nil, err
		}

		// Rounding UP on the required input: the pool must never be short-changed on a
		// buy, and a floor here would let a caller extract a unit for free per bin.
		var requiredIn *uint256.Int
		if xForY {
			requiredIn, err = XFromQuoteUp(takenOut, price, s.params.DecimalsX, s.params.DecimalsY)
		} else {
			requiredIn, err = QuoteFromXUp(takenOut, price, s.params.DecimalsX, s.params.DecimalsY)
		}
		if err != nil {
			return nil, err
		}

		var summed uint256.Int
		if _, overflow := summed.AddOverflow(netAmountIn, requiredIn); overflow {
			return nil, fmt.Errorf("%w: exact-out netAmountIn", ErrOverflow)
		}
		netAmountIn = &summed
		remainingOut = new(uint256.Int).Sub(remainingOut, takenOut)

		if remainingOut.IsZero() {
			cursor = id
		} else if xForY {
			cursor = id - 1
		} else {
			cursor = id + 1
		}
		finalID = id
	}

	feeDenominator := new(uint256.Int).Sub(FeePrecision, s.TotalFeeRate())

	var grossed uint256.Int
	if _, overflow := grossed.MulOverflow(netAmountIn, FeePrecision); overflow {
		return nil, fmt.Errorf("%w: exact-out fee inversion", ErrOverflow)
	}
	amountIn, err := ceilDiv(&grossed, feeDenominator)
	if err != nil {
		return nil, err
	}

	// Fee is derived by re-applying the forward formula rather than as (amountIn -
	// netAmountIn). The ceil above means those two differ by a unit at some sizes, and the
	// contract reports this one.
	back, err := netOfFee(amountIn, feeDenominator)
	if err != nil {
		return nil, err
	}
	feeAmount := new(uint256.Int).Sub(amountIn, back)

	return &ExactOutQuote{
		AmountIn: amountIn, FeeAmount: feeAmount,
		NetAmountIn: netAmountIn, FinalID: finalID,
	}, nil
}
