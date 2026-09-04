package ilyris

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	tokX = "0x0bd7d308f8e1639fab988df18a8011f41eacad73" // WETH, 18dp
	tokY = "0x5fc5360d0400a0fd4f2af552add042d716f1d168" // USDG, 6dp
)

// A small book straddling the active bin: Y at and below, X at and above.
func newTestSim() *PoolSimulator {
	e18 := func(n int64) *big.Int {
		return new(big.Int).Mul(big.NewInt(n), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	}
	e6 := func(n int64) *big.Int {
		return new(big.Int).Mul(big.NewInt(n), big.NewInt(1_000_000))
	}
	bins := []bin{
		{ID: 7794, ReserveX: big.NewInt(0), ReserveY: e6(500)},
		{ID: 7795, ReserveX: big.NewInt(0), ReserveY: e6(500)},
		{ID: 7796, ReserveX: e18(1), ReserveY: e6(500)},
		{ID: 7797, ReserveX: e18(1), ReserveY: big.NewInt(0)},
		{ID: 7798, ReserveX: e18(1), ReserveY: big.NewInt(0)},
	}
	sumX, sumY := big.NewInt(0), big.NewInt(0)
	for _, b := range bins {
		sumX.Add(sumX, b.ReserveX)
		sumY.Add(sumY, b.ReserveY)
	}
	return &PoolSimulator{
		Pool: pool.Pool{Info: pool.PoolInfo{
			Address:  "0x90d0950065c567b9324a08a9aae8a28890fbab16",
			Exchange: DexType,
			Type:     DexType,
			Tokens:   []string{tokX, tokY},
			Reserves: []*big.Int{sumX, sumY},
		}},
		binStepBps:   10,
		activeID:     7796,
		decimalsX:    18,
		decimalsY:    6,
		bins:         bins,
		totalFeeRate: 3_000_000, // 0.30% in FEE_PRECISION (1e9) units
	}
}

// The assertion that justifies this whole module. It is compile-time, so this test exists to
// state WHY it matters rather than to add coverage: their interface is 14 methods and a
// transcription of it looks right until their CI disagrees in public.
func TestSatisfiesTheirInterface(t *testing.T) {
	var _ pool.IPoolSimulator = newTestSim()
}

func TestCalcAmountOutSellingQuote(t *testing.T) {
	s := newTestSim()
	res, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: big.NewInt(10_000_000)}, // 10 USDG
		TokenOut:      tokX,
	})
	if err != nil {
		t.Fatalf("quote failed: %v", err)
	}
	if res.TokenAmountOut.Amount.Sign() <= 0 {
		t.Fatalf("non-positive amountOut: %s", res.TokenAmountOut.Amount)
	}
	// Fee is denominated in the INPUT token, because that is where BinPool takes it.
	if res.Fee.Token != tokY {
		t.Fatalf("fee should be in the input token, got %s", res.Fee.Token)
	}
	if res.Gas < BaseSwapGas {
		t.Fatalf("gas below the measured floor: %d", res.Gas)
	}
}

// An unfillable quote must be an ERROR. A zero would be routed as a genuine offer of nothing
// and rank us last instead of skipping us.
func TestUnfillableIsAnErrorNotAZero(t *testing.T) {
	s := newTestSim()
	huge := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: huge},
		TokenOut:      tokX,
	}); err == nil {
		t.Fatal("expected an error for an amount the book cannot fill")
	}
}

// The guard is invisible to quoteExactIn on chain, so the simulator must model it or we route
// into a reverting swap.
func TestGuardBlocksQuotes(t *testing.T) {
	s := newTestSim()
	s.guardSwapsPaused = true
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: big.NewInt(1_000_000)},
		TokenOut:      tokX,
	}); err != ErrSwapsPaused {
		t.Fatalf("paused guard must block the quote, got %v", err)
	}

	s2 := newTestSim()
	s2.blockTimestamp = 1000
	s2.guardFreezeEnd = 2000
	if _, err := s2.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: big.NewInt(1_000_000)},
		TokenOut:      tokX,
	}); err != ErrCorporateActionFreeze {
		t.Fatalf("active freeze must block the quote, got %v", err)
	}
}

// Addresses arrive checksummed from our manifest and lowercased from their loader. Their own
// GetTokenIndex compares with ==, so folding case here is what stops one source silently
// failing.
func TestTokenLookupIsCaseInsensitive(t *testing.T) {
	s := newTestSim()
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0x5FC5360D0400A0FD4F2AF552ADD042D716F1D168", Amount: big.NewInt(1_000_000)},
		TokenOut:      "0x0BD7D308F8E1639FAB988DF18A8011F41EACAD73",
	}); err != nil {
		t.Fatalf("checksummed addresses must resolve: %v", err)
	}
}

// CloneState must deep-copy what UpdateBalance mutates. The base returns nil, which silently
// breaks split routing -- the aggregator cannot restore state to try a second path.
func TestCloneStateIsDeepAndNotNil(t *testing.T) {
	s := newTestSim()
	c := s.CloneState()
	if c == nil {
		t.Fatal("CloneState returned nil - split routing would break")
	}
	clone := c.(*PoolSimulator)
	clone.bins[0].ReserveY.SetInt64(1)
	clone.Info.Reserves[1].SetInt64(1)
	clone.activeID = 1

	if s.bins[0].ReserveY.Int64() == 1 {
		t.Fatal("bin reserves are shared with the clone")
	}
	if s.Info.Reserves[1].Int64() == 1 {
		t.Fatal("Info.Reserves is shared with the clone")
	}
	if s.activeID == 1 {
		t.Fatal("activeID leaked from the clone")
	}
}

// UpdateBalance must add the NET input. Adding the gross would inflate reserves by the fee on
// every swap and drift the book from chain a little more each time.
func TestUpdateBalanceAddsNetOfFee(t *testing.T) {
	s := newTestSim()
	before := new(big.Int).Set(s.Info.Reserves[1])

	in := big.NewInt(10_000_000)
	fee := big.NewInt(30_000)
	out := big.NewInt(1_000_000_000_000)

	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokY, Amount: in},
		TokenAmountOut: pool.TokenAmount{Token: tokX, Amount: out},
		Fee:            pool.TokenAmount{Token: tokY, Amount: fee},
		SwapInfo:       SwapInfo{NewActiveID: 7795, XForY: false, BinsCrossed: 2},
	})

	want := new(big.Int).Add(before, new(big.Int).Sub(in, fee))
	if s.Info.Reserves[1].Cmp(want) != 0 {
		t.Fatalf("reserve should grow by NET input: got %s want %s", s.Info.Reserves[1], want)
	}
	if s.activeID != 7795 {
		t.Fatalf("activeID not applied: %d", s.activeID)
	}
}

// A SwapInfo that is not ours must abort the update, not continue into a nil dereference the
// way the liquiditybookv21 template does.
func TestUpdateBalanceIgnoresForeignSwapInfo(t *testing.T) {
	s := newTestSim()
	before := new(big.Int).Set(s.Info.Reserves[1])
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokY, Amount: big.NewInt(1_000_000)},
		TokenAmountOut: pool.TokenAmount{Token: tokX, Amount: big.NewInt(1)},
		Fee:            pool.TokenAmount{Token: tokY, Amount: big.NewInt(0)},
		SwapInfo:       "not ours",
	})
	if s.Info.Reserves[1].Cmp(before) != 0 {
		t.Fatal("a foreign SwapInfo must leave the book untouched")
	}
}

// Gas must scale with bins crossed. A flat constant flatters large swaps and penalises small
// ones, which is how a router ends up preferring the wrong venue.
func TestGasScalesWithBinsCrossed(t *testing.T) {
	if gasFor(1) != BaseSwapGas {
		t.Fatalf("single-bin swap should be the base cost, got %d", gasFor(1))
	}
	if gasFor(3) != BaseSwapGas+2*PerExtraBinGas {
		t.Fatalf("three bins should add two increments, got %d", gasFor(3))
	}
}

// A freezeEnd in the past is not a freeze. BlockTimestamp == 0 would make any
// nonzero freezeEnd look active forever, which is why the tracker must pin the
// header timestamp of the same block as the book.
func TestExpiredFreezeIsNotFrozen(t *testing.T) {
	s := newTestSim()
	s.guardFreezeEnd = 1_700_000_000
	s.blockTimestamp = 1_700_000_000
	if err := s.blocked(); err != nil {
		t.Fatalf("freezeEnd == BlockTimestamp must not freeze, got %v", err)
	}
	s.blockTimestamp = 1_700_000_001
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokY, Amount: big.NewInt(1_000_000)},
		TokenOut:      tokX,
	}); err != nil {
		t.Fatalf("expired freeze must not block a quote, got %v", err)
	}
	s.blockTimestamp = 1_699_999_999
	if err := s.blocked(); err != ErrCorporateActionFreeze {
		t.Fatalf("freezeEnd still in the future must freeze, got %v", err)
	}
}

func binY(s *PoolSimulator) *big.Int {
	sum := new(big.Int)
	for _, b := range s.bins {
		sum.Add(sum, b.ReserveY)
	}
	return sum
}

func binX(s *PoolSimulator) *big.Int {
	sum := new(big.Int)
	for _, b := range s.bins {
		sum.Add(sum, b.ReserveX)
	}
	return sum
}

// After a swap, the bins that were crossed must shrink. If UpdateBalance only
// moved aggregate reserves + activeID, a split/multi-hop re-quote would see the
// original book and pay out more than is left.
func TestSequentialSwapsDoNotRequoteSpentBins(t *testing.T) {
	in := new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil)) // 0.5 X

	s := newTestSim()
	first, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokX, Amount: in},
		TokenOut:      tokY,
	})
	if err != nil {
		t.Fatalf("first quote: %v", err)
	}
	si, ok := first.SwapInfo.(SwapInfo)
	if !ok || len(si.Fills) == 0 {
		t.Fatal("quote must hand back the bins it crossed")
	}

	yBefore := binY(s)
	s.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokX, Amount: in},
		TokenAmountOut: pool.TokenAmount{Token: tokY, Amount: first.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: tokX, Amount: first.Fee.Amount},
		SwapInfo:       first.SwapInfo,
	})
	yAfter := binY(s)
	if yAfter.Cmp(yBefore) >= 0 {
		t.Fatalf("bin Y did not shrink after paying out Y: before=%s after=%s", yBefore, yAfter)
	}
	wantY := new(big.Int).Sub(yBefore, first.TokenAmountOut.Amount)
	if yAfter.Cmp(wantY) != 0 {
		t.Fatalf("bin Y should fall by amountOut: got %s want %s", yAfter, wantY)
	}

	second, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokX, Amount: in},
		TokenOut:      tokY,
	})
	if err != nil {
		if err != ErrInsufficientLiquidity {
			t.Fatalf("second quote: %v", err)
		}
		// Remaining bins cannot fill the same size — that is the spent-book signal.
	} else {
		if second.TokenAmountOut.Amount.Cmp(yAfter) > 0 {
			t.Fatalf("second quote paid %s Y but bins only hold %s", second.TokenAmountOut.Amount, yAfter)
		}
		if second.TokenAmountOut.Amount.Cmp(first.TokenAmountOut.Amount) >= 0 {
			t.Fatalf("second quote did not shrink after consuming bins: first=%s second=%s",
				first.TokenAmountOut.Amount, second.TokenAmountOut.Amount)
		}
	}

	// Split routing: clone, take the first leg, then quote the second from the clone.
	base := newTestSim()
	leg := base.CloneState().(*PoolSimulator)
	q1, err := leg.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokX, Amount: in},
		TokenOut:      tokY,
	})
	if err != nil {
		t.Fatalf("split first leg: %v", err)
	}
	leg.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: tokX, Amount: in},
		TokenAmountOut: pool.TokenAmount{Token: tokY, Amount: q1.TokenAmountOut.Amount},
		Fee:            pool.TokenAmount{Token: tokX, Amount: q1.Fee.Amount},
		SwapInfo:       q1.SwapInfo,
	})
	remaining := binY(leg)
	q2, err := leg.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokX, Amount: in},
		TokenOut:      tokY,
	})
	if err != nil {
		if err != ErrInsufficientLiquidity {
			t.Fatalf("split second leg: %v", err)
		}
	} else if q2.TokenAmountOut.Amount.Cmp(remaining) > 0 {
		t.Fatalf("cloned second leg paid %s Y but bins only hold %s", q2.TokenAmountOut.Amount, remaining)
	}
	if base.bins[2].ReserveY.Cmp(leg.bins[2].ReserveY) == 0 {
		t.Fatal("clone's bins were not mutated independently of the original")
	}
}
