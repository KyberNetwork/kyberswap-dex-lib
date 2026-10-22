package algebrav1

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	tickspkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

var errMismatch = &ephemeral.RevertError{Err: &ephemeral.ContractError{Name: "TickTableMismatch"}}

// fakeTable answers lens reads for pools whose tick table is indexed one way: read that way, the
// pool's ticks; read the other way, a mismatch, or no ticks when there are none or they all lie
// past the words a read by tick over the spacing covers.
type fakeTable struct {
	raw        bool
	spacing    int
	beyond     bool
	ticks      []tickspkg.Tick
	reads      []bool
	readBlocks []*big.Int
}

func (f *fakeTable) read(_ context.Context, _ string, block *big.Int, raw bool) (*lensSnapshot, error) {
	f.reads = append(f.reads, raw)
	f.readBlocks = append(f.readBlocks, block)
	s := &lensSnapshot{block: 100, tickSpacing: f.spacing, tick: 0, liquidity: new(big.Int)}
	if raw == f.raw || f.spacing == 1 {
		s.ticks = f.ticks
		for _, tk := range f.ticks {
			if tk.TickIdx <= s.tick {
				s.liquidity.Add(s.liquidity, tk.LiquidityNet)
			}
		}
		return s, nil
	}
	if len(f.ticks) == 0 || f.raw && f.beyond {
		return s, nil
	}
	return nil, errMismatch
}

func ticks(pairs ...int64) []tickspkg.Tick {
	var out []tickspkg.Tick
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, tickspkg.Tick{TickIdx: int(pairs[i]), LiquidityNet: big.NewInt(pairs[i+1]),
			LiquidityGross: new(big.Int).Abs(big.NewInt(pairs[i+1]))})
	}
	return out
}

func newLensTracker(f *fakeTable) *PoolTracker {
	t := NewPoolTracker(&Config{UseLens: true}, nil, nil)
	t.readLens = f.read
	return t
}

// TestSnapshotLearnsTheTickTable: each way a first read can go settles the exchange's table, or
// leaves it to a later pool when the read cannot tell.
func TestSnapshotLearnsTheTickTable(t *testing.T) {
	pool := ticks(-120, 5, 600, -5)
	for _, tc := range []struct {
		name  string
		table fakeTable
		reads []bool
		want  tickTable
	}{
		{"indexed by tick over the spacing", fakeTable{spacing: 60, ticks: pool}, []bool{false}, tickTableCompressed},
		{"indexed by tick, a set bit read the wrong way", fakeTable{raw: true, spacing: 60, ticks: pool},
			[]bool{false, true}, tickTableRaw},
		{"indexed by tick, every tick past the words read", fakeTable{raw: true, spacing: 60, beyond: true,
			ticks: pool}, []bool{false, true}, tickTableRaw},
		{"spacing 1 indexes both ways alike", fakeTable{raw: true, spacing: 1, ticks: pool}, []bool{false},
			tickTableUnknown},
		{"no ticks either way", fakeTable{spacing: 60}, []bool{false, true}, tickTableUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newLensTracker(&tc.table)
			s, err := tr.snapshot(t.Context(), "0xpool", nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(s.ticks) != len(tc.table.ticks) {
				t.Errorf("%d ticks, want %d", len(s.ticks), len(tc.table.ticks))
			}
			if got := tr.tickTable.Load(); got != tc.want {
				t.Errorf("table %d, want %d", got, tc.want)
			}
			if len(tc.table.reads) != len(tc.reads) {
				t.Fatalf("reads %v, want %v", tc.table.reads, tc.reads)
			}
			for i := range tc.reads {
				if tc.table.reads[i] != tc.reads[i] {
					t.Errorf("reads %v, want %v", tc.table.reads, tc.reads)
				}
			}
		})
	}
}

// TestSnapshotReadsBeyondAtTheFirstReadsBlock: the second read of a pool whose first found nothing
// is of the same block, so the two describe one state.
func TestSnapshotReadsBeyondAtTheFirstReadsBlock(t *testing.T) {
	f := &fakeTable{raw: true, spacing: 60, beyond: true, ticks: ticks(-198240, 7, -198000, -7)}
	if _, err := newLensTracker(f).snapshot(t.Context(), "0xpool", nil); err != nil {
		t.Fatal(err)
	}
	if f.readBlocks[0] != nil || f.readBlocks[1] == nil || f.readBlocks[1].Uint64() != 100 {
		t.Errorf("read at blocks %v, want the latest then 100", f.readBlocks)
	}
}

// TestSnapshotKeepsTheLearntTable: once settled, a pool is read the exchange's way alone, and one
// that is not indexed that way is read the other way without unsettling the exchange.
func TestSnapshotKeepsTheLearntTable(t *testing.T) {
	f := &fakeTable{raw: true, spacing: 60, ticks: ticks(-120, 5, 600, -5)}
	tr := newLensTracker(f)
	tr.tickTable.Store(tickTableRaw)
	if _, err := tr.snapshot(t.Context(), "0xpool", nil); err != nil {
		t.Fatal(err)
	}
	if len(f.reads) != 1 || !f.reads[0] {
		t.Errorf("reads %v, want one by tick", f.reads)
	}

	other := &fakeTable{spacing: 60, ticks: ticks(-120, 5, 600, -5)}
	tr.readLens = other.read
	if _, err := tr.snapshot(t.Context(), "0xother", nil); err != nil {
		t.Fatal(err)
	}
	if len(other.reads) != 2 || !other.reads[0] || other.reads[1] {
		t.Errorf("reads %v, want by tick then over the spacing", other.reads)
	}
	if got := tr.tickTable.Load(); got != tickTableRaw {
		t.Errorf("table %d, want it still by tick", got)
	}
}

// TestSnapshotChecksTheTicks: a set that does not add up to the pool's liquidity is refused.
func TestSnapshotChecksTheTicks(t *testing.T) {
	f := &fakeTable{spacing: 60, ticks: ticks(-120, 5, 600, -4)}
	if _, err := newLensTracker(f).snapshot(t.Context(), "0xpool", nil); err == nil {
		t.Fatal("ticks summing to 1 accepted")
	}
}

// TestTicksFromLensReplacesTheSet: the lens's set replaces the stored one, dropping a tick the pool
// no longer has, and is read at the block the logs' ticks were (the ticks-based pool's), the
// latest when that is none.
func TestTicksFromLensReplacesTheSet(t *testing.T) {
	f := &fakeTable{spacing: 60, ticks: ticks(-120, 5, 600, -5)}
	tr := newLensTracker(f)
	p := tickspkg.TicksBasedPool{Address: "0xpool", BlockNumber: 95, Ticks: map[int]tickspkg.Tick{
		-120: ticks(-120, 3)[0], 60: ticks(60, 9)[0]}}

	if !tr.ticksFromLens(t.Context(), &p) {
		t.Fatal("lens read not used")
	}
	if len(p.Ticks) != 2 || p.Ticks[-120].LiquidityNet.Int64() != 5 || p.Ticks[600].LiquidityNet.Int64() != -5 {
		t.Errorf("ticks %v", p.Ticks)
	}
	if got := f.readBlocks[0]; got == nil || got.Uint64() != 95 {
		t.Errorf("read at %v, want 95", got)
	}

	// Another pool: the guard holds one pool back for a cooldown after its read.
	p.Address, p.BlockNumber = "0xother", 0
	if !tr.ticksFromLens(t.Context(), &p) || f.readBlocks[len(f.readBlocks)-1] != nil {
		t.Error("ticks named no block, yet the lens read was pinned")
	}

	if NewPoolTracker(&Config{}, nil, nil).ticksFromLens(t.Context(), &p) {
		t.Error("a tracker without the lens read through it")
	}
}

// TestResetPoolState: a reset drops what the logs built and keeps the rest, so the next update
// reads the pool whole.
func TestResetPoolState(t *testing.T) {
	extra, _ := json.Marshal(Extra{Liquidity: big.NewInt(7), GlobalState: GlobalState{Price: big.NewInt(1),
		Tick: big.NewInt(3)}, TickSpacing: 60})
	p := entity.Pool{Address: "0xpool", Extra: string(extra)}
	if !tickspkg.HasState(p) {
		t.Fatal("pool without state before the reset")
	}

	if _, ok := NewPoolTracker(&Config{}, nil, nil).ResetPoolState(p, 50); ok {
		t.Error("reset without the lens")
	}
	got, ok := newLensTracker(&fakeTable{}).ResetPoolState(p, 50)
	if !ok || tickspkg.HasState(got) || tickspkg.LensDue(got) != 50 {
		t.Fatalf("reset to %s", got.Extra)
	}
	var kept Extra
	if err := json.Unmarshal([]byte(got.Extra), &kept); err != nil || kept.TickSpacing != 60 ||
		kept.GlobalState.Tick.Int64() != 3 {
		t.Errorf("kept %s", got.Extra)
	}
}

// TestStatelessPoolsAreScheduled: a pool without state is due when the guard would take it.
func TestStatelessPoolsAreScheduled(t *testing.T) {
	tr := NewPoolTracker(&Config{UseLens: true}, nil, nil)
	if at := tr.NextUpdateAt(entity.Pool{Address: "0xpool"}); at == 0 {
		t.Error("pool without state not scheduled")
	}
	extra, _ := json.Marshal(Extra{Liquidity: big.NewInt(7)})
	if at := tr.NextUpdateAt(entity.Pool{Address: "0xpool", Extra: string(extra)}); at != 0 {
		t.Error("pool with state scheduled")
	}
}
