package ilyris

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// fakeChain lets the parts that carry bugs be tested without a node.
type fakeChain struct {
	state          RawPoolState
	guard          RawGuardState
	stateErr       error
	guardErr       error
	stateCalls     int
	guardCalls     int
	lastGuardBlock uint64
	pools          []string
	total          int
	poolsErr       error
}

func (f *fakeChain) PoolState(_ context.Context, _ string, _ uint32) (RawPoolState, error) {
	f.stateCalls++
	return f.state, f.stateErr
}
func (f *fakeChain) GuardState(_ context.Context, _ string, blockNumber uint64) (RawGuardState, error) {
	f.guardCalls++
	f.lastGuardBlock = blockNumber
	g := f.guard
	g.BlockNumber = blockNumber
	return g, f.guardErr
}
func (f *fakeChain) FactoryPools(_ context.Context, _ string, offset, limit int) ([]string, int, error) {
	if f.poolsErr != nil {
		return nil, 0, f.poolsErr
	}
	if offset >= len(f.pools) {
		return nil, f.total, nil
	}
	end := offset + limit
	if end > len(f.pools) {
		end = len(f.pools)
	}
	return f.pools[offset:end], f.total, nil
}

func liveLikeChain() *fakeChain {
	return &fakeChain{
		state: RawPoolState{
			TokenX: "0x0Bd7D308f8E1639FAb988df18A8011f41EAcAD73", DecimalsX: 18,
			TokenY: "0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168", DecimalsY: 6,
			BinStepBps: 10, ActiveID: 7796, TotalFeeRate: 3_000_000,
			MarketGuard: "0xDd74981476f81c8e45e962Af6DF886a3c5788816",
			Bins: []RawBin{
				{ID: 7795, ReserveX: big.NewInt(0), ReserveY: big.NewInt(500_000_000)},
				{ID: 7796, ReserveX: big.NewInt(1e18), ReserveY: big.NewInt(500_000_000)},
			},
			BlockNumber: 43307616, BlockTimestamp: 1_700_000_000,
		},
	}
}

// THE FAILURE THIS ADAPTER EXISTS TO AVOID. Their service hands a tracker recent logs, never
// history. A pool stored with no book must trigger a full RPC refresh -- otherwise it folds
// logs into nothing, stays empty forever, and we are listed and never routed with nothing
// reporting an error anywhere.
func TestEmptyPoolTriggersAColdStart(t *testing.T) {
	c := liveLikeChain()
	tr := NewPoolTracker(c)

	got, err := tr.GetNewPoolState(context.Background(),
		entity.Pool{Address: "0xpool", Tokens: []*entity.PoolToken{{}, {}}},
		pool.GetNewPoolStateParams{}) // no logs at all
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if c.stateCalls == 0 {
		t.Fatal("no chain read - the book would have stayed empty forever")
	}
	var ex Extra
	if err := json.Unmarshal([]byte(got.Extra), &ex); err != nil {
		t.Fatalf("extra: %v", err)
	}
	if len(ex.Bins) == 0 {
		t.Fatal("cold start produced no bins")
	}
	if ex.ActiveID != 7796 {
		t.Fatalf("activeId not carried: %d", ex.ActiveID)
	}
}

// Bins present but all zero is the same situation as no bins.
func TestAllZeroBinsAlsoColdStart(t *testing.T) {
	empty, _ := json.Marshal(Extra{Bins: []BinJSON{{ID: 1, ReserveX: "0", ReserveY: "0"}}})
	if !needsBootstrap(entity.Pool{Extra: string(empty), StaticExtra: "{}"}) {
		t.Fatal("a book of empty bins must be treated as needing a bootstrap")
	}
}

// An unreadable guard must fail CLOSED. We cannot prove swaps are open, and quoting into a
// reverting swap reads as our pool being broken rather than closed.
func TestUnreadableGuardFailsClosed(t *testing.T) {
	c := liveLikeChain()
	c.guardErr = errors.New("rpc down")
	tr := NewPoolTracker(c)

	got, err := tr.BootstrapPoolState(context.Background(),
		entity.Pool{Address: "0xpool", Tokens: []*entity.PoolToken{{}, {}}},
		pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	var ex Extra
	_ = json.Unmarshal([]byte(got.Extra), &ex)
	if !ex.GuardSwapsPaused {
		t.Fatal("unreadable guard must leave swaps marked paused")
	}
}

// The guard ADDRESS is owner-mutable (setMarketGuard), so it must be re-read every refresh
// rather than cached from pool creation.
func TestGuardIsReadEveryRefresh(t *testing.T) {
	c := liveLikeChain()
	tr := NewPoolTracker(c)
	p := entity.Pool{Address: "0xpool", Tokens: []*entity.PoolToken{{}, {}}}
	for i := 0; i < 3; i++ {
		p, _ = tr.BootstrapPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	}
	if c.guardCalls != 3 {
		t.Fatalf("guard should be read on every refresh, got %d reads", c.guardCalls)
	}
}

// A transient RPC failure must leave the stored pool untouched. Replacing a good book with an
// empty one would quietly delist us.
func TestFailedReadLeavesThePoolUnchanged(t *testing.T) {
	c := liveLikeChain()
	c.stateErr = errors.New("rpc down")
	tr := NewPoolTracker(c)

	before := entity.Pool{Address: "0xpool", Extra: `{"activeId":1,"bins":[{"id":1,"x":"5","y":"5"}]}`}
	got, err := tr.BootstrapPoolState(context.Background(), before, pool.GetNewPoolStateParams{})
	if err == nil {
		t.Fatal("expected the RPC error to surface")
	}
	if got.Extra != before.Extra {
		t.Fatal("a failed read must not overwrite the stored book")
	}
}

// StaticExtra is immutable. Rewriting it every refresh would let one bad read silently
// redefine the pool's decimals.
func TestStaticExtraIsWrittenOnlyOnce(t *testing.T) {
	c := liveLikeChain()
	tr := NewPoolTracker(c)
	p := entity.Pool{Address: "0xpool", Tokens: []*entity.PoolToken{{}, {}},
		StaticExtra: `{"binStepBps":25,"decimalsX":18,"decimalsY":6}`}
	got, _ := tr.BootstrapPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	if got.StaticExtra != p.StaticExtra {
		t.Fatalf("StaticExtra was rewritten: %s", got.StaticExtra)
	}
}

// A refreshed pool must be directly constructible into a working simulator. This is the seam
// where a tracker and a simulator most easily disagree about the wire format.
func TestRefreshedPoolBuildsAWorkingSimulator(t *testing.T) {
	c := liveLikeChain()
	tr := NewPoolTracker(c)
	p, err := tr.BootstrapPoolState(context.Background(),
		entity.Pool{
			Address:  "0x90d0950065c567b9324a08a9aae8a28890fbab16",
			Exchange: DexType, Type: DexType,
			Tokens: []*entity.PoolToken{{}, {}},
		}, pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	s, err := NewPoolSimulator(p)
	if err != nil {
		t.Fatalf("tracker output did not build a simulator: %v", err)
	}
	if _, err := s.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: s.Info.Tokens[1], Amount: big.NewInt(1_000_000)},
		TokenOut:      s.Info.Tokens[0],
	}); err != nil {
		t.Fatalf("refreshed pool could not quote: %v", err)
	}
}

// ---- lister ----

// The cursor is an offset into an append-only array, which is what makes resuming safe.
func TestListerAdvancesItsCursor(t *testing.T) {
	c := &fakeChain{pools: []string{"0xa", "0xb", "0xc"}, total: 3}
	u := NewPoolsListUpdater(c, "0xfactory", DexType)
	u.limit = 2

	first, md, err := u.GetNewPools(context.Background(), nil)
	if err != nil || len(first) != 2 {
		t.Fatalf("first page: %d pools, err %v", len(first), err)
	}
	second, _, err := u.GetNewPools(context.Background(), md)
	if err != nil || len(second) != 1 {
		t.Fatalf("second page: %d pools, err %v", len(second), err)
	}
	if second[0].Address != "0xc" {
		t.Fatalf("cursor did not resume correctly: %s", second[0].Address)
	}
}

// A failed enumeration must hand the cursor back UNCHANGED. Advancing past pools we never read
// would skip them permanently -- the array is append-only, so nothing revisits that range.
func TestListerDoesNotAdvancePastAFailure(t *testing.T) {
	c := &fakeChain{poolsErr: errors.New("rpc down")}
	u := NewPoolsListUpdater(c, "0xfactory", DexType)
	in, _ := json.Marshal(Metadata{Offset: 7})
	_, out, err := u.GetNewPools(context.Background(), in)
	if err == nil {
		t.Fatal("expected the error to surface")
	}
	if string(out) != string(in) {
		t.Fatalf("cursor moved despite a failure: %s", out)
	}
}

// A corrupt cursor must not silently restart from zero -- that re-emits every pool as new on
// every round, forever.
func TestCorruptCursorIsRefusedNotReset(t *testing.T) {
	c := &fakeChain{pools: []string{"0xa"}, total: 1}
	u := NewPoolsListUpdater(c, "0xfactory", DexType)
	if _, _, err := u.GetNewPools(context.Background(), []byte("{not json")); err == nil {
		t.Fatal("a corrupt cursor must surface, not reset the scan")
	}
}

// Bins and the market guard must be pinned to one block. A guard from a later
// block can disagree with the book (freeze already lifted, or not yet) and the
// quote either routes into a revert or skips an open pool.
func TestBinsAndGuardShareOneBlock(t *testing.T) {
	c := liveLikeChain()
	tr := NewPoolTracker(c)
	got, err := tr.BootstrapPoolState(context.Background(),
		entity.Pool{Address: "0xpool", Tokens: []*entity.PoolToken{{}, {}}},
		pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if c.lastGuardBlock != c.state.BlockNumber {
		t.Fatalf("guard fetched at block %d, bins at %d", c.lastGuardBlock, c.state.BlockNumber)
	}
	if got.BlockNumber != c.state.BlockNumber {
		t.Fatalf("entity block %d, bins at %d", got.BlockNumber, c.state.BlockNumber)
	}
	var ex Extra
	if err := json.Unmarshal([]byte(got.Extra), &ex); err != nil {
		t.Fatalf("extra: %v", err)
	}
	if ex.BlockNumber != got.BlockNumber {
		t.Fatalf("extra block %d != entity block %d", ex.BlockNumber, got.BlockNumber)
	}

	st, g, err := tr.FetchRPCData(context.Background(), entity.Pool{Address: "0xpool"})
	if err != nil {
		t.Fatalf("FetchRPCData: %v", err)
	}
	if st.BlockNumber != g.BlockNumber {
		t.Fatalf("FetchRPCData mixed blocks: bins %d guard %d", st.BlockNumber, g.BlockNumber)
	}
	if st.BlockNumber == 0 {
		t.Fatal("expected a pinned block number")
	}
}
