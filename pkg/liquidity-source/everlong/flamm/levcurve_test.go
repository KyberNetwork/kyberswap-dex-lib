package everlongflamm

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// The Rust<->Solidity leverage-curve parity tape, replayed against the Go port with the assertions of
// test/flamm/lev/CollRebalancerMathLevCurveParity.t.sol @ c104 80abd43. The tape carries the Rust twin's
// answer; the expected SOLIDITY answer is reconstructed per row class exactly as the Solidity gate's
// _expectedOut/_assertRow do, so every row pins the deployed library's payout wei for wei.
const (
	levTapeArchive        = "testdata/lev_curve_tape_v1.tar.gz"
	levTapeArchiveSha256  = "18fe3e2aa02cce91f8b312f95730b2ef556363e271e82dbbc12d1d107ce29437"
	levTapeManifestSha256 = "9df07c65ea45563c0f343ed98290df1fb10fa1a3ea870dbc3c9bf51c26a058f6"
	levTapeStem           = "levcurve_c1_tape_v1"
	levTapeStride         = 13
)

const (
	levTapeKind = iota
	levTapeCollateral
	levTapeDebt
	levTapePrice
	levTapeSpread
	levTapeAmountIn
	levTapeRustOut
	levTapeRustNewCollateral
	levTapeRustNewDebt
	levTapeClass
	levTapeAnchor
	levTapeOutGross
	levTapeGrossCapped
)

const (
	levClsExact = iota
	levClsCrossing
	levClsDustGuard
	levClsNoCappedPortion
	levClsLiveness
)

type levTapeClasses struct {
	Exact           int `json:"EXACT"`
	Crossing        int `json:"PRORATA_CROSSING"`
	DustGuard       int `json:"PRORATA_DUST_GUARD"`
	NoCappedPortion int `json:"PRORATA_NO_CAPPED_PORTION"`
	Liveness        int `json:"LIVENESS_EXCLUSION"`
}

type levTapeManifest struct {
	Schema     string         `json:"schema"`
	Stride     int            `json:"stride"`
	RustSha256 string         `json:"rust_sha256"`
	Rows       int            `json:"rows"`
	Classes    levTapeClasses `json:"classes"`
	Blocks     []struct {
		Block  string `json:"block"`
		File   string `json:"file"`
		Rows   int    `json:"rows"`
		Sha256 string `json:"sha256"`
	} `json:"blocks"`
}

type levTapeBlock struct {
	Block   string         `json:"block"`
	Rows    int            `json:"rows"`
	Classes levTapeClasses `json:"classes"`
	V       []string       `json:"v"`
}

func levReadTape(t *testing.T) map[string][]byte {
	t.Helper()
	raw, err := os.ReadFile(levTapeArchive)
	require.NoError(t, err)
	sum := sha256.Sum256(raw)
	require.Equal(t, levTapeArchiveSha256, hex.EncodeToString(sum[:]), "tape archive is not the pinned one")
	gz, err := gzip.NewReader(strings.NewReader(string(raw)))
	require.NoError(t, err)
	tr := tar.NewReader(gz)
	files := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		body, err := io.ReadAll(tr)
		require.NoError(t, err)
		files[filepath.Base(hdr.Name)] = body
	}
	return files
}

func levSha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return "0x" + hex.EncodeToString(sum[:])
}

func TestLevCurveParityTape(t *testing.T) {
	t.Parallel()
	files := levReadTape(t)
	manifestRaw := files[levTapeStem+".manifest.json"]
	require.Equal(t, "0x"+levTapeManifestSha256, levSha256Hex(manifestRaw), "tape manifest is not the pinned one")

	var m levTapeManifest
	require.NoError(t, json.Unmarshal(manifestRaw, &m))
	require.Equal(t, "levcurve-c1-parity-v1", m.Schema)
	require.Equal(t, levTapeStride, m.Stride)
	require.Equal(t, "0x894a710140d9070849665cb63597847d2f2252a5d6962f27e0ad0f3a0a1624fd", m.RustSha256)
	require.Equal(t, levTapeClasses{18_876, 3_102, 194, 185, 108}, m.Classes)
	require.Equal(t, 22_465, m.Rows)
	require.Len(t, m.Blocks, 16)

	total := 0
	for i, b := range m.Blocks {
		b := b
		t.Run(fmt.Sprintf("%02d_%s", i, b.Block), func(t *testing.T) {
			raw, ok := files[b.File]
			require.True(t, ok, "block file missing from the archive")
			require.Equal(t, b.Sha256, levSha256Hex(raw), "block file does not match the digest the manifest pins")
			var blk levTapeBlock
			require.NoError(t, json.Unmarshal(raw, &blk))
			require.Equal(t, b.Block, blk.Block)
			require.Len(t, blk.V, blk.Rows*levTapeStride)
			require.Equal(t, b.Rows, blk.Rows)
			v := make([]*uint256.Int, len(blk.V))
			for k, s := range blk.V {
				v[k] = uint256.MustFromDecimal(s)
			}
			keep := sampleRows(blk.Rows, rowSampling{stride: 32, rare: 64}, func(i int) (string, string) {
				r := v[i*levTapeStride : (i+1)*levTapeStride]
				return r[levTapeKind].Dec(), fmt.Sprintf("%s|%v|%v", r[levTapeClass].Dec(), r[levTapeRustOut].IsZero(),
					r[levTapeAnchor].IsZero())
			})
			got := levReplayTapeRows(t, v, blk.Rows, keep)
			if !fixturesSampled() {
				require.Equal(t, blk.Classes, got, "class tally for this block moved")
			}
		})
		total += b.Rows
	}
	require.Equal(t, m.Rows, total)
}

func levReplayTapeRows(t *testing.T, v []*uint256.Int, rows int, keep []bool) levTapeClasses {
	var tally levTapeClasses
	for i := 0; i < rows; i++ {
		if !keep[i] {
			continue
		}
		r := v[i*levTapeStride : (i+1)*levTapeStride]
		var out, newCollateral, newDebt *uint256.Int
		if r[levTapeKind].IsZero() {
			out, newCollateral, newDebt = levLeverageQuote(r[levTapeCollateral], r[levTapeDebt], r[levTapePrice],
				levLeverageRatioWad, r[levTapeSpread], r[levTapeAmountIn])
		} else {
			out, newCollateral, newDebt = levDeleverageQuote(r[levTapeCollateral], r[levTapeDebt], r[levTapePrice],
				levLeverageRatioWad, r[levTapeSpread], r[levTapeAmountIn])
		}
		msg := fmt.Sprintf("row %d %v", i, r)
		levAssertTapeSelfConsistent(t, r, msg)
		levAssertTapeStructural(t, r, out, msg)
		levAssertTapeRow(t, r, out, newCollateral, newDebt, levTapeExpectedOut(r), msg)
		switch r[levTapeClass].Uint64() {
		case levClsExact:
			tally.Exact++
		case levClsCrossing:
			tally.Crossing++
		case levClsDustGuard:
			tally.DustGuard++
		case levClsNoCappedPortion:
			tally.NoCappedPortion++
		case levClsLiveness:
			tally.Liveness++
		default:
			t.Fatalf("%s: tape declares a class this gate has no sanction for", msg)
		}
	}
	return tally
}

// levTapeCliffSpread is the Rust twin's effective_deleverage_spread cliff the library replaced.
func levTapeCliffSpread(cv, debt, posted *uint256.Int) *uint256.Int {
	if !levMul(levTwo, debt).Lt(cv) && posted.Gt(levTargetSpreadCapPpm) {
		return levTargetSpreadCapPpm
	}
	return posted
}

func levTapeMarked(collateral, price *uint256.Int) *uint256.Int {
	if price.IsZero() {
		return uZero
	}
	return levMulDivFloor(collateral, price, uWad)
}

func levAssertTapeSelfConsistent(t *testing.T, r []*uint256.Int, msg string) {
	rustOut, outGross := r[levTapeRustOut], r[levTapeOutGross]
	if !outGross.IsZero() {
		rate := r[levTapeSpread]
		if !r[levTapeKind].IsZero() {
			rate = levTapeCliffSpread(levTapeMarked(r[levTapeCollateral], r[levTapePrice]), r[levTapeDebt], rate)
		}
		require.Equal(t, rustOut, levMulDivFloor(outGross, levSub(uPpm, rate), uPpm), "tape outGross does not reproduce the Rust answer: "+msg)
		require.False(t, rustOut.IsZero(), msg)
	}
	switch {
	case rustOut.IsZero():
		require.Equal(t, r[levTapeCollateral], r[levTapeRustNewCollateral], msg)
		require.Equal(t, r[levTapeDebt], r[levTapeRustNewDebt], msg)
	case r[levTapeKind].IsZero():
		require.Equal(t, levAdd(r[levTapeCollateral], r[levTapeAmountIn]), r[levTapeRustNewCollateral], msg)
		require.Equal(t, levAdd(r[levTapeDebt], rustOut), r[levTapeRustNewDebt], msg)
	default:
		require.Equal(t, levSub(r[levTapeCollateral], rustOut), r[levTapeRustNewCollateral], msg)
		require.Equal(t, levSub(r[levTapeDebt], r[levTapeAmountIn]), r[levTapeRustNewDebt], msg)
	}
}

func levAssertTapeStructural(t *testing.T, r []*uint256.Int, out *uint256.Int, msg string) {
	cls := r[levTapeClass].Uint64()
	if r[levTapeKind].IsZero() {
		require.EqualValues(t, levClsExact, cls, "a leverage row diverged: "+msg)
	}
	if !r[levTapeSpread].Gt(levTargetSpreadCapPpm) {
		require.EqualValues(t, levClsExact, cls, "a row at or below the spread cap diverged: "+msg)
	}
	if r[levTapeAnchor].IsZero() {
		require.EqualValues(t, levClsExact, cls, "a row off the strict-anchor branch diverged: "+msg)
	}
	if r[levTapeRustOut].IsZero() {
		require.True(t, out.IsZero(), "filled where the Rust twin declined: "+msg)
	} else {
		require.False(t, out.Gt(r[levTapeRustOut]), "released MORE than the Rust twin: "+msg)
	}
	if !r[levTapeKind].IsZero() && !r[levTapeAnchor].IsZero() {
		cv := levTapeMarked(r[levTapeCollateral], r[levTapePrice])
		debt := r[levTapeDebt]
		require.Equal(t, !levMul(levThree, debt).Lt(r[levTapeAnchor]), !levMul(levTwo, debt).Lt(cv), msg)
	}
}

func levTapeExpectedOut(r []*uint256.Int) *uint256.Int {
	spread, outGross, grossCapped := r[levTapeSpread], r[levTapeOutGross], r[levTapeGrossCapped]
	switch r[levTapeClass].Uint64() {
	case levClsExact:
		return r[levTapeRustOut]
	case levClsCrossing:
		out := levMulDivFloor(grossCapped, levSub(uPpm, levTargetSpreadCapPpm), uPpm)
		return out.Add(out, levMulDivFloor(levSub(outGross, grossCapped), levSub(uPpm, spread), uPpm))
	case levClsDustGuard, levClsNoCappedPortion:
		return levMulDivFloor(outGross, levSub(uPpm, spread), uPpm)
	}
	return uZero
}

func levAssertTapeRow(t *testing.T, r []*uint256.Int, out, newCollateral, newDebt, expected *uint256.Int, msg string) {
	cls := r[levTapeClass].Uint64()
	collateral, debt, spread := r[levTapeCollateral], r[levTapeDebt], r[levTapeSpread]
	anchor, outGross, grossCapped, rustOut := r[levTapeAnchor], r[levTapeOutGross], r[levTapeGrossCapped], r[levTapeRustOut]

	if cls == levClsExact {
		require.Equal(t, rustOut, out, "EXACT payout: "+msg)
		require.Equal(t, r[levTapeRustNewCollateral], newCollateral, "EXACT collateral: "+msg)
		require.Equal(t, r[levTapeRustNewDebt], newDebt, "EXACT debt: "+msg)
		return
	}
	require.EqualValues(t, 1, r[levTapeKind].Uint64(), msg)
	require.True(t, spread.Gt(levTargetSpreadCapPpm), msg)
	require.False(t, anchor.IsZero(), msg)
	require.False(t, levMul(levThree, debt).Lt(anchor), msg)
	require.False(t, outGross.IsZero(), msg)

	switch cls {
	case levClsCrossing:
		require.False(t, anchor.Lt(levDustAnchorFloor), msg)
		require.False(t, grossCapped.IsZero(), msg)
		require.True(t, grossCapped.Lt(outGross), msg)
		require.Equal(t, expected, out, "crossing payout is not the pro-rata blend: "+msg)
		require.True(t, out.Lt(rustOut), "crossing row reproduces the cliff: "+msg)
	case levClsDustGuard:
		require.True(t, anchor.Lt(levDustAnchorFloor), msg)
		require.True(t, grossCapped.IsZero(), msg)
		require.Equal(t, expected, out, "dust-guard payout: "+msg)
		require.True(t, out.Lt(rustOut), msg)
		require.False(t, levSub(rustOut, out).GtUint64(2), msg)
	case levClsNoCappedPortion:
		require.False(t, anchor.Lt(levDustAnchorFloor), msg)
		require.True(t, grossCapped.IsZero(), msg)
		require.Equal(t, expected, out, "no-capped-portion payout: "+msg)
		require.True(t, out.Lt(rustOut), msg)
	default:
		require.EqualValues(t, 2, outGross.Uint64(), msg)
		require.EqualValues(t, 1, grossCapped.Uint64(), msg)
		require.EqualValues(t, 1, rustOut.Uint64(), msg)
		require.True(t, out.IsZero(), msg)
		require.Equal(t, collateral, newCollateral, msg)
		require.Equal(t, debt, newDebt, msg)
		return
	}
	require.Equal(t, levSub(collateral, out), newCollateral, msg)
	require.Equal(t, r[levTapeRustNewDebt], newDebt, msg)
}

type levCurveForkFixture struct {
	Block        uint64          `json:"block"`
	Math         string          `json:"math"`
	FrozenParams levFixtureUints `json:"frozenParams"`
	Stride       int             `json:"stride"`
	V            levFixtureUints `json:"v"`
}

// TestLevCurveForkFixture replays a grid called on the deployed CollRebalancerMath bytecode on Base
// (testdata/gen/LevForkFixture.t.sol) and pins the Go constants to its frozenParams() (sampled under -race,
// fixture_sample_test.go).
func TestLevCurveForkFixture(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/lev_curve_fork_fixture.json.gz")
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	require.NoError(t, err)
	var fx levCurveForkFixture
	require.NoError(t, json.NewDecoder(gz).Decode(&fx))
	require.True(t, strings.EqualFold("0xC002d0731E6a2E6e80Be754779bCEf6B01Aff0bb", fx.Math), fx.Math)

	// LevCurveParams: the 13-word curve, hZero, leverageRatioWad, targetSpreadCapPpm, then the four
	// caller-owned fields the library leaves zero.
	require.Len(t, fx.FrozenParams, 20)
	for i, want := range levCurveFrozenParams() {
		require.Equal(t, fx.FrozenParams[i], want, "frozenParams word %d", i)
	}
	for i := 16; i < 20; i++ {
		require.True(t, fx.FrozenParams[i].IsZero())
	}

	require.Equal(t, 9, fx.Stride)
	require.Zero(t, len(fx.V)%fx.Stride)
	var counts [3]int
	var filled [2]int
	n := len(fx.V) / fx.Stride
	keep := sampleRows(n, rowSampling{stride: 16, rare: 64}, func(i int) (string, string) {
		r := fx.V[i*fx.Stride : (i+1)*fx.Stride]
		return r[0].Dec(), fmt.Sprintf("%v|%v", r[6].IsZero(), r[8].IsZero())
	})
	for i := 0; i < n; i++ {
		if !keep[i] {
			continue
		}
		r := fx.V[i*fx.Stride : (i+1)*fx.Stride]
		msg := fmt.Sprintf("row %d %v", i, r)
		kind := r[0].Uint64()
		counts[kind]++
		switch kind {
		case 0, 1:
			quote := levLeverageQuote
			if kind == 1 {
				quote = levDeleverageQuote
			}
			out, newCollateral, newDebt := quote(r[1], r[2], r[3], levLeverageRatioWad, r[4], r[5])
			require.Equal(t, r[6], out, msg)
			require.Equal(t, r[7], newCollateral, msg)
			require.Equal(t, r[8], newDebt, msg)
			if !out.IsZero() {
				filled[kind]++
			}
		case 2:
			xAnchor, baseX := levAnchorAndBase(r[1], r[2], r[3], levLeverageRatioWad)
			require.Equal(t, r[6], xAnchor, msg)
			require.Equal(t, r[7], baseX, msg)
			require.Equal(t, !r[8].IsZero(), levIsStateSafe(r[1], r[2], r[3], xAnchor, levLeverageRatioWad), msg)
		default:
			t.Fatal(msg)
		}
	}
	t.Logf("block %d: %d leverage (%d filled), %d deleverage (%d filled), %d anchor rows", fx.Block,
		counts[0], filled[0], counts[1], filled[1], counts[2])
	if !fixturesSampled() {
		require.Greater(t, filled[0], 500)
		require.Greater(t, filled[1], 500)
	}
}
