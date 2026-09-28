package everlongflamm

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// The composed pool core over edge grids and sequences (testdata/edges/core_edge_*.jsonl.gz) from
// testdata/gen/CoreEdge*.sol, written without the CoreE2E generators (own dump, own scenarios) on Base forks.
// Revert data is mapped through a selector table rebuilt here from each sentinel's Solidity signature, not through
// the port's hard-coded bytes.

// coreEdgeLoad returns the non-empty JSONL lines of a (gzipped) fixture, checking its sha256 when digest is set.
func coreEdgeLoad(t *testing.T, path, digest string) [][]byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	if digest != "" {
		sum := sha256.Sum256(raw)
		require.Equal(t, digest, hex.EncodeToString(sum[:]), "%s is not the pinned fixture", path)
	}
	var rd io.Reader = bytes.NewReader(raw)
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(rd)
		require.NoError(t, err)
		rd = zr
	}
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	var out [][]byte
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) > 0 {
			out = append(out, append([]byte(nil), sc.Bytes()...))
		}
	}
	require.NoError(t, sc.Err())
	return out
}

var (
	coreEdgeSelectorsOnce sync.Once
	coreEdgeSelectors     map[[4]byte]error
	coreEdgeSelectorsErr  string
)

// coreEdgeSelectorTable rebuilds selector -> sentinel from the signatures carried in the sentinels' messages and
// asserts the port's revertSelectors agrees entry for entry.
func coreEdgeSelectorTable(t *testing.T) map[[4]byte]error {
	t.Helper()
	coreEdgeSelectorsOnce.Do(func() {
		m := map[[4]byte]error{}
		for key, e := range revertSelectors {
			sig := strings.TrimPrefix(e.Error(), "everlong-flamm: ")
			var sel [4]byte
			copy(sel[:], crypto.Keccak256([]byte(sig))[:4])
			if sel != key && coreEdgeSelectorsErr == "" {
				coreEdgeSelectorsErr = fmt.Sprintf("revertSelectors key %x for %s is not its selector %x", key, sig, sel)
			}
			m[sel] = e
		}
		coreEdgeSelectors = m
	})
	require.Empty(t, coreEdgeSelectorsErr)
	return coreEdgeSelectors
}

var coreEdgeMorphoStrings = map[string]error{
	"max uint128 exceeded":    errMMMaxUint128Exceeded,
	"insufficient liquidity":  errMMInsufficientLiquidity,
	"insufficient collateral": errMMInsufficientCollateral,
	"inconsistent input":      errMMInconsistentInput,
	"zero assets":             errMMZeroAssets,
}

// coreEdgeRevert maps revert data to the port's error vocabulary (nil: unmapped).
func coreEdgeRevert(t *testing.T, h string) error {
	b, err := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	if err != nil {
		return nil
	}
	switch string(b) {
	case "":
		return errMulDivOverflow
	case "oracle down":
		return errMMOracleReverted
	case "down":
		return errMMIrmReverted
	}
	if len(b) < 4 {
		return nil
	}
	var sel [4]byte
	copy(sel[:], b[:4])
	switch sel {
	case [4]byte{0x4e, 0x48, 0x7b, 0x71}: // Panic(uint256)
		if len(b) != 36 {
			return nil
		}
		switch b[35] {
		case 0x11:
			return errPanicArithmetic
		case 0x12:
			return errPanicDivZero
		case 0x32:
			return errPanicIndex
		}
		return nil
	case [4]byte{0x08, 0xc3, 0x79, 0xa0}: // Error(string)
		if len(b) < 68 {
			return nil
		}
		n := new(uint256.Int).SetBytes(b[36:68]).Uint64()
		if uint64(len(b)) < 68+n {
			return nil
		}
		return coreEdgeMorphoStrings[string(b[68:68+n])]
	}
	return coreEdgeSelectorTable(t)[sel]
}

// coreEdgeWords decodes ABI return data into words.
func coreEdgeWords(h string) []uint256.Int {
	b, _ := hex.DecodeString(strings.TrimPrefix(h, "0x"))
	out := make([]uint256.Int, len(b)/32)
	for i := range out {
		out[i].SetBytes(b[32*i : 32*i+32])
	}
	return out
}

type coreEdgeReport struct {
	t      *testing.T
	fails  int
	checks int
	quiet  bool // sensitivity runs count mismatches without failing the test, and stop at the first
}

func (r *coreEdgeReport) fail(format string, args ...any) {
	r.t.Helper()
	r.fails++
	if !r.quiet && r.fails <= 60 {
		r.t.Errorf(format, args...)
	}
}

// coreEdgeMatch compares one call: same words, or both refusing with the same revert class.
func (r *coreEdgeReport) match(where string, chainOk bool, ret, revert string, goErr error, got []uint256.Int) {
	r.t.Helper()
	r.checks++
	if !chainOk {
		want := coreEdgeRevert(r.t, revert)
		if want == nil {
			r.fail("%s: unmapped chain revert %s (go err=%v words=%v)", where, revert, goErr, got)
		} else if !errors.Is(goErr, want) {
			r.fail("%s: chain reverts %v, go err=%v words=%v", where, want, goErr, got)
		}
		return
	}
	if goErr != nil {
		r.fail("%s: chain %v, go err %v", where, coreEdgeWords(ret), goErr)
		return
	}
	w := coreEdgeWords(ret)
	if len(w) < len(got) {
		r.fail("%s: chain returned %d words, go %d", where, len(w), len(got))
		return
	}
	for i := range got {
		if !w[i].Eq(&got[i]) {
			r.fail("%s: word %d chain %s go %s (chain %v go %v)", where, i, w[i].Dec(), got[i].Dec(), w[:len(got)], got)
			return
		}
	}
}

func coreEdgeState(t *testing.T, raw json.RawMessage) (*flammState, *flammReads) {
	t.Helper()
	var reads flammReads
	require.NoError(t, json.Unmarshal(raw, &reads))
	s, err := reads.baseState()
	require.NoError(t, err)
	return s, &reads
}

func coreEdgeSwapWords(s *flammState, sell bool, a *uint256.Int, now uint64) ([]uint256.Int, error) {
	p, err := s.previewSwap(sell, a, now)
	if err != nil {
		return nil, err
	}
	return []uint256.Int{p.UsedNative, p.NetNative, p.FeeWad}, nil
}

func coreEdgeLeverWords(s *flammState, up bool, a *uint256.Int, now uint64) ([]uint256.Int, error) {
	r, err := s.previewLever(up, a, now)
	if err != nil {
		return nil, err
	}
	return []uint256.Int{r.AmountInUsed, r.AmountOut, r.SpreadPpm, r.CrAfterWad}, nil
}

// coreEdgeDiff lists every differing field of two states except the snapshot stamps and the per-call price frame.
func coreEdgeDiff(got, want *flammState, skip map[string]bool) []string {
	g, w := *got.clone(), *want.clone()
	for _, s := range []*flammState{&g, &w} {
		s.Block, s.Timestamp = 0, 0
		s.Pool.PriceWad, s.Pool.CrossWad = nil, nil
		s.Router.TransientRepay = nil
	}
	var d []string
	gv, wv := reflect.ValueOf(g), reflect.ValueOf(w)
	var walk func(path string, a, b reflect.Value)
	walk = func(path string, a, b reflect.Value) {
		if skip[path] {
			return
		}
		switch a.Kind() {
		case reflect.Struct:
			if a.Type() == reflect.TypeOf(uint256.Int{}) {
				x, y := a.Interface().(uint256.Int), b.Interface().(uint256.Int)
				if !x.Eq(&y) {
					d = append(d, fmt.Sprintf("%s: go %s chain %s", path, x.Dec(), y.Dec()))
				}
				return
			}
			for i := 0; i < a.NumField(); i++ {
				walk(path+"."+a.Type().Field(i).Name, a.Field(i), b.Field(i))
			}
		case reflect.Pointer:
			if a.IsNil() || b.IsNil() {
				if a.IsNil() != b.IsNil() {
					d = append(d, fmt.Sprintf("%s: go nil %v chain nil %v", path, a.IsNil(), b.IsNil()))
				}
				return
			}
			walk(path, a.Elem(), b.Elem())
		case reflect.Slice:
			if a.Len() != b.Len() {
				d = append(d, fmt.Sprintf("%s: len go %d chain %d", path, a.Len(), b.Len()))
				return
			}
			for i := 0; i < a.Len(); i++ {
				walk(fmt.Sprintf("%s[%d]", path, i), a.Index(i), b.Index(i))
			}
		default:
			if !reflect.DeepEqual(a.Interface(), b.Interface()) {
				d = append(d, fmt.Sprintf("%s: go %v chain %v", path, a.Interface(), b.Interface()))
			}
		}
	}
	walk("s", gv, wv)
	return d
}

type coreEdgeGridRow struct {
	K   string          `json:"k"`
	Tag string          `json:"tag"`
	S   json.RawMessage `json:"s"`
	D   int             `json:"d"`
	A   uint256.Int     `json:"a"`
	C   uint256.Int     `json:"c"`
	P   uint256.Int     `json:"p"`
	R   *string         `json:"r"`
	E   *string         `json:"e"`
	Msg string          `json:"msg"`
}

// coreEdgeGridFixtures: pool previewSwap / previewLever and router.fundingCeiling at scenario states
// (testdata/gen/CoreEdgeGrid.t.sol).
var coreEdgeGridFixtures = map[string]string{
	"51302915": "946d384674b004378198eb5fd023d28fea944ebcf6d626c34994561149d6eba2",
	"51324800": "8c085a8183d9de9a6edbe7fdab5d08e276141a598739bd4a524baf814fe8f570",
	"51326000": "463868de0f612bdb55be5cad602a5911b6a9d08b3f2904cb578c42f9edc7d3b4",
}

func coreEdgeFixturePath(kind, blk string) string {
	return fmt.Sprintf("testdata/edges/core_edge_%s_%s.jsonl.gz", kind, blk)
}

func TestCoreEdgeGrid(t *testing.T) {
	t.Parallel()
	blocks := make([]string, 0, len(coreEdgeGridFixtures))
	for b := range coreEdgeGridFixtures {
		blocks = append(blocks, b)
	}
	sort.Strings(blocks)
	for _, blk := range blocks {
		t.Run(blk, func(t *testing.T) {
			rep := &coreEdgeReport{t: t}
			classes := coreEdgeReplayGrid(t, rep, blk, "", nil)
			t.Logf("block %s: %d checks, %d mismatches; classes %v", blk, rep.checks, rep.fails, classes)
		})
	}
}

// coreEdgeReplayGrid replays one block's grid (only one scenario when onlyTag is set); mutate (sensitivity runs) may
// perturb a freshly built state.
func coreEdgeReplayGrid(t *testing.T, rep *coreEdgeReport, blk, onlyTag string, mutate func(tag string, s *flammState)) map[string]int {
	path, digest := coreEdgeFixturePath("grid", blk), coreEdgeGridFixtures[blk]
	states := map[string]*flammState{}
	nows := map[string]uint64{}
	classes := map[string]int{}
	lines := coreEdgeLoad(t, path, digest)
	sm := rowSampling{stride: 32, rare: 16, edges: true}
	if onlyTag != "" {
		// a sensitivity run replays every row of its scenario
		tagged := []byte(fmt.Sprintf(`"tag":%q`, onlyTag))
		lines = slices.DeleteFunc(lines, func(l []byte) bool { return !bytes.Contains(l, tagged) })
		sm.stride = 1
	}
	rows := make([]coreEdgeGridRow, len(lines))
	for i, line := range lines {
		require.NoError(t, json.Unmarshal(line, &rows[i]))
	}
	keep := sampleRows(len(rows), sm, func(i int) (string, string) {
		row := &rows[i]
		if row.K != "sw" && row.K != "lv" && row.K != "fc" {
			return "", ""
		}
		class := "ok"
		if row.E != nil {
			class = (*row.E + "0000000000")[:10]
		}
		return fmt.Sprintf("%s|%s|%d", row.Tag, row.K, row.D), class
	})
	for i := range rows {
		row := rows[i]
		if rep.quiet && rep.fails > 0 {
			break
		}
		if !keep[i] || (onlyTag != "" && row.Tag != onlyTag) {
			continue
		}
		switch row.K {
		case "note":
			continue
		case "state":
			s, reads := coreEdgeState(t, row.S)
			states[row.Tag], nows[row.Tag] = s, reads.Timestamp
			pos, err := s.Router.positions(reads.Timestamp)
			rep.checks++
			if err != nil {
				rep.fail("%s/%s: positions %v", blk, row.Tag, err)
			} else {
				var g uint256.Int
				g.Add(&s.Pool.Physical, &pos.TotalColl)
				if !g.Eq(&reads.Pool.Gross) {
					rep.fail("%s/%s: gross chain %s go %s", blk, row.Tag, reads.Pool.Gross.Dec(), g.Dec())
				}
			}
			if mutate != nil {
				mutate(row.Tag, s)
			}
			continue
		}
		s := states[row.Tag]
		require.NotNil(t, s, row.Tag)
		now := nows[row.Tag]
		var got []uint256.Int
		var err error
		where := fmt.Sprintf("%s/%s %s d=%d a=%s", blk, row.Tag, row.K, row.D, row.A.Dec())
		switch row.K {
		case "sw":
			got, err = coreEdgeSwapWords(s, row.D == 1, &row.A, now)
		case "lv":
			got, err = coreEdgeLeverWords(s, row.D == 1, &row.A, now)
		case "fc":
			where = fmt.Sprintf("%s/%s fc c=%s p=%s", blk, row.Tag, row.C.Dec(), row.P.Dec())
			var f uint256.Int
			f, err = s.Router.fundingCeiling(0, &row.C, &row.P, now)
			got = []uint256.Int{f}
		default:
			t.Fatalf("row kind %q", row.K)
		}
		rep.match(where, row.R != nil, coreEdgeStr(row.R), coreEdgeStr(row.E), err, got)
		cls := "ok"
		if row.E != nil {
			cls = (*row.E + "0000000000")[:10]
		}
		classes[row.K+" "+cls]++
	}
	return classes
}

// TestCoreEdgeGridSensitivity proves the replays can fail: each small perturbation of a state input the scenario
// exercises must produce mismatches (the unperturbed replay is TestCoreEdgeGrid, which requires none).
func TestCoreEdgeGridSensitivity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		tag    string
		mutate func(s *flammState)
	}{
		{"seq_grace_eq", func(s *flammState) { s.Feed.SequencerGrace.SubUint64(&s.Feed.SequencerGrace, 1) }},
		{"cb_age_eq", func(s *flammState) { s.Feed.Asset.Heartbeat.SubUint64(&s.Feed.Asset.Heartbeat, 1) }},
		{"usdc_age_eq", func(s *flammState) { s.Feed.Loans[0].Heartbeat.SubUint64(&s.Feed.Loans[0].Heartbeat, 1) }},
		{"peg_lo_eq", func(s *flammState) { s.Feed.Loans[0].PegBandWad.SubUint64(&s.Feed.Loans[0].PegBandWad, 1) }},
		{"armed_min", func(s *flammState) {
			sp := s.Hooks.Spread.EverlongSpread
			sp.Spread.AddUint64(&sp.Spread, 1)
		}},
		{"spread_age_eq", func(s *flammState) {
			sp := s.Hooks.Spread.EverlongSpread
			sp.MaxSpreadAge.SubUint64(&sp.MaxSpreadAge, 1)
		}},
		{"degrade_zero", func(s *flammState) { s.LastLeverSpreadPpm.SetUint64(17_500) }},
		{"live", func(s *flammState) {
			h := s.Hooks.Swap.EverlongSwap
			h.ReserveVolatile.AddUint64(&h.ReserveVolatile, 1)
		}},
		{"ratecap_0", func(s *flammState) {
			v := &s.Router.Venues[0]
			v.MaxBorrowRateWad.SubUint64(&v.MaxBorrowRateWad, 1)
		}},
		{"irm_dt_3600", func(s *flammState) { s.Router.Venues[0].Morpho.IrmReadable = true }},
		{"morpho_fee", func(s *flammState) {
			m := &s.Router.Venues[0].Morpho
			m.Market.LastUpdate.SubUint64(&m.Market.LastUpdate, 86_400)
		}},
		{"debtcap_tight", func(s *flammState) {
			v := &s.Router.Venues[0]
			v.DebtCap.AddUint64(&v.DebtCap, 1)
		}},
		{"ltv_low_1", func(s *flammState) { s.Pool.RoomEpsilonWad.AddUint64(&s.Pool.RoomEpsilonWad, 1e15) }},
		{"whale_dt1", func(s *flammState) {
			m := &s.Router.Venues[0].Morpho
			m.Market.TotalSupplyAssets.AddUint64(&m.Market.TotalSupplyAssets, 1)
		}},
	}
	for _, c := range cases {
		t.Run(c.tag, func(t *testing.T) {
			rep := &coreEdgeReport{t: t, quiet: true}
			coreEdgeReplayGrid(t, rep, "51324800", c.tag, func(tag string, s *flammState) {
				if tag == c.tag {
					c.mutate(s)
				}
			})
			require.NotZero(t, rep.fails, "perturbing %s changed no row", c.tag)
		})
	}
}
