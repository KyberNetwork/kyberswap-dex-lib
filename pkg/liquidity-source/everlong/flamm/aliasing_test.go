package everlongflamm

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// aliasingGlobals lists every package-level word a function could hand out by pointer;
// TestAliasingGlobalsComplete keeps it in sync with the var blocks of the non-test files.
func aliasingGlobals() map[string]*uint256.Int {
	return map[string]*uint256.Int{
		"uZero": uZero, "uOne": uOne, "uWad": uWad, "uWadSquared": uWadSquared, "uQ96": uQ96, "uPpm": uPpm,
		"uBps":       uBps,
		"maxUint256": maxUint256,

		"almMinXWad": almMinXWad, "almMaxXWad": almMaxXWad, "almMinAWad": almMinAWad, "almMaxAWad": almMaxAWad,
		"almHalfWad": almHalfWad, "hookKappaSeed": hookKappaSeed,

		"feeSkewRampWad": feeSkewRampWad, "feeSigmaScale": feeSigmaScale, "feeFour": feeFour,
		"feeLnP0": feeLnP0, "feeLnP1": feeLnP1, "feeLnP2": feeLnP2, "feeLnP3": feeLnP3, "feeLnP4": feeLnP4,
		"feeLnP5": feeLnP5, "feeLnP6": feeLnP6, "feeLnQ0": feeLnQ0, "feeLnQ1": feeLnQ1, "feeLnQ2": feeLnQ2,
		"feeLnQ3": feeLnQ3, "feeLnQ4": feeLnQ4, "feeLnQ5": feeLnQ5, "feeLnQ6": feeLnQ6, "feeLnS": feeLnS,
		"feeLnK": feeLnK, "feeLnC": feeLnC,

		"gateMonotoneSlackWad": gateMonotoneSlackWad, "gateReleaseHysteresis": gateReleaseHysteresis,
		"gateWadPlusSlack": gateWadPlusSlack, "gateFeatureSupplyLending": gateFeatureSupplyLending,
		"gateIntMinAbs": gateIntMinAbs, "mmIrmStaleGrace": mmIrmStaleGrace,

		"mmGraceRateWad": mmGraceRateWad, "mmVirtualShares": mmVirtualShares, "mmVirtualAssets": mmVirtualAssets,
		"mmOraclePriceScale": mmOraclePriceScale, "mmMaxUint128": mmMaxUint128, "mmTwoWad": mmTwoWad,
		"mmThreeWad": mmThreeWad, "mmQuarantineCountWad": mmQuarantineCountWad, "mmMaxUint256": mmMaxUint256,

		"levMaxInput": levMaxInput, "levLeverageRatioWad": levLeverageRatioWad, "levHZero": levHZero,
		"levHJoin": levHJoin, "levHWall": levHWall, "levWidth": levWidth, "levDJoin": levDJoin, "levDWall": levDWall,
		"levP0": levP0, "levP1": levP1, "levP2": levP2, "levP3": levP3,
		"levQ1": levQ1, "levQ2": levQ2, "levQ3": levQ3, "levQ4": levQ4,
		"levTargetSpreadCapPpm": levTargetSpreadCapPpm, "levDustAnchorFloor": levDustAnchorFloor,
		"levHalfLawOffset": levHalfLawOffset, "levThreeWad": levThreeWad, "levWadMinusOne": levWadMinusOne,
		"levTwo": levTwo, "levThree": levThree, "levSix": levSix, "levEight": levEight,
		"levRhoWallNumerator": levRhoWallNumerator, "levRhoDenominator": levRhoDenominator,
		"levHWallMinusDWall": levHWallMinusDWall, "levCrCeilingWad": levCrCeilingWad,
		"levMaxConcessionPpm": levMaxConcessionPpm, "levPriceBandUpperWad": levPriceBandUpperWad,
		"levPriceBandLowerWad": levPriceBandLowerWad,

		"leverCrFloorWad": leverCrFloorWad, "leverSpreadFloorPpm": leverSpreadFloorPpm,
		"leverSpreadCeilingPpm": leverSpreadCeilingPpm, "leverBandToPpm": leverBandToPpm,
		"feedMaxPriceWad": feedMaxPriceWad,
	}
}

// TestAliasingGlobalsComplete fails when a non-test file declares a package-level uint256 word that
// aliasingGlobals does not list, so TestAliasingReturnedWords keeps covering every constant.
func TestAliasingGlobalsComplete(t *testing.T) {
	t.Parallel()
	listed := aliasingGlobals()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		require.NoError(t, err)
		f, err := parser.ParseFile(fset, name, src, 0)
		require.NoError(t, err)
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, id := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					init := string(src[fset.Position(vs.Values[i].Pos()).Offset:fset.Position(vs.Values[i].End()).Offset])
					_, aliasOfListed := listed[init]
					isWord := strings.Contains(init, "uint256.") || strings.Contains(init, "big256.") || aliasOfListed
					if isWord && !strings.HasPrefix(init, "errors.New(") {
						require.Contains(t, listed, id.Name, "%s: package word %s is not in aliasingGlobals", name, id.Name)
					}
				}
			}
		}
	}
}

// aliasingClobber overwrites every word so a shared pointer shows up as a changed constant or input.
func aliasingClobber(ws ...*uint256.Int) {
	for _, w := range ws {
		w.SetAllOne()
	}
}

// TestAliasingReturnedWords pins that no returned pointer is a package constant, a big256 constant or a caller's
// input: every result the lev ports hand out, on the refusal and the zero branches included, is written over
// and the constants and inputs must be unchanged afterwards.
func TestAliasingReturnedWords(t *testing.T) {
	globals := aliasingGlobals()
	before := make(map[string]uint256.Int, len(globals))
	for name, w := range globals {
		before[name] = *w
	}
	big256Words := [...]*uint256.Int{big256.BONE, big256.TenPow(21), big256.TenPow(30), big256.TenPow(36),
		big256.U2Pow96, big256.UMaxU128, big256.UMax}
	big256Before := make([]uint256.Int, len(big256Words))
	for i, w := range big256Words {
		big256Before[i] = *w
	}
	// A failure must not leak clobbered constants into the tests that run after this one.
	t.Cleanup(func() {
		for name, w := range globals {
			*w = before[name]
		}
		for i, w := range big256Words {
			*w = big256Before[i]
		}
	})

	// The package words are copies of big256's shared words, never the same pointer.
	for _, pair := range [][2]*uint256.Int{
		{uWad, big256.BONE}, {uWadSquared, big256.TenPow(36)}, {uQ96, big256.U2Pow96},
		{almMaxAWad, big256.TenPow(21)}, {hookKappaSeed, big256.TenPow(30)},
		{mmOraclePriceScale, big256.TenPow(36)}, {mmMaxUint128, big256.UMaxU128}, {mmMaxUint256, big256.UMax},
	} {
		require.NotSame(t, pair[0], pair[1])
		require.Equal(t, pair[0], pair[1])
	}

	_, rows := levLoadHookFixture(t, "testdata/lev_hook_local_fixture.json.gz")
	// Rows 0 and 1 are the VenueGolden leverUp and leverDown fills; row 596 frames with D <= 0 (xAnchor zero).
	for _, k := range []int{0, 1, 596} {
		ctx, book := rows[k].context()
		ctxIn := ctx.AmountIn
		f, err := levFrameFor(&ctx.Pool, book)
		require.NoError(t, err)
		aliasingClobber(f.Cv, f.V, f.S, f.XAnchor, (*uint256.Int)(f.D))
		if k == 596 {
			continue
		}
		fill, err := levQuote(ctx, book)
		require.NoError(t, err)
		aliasingClobber(fill.AmountInUsed, fill.GrossOut, fill.VirtualLegL18, fill.CrAfterWad)
		require.Equal(t, ctxIn, ctx.AmountIn, "row %d: levFill.AmountInUsed aliases ctx.AmountIn", k)
	}

	// Library refusals and zero branches.
	u := uint256.NewInt
	coll, debt, price := u(1000), u(10), u(0)
	a, b, c := levDeleverageQuote(coll, debt, price, levLeverageRatioWad, u(100), u(5))
	aliasingClobber(a, b, c)
	a, b, c = levLeverageQuote(coll, debt, price, levLeverageRatioWad, u(100), u(5))
	aliasingClobber(a, b, c)
	require.Equal(t, u(1000), coll)
	require.Equal(t, u(10), debt)
	cv, _ := levMarkedValue(coll, price)
	aliasingClobber(cv)
	_, anchor, h, _ := levStrictAnchor(u(1), u(1), u(1))
	aliasingClobber(anchor, h)
	_, anchor, h, _ = levStrictAnchor(u(0), u(0), levLeverageRatioWad)
	aliasingClobber(anchor, h)
	_, anchor, h, _ = levStrictAnchor(u(1e18), u(1), levLeverageRatioWad)
	aliasingClobber(anchor, h)
	aliasingClobber(levHalfLawAnchor(u(1), u(1e18)))
	_, w := levCvRequiredOnAnchor(u(1), u(1e18))
	aliasingClobber(w)
	_, w = levDebtCapOnAnchor(u(1e18), u(1))
	aliasingClobber(w)
	aliasingClobber(levAnchorBestEffortCv(u(1), u(1), u(1)))
	x, y := levAnchorAndBase(u(1), u(1), u(0), levLeverageRatioWad)
	aliasingClobber(x, y)
	posted := u(20_000)
	aliasingClobber(levDeleverageSpread(u(1), u(1), posted))
	aliasingClobber(levDeleverageSpread(u(1e18), u(1), posted))
	o, s := levDeleverageProRata(u(1e18), u(1e18), u(1), u(1), u(1e18), u(1e17), posted)
	aliasingClobber(o, s)
	o, s = levDeleverageProRata(u(100), u(1e18), u(1e18), u(1e18), u(1e18), u(1e17), posted)
	aliasingClobber(o, s)
	o, s = levDeleverageProRata(u(1000), u(1e18), u(1e18), u(1e18), u(1e18), u(1e17), posted)
	aliasingClobber(o, s)
	require.Equal(t, u(20_000), posted)
	params := levCurveFrozenParams()
	aliasingClobber(params[:]...)

	for name, w := range globals {
		require.Equal(t, before[name], *w, "package constant %s was written through a returned pointer", name)
	}
	for i, w := range big256Words {
		require.Equal(t, big256Before[i], *w, "big256 constant %d was written through a returned pointer", i)
	}
}
