package everlongflamm

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// guardVectorsHash pins the guard's shared vectors (testdata/held_repay_guard_vectors.json): the same bytes in every
// client that routes these pools.
const guardVectorsHash = "0x7758eedfa75e7fecaa9d8fc41f194908a2f9eb551c965505e6400affbec50ba8"

// Every shared case keeps the range it states. A case's point is a settled point (x, y), the pool's own answer at x
// (paid) and the label x was priced at: here a priced probe at x.
func TestHeldRepayGuardVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/held_repay_guard_vectors.json")
	require.NoError(t, err)
	require.Equal(t, guardVectorsHash, crypto.Keccak256Hash(raw).Hex(), "the shared bytes")
	type point struct {
		X     string `json:"x"`
		Y     string `json:"y"`
		Paid  string `json:"paid"`
		Label string `json:"label"`
	}
	var doc struct {
		Cases []struct {
			Name    string   `json:"name"`
			Points  []point  `json:"points"`
			Legs    []string `json:"legs"`
			Band    string   `json:"band"`
			Minimum *string  `json:"minimum"`
			Sole    bool     `json:"sole"`
			Keep    [2]int   `json:"keep"`
		} `json:"cases"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Cases)
	num := func(s string) *big.Int {
		v, ok := new(big.Int).SetString(s, 10)
		require.True(t, ok, s)
		return v
	}
	kinds := map[string]int{}
	for _, c := range doc.Cases {
		var x, y []*big.Int
		var probes []sampled
		for _, p := range c.Points {
			x, y = append(x, num(p.X)), append(y, num(p.Y))
			probes = append(probes, sampled{ask: num(p.X), used: num(p.X), out: num(p.Paid), fee: num(p.Label),
				status: statusPriced})
		}
		var legs *[8]*big.Int
		if c.Legs != nil {
			require.Len(t, c.Legs, 8, c.Name)
			legs = new([8]*big.Int)
			for k := range legs {
				legs[k] = num(c.Legs[k])
			}
		}
		g := heldRepay{sole: c.Sole}
		if c.Minimum != nil {
			g.floor = num(*c.Minimum)
		}
		lo, hi := keepHeld(x, y, probes, legs, num(c.Band), g)
		require.Equal(t, c.Keep, [2]int{lo, hi}, c.Name)
		switch n := len(c.Points); {
		case hi == 0:
			kinds["none"]++
		case lo == 0 && hi == n:
			kinds["whole"]++
		case lo == 0:
			kinds["custody"]++
		default:
			kinds["past the window"]++
		}
	}
	require.Len(t, kinds, 4, "every outcome among the cases")
	t.Logf("%d cases: %v", len(doc.Cases), kinds)
}

// A cut keeps its points, no flat span under its first, and the input ceiling only where it keeps the top.
func TestCutHeldKeepsOnePiece(t *testing.T) {
	b := func(v int64) *big.Int { return big.NewInt(v) }
	s := settled{x: []*big.Int{b(1), b(2), b(3), b(4)}, y: []*big.Int{b(10), b(20), b(30), b(40)},
		f: []*big.Int{b(0), b(0), b(0), b(0)}, flat: []bool{false, true, true, false}, capped: true}
	high := cutHeld(s, 2, 4)
	require.Equal(t, []*big.Int{b(3), b(4)}, high.x)
	require.Equal(t, []bool{false, false}, high.flat, "no span under the first point")
	require.True(t, high.capped, "the top stays")
	require.Equal(t, []bool{false, true, true, false}, s.flat, "the settled ladder unchanged")
	low := cutHeld(s, 0, 2)
	require.Equal(t, []*big.Int{b(10), b(20)}, low.y)
	require.False(t, low.capped, "cut under the ceiling")
	require.Equal(t, s, cutHeld(s, 0, 4))
	require.Empty(t, cutHeld(s, 0, 0).x)
}

// listaWorld is tierWorld on BSC, its USDC lent by a Lista Moolah venue against cbBTC whose minimum is 1,000,000
// USDC, the pool holding 1,000,000 USDC and posting 500,000 more: F is 1,500,000 USDC and E 2,510,000.
func listaWorld() (*world, *fakePool, *Config) {
	w, fp := tierWorld()
	cfg := testConfig()
	cfg.ChainID = valueobject.ChainIDBSC
	fp.venues = []venueView{venue(3, 0, 1, 0)}
	w.minimum = big.NewInt(1_000_000_000_000)
	fp.legs[1] = [4]*big.Int{big.NewInt(1_000_000_000_000), new(big.Int), big.NewInt(500_000_000_000), new(big.Int)}
	return w, fp, cfg
}

// The guard on a refresh: a sale paying USDC that Lista lends keeps the points paying at least E, or those its
// custody pays, or the whole ladder where no draw can leave the debt under its minimum; the buy, whose output no
// held-repay venue lends, is never cut.
func TestTrackerKeepsASaleOutOfListasWindow(t *testing.T) {
	const custody, freed, edge = 1_000_000_000_000, 1_500_000_000_000, 2_510_000_000_000
	ladderOf := func(t *testing.T, edit func(*world, *fakePool, *Config)) map[uint8]Ladder {
		w, fp, cfg := listaWorld()
		if edit != nil {
			edit(w, fp, cfg)
		}
		n, pools := listed(t, w, cfg)
		p, err := track(t, n, cfg, pools[0])
		require.NoError(t, err)
		return lo.SliceToMap(extraOf(t, p).Ladders, func(l Ladder) (uint8, Ladder) { return l.In, l })
	}
	// the control: an Aave V3 venue in Lista's place, which holds no minimum
	open := ladderOf(t, func(_ *world, fp *fakePool, _ *Config) { fp.venues = []venueView{venue(1, 0, 1, 0)} })
	whole := open[0].Points
	require.NotEmpty(t, whole)
	require.Positive(t, whole[len(whole)-1][1].CmpBig(big.NewInt(edge)), "the sale pays past E")
	require.Negative(t, whole[0][1].CmpBig(big.NewInt(custody)), "and from custody")
	firstAt := func(v int64) int {
		return lo.IndexOf(lo.Map(whole, func(p Point, _ int) bool { return !p[1].Lt(uint256.NewInt(uint64(v))) }), true)
	}
	from, past := firstAt(edge), firstAt(freed+1)
	require.Positive(t, from)
	require.Positive(t, past)

	const (
		high         = iota // the points from the first paying at least E
		custodyPiece        // a prefix of the ladder, each point inside custody
		all                 // the whole ladder
		none                // no sale
	)
	for name, tc := range map[string]struct {
		edit  func(*world, *fakePool, *Config)
		piece int
		from  int // the high piece's first point, where not the window's
		buy   bool
	}{
		"past the window":                    {piece: high, buy: true},
		"no minimum read":                    {edit: func(w *world, _ *fakePool, _ *Config) { w.minimum = nil }, piece: custodyPiece, buy: true},
		"minimumDebt reverts":                {edit: func(w *world, _ *fakePool, _ *Config) { w.badMinimum = true }, piece: custodyPiece, buy: true},
		"a second lender of USDC":            {edit: func(_ *world, fp *fakePool, _ *Config) { fp.venues = append(fp.venues, venue(1, 0, 1, 0)) }, piece: custodyPiece, buy: true},
		"owing its minimum and the slack":    {edit: owing(1_010_000_000_000 + releaseSlack), piece: all, buy: true},
		"owing one unit less: past F + 1":    {edit: owing(1_010_000_000_000 + releaseSlack - 1), piece: high, from: past, buy: true},
		"capabilities() unread":              {edit: func(w *world, _ *fakePool, _ *Config) { w.badCaps[3] = true }, piece: none},
		"the request reverts":                {edit: func(_ *world, fp *fakePool, _ *Config) { fp.noRequest = true }, piece: none},
		"the config leaves Lista out":        {edit: func(_ *world, _ *fakePool, cfg *Config) { cfg.VenueKinds = []uint8{1, 2} }, piece: none},
		"Lista on Base, a chain without it":  {edit: func(_ *world, _ *fakePool, cfg *Config) { cfg.ChainID = valueobject.ChainIDBase }, piece: none},
		"Lista lending cbBTC, the buy's out": {edit: func(_ *world, fp *fakePool, _ *Config) { fp.venues = []venueView{venue(3, 1, 0, 0)} }, piece: all, buy: true},
	} {
		t.Run(name, func(t *testing.T) {
			got := ladderOf(t, tc.edit)
			sale, saleOk := got[0]
			_, buyOk := got[1]
			require.Equal(t, tc.buy, buyOk, "the buy")
			switch tc.piece {
			case none:
				require.False(t, saleOk, "no sale")
			case all:
				require.Equal(t, whole, sale.Points)
			case high:
				require.Equal(t, whole[max(tc.from, from*lo.Ternary(tc.from == 0, 1, 0)):], sale.Points)
			case custodyPiece:
				require.True(t, saleOk)
				require.Equal(t, whole[:len(sale.Points)], sale.Points, "a prefix")
				for _, p := range sale.Points {
					require.True(t, p[1].Lt(uint256.NewInt(custody)), "%v inside custody", p)
				}
			}
		})
	}

	// a curve alone: no walk carries a tier label, so no walk reads the pool's request; the tracker reads it for the
	// sale, whose output Lista lends, and the guard cuts the sale as it does where a walk read it
	curve := ladderOf(t, func(w *world, fp *fakePool, cfg *Config) {
		curveOnly(w, fp, cfg)
		fp.venues = []venueView{venue(1, 0, 1, 0)}
	})[0].Points
	at := lo.IndexOf(lo.Map(curve, func(p Point, _ int) bool { return !p[1].Lt(uint256.NewInt(edge)) }), true)
	require.Positive(t, at, "the curve's sale pays past E, and under it first")
	got := ladderOf(t, curveOnly)
	require.Equal(t, curve[at:], got[0].Points, "a curve alone: the points from the first paying at least E")
	require.Contains(t, got, uint8(1), "and the buy")
	got = ladderOf(t, func(w *world, fp *fakePool, cfg *Config) { curveOnly(w, fp, cfg); fp.noRequest = true })
	require.NotContains(t, got, uint8(0), "a curve alone, the request refused: no sale")
	require.Contains(t, got, uint8(1), "and the buy")
}

// curveOnly takes the leverage route out of tierWorld: no walk carries a tier label, so no walk reads the pool's
// request.
func curveOnly(_ *world, fp *fakePool, _ *Config) {
	sale, buy := oracle(1_000, 1, 100, 1_000_000_000), oracle(1, 1_000, 100, 100_000_000_000)
	fp.fee = nil // every fill the curve's label
	fp.quote = func(from common.Address, i, j int, a *big.Int) (*big.Int, *big.Int, bool) {
		switch {
		case from != defaultTaker:
			return nil, nil, false
		case i == 0 && a.Cmp(big.NewInt(5_000_000_000)) <= 0:
			return sale(from, i, j, a)
		case i == 1:
			return buy(from, i, j, minBig(a, big.NewInt(1_000_000_000_000)))
		}
		return nil, nil, false
	}
}

func owing(owed int64) func(*world, *fakePool, *Config) {
	return func(_ *world, fp *fakePool, _ *Config) { fp.legs[1][3] = big.NewInt(owed) }
}

// The repay rule on a buy whose input Lista lends: a repay is certain only where Lista is the input's only lender, its minimum
// read, and the debt past it by the margin, floor x 1.01 + 4; else the paths without a repay stay allowed.
func TestChargeRepaysAListaDebtOnlyPastItsMinimum(t *testing.T) {
	margin := int64(1_010_000_000_000 + releaseSlack)
	for name, tc := range map[string]struct {
		edit  func(*world, *fakePool, *Config)
		repay bool
	}{
		"one unit past the margin": {edit: owing(margin + 1), repay: true},
		"at the margin":            {edit: owing(margin)},
		"no minimum read":          {edit: func(w *world, fp *fakePool, c *Config) { owing(margin+1)(w, fp, c); w.minimum = nil }},
		"a second lender of USDC": {edit: func(w *world, fp *fakePool, c *Config) {
			owing(margin+1)(w, fp, c)
			fp.venues = append(fp.venues, venue(1, 0, 1, 0))
		}},
		"repays refused at Lista": {edit: func(w *world, fp *fakePool, c *Config) {
			owing(margin+1)(w, fp, c)
			fp.acts = map[int]uint8{0: 15 &^ canRepay}
		}},
		"Aave alone, the same debt": {edit: func(w *world, fp *fakePool, c *Config) {
			owing(margin)(w, fp, c)
			fp.venues = []venueView{venue(1, 0, 1, 0)}
		}, repay: true},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp, cfg := listaWorld()
			tc.edit(w, fp, cfg)
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			buy, ok := lo.Find(extraOf(t, p).Ladders, func(l Ladder) bool { return l.In == 1 })
			require.True(t, ok)
			require.Equal(t, owedSome, buy.OwedIn)
			require.Equal(t, tc.repay, buy.Repay)
		})
	}
	floor := uint256.NewInt(1_000_000)
	require.False(t, heldRepayOk(uint256.NewInt(1_010_004), floor))
	require.True(t, heldRepayOk(uint256.NewInt(1_010_005), floor))
	require.False(t, heldRepayOk(uint256.NewInt(4), uint256.NewInt(0)), "a zero floor still needs the slack")
	require.True(t, heldRepayOk(uint256.NewInt(5), uint256.NewInt(0)))
}
