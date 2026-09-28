package everlongflamm

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// End-to-end parity of the composed port (state.go, pricefeed.go, swap.go, lever.go over the hook, financing and
// leverage modules) with the DEPLOYED c104 pool on Base. testdata/core_e2e_grid_<block>.jsonl.gz is written by
// testdata/gen/CoreE2EGrid.t.sol at blocks 51302915, 51313000 and 51324800: per scenario one complete state read
// the way the tracker reads it (state_reads.go) and pool.previewSwap / pool.previewLever over log grids and
// unit-by-unit windows around every class boundary. testdata/core_e2e_seq_<block>.jsonl.gz
// (testdata/gen/CoreE2ESeq.t.sol) executes real swap / leverUp / leverDown sequences from funded accounts with
// admin moves and time warps between them; the port replays each sequence from its own post-states and must
// match every return, every event and the chain's post-state field for field. No tolerance anywhere.

var e2eBlocks = []string{"51302915", "51313000", "51324800"}

type e2eRow struct {
	K   string          `json:"k"`
	Tag string          `json:"tag"`
	S   json.RawMessage `json:"s"`
	D   int             `json:"d"`
	A   uint256.Int     `json:"a"`
	R   []uint256.Int   `json:"r"`
	E   *string         `json:"e"`
}

func e2eLines(t *testing.T, path string) [][]byte {
	t.Helper()
	lines, err := readLines(path)
	require.NoError(t, err)
	return lines
}

// e2eDigests pins every end-to-end fixture (sha256 of the stored file; testdata/README.md section 4).
var e2eDigests = map[string]string{
	"core_e2e_grid_51302915.jsonl.gz": "ae588fcaf6e08b24b5d7e7b3f12c491b89c73f9e155ceb8641090c5a6004ddd5",
	"core_e2e_grid_51313000.jsonl.gz": "4e21a865486e59b4fc0eae66290ed509fc1afe88254c558b67cad6b7e62258ee",
	"core_e2e_grid_51324800.jsonl.gz": "2d053f484f935e3e715c8f3e8655ef6a32f08c4255a6d8d8d2622f0d303130b1",
	"core_e2e_seq_51302915.jsonl.gz":  "9ae2f16137c0a744dd730758ad97537bb8533ce6a0868367d1113ff7c35c1116",
	"core_e2e_seq_51313000.jsonl.gz":  "e2d4569b540b7853b49e04507e7ba485439aae3ebb26621c28352b411a92536f",
	"core_e2e_seq_51324800.jsonl.gz":  "6529b264e751e5026344539a039b45bb90542d56b5f501d5ed3f3f5c05f64d45",
}

func e2eFixture(name string) string {
	return "testdata/" + name
}

func TestCoreE2EFixtureDigests(t *testing.T) {
	t.Parallel()
	for name, want := range e2eDigests {
		raw, err := os.ReadFile(e2eFixture(name))
		require.NoError(t, err)
		sum := sha256.Sum256(raw)
		require.Equal(t, want, hex.EncodeToString(sum[:]), name)
	}
}

// e2eRevert maps on-chain revert data onto the port's error vocabulary; nil is unmapped (the row fails).
func e2eRevert(h string) error {
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	if err != nil {
		return nil
	}
	if len(b) == 0 {
		return errMulDivOverflow
	}
	if len(b) < 4 {
		return nil
	}
	var sel [4]byte
	copy(sel[:], b[:4])
	if sel == [4]byte{0x4e, 0x48, 0x7b, 0x71} && len(b) == 36 {
		switch b[35] {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		case 0x32:
			return errPanicIndex
		}
		return nil
	}
	return revertSelectors[sel]
}

type e2eReport struct {
	t    *testing.T
	n    int
	rows int
}

func (r *e2eReport) fail(format string, args ...any) {
	r.t.Helper()
	r.n++
	if r.n <= 40 {
		r.t.Errorf(format, args...)
	}
}

// e2eCompare checks one call: both answer with the same words, or both refuse with the same revert class.
func e2eCompare(rep *e2eReport, where string, row *e2eRow, goErr error, got []uint256.Int) {
	rep.rows++
	if row.E != nil {
		want := e2eRevert(*row.E)
		if want == nil {
			rep.fail("%s: unmapped chain revert %s (go %v)", where, *row.E, goErr)
		} else if !errors.Is(goErr, want) {
			rep.fail("%s: chain revert %v, go err=%v words=%v", where, want, goErr, got)
		}
		return
	}
	if goErr != nil {
		rep.fail("%s: chain %v, go err %v", where, row.R, goErr)
		return
	}
	for i := range row.R {
		if !row.R[i].Eq(&got[i]) {
			rep.fail("%s: word %d chain %s go %s (chain %v go %v)", where, i, row.R[i].Dec(), got[i].Dec(), row.R, got)
			return
		}
	}
}

func e2eState(t *testing.T, raw json.RawMessage) (*flammState, *flammReads, map[string]json.RawMessage) {
	t.Helper()
	var reads flammReads
	require.NoError(t, json.Unmarshal(raw, &reads))
	var views struct {
		Views map[string]json.RawMessage `json:"views"`
	}
	require.NoError(t, json.Unmarshal(raw, &views))
	s, err := reads.baseState()
	require.NoError(t, err)
	return s, &reads, views.Views
}

func e2ePreviewSwapWords(s *flammState, sell bool, amount *uint256.Int, now uint64) ([]uint256.Int, error) {
	p, err := s.previewSwap(sell, amount, now)
	if err != nil {
		return nil, err
	}
	return []uint256.Int{p.UsedNative, p.NetNative, p.FeeWad}, nil
}

func e2ePreviewLeverWords(s *flammState, up bool, amount *uint256.Int, now uint64) ([]uint256.Int, error) {
	r, err := s.previewLever(up, amount, now)
	if err != nil {
		return nil, err
	}
	return []uint256.Int{r.AmountInUsed, r.AmountOut, r.SpreadPpm, r.CrAfterWad}, nil
}

// e2eAttest recomputes the deployed views the dump carries from the built state.
func e2eAttest(rep *e2eReport, where string, s *flammState, reads *flammReads, views map[string]json.RawMessage,
	now uint64) {
	var v struct {
		PeekCross struct {
			Ok       bool        `json:"ok"`
			PriceWad uint256.Int `json:"priceWad"`
			Ts       uint256.Int `json:"ts"`
		} `json:"peekCross"`
		PegOk        bool           `json:"pegOk"`
		LoanPosition [3]uint256.Int `json:"loanPosition"`
		Positions    struct {
			Coll      []uint256.Int `json:"coll"`
			Sup       []uint256.Int `json:"sup"`
			Debt      []uint256.Int `json:"debt"`
			TotalColl uint256.Int   `json:"totalColl"`
		} `json:"positions"`
		TotalAssets struct {
			V   *uint256.Int `json:"v"`
			Err *string      `json:"err"`
		} `json:"totalAssets"`
	}
	raw, _ := json.Marshal(views)
	if err := json.Unmarshal(raw, &v); err != nil {
		rep.fail("%s: views: %v", where, err)
		return
	}
	rep.rows++
	ok, p, ts := s.Feed.peekCross(&s.Feed.Asset, &s.Feed.Loans[0], now)
	if ok != v.PeekCross.Ok || !p.Eq(&v.PeekCross.PriceWad) || ts != v.PeekCross.Ts.Uint64() {
		rep.fail("%s: peekCross chain %+v go (%v %s %d)", where, v.PeekCross, ok, p.Dec(), ts)
	}
	if peg, err := s.pegOk(0, now); err != nil || peg != v.PegOk {
		rep.fail("%s: pegOk chain %v go %v %v", where, v.PegOk, peg, err)
	}
	pos, err := s.Router.positions(now)
	if err != nil {
		rep.fail("%s: positions: %v", where, err)
		return
	}
	if fmt.Sprint(pos.Coll, pos.Sup, pos.Debt, pos.TotalColl.Dec()) !=
		fmt.Sprint(v.Positions.Coll, v.Positions.Sup, v.Positions.Debt, v.Positions.TotalColl.Dec()) {
		rep.fail("%s: positions chain %+v go %+v", where, v.Positions, pos)
	}
	var gross uint256.Int
	gross.Add(&s.Pool.Physical, &pos.TotalColl)
	if !gross.Eq(&reads.Pool.Gross) {
		rep.fail("%s: gross chain %s go %s", where, reads.Pool.Gross.Dec(), gross.Dec())
	}
	_, sup, debt, err := s.Router.position(0, now)
	if err != nil || !s.Pool.Loans[0].Liquid.Eq(&v.LoanPosition[0]) || !sup.Eq(&v.LoanPosition[1]) ||
		!debt.Eq(&v.LoanPosition[2]) {
		rep.fail("%s: loanPosition chain %v go (%s %s %s) %v", where, v.LoanPosition, s.Pool.Loans[0].Liquid.Dec(),
			sup.Dec(), debt.Dec(), err)
	}
	pool := s.Pool
	b, err := s.priced(&pool, now)
	var nav uint256.Int
	if err == nil {
		nav, err = gateNavAt(&b)
	}
	if v.TotalAssets.V != nil {
		if err != nil || !nav.Eq(v.TotalAssets.V) {
			rep.fail("%s: totalAssets chain %s go %s %v", where, v.TotalAssets.V.Dec(), nav.Dec(), err)
		}
	} else if want := e2eRevert(*v.TotalAssets.Err); want == nil || !errors.Is(err, want) {
		rep.fail("%s: totalAssets chain revert %s go %v", where, *v.TotalAssets.Err, err)
	}
}

// e2eRowKey is a grid row's sampling stream (scenario, entry, direction) and outcome class (sampleRows): the revert
// selector, a fill, or a fill that used less than its input.
func e2eRowKey(row *e2eRow) (stream, class string) {
	if row.K != "sw" && row.K != "lv" {
		return "", ""
	}
	stream = fmt.Sprintf("%s|%s|%d", row.Tag, row.K, row.D)
	switch {
	case row.E != nil:
		return stream, (*row.E + "0000000000")[:10]
	case len(row.R) > 0 && !row.R[0].Eq(&row.A):
		return stream, "partial"
	}
	return stream, "ok"
}

// TestCoreE2EPreviewGrids replays every recorded previewSwap / previewLever against the scenario's state (sampled
// under -race, fixture_sample_test.go).
func TestCoreE2EPreviewGrids(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{}
	defer func() {
		// Every revert class the pool's swap and leverage previews can reach on this deployment must have fired
		// somewhere in the grids (FeeMismatch / FillMismatch / PriceUnchecked for loan 0 cannot).
		for _, sel := range []string{"0xfe85bb51", "0x77417454", "0xf9b4678a", "0x26363b73", "0x84f5270a", "0xc6520de3",
			"0x78d612f2", "0xc81f1209", "0x28851730", "0x7c3fa3af", "0x9e87fac8", "0x6d2d9f49", "0x81927929",
			"0x032b3d00", "0xc3734dc2", "0x85c2be22", "0xcd215006", "0x2c5211c6", "0xd6f8f89c", "0x2ea2dce8",
			"0x4e487b71", "0x00000000", "ok"} {
			require.True(t, seen[sel], "no grid row reached %s", sel)
		}
	}()
	for _, blk := range e2eBlocks {
		t.Run(blk, func(t *testing.T) {
			rep := &e2eReport{t: t}
			states := map[string]*flammState{}
			nows := map[string]uint64{}
			counts := map[string]int{}
			lines := e2eLines(t, e2eFixture("core_e2e_grid_"+blk+".jsonl.gz"))
			rows := make([]e2eRow, len(lines))
			for i, line := range lines {
				require.NoError(t, json.Unmarshal(line, &rows[i]))
			}
			keep := sampleRows(len(rows), rowSampling{stride: 64, edges: true}, func(i int) (string, string) { return e2eRowKey(&rows[i]) })
			for i := range rows {
				if !keep[i] {
					continue
				}
				row := rows[i]
				switch row.K {
				case "state":
					s, reads, views := e2eState(t, row.S)
					states[row.Tag], nows[row.Tag] = s, reads.Timestamp
					e2eAttest(rep, row.Tag, s, reads, views, reads.Timestamp)
				case "sw", "lv":
					s := states[row.Tag]
					require.NotNil(t, s, row.Tag)
					var got []uint256.Int
					var err error
					if row.K == "sw" {
						got, err = e2ePreviewSwapWords(s, row.D == 1, &row.A, nows[row.Tag])
					} else {
						got, err = e2ePreviewLeverWords(s, row.D == 1, &row.A, nows[row.Tag])
					}
					e2eCompare(rep, fmt.Sprintf("%s %s d=%d a=%s", row.Tag, row.K, row.D, row.A.Dec()), &row, err, got)
					cls := "ok"
					if row.E != nil {
						cls = (*row.E + "0000000000")[:10]
					}
					counts[row.K+" "+cls]++
					seen[cls] = true
				}
			}
			if !fixturesSampled() {
				require.Greater(t, rep.rows, 20_000)
			}
			t.Logf("block %s: %d rows compared (%d of %d kept), %d mismatches; classes %v", blk, rep.rows, kept(keep),
				len(rows), rep.n, counts)
		})
	}
}

type e2eStep struct {
	K    string          `json:"k"`
	Seq  string          `json:"seq"`
	I    int             `json:"i"`
	Op   string          `json:"op"`
	Args []uint256.Int   `json:"args"`
	Ts   uint64          `json:"ts"`
	Ok   bool            `json:"ok"`
	R    []uint256.Int   `json:"r"`
	E    *string         `json:"e"`
	Ev   []e2eEvent      `json:"ev"`
	Post json.RawMessage `json:"post"`
	S    json.RawMessage `json:"s"`
	// preview rows interleaved in a sequence
	D int         `json:"d"`
	A uint256.Int `json:"a"`
}

type e2eEvent struct {
	Name string        `json:"name"`
	W    []uint256.Int `json:"w"`
}

// e2eNormalized drops what is not state: the price frame priced writes and the per-transaction repay snapshot.
func e2eNormalized(s *flammState) flammState {
	c := *s.clone()
	c.Pool.PriceWad, c.Pool.CrossWad = nil, nil
	c.Router.TransientRepay = nil
	return c
}

// e2eDiff lists the fields where two states differ.
func e2eDiff(got, want *flammState) []string {
	g, w := e2eNormalized(got), e2eNormalized(want)
	var d []string
	check := func(name string, a, b any) {
		if !reflect.DeepEqual(a, b) {
			d = append(d, fmt.Sprintf("%s: go %+v chain %+v", name, a, b))
		}
	}
	check("pool.physical", g.Pool.Physical, w.Pool.Physical)
	check("pool.loans", g.Pool.Loans, w.Pool.Loans)
	check("pool.dials", []uint256.Int{g.Pool.LtvWad, g.Pool.PhiWad, g.Pool.RoomEpsilonWad, g.Pool.Features},
		[]uint256.Int{w.Pool.LtvWad, w.Pool.PhiWad, w.Pool.RoomEpsilonWad, w.Pool.Features})
	check("switches", []bool{g.Paused, g.LevPaused, g.Hooks.hasLeverage(), g.Hooks.hasSpread()},
		[]bool{w.Paused, w.LevPaused, w.Hooks.hasLeverage(), w.Hooks.hasSpread()})
	check("fee bounds", []uint256.Int{g.FeeFloorWad, g.FeeCapWad}, []uint256.Int{w.FeeFloorWad, w.FeeCapWad})
	check("shareSupply", g.ShareSupply, w.ShareSupply)
	check("lastLeverSpreadPpm", g.LastLeverSpreadPpm, w.LastLeverSpreadPpm)
	check("hook", g.Hooks.Swap, w.Hooks.Swap)
	check("spread", g.Hooks.Spread, w.Hooks.Spread)
	check("hook set", []any{g.Hooks.Addrs, g.Hooks.Leverage}, []any{w.Hooks.Addrs, w.Hooks.Leverage})
	check("feed", g.Feed, w.Feed)
	check("poolAsset", g.PoolAsset, w.PoolAsset)
	check("block", g.Block, w.Block)
	for i := range w.Router.Venues {
		if i < len(g.Router.Venues) {
			check(fmt.Sprintf("router.venue[%d]", i), g.Router.Venues[i], w.Router.Venues[i])
		}
	}
	g.Router.Venues, w.Router.Venues = nil, nil
	check("router", g.Router, w.Router)
	return d
}

func e2eWord(b bool) uint256.Int {
	if b {
		return *uint256.NewInt(1)
	}
	return uint256.Int{}
}

// TestCoreE2ESequences replays every executed sequence from the port's own post-states.
func TestCoreE2ESequences(t *testing.T) {
	t.Parallel()
	for _, blk := range e2eBlocks {
		t.Run(blk, func(t *testing.T) {
			rep := &e2eReport{t: t}
			var s *flammState
			var now uint64
			var seq string
			executed := map[string]int{}
			for _, line := range e2eLines(t, e2eFixture("core_e2e_seq_"+blk+".jsonl.gz")) {
				var st e2eStep
				require.NoError(t, json.Unmarshal(line, &st))
				switch st.K {
				case "seq":
					var reads *flammReads
					var views map[string]json.RawMessage
					s, reads, views = e2eState(t, st.S)
					now, seq = reads.Timestamp, st.Seq
					e2eAttest(rep, seq+" start", s, reads, views, now)
					continue
				case "sw", "lv":
					row := e2eRow{K: st.K, D: st.D, A: st.A, R: st.R, E: st.E}
					var got []uint256.Int
					var err error
					if st.K == "sw" {
						got, err = e2ePreviewSwapWords(s, st.D == 1, &st.A, now)
					} else {
						got, err = e2ePreviewLeverWords(s, st.D == 1, &st.A, now)
					}
					e2eCompare(rep, fmt.Sprintf("%s preview %s d=%d a=%s", seq, st.K, st.D, st.A.Dec()), &row, err, got)
					continue
				}
				where := fmt.Sprintf("%s step %d %s %v", seq, st.I, st.Op, st.Args)
				now = st.Ts
				want, wantReads, wantViews := e2eState(t, st.Post)
				var err error
				var post *flammState
				var words []uint256.Int
				var events []e2eEvent
				switch st.Op {
				case "swap":
					sell := !st.Args[0].IsZero()
					tokenIn, tokenOut := s.PoolAsset, s.Pool.Loans[0].Token
					if !sell {
						tokenIn, tokenOut = tokenOut, tokenIn
					}
					if !st.Args[4].IsZero() {
						tokenIn, tokenOut = s.Pool.Loans[0].Token, s.Pool.Loans[0].Token
					}
					var r *swapResult
					r, post, err = s.executeSwap(tokenIn, tokenOut, &st.Args[1], &st.Args[2], now-st.Args[3].Uint64(), now)
					if err == nil {
						words = []uint256.Int{r.AmountInUsed, r.AmountOut}
						events = []e2eEvent{{Name: "Swap", W: []uint256.Int{e2eWord(r.PoolAssetIn), r.AmountInUsed,
							r.AmountOut, r.FeeOut, r.FeeWad, r.SpotAfterWad}}}
						if blk == "51302915" && seq == "real_sell" {
							require.Equal(t, "15000", r.AmountInUsed.Dec())
							require.Equal(t, "11301759", r.AmountOut.Dec(), "the real sell of tx 0x46c3cd72...")
						}
					}
				case "leverUp", "leverDown":
					var r *leverResult
					r, post, err = s.executeLever(st.Op == "leverUp", &st.Args[0], &st.Args[1], now, now)
					if err == nil {
						words = []uint256.Int{r.AmountInUsed, r.AmountOut}
						events = []e2eEvent{{Name: map[bool]string{true: "LeverUp", false: "LeverDown"}[r.Up],
							W: []uint256.Int{r.AmountInUsed, r.AmountOut, r.SpreadPpm, r.CrAfterWad}}}
					}
				default:
					post = s.clone()
					switch st.Op {
					case "setLoanConfig":
						l := &post.Pool.Loans[0]
						l.SwapPriceBandWad, l.FeeFloorWad = st.Args[0], st.Args[1]
						l.MaxSwapNotional, l.ReserveTarget = st.Args[2], st.Args[3]
					case "setLevPaused":
						post.LevPaused = !st.Args[0].IsZero()
					case "setPaused":
						post.Paused = !st.Args[0].IsZero()
					case "setFeeBounds":
						post.FeeFloorWad, post.FeeCapWad = st.Args[0], st.Args[1]
					case "setSpread":
						post.Hooks.Spread.EverlongSpread.Spread = st.Args[0]
						post.Hooks.Spread.EverlongSpread.LastSetTs.SetUint64(now)
					case "setMaxSpreadAge":
						post.Hooks.Spread.EverlongSpread.MaxSpreadAge = st.Args[0]
					case "warp":
					case "mockFeeds", "irmDown", "storePin":
						// Tracker inputs: a new aggregator round, a rate model that stopped answering, a pin.
						post.Feed = want.Feed.clone()
						post.Router.PinLtvWad = want.Router.PinLtvWad
						for i := range post.Router.Venues {
							v, w := &post.Router.Venues[i].Morpho, &want.Router.Venues[i].Morpho
							v.IrmReadable, v.OracleOk, v.OraclePrice, v.OracleZero = w.IrmReadable, w.OracleOk, w.OraclePrice, w.OracleZero
						}
					default:
						t.Fatalf("%s: unknown op", where)
					}
				}
				rep.rows++
				if st.Ok != (err == nil) {
					rep.fail("%s: chain ok=%v e=%v, go err %v", where, st.Ok, st.E, err)
				} else if !st.Ok {
					if w := e2eRevert(*st.E); w == nil || !errors.Is(err, w) {
						rep.fail("%s: chain revert %s, go %v", where, *st.E, err)
					}
					post = s
				} else {
					for i := range st.R {
						if !st.R[i].Eq(&words[i]) {
							rep.fail("%s: return %d chain %s go %s", where, i, st.R[i].Dec(), words[i].Dec())
						}
					}
					if len(st.Ev) != len(events) {
						rep.fail("%s: chain events %+v go %+v", where, st.Ev, events)
					} else {
						for i := range st.Ev {
							if st.Ev[i].Name != events[i].Name || fmt.Sprint(st.Ev[i].W) != fmt.Sprint(events[i].W) {
								rep.fail("%s: event chain %+v go %+v", where, st.Ev[i], events[i])
							}
						}
					}
					if st.Op == "swap" || st.Op == "leverUp" || st.Op == "leverDown" {
						executed[st.Op]++
					}
				}
				post.Timestamp = want.Timestamp // entries run at the step's timestamp; the stamp is the reader's
				if d := e2eDiff(post, want); len(d) != 0 {
					rep.fail("%s: post-state differs:\n  %s", where, strings.Join(d, "\n  "))
					post = want // keep replaying from the chain's state so one divergence reports once
				}
				e2eAttest(rep, where+" post", post, wantReads, wantViews, now)
				s = post
			}
			t.Logf("block %s: %d checks, %d mismatches, executed %v", blk, rep.rows, rep.n, executed)
			require.GreaterOrEqual(t, executed["swap"], 30)
			require.GreaterOrEqual(t, executed["leverUp"], 2)
			require.GreaterOrEqual(t, executed["leverDown"], 3)
		})
	}
}
