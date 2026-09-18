package everlongflamm

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// Dense fixture replays under the race detector.
//
// Every fixture row is replayed by default. The race detector instruments every limb operation of the uint256
// arithmetic, which makes the dense replays (the preview grids, the curve and hook grids, the leverage tapes, the
// scenario sweeps) some twenty times slower, so under -race -- how the upstream CI runs `go test` -- they replay a
// sample: sampleRows keeps, in every stream of a fixture (a scenario, operation and direction), every stride-th row,
// every row of a rare outcome class (a revert selector, a fill, a partial fill) and, along grids ordered by amount,
// both rows of every change of class; a sweep without recorded classes (sampleCase) keeps every stride-th case,
// offset per scenario so every case runs somewhere. Sequences and sensitivity runs are never sampled.
// EVERLONG_FLAMM_FIXTURES=full replays every row of every fixture, under -race too, and =sampled samples without it.
//
// A sample that reaches the same statements as the full suite does not catch the same faults: what separates a
// correct fill from a wrong one on a grid ordered by amount is the row on either side of a change of class, which
// is where a gate starts binding. So every amount-ordered grid sets `edges`, which keeps both of them -- without
// it the ok -> PriceBand row of the core edge grid is dropped, and that row is the only one that separates the
// lever-up taker leg (lever.go, the floor FLAMMLeverLib.sol:103-105 performs) from the curve's gross output. The
// streams with no amount order (the module tapes) keep a stride and every rare class instead.

func fixturesSampled() bool {
	switch os.Getenv("EVERLONG_FLAMM_FIXTURES") {
	case "full":
		return false
	case "sampled":
		return true
	}
	return raceEnabled
}

// fixturesFull reports whether the run was asked for every row. It is not the negation of fixturesSampled: a test
// whose default keeps a fraction of its fixture even without -race (TestSimulatorPreviewGrid keeps every third
// preview row, because each row there is a full settlement and the grids hold 82,545 of them) replays all of it
// only under EVERLONG_FLAMM_FIXTURES=full, so that the flag means what it says.
func fixturesFull() bool {
	return os.Getenv("EVERLONG_FLAMM_FIXTURES") == "full"
}

// rowSampling is how sampleRows thins one fixture: every stride-th row of each stream, every row of a class the
// stream holds at most rare rows of, and with edges both rows of every change of class along the stream (for grids
// ordered by amount, where a change of class is an edge the chain recorded).
type rowSampling struct {
	stride, rare int
	edges        bool
}

// sampleRows marks which of n rows a replay keeps: all of them unless fixtures are sampled, otherwise the first and
// last row of each stream and the rows rowSampling selects. key names row i's stream and class; a row with an empty
// stream (a state, a note) is always kept.
func sampleRows(n int, sm rowSampling, key func(i int) (stream, class string)) []bool {
	keep := make([]bool, n)
	if !fixturesSampled() || sm.stride <= 1 {
		for i := range keep {
			keep[i] = true
		}
		return keep
	}
	streams, classes := make([]string, n), make([]string, n)
	sizes := map[[2]string]int{}
	for i := 0; i < n; i++ {
		streams[i], classes[i] = key(i)
		sizes[[2]string{streams[i], classes[i]}]++
	}
	type last struct {
		i     int
		class string
		n     int
	}
	seen := map[string]*last{}
	for i := 0; i < n; i++ {
		stream, class := streams[i], classes[i]
		if stream == "" {
			keep[i] = true
			continue
		}
		l, ok := seen[stream]
		switch {
		case !ok:
			keep[i] = true
			l = &last{}
			seen[stream] = l
		case sm.edges && l.class != class:
			keep[l.i], keep[i] = true, true
		case l.n%sm.stride == 0, sizes[[2]string{stream, class}] <= sm.rare:
			keep[i] = true
		}
		l.i, l.class = i, class
		l.n++
	}
	for _, l := range seen {
		keep[l.i] = true
	}
	return keep
}

// sampleCase reports whether case i of scenario combo runs: every case unless fixtures are sampled, otherwise every
// stride-th, offset by combo.
func sampleCase(i, combo, stride int) bool {
	return !fixturesSampled() || stride <= 1 || (i+combo)%stride == 0
}

// kept counts the marked rows.
func kept(keep []bool) int {
	n := 0
	for _, k := range keep {
		if k {
			n++
		}
	}
	return n
}

// readFixture is a fixture's content, decompressed when it is stored gzipped (testdata/README.md: every generated
// fixture above a few tens of kilobytes is, and its digest table pins the stored bytes and the content both).
func readFixture(t testing.TB, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	if !strings.HasSuffix(path, ".gz") {
		return raw
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	require.NoError(t, err)
	out, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	return out
}

// readLines reads the non-blank lines of a (gzipped) JSON-lines fixture.
func readLines(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var rd io.Reader = f
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
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
	return out, sc.Err()
}

// gridStates caches the scenario states of each core end-to-end grid (gridReads): the raw JSON per tag, decoded
// afresh on every call so no caller shares a mutable state.
var gridStates sync.Map // block -> *gridStateSet

type gridStateSet struct {
	once sync.Once
	raw  map[string]json.RawMessage
	err  error
}

func gridStateJSON(t testing.TB, block, tag string) (json.RawMessage, bool) {
	t.Helper()
	v, _ := gridStates.LoadOrStore(block, &gridStateSet{})
	set := v.(*gridStateSet)
	set.once.Do(func() {
		set.raw = map[string]json.RawMessage{}
		lines, err := readLines(e2eFixture("core_e2e_grid_" + block + ".jsonl.gz"))
		if err != nil {
			set.err = err
			return
		}
		for _, line := range lines {
			if !bytes.Contains(line, []byte(`"k":"state"`)) {
				continue
			}
			var row e2eRow
			if set.err = json.Unmarshal(line, &row); set.err != nil {
				return
			}
			if _, dup := set.raw[row.Tag]; !dup {
				set.raw[row.Tag] = row.S
			}
		}
	})
	require.NoError(t, set.err)
	raw, ok := set.raw[tag]
	return raw, ok
}
