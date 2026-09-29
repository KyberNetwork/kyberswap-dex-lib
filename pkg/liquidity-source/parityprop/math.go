package parityprop

import "github.com/holiman/uint256"

// priceScale is PmmPool.sol's PRICE_SCALE constant (1e8) -- the fixed-point
// scale the oracle's bid/mid/ask are quoted in.
var priceScale = uint256.NewInt(1e8)

// bpsDenominator is PmmPool.sol's BPS constant (10_000), the fee/spread
// basis-point denominator.
var bpsDenominator = uint256.NewInt(bps)

// QuoteParams mirrors the values PmmPool._quote() reads from oracle/pool
// state for a single quote. Paused and zero-amount are checked by the
// caller (pool_simulator.go) before this is invoked; everything from the
// oracle-sanity check onward lives here, in the same order as _quote().
type QuoteParams struct {
	// TokenInIsBase is true when selling base for quote (fills at Bid);
	// false when buying base with quote (fills at Ask).
	TokenInIsBase bool
	AmountIn      *uint256.Int

	Bid, Mid, Ask  uint64
	PriceUpdatedAt uint64
	NowTimestamp   uint64
	MaxStaleness   uint32
	MaxSpreadBps   uint16
	FeeBps         uint16

	BaseScale, QuoteScale *uint256.Int

	MaxSwapNotional, MaxBlockNotional *uint256.Int
	// AccumulatedBlockNotional is the notional already spent in the current
	// block (0 if the last swap was in a different block). The caller
	// decides whether to reset or accumulate; this only adds Notional to
	// whatever it's given.
	AccumulatedBlockNotional *uint256.Int

	OutBalance      *uint256.Int
	OutReserveFloor *uint256.Int
}

// QuoteResult is what CalcAmountOut needs to build pool.CalcAmountOutResult
// and SwapInfo.
type QuoteResult struct {
	AmountOut *uint256.Int
	Fee       *uint256.Int
	Notional  *uint256.Int
}

// Quote ports PmmPool.sol's _quote() from the oracle-sanity check onward.
// Guard order, operation grouping, and rounding (integer division = floor,
// same as Solidity) match the source exactly. Overflow is checked at each
// individual multiplication/addition, matching Solidity 0.8's checked
// arithmetic (panic 0x11), rather than using a wide-precision mulDiv that
// could produce an answer the real contract would never reach.
func Quote(p QuoteParams) (*QuoteResult, error) {
	bid := uint256.NewInt(p.Bid)
	mid := uint256.NewInt(p.Mid)
	ask := uint256.NewInt(p.Ask)

	// if (bid == 0 || bid > mid || mid > ask) revert InvalidOraclePrices();
	if bid.IsZero() || bid.Cmp(mid) > 0 || mid.Cmp(ask) > 0 {
		return nil, ErrInvalidOraclePrices
	}

	// if (block.timestamp > uint256(priceUpdatedAt) + p.maxStaleness) revert StalePrice();
	// PriceUpdatedAt (uint64) + MaxStaleness (uint32) can never overflow uint256, so a plain Add is safe here.
	staleAt := new(uint256.Int).Add(uint256.NewInt(p.PriceUpdatedAt), uint256.NewInt(uint64(p.MaxStaleness)))
	if uint256.NewInt(p.NowTimestamp).Cmp(staleAt) > 0 {
		return nil, ErrStalePrice
	}

	// if (uint256(ask - bid) * BPS > uint256(p.maxSpreadBps) * mid) revert SpreadTooWide();
	// Both products fit well within uint256, so plain Muls are safe here too.
	askMinusBid := new(uint256.Int).Sub(ask, bid) // ask >= mid >= bid already established above, so this never wraps
	spreadLHS := new(uint256.Int).Mul(askMinusBid, bpsDenominator)
	spreadRHS := new(uint256.Int).Mul(uint256.NewInt(uint64(p.MaxSpreadBps)), mid)
	if spreadLHS.Cmp(spreadRHS) > 0 {
		return nil, ErrSpreadTooWide
	}

	var gross, notional uint256.Int
	if p.TokenInIsBase {
		// gross = amountIn * bid * quoteScale / (PRICE_SCALE * baseScale);
		g, err := mulMulDiv(p.AmountIn, bid, p.QuoteScale, priceScale, p.BaseScale)
		if err != nil {
			return nil, err
		}
		gross = *g
		notional = gross
	} else {
		// gross = amountIn * PRICE_SCALE * baseScale / (ask * quoteScale);
		g, err := mulMulDiv(p.AmountIn, priceScale, p.BaseScale, ask, p.QuoteScale)
		if err != nil {
			return nil, err
		}
		gross = *g
		notional = *p.AmountIn
	}

	// if (notional > p.maxSwapNotional) revert SwapTooLarge();
	if notional.Cmp(p.MaxSwapNotional) > 0 {
		return nil, ErrSwapTooLarge
	}

	// uint256 accumulated = ...; if (accumulated + notional > p.maxBlockNotional) revert BlockCapExceeded();
	accumulated := p.AccumulatedBlockNotional
	if accumulated == nil {
		accumulated = new(uint256.Int)
	}
	var totalBlockNotional uint256.Int
	if _, overflow := totalBlockNotional.AddOverflow(accumulated, &notional); overflow {
		return nil, ErrOverflow
	}
	if totalBlockNotional.Cmp(p.MaxBlockNotional) > 0 {
		return nil, ErrBlockCapExceeded
	}

	// fee = gross * p.feeBps / BPS; amountOut = gross - fee;
	var feeNumerator uint256.Int
	if _, overflow := feeNumerator.MulOverflow(&gross, uint256.NewInt(uint64(p.FeeBps))); overflow {
		return nil, ErrOverflow
	}
	fee := new(uint256.Int).Div(&feeNumerator, bpsDenominator)
	// feeBps <= BPS is enforced admin-side (_setParams' MAX_FEE_BPS=500 bound), so fee <= gross always.
	amountOut := new(uint256.Int).Sub(&gross, fee)

	// if (amountOut == 0) revert ZeroAmount();
	if amountOut.IsZero() {
		return nil, ErrZeroAmount
	}

	// if (outBalance < amountOut + floor) revert InsufficientReserve();
	var floorSum uint256.Int
	if _, overflow := floorSum.AddOverflow(amountOut, p.OutReserveFloor); overflow {
		return nil, ErrOverflow
	}
	if p.OutBalance.Cmp(&floorSum) < 0 {
		return nil, ErrInsufficientReserve
	}

	notionalCopy := notional
	return &QuoteResult{AmountOut: amountOut, Fee: fee, Notional: &notionalCopy}, nil
}

// mulMulDiv computes floor(a*b*c / (d*e)), checking overflow at each
// individual multiplication in the same left-to-right order Solidity
// evaluates `a * b * c / (d * e)`. It deliberately doesn't use a
// wide-precision mulDiv: an input that overflows any intermediate product
// reverts on-chain, and the simulator needs to fail the same way.
func mulMulDiv(a, b, c, d, e *uint256.Int) (*uint256.Int, error) {
	var step1, step2, denom uint256.Int

	if _, overflow := step1.MulOverflow(a, b); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := step2.MulOverflow(&step1, c); overflow {
		return nil, ErrOverflow
	}
	if _, overflow := denom.MulOverflow(d, e); overflow {
		return nil, ErrOverflow
	}
	if denom.IsZero() {
		return nil, ErrOverflow
	}
	return new(uint256.Int).Div(&step2, &denom), nil
}
