package parityswapprop

import (
	"testing"

	"github.com/holiman/uint256"
)

// Live on-chain values captured from a real swap: pool
// 0xD778f470c69bCe130D1Cef08852F34Bf296B4E67 (WETH/USDG), oracle
// 0xC484F39B1C25fC7FCB140fbC0824A6ff9143e405, block ~35123 series.
const (
	liveBid              = uint64(190510625160)
	liveMid              = uint64(190584000000)
	liveAsk              = uint64(190657374840)
	liveBaseScale        = "1000000000000000000" // 10**18, WETH
	liveQuoteScale       = "1000000"             // 10**6, USDG
	liveMaxSwapNotional  = "500000000"
	liveMaxBlockNotional = "1500000000"
	liveMinBaseReserve   = "25000000000000000"
	liveMinQuoteReserve  = "50000000"
)

func mustU256(s string) *uint256.Int {
	v, err := uint256.FromDecimal(s)
	if err != nil {
		panic(err)
	}
	return v
}

// baseQuoteParams returns a QuoteParams pre-filled with the live oracle
// price/params, a fresh (non-stale, in-window) timestamp, and no prior
// same-block notional -- individual tests override only what they need to
// exercise.
func baseQuoteParams() QuoteParams {
	return QuoteParams{
		Bid:                      liveBid,
		Mid:                      liveMid,
		Ask:                      liveAsk,
		PriceUpdatedAt:           1_000_000,
		NowTimestamp:             1_000_000,
		MaxStaleness:             8,
		MaxSpreadBps:             50,
		FeeBps:                   0,
		BaseScale:                mustU256(liveBaseScale),
		QuoteScale:               mustU256(liveQuoteScale),
		MaxSwapNotional:          mustU256(liveMaxSwapNotional),
		MaxBlockNotional:         mustU256(liveMaxBlockNotional),
		AccumulatedBlockNotional: uint256.NewInt(0),
		OutBalance:               mustU256("500000000"),
		OutReserveFloor:          mustU256(liveMinQuoteReserve),
	}
}

// TestQuote_BaseForQuote_LiveOnChainMatch reproduces the executed on-chain
// swap: 0.1 WETH in -> 190510625 USDG out, fee=0 (feeBps=0 live). Matches
// getAmountOut/previewSwap, the Swap event, and the pool's reserve delta
// exactly -- see the smoke-test simulation file cited above.
func TestQuote_BaseForQuote_LiveOnChainMatch(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000") // 0.1 WETH

	res, err := Quote(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertU256Eq(t, "amountOut", res.AmountOut, "190510625")
	assertU256Eq(t, "fee", res.Fee, "0")
	assertU256Eq(t, "notional", res.Notional, "190510625")
}

// TestQuote_QuoteForBase_LiveOnChainMatch reproduces the reverse-direction
// view-call sanity check: 100 USDG in -> 52450108517396808 WETH out
// (view-only; independently recomputed from the same formula, not
// executed on-chain).
func TestQuote_QuoteForBase_LiveOnChainMatch(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = false
	p.AmountIn = mustU256("100000000") // 100 USDG
	p.OutBalance = mustU256("250000000000000000")
	p.OutReserveFloor = mustU256(liveMinBaseReserve)

	res, err := Quote(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertU256Eq(t, "amountOut", res.AmountOut, "52450108517396808")
	assertU256Eq(t, "fee", res.Fee, "0")
	assertU256Eq(t, "notional", res.Notional, "100000000") // notional = amountIn on this side
}

// TestQuote_FeeApplied is synthetic (feeBps is 0 live) but exercises the
// fee = gross*feeBps/BPS (floor) / amountOut = gross-fee arithmetic PmmPool.sol
// applies to the output token.
func TestQuote_FeeApplied(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000") // 0.1 WETH, gross=190510625
	p.FeeBps = 30                               // 0.3%

	res, err := Quote(p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// fee = 190510625*30/10000 = 571531 (floor); amountOut = 190510625-571531.
	assertU256Eq(t, "fee", res.Fee, "571531")
	assertU256Eq(t, "amountOut", res.AmountOut, "189939094")
}

func TestQuote_InvalidOraclePrices(t *testing.T) {
	cases := []struct {
		name          string
		bid, mid, ask uint64
	}{
		{"bid zero", 0, liveMid, liveAsk},
		{"bid greater than mid", 200, 100, liveAsk},
		{"mid greater than ask", liveBid, 300, 200},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := baseQuoteParams()
			p.TokenInIsBase = true
			p.AmountIn = mustU256("100000000000000000")
			p.Bid, p.Mid, p.Ask = tc.bid, tc.mid, tc.ask

			_, err := Quote(p)
			if err != ErrInvalidOraclePrices {
				t.Fatalf("want ErrInvalidOraclePrices, got %v", err)
			}
		})
	}
}

func TestQuote_StalePrice(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000")
	p.PriceUpdatedAt = 1000
	p.MaxStaleness = 8
	p.NowTimestamp = 1009 // 1009 > 1000+8

	_, err := Quote(p)
	if err != ErrStalePrice {
		t.Fatalf("want ErrStalePrice, got %v", err)
	}
}

func TestQuote_NotStale_AtExactBoundary(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000")
	p.PriceUpdatedAt = 1000
	p.MaxStaleness = 8
	p.NowTimestamp = 1008 // == updatedAt+maxStaleness, not stale (guard is strictly ">")

	if _, err := Quote(p); err != nil {
		t.Fatalf("unexpected error at exact staleness boundary: %v", err)
	}
}

func TestQuote_SpreadTooWide(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000")
	// bid=100e8, mid=105e8, ask=110e8 -> spread(ask-bid)=10e8.
	p.Bid, p.Mid, p.Ask = 100_00000000, 105_00000000, 110_00000000
	p.MaxSpreadBps = 100 // 1%: (10e8)*10000=1e13 > 100*105e8=1.05e11

	_, err := Quote(p)
	if err != ErrSpreadTooWide {
		t.Fatalf("want ErrSpreadTooWide, got %v", err)
	}
}

func TestQuote_SwapTooLarge(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000") // gross/notional = 190510625
	p.MaxSwapNotional = uint256.NewInt(1)       // far below notional

	_, err := Quote(p)
	if err != ErrSwapTooLarge {
		t.Fatalf("want ErrSwapTooLarge, got %v", err)
	}
}

func TestQuote_BlockCapExceeded(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000") // notional = 190510625
	p.MaxBlockNotional = mustU256("200000000")
	p.AccumulatedBlockNotional = mustU256("100000000") // 100000000+190510625 > 200000000

	_, err := Quote(p)
	if err != ErrBlockCapExceeded {
		t.Fatalf("want ErrBlockCapExceeded, got %v", err)
	}
}

func TestQuote_NilAccumulatedBlockNotional_TreatedAsZero(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000")
	p.AccumulatedBlockNotional = nil // e.g. first swap this block

	if _, err := Quote(p); err != nil {
		t.Fatalf("unexpected error with nil AccumulatedBlockNotional: %v", err)
	}
}

// TestQuote_BaseForQuote_GrossOverflowPropagates covers the base-for-quote
// branch's mulMulDiv error-propagation path (Quote must return the error,
// not swallow it): AmountIn*Bid alone overflows 2**256-1.
func TestQuote_BaseForQuote_GrossOverflowPropagates(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.Bid, p.Mid, p.Ask = 2, 2, 2
	p.MaxSpreadBps = 0
	p.AmountIn = new(uint256.Int).Not(uint256.NewInt(0)) // MaxUint256

	_, err := Quote(p)
	if err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

// TestQuote_QuoteForBase_GrossOverflowPropagates is the same propagation
// check for the quote-for-base branch: AmountIn*PRICE_SCALE alone overflows.
func TestQuote_QuoteForBase_GrossOverflowPropagates(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = false
	p.Bid, p.Mid, p.Ask = 1, 1, 1
	p.MaxSpreadBps = 0
	p.AmountIn = new(uint256.Int).Not(uint256.NewInt(0)) // MaxUint256

	_, err := Quote(p)
	if err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

// TestQuote_ZeroAmountOut covers the amountOut==0 guard: an AmountIn small
// enough that integer division floors gross (and therefore amountOut) to 0.
func TestQuote_ZeroAmountOut(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.Bid, p.Mid, p.Ask = 1, 1, 1
	p.MaxSpreadBps = 0
	p.BaseScale = uint256.NewInt(1_000_000_000_000_000_000) // 1e18
	p.QuoteScale = uint256.NewInt(1)
	p.AmountIn = uint256.NewInt(1) // gross = 1*1*1/(1e8*1e18) = 0

	_, err := Quote(p)
	if err != ErrZeroAmount {
		t.Fatalf("want ErrZeroAmount, got %v", err)
	}
}

func TestQuote_InsufficientReserve(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = true
	p.AmountIn = mustU256("100000000000000000") // amountOut = 190510625
	p.OutBalance = mustU256("100000000")        // < amountOut + floor
	p.OutReserveFloor = uint256.NewInt(0)

	_, err := Quote(p)
	if err != ErrInsufficientReserve {
		t.Fatalf("want ErrInsufficientReserve, got %v", err)
	}
}

// TestQuote_FeeNumeratorOverflow is a synthetic (not achievable with any
// real token's decimals) construction that drives gross to ~2**246 via an
// oversized BaseScale so that gross*feeBps (feeBps at its uint16 max)
// exceeds 2**256-1, exercising the overflow-checked fee computation --
// mirroring that Solidity 0.8 would revert (panic 0x11) rather than wrap.
func TestQuote_FeeNumeratorOverflow(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = false
	p.Bid, p.Mid, p.Ask = 1, 1, 1
	p.MaxSpreadBps = 0
	p.AmountIn = uint256.NewInt(1)
	p.BaseScale = new(uint256.Int).Lsh(uint256.NewInt(1), 220) // 2**220
	p.QuoteScale = uint256.NewInt(1)
	p.MaxSwapNotional = mustU256("1000")
	p.MaxBlockNotional = mustU256("1000")
	p.AccumulatedBlockNotional = uint256.NewInt(0)
	p.FeeBps = 65535 // uint16 max

	_, err := Quote(p)
	if err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

// TestQuote_BlockNotionalAddOverflow drives AccumulatedBlockNotional to
// within a few wei of 2**256-1 so adding even a tiny Notional overflows the
// running block-notional accumulator.
func TestQuote_BlockNotionalAddOverflow(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = false // notional = AmountIn directly on this side
	p.Bid, p.Mid, p.Ask = 1, 1, 1
	p.MaxSpreadBps = 0
	p.AmountIn = uint256.NewInt(10)
	p.BaseScale = uint256.NewInt(1)
	p.QuoteScale = uint256.NewInt(1)
	p.MaxSwapNotional = new(uint256.Int).Not(uint256.NewInt(0)) // MaxUint256, so notional=10 always passes
	p.MaxBlockNotional = new(uint256.Int).Not(uint256.NewInt(0))
	maxU256 := new(uint256.Int).Not(uint256.NewInt(0))
	p.AccumulatedBlockNotional = new(uint256.Int).Sub(maxU256, uint256.NewInt(5)) // MaxUint256-5 + 10 overflows

	_, err := Quote(p)
	if err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

// TestQuote_ReserveFloorAddOverflow constructs an amountOut within a few wei
// of 2**256-1 (via an oversized BaseScale, feeBps=0 so amountOut==gross
// exactly) and a floor chosen so amountOut+floor wraps past 2**256-1.
func TestQuote_ReserveFloorAddOverflow(t *testing.T) {
	p := baseQuoteParams()
	p.TokenInIsBase = false
	p.Bid, p.Mid, p.Ask = 1, 1, 1
	p.MaxSpreadBps = 0
	p.AmountIn = uint256.NewInt(1)
	p.BaseScale = new(uint256.Int).Lsh(uint256.NewInt(1), 229) // just under the mulMulDiv overflow ceiling for this input
	p.QuoteScale = uint256.NewInt(1)
	p.FeeBps = 0
	p.MaxSwapNotional = mustU256("1000")
	p.MaxBlockNotional = mustU256("1000")
	p.AccumulatedBlockNotional = uint256.NewInt(0)

	// Precompute the exact gross this input produces (feeBps=0 => amountOut == gross)
	// using the identical formula, purely to derive a floor that overflows against it.
	expGross, err := mulMulDiv(p.AmountIn, priceScale, p.BaseScale, uint256.NewInt(p.Ask), p.QuoteScale)
	if err != nil {
		t.Fatalf("setup: mulMulDiv unexpectedly failed: %v", err)
	}
	maxU256 := new(uint256.Int).Not(uint256.NewInt(0))
	p.OutReserveFloor = new(uint256.Int).Sub(maxU256, expGross) // amountOut + floor == MaxUint256 exactly...
	p.OutReserveFloor.AddUint64(p.OutReserveFloor, 1)           // ...+1 wraps past it

	_, err = Quote(p)
	if err != ErrOverflow {
		t.Fatalf("want ErrOverflow, got %v", err)
	}
}

func TestMulMulDiv(t *testing.T) {
	maxU256 := new(uint256.Int).Not(uint256.NewInt(0))
	one := uint256.NewInt(1)

	t.Run("step1 overflow", func(t *testing.T) {
		_, err := mulMulDiv(maxU256, uint256.NewInt(2), one, one, one)
		if err != ErrOverflow {
			t.Fatalf("want ErrOverflow, got %v", err)
		}
	})

	t.Run("step2 overflow", func(t *testing.T) {
		half := new(uint256.Int).Rsh(maxU256, 1) // ~2**255, a*b=half*1=half (safe)
		_, err := mulMulDiv(half, one, uint256.NewInt(3), one, one)
		if err != ErrOverflow {
			t.Fatalf("want ErrOverflow, got %v", err)
		}
	})

	t.Run("denom overflow", func(t *testing.T) {
		_, err := mulMulDiv(one, one, one, maxU256, uint256.NewInt(2))
		if err != ErrOverflow {
			t.Fatalf("want ErrOverflow, got %v", err)
		}
	})

	t.Run("denom zero", func(t *testing.T) {
		_, err := mulMulDiv(one, one, one, uint256.NewInt(0), uint256.NewInt(5))
		if err != ErrOverflow {
			t.Fatalf("want ErrOverflow, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		// floor(6*7*2 / (3*1)) = floor(84/3) = 28.
		res, err := mulMulDiv(uint256.NewInt(6), uint256.NewInt(7), uint256.NewInt(2), uint256.NewInt(3), one)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertU256Eq(t, "mulMulDiv result", res, "28")
	})
}

func assertU256Eq(t *testing.T, label string, got *uint256.Int, want string) {
	t.Helper()
	w := mustU256(want)
	if got == nil || got.Cmp(w) != 0 {
		t.Fatalf("%s: got %v, want %s", label, got, want)
	}
}
