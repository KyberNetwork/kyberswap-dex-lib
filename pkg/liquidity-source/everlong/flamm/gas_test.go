package everlongflamm

import (
	"bufio"
	"fmt"
	"math"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The test families' builds (testFamilies), and their cells and second-fill charges: this package's tests add them to
// the generated table, on Base for kinds 0 and 1, with the replay fixtures' builds, and on BSC for kinds 1 to 3; and
// the swap hook's cells alone on Ethereum for kind 0, a kind with cells and no second-fill charge. A kind's charges
// cost 200,000 more per kind number, and the pair hook's 1,000 more than the swap hook's.
var testHookFamily, testPairFamily = crypto.Keccak256Hash(hookCode(0)).Hex(), crypto.Keccak256Hash(pairHookCode).Hex()

var testPaths = []struct {
	sale bool
	path uint8
	gas  int64
}{
	{true, 0, 250_000}, {true, pathOut, 400_000}, {true, pathOut | pathFirst, 600_000},
	{true, pathOut | pathOwing, 500_000}, {false, 0, 300_000}, {false, pathRepay, 350_000},
	{false, pathOut | pathRepay, 450_000},
}

// fixtureFamilies are the builds of the replay fixtures' hooks (testdata/base-fork*.json.gz), which the table does
// not measure: their ladders carry the test cells too.
var fixtureFamilies = []string{
	"0x1121cc662d8bf9691898d7537037c3bccffcda9df242918e4161abe7bbcc00c1",
	"0x659fac288160a9453c24b8c414379bef794b3be3d8bf37bf7209d3d870755de7",
}

func init() {
	add := func(fam string, chain valueobject.ChainID, second bool, kinds ...uint8) {
		for _, kind := range kinds {
			for _, p := range testPaths {
				gasCells = append(gasCells, gasCell{fam, chain, kind, p.sale, p.path, testGas(fam, kind, p.gas)})
			}
			for _, sale := range lo.Ternary(second, []bool{true, false}, nil) {
				gasSecondFills = append(gasSecondFills, gasSecondFill{fam, chain, kind, sale, testSecondGas(fam, kind,
					sale)})
			}
		}
	}
	for _, fam := range append([]string{testHookFamily, testPairFamily}, fixtureFamilies...) {
		add(fam, valueobject.ChainIDBase, true, 0, 1)
	}
	add(testHookFamily, valueobject.ChainIDBSC, true, 1, 2, 3)
	add(testPairFamily, valueobject.ChainIDBSC, true, 1, 2, 3)
	add(testHookFamily, valueobject.ChainIDEthereum, false, 0)
}

func testGas(fam string, kind uint8, g int64) int64 {
	g += 200_000 * int64(kind)
	if fam == testPairFamily {
		g += 1_000
	}
	return g
}

// testSecondGas is a test kind's second-fill charge: under every cell of its side.
func testSecondGas(fam string, kind uint8, sale bool) int64 {
	return testGas(fam, kind, lo.Ternary[int64](sale, 100_000, 90_000))
}

// testSecond is the second-fill charge a ladder of the side carries where the pair's live venues are of `kinds`: the
// least of theirs.
func testSecond(fam string, sale bool, kinds ...uint8) int64 {
	return testSecondGas(fam, slices.Min(kinds), sale)
}

// testCells are the cells a ladder of the side carries where the pair's live venues are of `kinds`: each path at
// its least, by path.
func testCells(fam string, sale bool, kinds ...uint8) []Cell {
	var cs []Cell
	for _, p := range testPaths {
		if p.sale == sale {
			g := testGas(fam, slices.Min(kinds), p.gas)
			cs = append(cs, Cell{Path: p.path, Gas: g})
		}
	}
	slices.SortFunc(cs, func(a, b Cell) int { return int(a.Path) - int(b.Path) })
	return cs
}

// chargeOf is the charge the simulator's curve gives a fill paying q on a ladder with these reads; a ladder with no
// second-fill charge is given one over its cells, which the curve takes down to their least.
func chargeOf(t *testing.T, l Ladder, q *uint256.Int, moved bool) int64 {
	t.Helper()
	return chargeAt(t, l, q, moved, false)
}

// chargeAt is chargeOf with the route's swap limit saying whether another pool of the source filled before (after).
func chargeAt(t *testing.T, l Ladder, q *uint256.Int, moved, after bool) int64 {
	t.Helper()
	l.Points = []Point{{uint256.NewInt(1), uint256.NewInt(1)}}
	if l.Second == 0 {
		l.Second = math.MaxInt64
	}
	c, err := newCurve(l)
	require.NoError(t, err)
	return c.charge(q, moved, after)
}

// Each edge of the path rules (output custody, first or owing borrow, repay), on a sale's cells (custody 100, freed 200: held, supplied and posted output) and a buy's.
func TestChargeTakesTheLeastOfThePathsItsReadsAllow(t *testing.T) {
	sale, buy := testCells(testHookFamily, true, 0), testCells(testHookFamily, false, 0)
	u := uint256.NewInt
	for name, tc := range map[string]struct {
		l    Ladder
		q    uint64
		want int64
	}{
		"no reads: the least":                      {l: Ladder{Cells: sale}, q: 1_000, want: 250_000},
		"within custody: any path":                 {l: Ladder{Cells: sale, Held: u(100)}, q: 100, want: 250_000},
		"past custody, a sale: a venue pays":       {l: Ladder{Cells: sale, Held: u(100)}, q: 101, want: 400_000},
		"past custody: a venue pays":               {l: Ladder{Cells: buy, Held: u(100)}, q: 101, want: 450_000},
		"within custody, a buy: custody":           {l: Ladder{Cells: buy, Held: u(100)}, q: 100, want: 300_000},
		"within what the pool frees: no borrow":    {l: Ladder{Cells: sale, Held: u(100), Freed: u(200)}, q: 200, want: 400_000},
		"within the slack: no borrow":              {l: Ladder{Cells: sale, Held: u(100), Freed: u(20_000)}, q: 20_002, want: 400_000},
		"past the slack: a borrow, either kind":    {l: Ladder{Cells: sale, Held: u(100), Freed: u(20_000)}, q: 20_003, want: 500_000},
		"past it, owing nothing: a first borrow":   {l: Ladder{Cells: sale, Freed: u(200), OwedOut: owedNone}, q: 201, want: 600_000},
		"past it, owing: an owing borrow":          {l: Ladder{Cells: sale, Freed: u(200), OwedOut: owedSome}, q: 201, want: 500_000},
		"owing, a second lender: either":           {l: Ladder{Cells: sale, Freed: u(200), OwedOut: owedSome, FirstOwing: true}, q: 201, want: 500_000},
		"owing nothing, collateral posted: either": {l: Ladder{Cells: sale, Freed: u(200), OwedOut: owedNone, PostedIn: true}, q: 201, want: 500_000},
		"owing, collateral posted: an owing one":   {l: Ladder{Cells: sale, Freed: u(200), OwedOut: owedSome, PostedIn: true}, q: 201, want: 500_000},
		"collateral posted, within what it frees":  {l: Ladder{Cells: sale, Held: u(100), Freed: u(200), OwedOut: owedNone, PostedIn: true}, q: 200, want: 400_000},
		"the input owes nothing: no repay":         {l: Ladder{Cells: buy, OwedIn: owedNone}, q: 1, want: 300_000},
		"the input owes, repay certain":            {l: Ladder{Cells: buy, OwedIn: owedSome, Repay: true}, q: 1, want: 350_000},
		"the input owes, repay not certain":        {l: Ladder{Cells: buy, OwedIn: owedSome}, q: 1, want: 300_000},
		"repay certain, past custody":              {l: Ladder{Cells: buy, Held: u(100), OwedIn: owedSome, Repay: true}, q: 101, want: 450_000},
		"no repay, past custody: no cell, least":   {l: Ladder{Cells: buy, Held: u(100), OwedIn: owedNone}, q: 101, want: 300_000},
		"a buy past what it frees: no cell, least": {l: Ladder{Cells: buy, Held: u(1), Freed: u(1)}, q: 2, want: 300_000},
	} {
		t.Run(name, func(t *testing.T) {
			q := uint256.NewInt(tc.q)
			require.Equal(t, tc.want, chargeOf(t, tc.l, q, false))
			require.Equal(t, slices.MinFunc(tc.l.Cells, func(a, b Cell) int { return int(a.Gas - b.Gas) }).Gas,
				chargeOf(t, tc.l, q, true), "after a fill, a second-fill charge over the cells: the least")
			l := tc.l
			l.Second = 1_000
			require.Equal(t, int64(1_000), chargeOf(t, l, q, true), "after a fill: the second-fill charge")
			require.Equal(t, tc.want, chargeOf(t, l, q, false), "a first fill: its cells")
		})
	}
}

// A ladder saved before the charge (no cells) or before its second-fill charge, or with a read out of range, is not
// quoted: the pool fails to load and ages out, never charged a figure of an older table.
func TestSimulatorRefusesALadderWithoutCells(t *testing.T) {
	for name, l := range map[string]Ladder{
		"no cells":              {In: 0, Out: 1, Second: 1},
		"a cell at zero":        {In: 0, Out: 1, Cells: []Cell{{Path: 0, Gas: 0}}, Second: 1},
		"an owed state":         {In: 0, Out: 1, Cells: []Cell{{Gas: 1}}, Second: 1, OwedIn: 3},
		"no second-fill charge": {In: 0, Out: 1, Cells: []Cell{{Gas: 1}}},
	} {
		t.Run(name, func(t *testing.T) {
			p := testEntity(time.Now().Unix())
			l.Points = pts(100, 990, 200, 1970)
			extra, err := json.Marshal(Extra{HaircutBps: 1, Ladders: []Ladder{l}})
			require.NoError(t, err)
			p.Extra = string(extra)
			_, err = NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.ErrorIs(t, err, ErrInvalidState)
		})
	}
	p := testEntity(time.Now().Unix())
	p.Extra = `{"h":1,"l":[{"i":0,"j":1,"p":[["100","990"],["200","1970"]],"g":600000}]}`
	_, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.ErrorIs(t, err, ErrInvalidState, "a state saved with the old figure")
}

// The simulator charges each fill at its payout from the direction's start, and every fill after another on the
// pool, in the same transaction, at its second-fill charge.
func TestSimulatorChargesEachFillByItsPayout(t *testing.T) {
	p := testEntity(time.Now().Unix())
	extra, err := json.Marshal(Extra{HaircutBps: 1, Ladders: []Ladder{
		{In: 0, Out: 1, Points: pts(100_000, 99_000_000, 1_000_000, 985_000_000), Cells: testCells(testHookFamily,
			true, 0), Second: testSecond(testHookFamily, true, 0), Held: uint256.NewInt(99_000_000),
			Freed: uint256.NewInt(500_000_000), OwedOut: owedNone},
		{In: 1, Out: 0, Points: pts(100_000_000, 99_000, 1_000_000_000, 985_000), Cells: testCells(testHookFamily,
			false, 0), Second: testSecond(testHookFamily, false, 0), Held: uint256.NewInt(1_000_000)},
	}})
	require.NoError(t, err)
	p.Extra = string(extra)
	s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
	require.NoError(t, err)
	res, err := calc(t, s, cbBTC, 100_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64(250_000), res.Gas, "within custody")
	res, err = calc(t, s, cbBTC, 1_000_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64(600_000), res.Gas, "past what the pool frees, owing nothing: a first borrow")
	res, err = calc(t, s, usdc, 100_000_000, cbBTC)
	require.NoError(t, err)
	require.Equal(t, int64(300_000), res.Gas, "a buy within custody")

	fill(t, s, usdc, 100_000_000, cbBTC)
	res, err = calc(t, s, cbBTC, 1_000_000, usdc)
	require.NoError(t, err)
	require.Equal(t, int64(100_000), res.Gas, "after a fill on the pool: the second fill")
	c := s.CloneState().(*PoolSimulator)
	require.True(t, c.moved())
}

// The kind gate and the venue reads of the charge, on testWorld's pool: a pair is routed only where every
// venue record read, some live venue lends or holds one of its tokens, and every such venue is of a kind the chain
// routes with cells and a second-fill charge for the hook's build and the side.
func TestTrackerRoutesOnlyKindsWithCells(t *testing.T) {
	nine := make([]venueView, 9) // past the first read's records: the tenth, a cbBTC/USDT venue
	for k := range nine {
		nine[k] = venue(0, 0, 1, 0)
	}
	for name, tc := range map[string]struct {
		venues []venueView
		bad    map[int]bool
		edit   func(cfg *Config)
		kinds  []uint8    // the live kinds whose cells the ladders carry; none: not routed
		dirs   [][2]uint8 // the directions routed; nil: both of the hook pair
	}{
		"Morpho Blue on cbBTC/USDC":                {venues: []venueView{venue(0, 0, 1, 0)}, kinds: []uint8{0}},
		"no venue":                                 {},
		"an Aave V3 venue retired beside it":       {venues: []venueView{venue(0, 0, 1, 0), venue(1, 0, 1, 1)}, kinds: []uint8{0}},
		"Aave V3 on cbBTC/USDT beside it":          {venues: []venueView{venue(0, 0, 1, 0), venue(1, 0, 2, 0)}, kinds: []uint8{0, 1}},
		"a fenced venue alone":                     {venues: []venueView{venue(1, 0, 1, 2)}},
		"a venue on USDT alone":                    {venues: []venueView{venue(1, 2, 2, 0)}},
		"a kind the chain does not route":          {venues: []venueView{venue(0, 0, 1, 0), venue(2, 0, 1, 0)}},
		"Lista Moolah on Base":                     {venues: []venueView{venue(3, 0, 1, 0)}},
		"a kind no table names":                    {venues: []venueView{venue(9, 0, 2, 0)}},
		"a record that does not read":              {venues: []venueView{venue(0, 0, 1, 0), venue(0, 2, 0, 0)}, bad: map[int]bool{1: true}},
		"a record past the first read":             {venues: append(slices.Clone(nine), venue(1, 0, 2, 0)), kinds: []uint8{0, 1}},
		"a record past the first read, not routed": {venues: append(slices.Clone(nine), venue(2, 0, 2, 0))},
		"the config narrows the kinds":             {venues: []venueView{venue(0, 0, 1, 0)}, edit: func(cfg *Config) { cfg.VenueKinds = []uint8{1} }},
		"the config cannot widen them":             {venues: []venueView{venue(2, 0, 1, 0)}, edit: func(cfg *Config) { cfg.VenueKinds = []uint8{2} }},
		"a chain the table has no cells of":        {venues: []venueView{venue(1, 0, 1, 0)}, edit: func(cfg *Config) { cfg.ChainID = valueobject.ChainIDEthereum }},
		"cells and no second-fill charge":          {venues: []venueView{venue(0, 0, 1, 0)}, edit: func(cfg *Config) { cfg.ChainID = valueobject.ChainIDEthereum }},
		"Venus Core on BSC":                        {venues: []venueView{venue(2, 0, 1, 0)}, edit: onBSC, kinds: []uint8{2}},
		"Aave V3 and Venus Core on BSC":            {venues: []venueView{venue(2, 0, 1, 0), venue(1, 0, 2, 0)}, edit: onBSC, kinds: []uint8{1, 2}},
		"Morpho Blue on BSC":                       {venues: []venueView{venue(0, 0, 1, 0)}, edit: onBSC},
		"Lista Moolah on BSC, its sale's legs unread": {venues: []venueView{venue(3, 0, 1, 0)}, edit: onBSC, kinds: []uint8{3},
			dirs: [][2]uint8{{1, 0}}},
		"no chain": {venues: []venueView{venue(0, 0, 1, 0)}, edit: func(cfg *Config) { cfg.ChainID = 0 }},
	} {
		t.Run(name, func(t *testing.T) {
			w, cfg := testWorld(), testConfig()
			fp := w.pools[common.HexToAddress(poolX)]
			fp.venues, fp.badVenue = tc.venues, tc.bad
			if tc.edit != nil {
				tc.edit(cfg)
			}
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			e := extraOf(t, p)
			if len(tc.kinds) == 0 {
				require.Empty(t, e.Ladders)
				return
			}
			require.ElementsMatch(t, lo.Ternary(tc.dirs == nil, [][2]uint8{{0, 1}, {1, 0}}, tc.dirs), directionsOf(e))
			for _, l := range e.Ladders {
				require.Equal(t, testCells(testHookFamily, l.In == 0, tc.kinds...), l.Cells)
				require.Equal(t, testSecond(testHookFamily, l.In == 0, tc.kinds...), l.Second)
			}
		})
	}
}

func onBSC(cfg *Config) { cfg.ChainID = valueobject.ChainIDBSC }

// BSC routes the kinds its table measured, Aave V3, Venus Core and Lista Moolah, from the cells generated for each
// build measured there: each kind a build lists has each side with a cell for every path a fill of it took, the hook's
// build lists all three, and a kind a build does not list has no cells, so its pools of that kind are not routed;
// Morpho Blue, which BSC does not have, has none.
func TestTheTableRoutesBSCsMeasuredKinds(t *testing.T) {
	require.Equal(t, []uint8{1, 2, 3}, chainKinds[valueobject.ChainIDBSC])
	builds := 0
	for _, f := range gasFamilies {
		var kinds []uint8
		for _, c := range gasCells {
			if c.family == f.CodeHash && c.chain == valueobject.ChainIDBSC && !slices.Contains(kinds, c.kind) {
				kinds = append(kinds, c.kind)
			}
		}
		if len(kinds) == 0 {
			continue
		}
		builds++
		require.Subset(t, chainKinds[valueobject.ChainIDBSC], kinds, f.Name)
		if f.CodeHash == bscBuild {
			require.ElementsMatch(t, chainKinds[valueobject.ChainIDBSC], kinds, "the hook's build lists every BSC kind")
		}
		for _, kind := range kinds {
			for _, sale := range []bool{true, false} {
				cs := cellsOf(f.CodeHash, valueobject.ChainIDBSC, kind, sale)
				require.NotEmpty(t, cs, "%s kind %d sale %v", f.Name, kind, sale)
				for _, c := range cs {
					require.Positive(t, c.Gas)
					require.Zero(t, c.Gas%1_000, "a base is rounded down to 1,000")
				}
			}
		}
		require.Empty(t, cellsOf(f.CodeHash, valueobject.ChainIDBSC, 0, true), "no Morpho Blue on BSC")
	}
	require.Equal(t, 2, builds, "two builds measured on BSC: the hook's and the idle-entry hook's")
}

// The builds the table measures: Base's and Ethereum's, and BSC's.
const measuredBuild, bscBuild = "0x4b085efb8bc456c765bffcdf17d95a5810765bc73b9917d0e5135efa55daeab8",
	"0x515c083b2bae27cadce7dbea69ca7245f327d5775db6521c5463d3e192d8b5eb"

// idleEntryBuild is the idle-entry hook's build (EverlongIdleEntryHook), measured on Base and BSC.
const idleEntryBuild = "0x79f4bdf891b3b00115ff3f89e17d5ad8fae87b65e3efc7fc3efcf7fb2b68a07c"

// Every key of the table, each build, chain, venue kind and side, has a custody cell, so a fill paid from custody is
// never charged a venue's path. Its input stays in custody too wherever the pool lends none of it (lending off, the
// input backing a debt, or its surplus over the reserve target under the supply batch), so a supply is never assumed.
func TestTheTableHasACustodyCellForEveryKey(t *testing.T) {
	type key struct {
		family string
		chain  valueobject.ChainID
		kind   uint8
		sale   bool
	}
	custody := map[key]bool{}
	for _, c := range gasCells {
		k := key{strings.ToLower(c.family), c.chain, c.kind, c.sale}
		custody[k] = custody[k] || c.path == 0
	}
	require.NotEmpty(t, custody)
	for k, ok := range custody {
		require.True(t, ok, "no custody cell for %+v", k)
	}
}

// measuredFill is the least net gas measured of a key's fills of one kind, on a local fork of its chain.
type measuredFill struct {
	name   string
	family string
	chain  valueobject.ChainID
	kind   uint8
	sale   bool
	net    int64
}

// A charge never over the net gas measured, on fills paid from custody, their input kept there too (its supply batch at its maximum where the pool would lend
// it), each the least measured of its key with its input untouched before the pool pulled it: within custody, with
// the legs unread, the charge is at or under it.
func TestTheTableChargesACustodyFillAtMostItsGas(t *testing.T) {
	for _, f := range []measuredFill{
		{"Base Morpho Blue sale", measuredBuild, valueobject.ChainIDBase, 0, true, 232_446},
		{"Base Morpho Blue buy", measuredBuild, valueobject.ChainIDBase, 0, false, 231_968},
		{"Base Aave V3 sale", measuredBuild, valueobject.ChainIDBase, 1, true, 232_495},
		{"Base Aave V3 buy", measuredBuild, valueobject.ChainIDBase, 1, false, 222_378},
		{"Ethereum Morpho Blue sale", measuredBuild, valueobject.ChainIDEthereum, 0, true, 215_826},
		{"Ethereum Morpho Blue buy", measuredBuild, valueobject.ChainIDEthereum, 0, false, 215_469},
		{"Ethereum Aave V3 sale", measuredBuild, valueobject.ChainIDEthereum, 1, true, 215_875},
		{"Ethereum Aave V3 buy", measuredBuild, valueobject.ChainIDEthereum, 1, false, 205_879},
		{"BSC Aave V3 sale", bscBuild, valueobject.ChainIDBSC, 1, true, 202_780},
		{"BSC Aave V3 buy", bscBuild, valueobject.ChainIDBSC, 1, false, 188_923},
		{"BSC Venus Core sale", bscBuild, valueobject.ChainIDBSC, 2, true, 320_032},
		{"BSC Venus Core buy", bscBuild, valueobject.ChainIDBSC, 2, false, 306_175},
		{"BSC Lista Moolah sale", bscBuild, valueobject.ChainIDBSC, 3, true, 217_314},
		{"BSC Lista Moolah buy", bscBuild, valueobject.ChainIDBSC, 3, false, 203_457},
		{"Base Morpho Blue sale, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, true, 268_200},
		{"Base Morpho Blue buy, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, false, 260_883},
		{"Base Aave V3 sale, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, true, 268_366},
		{"Base Aave V3 buy, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, false, 251_293},
		{"BSC Lista Moolah sale, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, true, 253_185},
		{"BSC Lista Moolah buy, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, false, 232_340},
	} {
		cells := cellsOf(f.family, f.chain, f.kind, f.sale)
		require.NotEmpty(t, cells, f.name)
		q := uint256.NewInt(1_000)
		g := chargeOf(t, Ladder{Cells: cells, Held: q}, q, false)
		require.LessOrEqual(t, g, f.net, "%s: charged %d", f.name, g)
	}
}

// A charge never over the net gas measured, on fills in a route's form: the input received earlier in the same transaction, the taker holding dust of it, and
// the output's token account warm and the recipient's balance written before the swap, as where an earlier leg paid
// the same token. Each is the least measured of its key and class at its state's reads: paid from custody, a first
// borrow, an owing borrow, or a repay with the output paid from custody.
func TestTheTableChargesARouteFillAtMostItsGas(t *testing.T) {
	q, zero := uint256.NewInt(1_000), uint256.NewInt(0)
	reads := map[string]Ladder{
		"custody":      {Held: q},
		"first borrow": {Held: zero, Freed: zero, OwedOut: owedNone, OwedIn: owedNone},
		"owing borrow": {Held: zero, Freed: zero, OwedOut: owedSome, OwedIn: owedNone},
		"repay":        {Held: q, OwedIn: owedSome, Repay: true},
	}
	for _, f := range []struct {
		measuredFill
		class string
	}{
		{measuredFill{"Base Morpho Blue sale, custody", measuredBuild, valueobject.ChainIDBase, 0, true, 209_046}, "custody"},
		{measuredFill{"Base Morpho Blue sale, first borrow", measuredBuild, valueobject.ChainIDBase, 0, true, 485_036}, "first borrow"},
		{measuredFill{"Base Morpho Blue sale, owing borrow", measuredBuild, valueobject.ChainIDBase, 0, true, 460_069}, "owing borrow"},
		{measuredFill{"Base Morpho Blue buy, custody", measuredBuild, valueobject.ChainIDBase, 0, false, 206_568}, "custody"},
		{measuredFill{"Base Morpho Blue buy, repay", measuredBuild, valueobject.ChainIDBase, 0, false, 330_085}, "repay"},
		{measuredFill{"Base Aave V3 sale, custody", measuredBuild, valueobject.ChainIDBase, 1, true, 209_095}, "custody"},
		{measuredFill{"Base Aave V3 sale, first borrow", measuredBuild, valueobject.ChainIDBase, 1, true, 989_226}, "first borrow"},
		{measuredFill{"Base Aave V3 sale, owing borrow", measuredBuild, valueobject.ChainIDBase, 1, true, 906_753}, "owing borrow"},
		{measuredFill{"Base Aave V3 buy, custody", measuredBuild, valueobject.ChainIDBase, 1, false, 196_978}, "custody"},
		{measuredFill{"Base Aave V3 buy, repay", measuredBuild, valueobject.ChainIDBase, 1, false, 525_395}, "repay"},
		{measuredFill{"Ethereum Morpho Blue sale, custody", measuredBuild, valueobject.ChainIDEthereum, 0, true, 192_426}, "custody"},
		{measuredFill{"Ethereum Morpho Blue sale, first borrow", measuredBuild, valueobject.ChainIDEthereum, 0, true, 469_725}, "first borrow"},
		{measuredFill{"Ethereum Morpho Blue sale, owing borrow", measuredBuild, valueobject.ChainIDEthereum, 0, true, 444_726}, "owing borrow"},
		{measuredFill{"Ethereum Morpho Blue buy, custody", measuredBuild, valueobject.ChainIDEthereum, 0, false, 190_069}, "custody"},
		{measuredFill{"Ethereum Morpho Blue buy, repay", measuredBuild, valueobject.ChainIDEthereum, 0, false, 314_337}, "repay"},
		{measuredFill{"Ethereum Aave V3 sale, custody", measuredBuild, valueobject.ChainIDEthereum, 1, true, 192_475}, "custody"},
		{measuredFill{"Ethereum Aave V3 sale, first borrow", measuredBuild, valueobject.ChainIDEthereum, 1, true, 941_669}, "first borrow"},
		{measuredFill{"Ethereum Aave V3 sale, owing borrow", measuredBuild, valueobject.ChainIDEthereum, 1, true, 853_952}, "owing borrow"},
		{measuredFill{"Ethereum Aave V3 buy, custody", measuredBuild, valueobject.ChainIDEthereum, 1, false, 180_479}, "custody"},
		{measuredFill{"Ethereum Aave V3 buy, repay", measuredBuild, valueobject.ChainIDEthereum, 1, false, 497_832}, "repay"},
		{measuredFill{"BSC Aave V3 sale, custody", bscBuild, valueobject.ChainIDBSC, 1, true, 187_880}, "custody"},
		{measuredFill{"BSC Aave V3 sale, first borrow", bscBuild, valueobject.ChainIDBSC, 1, true, 885_595}, "first borrow"},
		{measuredFill{"BSC Aave V3 sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 1, true, 793_006}, "owing borrow"},
		{measuredFill{"BSC Aave V3 buy, custody", bscBuild, valueobject.ChainIDBSC, 1, false, 174_023}, "custody"},
		{measuredFill{"BSC Aave V3 buy, repay", bscBuild, valueobject.ChainIDBSC, 1, false, 484_165}, "repay"},
		{measuredFill{"BSC Venus Core sale, custody", bscBuild, valueobject.ChainIDBSC, 2, true, 305_132}, "custody"},
		{measuredFill{"BSC Venus Core sale, first borrow", bscBuild, valueobject.ChainIDBSC, 2, true, 1_909_154}, "first borrow"},
		{measuredFill{"BSC Venus Core sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 2, true, 1_788_500}, "owing borrow"},
		{measuredFill{"BSC Venus Core buy, custody", bscBuild, valueobject.ChainIDBSC, 2, false, 291_275}, "custody"},
		{measuredFill{"BSC Venus Core buy, repay", bscBuild, valueobject.ChainIDBSC, 2, false, 807_987}, "repay"},
		{measuredFill{"BSC Lista Moolah sale, custody", bscBuild, valueobject.ChainIDBSC, 3, true, 202_414}, "custody"},
		{measuredFill{"BSC Lista Moolah sale, first borrow", bscBuild, valueobject.ChainIDBSC, 3, true, 823_495}, "first borrow"},
		{measuredFill{"BSC Lista Moolah sale, owing borrow", bscBuild, valueobject.ChainIDBSC, 3, true, 731_970}, "owing borrow"},
		{measuredFill{"BSC Lista Moolah buy, custody", bscBuild, valueobject.ChainIDBSC, 3, false, 188_557}, "custody"},
		{measuredFill{"BSC Lista Moolah buy, repay", bscBuild, valueobject.ChainIDBSC, 3, false, 443_142}, "repay"},
		{measuredFill{"Base Morpho Blue sale, custody, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, true, 244_800}, "custody"},
		{measuredFill{"Base Morpho Blue sale, first borrow, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, true, 513_721}, "first borrow"},
		{measuredFill{"Base Morpho Blue sale, owing borrow, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, true, 489_930}, "owing borrow"},
		{measuredFill{"Base Morpho Blue buy, custody, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, false, 235_483}, "custody"},
		{measuredFill{"Base Morpho Blue buy, repay, idle entry", idleEntryBuild, valueobject.ChainIDBase, 0, false, 368_677}, "repay"},
		{measuredFill{"Base Aave V3 sale, custody, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, true, 244_966}, "custody"},
		{measuredFill{"Base Aave V3 sale, first borrow, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, true, 1_062_255}, "first borrow"},
		{measuredFill{"Base Aave V3 sale, owing borrow, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, true, 977_011}, "owing borrow"},
		{measuredFill{"Base Aave V3 buy, custody, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, false, 225_893}, "custody"},
		{measuredFill{"Base Aave V3 buy, repay, idle entry", idleEntryBuild, valueobject.ChainIDBase, 1, false, 563_287}, "repay"},
		{measuredFill{"BSC Lista Moolah sale, custody, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, true, 238_285}, "custody"},
		{measuredFill{"BSC Lista Moolah sale, first borrow, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, true, 815_950}, "first borrow"},
		{measuredFill{"BSC Lista Moolah sale, owing borrow, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, true, 760_766}, "owing borrow"},
		{measuredFill{"BSC Lista Moolah buy, custody, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, false, 217_440}, "custody"},
		{measuredFill{"BSC Lista Moolah buy, repay, idle entry", idleEntryBuild, valueobject.ChainIDBSC, 3, false, 481_008}, "repay"},
	} {
		l, ok := reads[f.class]
		require.True(t, ok, f.class)
		l.Cells = cellsOf(f.family, f.chain, f.kind, f.sale)
		require.NotEmpty(t, l.Cells, f.name)
		g := chargeOf(t, l, q, false)
		require.LessOrEqual(t, g, f.net, "%s: charged %d", f.name, g)
	}
}

// Every key of the table, each build, chain, venue kind and side, has a second-fill charge, under its least cell: a
// kind without one is not routed.
func TestTheTableHasASecondFillForEveryKey(t *testing.T) {
	keys := 0
	for _, c := range gasCells {
		if !slices.ContainsFunc(gasFamilies, func(f HookFamily) bool { return strings.EqualFold(f.CodeHash, c.family) }) {
			continue // a test build's
		}
		g, ok := secondOf(c.family, c.chain, c.kind, c.sale)
		require.True(t, ok, "no second fill for %+v", c)
		require.Positive(t, g)
		require.Zero(t, g%1_000, "a base is rounded down to 1,000")
		require.Less(t, g, c.gas, "%+v", c)
		keys++
	}
	require.Positive(t, keys)
}

// A fill after another on the pool, by the same route in the same transaction, is charged at most the least net gas
// measured of a second swap of the pool in one transaction, of its kind and side: the BSC replay fixture's pools,
// each quoted at its first point in either direction after a first fill at the first point of either.
func TestASecondFillIsChargedAtMostItsMeasuredGas(t *testing.T) {
	fx := loadFixture(t, "testdata/bsc-fork.json.gz")
	least := map[string][2]int64{ // the sale's, the buy's
		"EverlongHook on Aave V3":      {75_380, 66_023},
		"EverlongHook on Venus Core":   {78_531, 69_174},
		"EverlongHook on Lista Moolah": {78_914, 69_557},
	}
	require.Len(t, fx.Pools, len(least))
	at := func(p entity.Pool, l Ladder) pool.CalcAmountOutParams {
		return pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: p.Tokens[l.In].Address,
			Amount: l.Points[0][0].ToBig()}, TokenOut: p.Tokens[l.Out].Address}
	}
	checked := 0
	for _, p := range fx.Pools {
		nets, ok := least[fx.Kinds[p.Address]]
		require.True(t, ok, fx.Kinds[p.Address])
		ladders := extraOf(t, p).Ladders
		require.Len(t, ladders, 2, fx.Kinds[p.Address])
		for _, first := range ladders {
			for _, next := range ladders {
				s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
				require.NoError(t, err)
				res, err := s.CalcAmountOut(at(p, first))
				require.NoError(t, err)
				s.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: at(p, first).TokenAmountIn,
					TokenAmountOut: *res.TokenAmountOut, Fee: *res.Fee, SwapInfo: res.SwapInfo})
				res, err = s.CalcAmountOut(at(p, next))
				if first.In == next.In {
					require.ErrorIs(t, err, ladder.ErrNoQuote, "a direction takes one fill per route")
					continue
				}
				require.NoError(t, err)
				want := nets[lo.Ternary(next.In == 0, 0, 1)]
				require.LessOrEqual(t, res.Gas, want, "%s, %d then %d", fx.Kinds[p.Address], first.In, next.In)
				checked++
			}
		}
	}
	require.Positive(t, checked)
}

// The venue reads of the borrow and repay rules, with the request's legs (tierWorld): a second live lender of the output allows a first
// borrow while it owes; collateral posted of the input allows an owing borrow while the output owes nothing; a repay
// is certain only where every venue lending the input is live and takes repays now.
func TestTrackerReadsWhatTheChargeNeedsOfTheVenues(t *testing.T) {
	owing := func(fp *fakePool) { fp.legs[1][3] = big.NewInt(5_000_000) } // USDC owes: a buy repays
	for name, tc := range map[string]struct {
		venues     []venueView
		acts       map[int]uint8
		badActs    map[int]bool
		owe        bool
		posted     bool // cbBTC posted: the sale's input
		firstOwing bool // the sale's
		repay      bool // the buy's
	}{
		"one lender, the input owing nothing": {venues: []venueView{venue(0, 0, 1, 0)}},
		"one lender, the input owing":         {venues: []venueView{venue(0, 0, 1, 0)}, owe: true, repay: true},
		"a repay refused now":                 {venues: []venueView{venue(0, 0, 1, 0)}, acts: map[int]uint8{0: 15 &^ canRepay}, owe: true},
		"its actions unread":                  {venues: []venueView{venue(0, 0, 1, 0)}, badActs: map[int]bool{0: true}, owe: true},
		"a fenced lender of the input":        {venues: []venueView{venue(0, 0, 1, 0), venue(0, 0, 1, 2)}, owe: true},
		"two lenders of the output":           {venues: []venueView{venue(0, 0, 1, 0), venue(1, 0, 1, 0)}, owe: true, firstOwing: true, repay: true},
		"collateral posted ahead":             {venues: []venueView{venue(0, 0, 1, 0)}, posted: true},
		"collateral posted, the output owing": {venues: []venueView{venue(0, 0, 1, 0)}, posted: true, owe: true, repay: true},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp := tierWorld()
			cfg := testConfig()
			fp.venues, fp.acts, fp.badActs = tc.venues, tc.acts, tc.badActs
			if tc.owe {
				owing(fp)
			}
			if tc.posted {
				fp.legs[0][2] = big.NewInt(100_000_000) // 1 cbBTC
			}
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			e := extraOf(t, p)
			require.ElementsMatch(t, [][2]uint8{{0, 1}, {1, 0}}, directionsOf(e))
			for _, l := range e.Ladders {
				require.NotNil(t, l.Freed, "the legs read")
				if l.In == 0 {
					require.Equal(t, tc.firstOwing, l.FirstOwing)
					require.Equal(t, tc.posted, l.PostedIn)
					require.Equal(t, owedNone, l.OwedIn)
					require.Equal(t, owedState(tc.owe), l.OwedOut)
				} else {
					require.False(t, l.PostedIn, "no USDC posted")
					require.Equal(t, tc.repay, l.Repay)
					require.Equal(t, owedState(tc.owe), l.OwedIn)
				}
			}
		})
	}
}

// The borrow rule against collateral posted ahead, from the request's legs to the fill's charge (tierWorld): a sale whose payout
// passes what the pool frees, with the output owing nothing, borrows; where its input has collateral posted, the
// pool posts only what the borrow needs past it, often nothing, unlike every first borrow measured, and the borrow
// costs no less than an owing one, so it is charged an owing borrow; with nothing posted, a first borrow.
func TestASaleAgainstCollateralPostedAheadIsChargedAnOwingBorrow(t *testing.T) {
	for name, tc := range map[string]struct {
		posted *big.Int // cbBTC posted
		want   int64
	}{
		"nothing posted: a first borrow":           {new(big.Int), testGas(testHookFamily, 0, 600_000)},
		"collateral posted ahead: an owing borrow": {big.NewInt(100_000_000), testGas(testHookFamily, 0, 500_000)},
	} {
		t.Run(name, func(t *testing.T) {
			w, fp := tierWorld()
			fp.legs[0][2] = tc.posted
			cfg := testConfig()
			n, pools := listed(t, w, cfg)
			p, err := track(t, n, cfg, pools[0])
			require.NoError(t, err)
			s, err := NewPoolSimulator(pool.FactoryParams{EntityPool: p})
			require.NoError(t, err)
			// 40 cbBTC pays about 3,958,000 USDC, past the 3,000,000 the pool holds of it and frees: a borrow
			res, err := calc(t, s, cbBTC, 4_000_000_000, usdc)
			require.NoError(t, err)
			require.Equal(t, tc.want, res.Gas)
		})
	}
}

func owedState(owes bool) uint8 {
	if owes {
		return owedSome
	}
	return owedNone
}

// A charge never over the net gas measured, over the measured rows: every filled row's charge, from this package's own code at what the client read at
// the row's state, is at or under the row's net gas (its swap call's gas less the refund the call earned); a second
// fill of the pool in one transaction (moved) is charged its key's second-fill charge, and a pool's first fill
// measured after another pool's fill in its transaction (after) is charged as with the route's swap limit saying
// so, its key's cells after another pool. The rows file is the generator's (a TSV: src chain family kind sale q held
// freed owedOut owedIn postedIn floor moved after net); the test is skipped without it. The worst under-charge of
// each chain, kind and side is logged, and how many rows are charged more than 100,000 under their net, in all
// and of the rows after another pool.
func TestChargeNeverOverAMeasuredFill(t *testing.T) {
	path := os.Getenv("FLAMM_GAS_ROWS")
	if path == "" {
		t.Skip("FLAMM_GAS_ROWS names no rows file")
	}
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	require.True(t, sc.Scan())
	require.Equal(t,
		"src\tchain\tfamily\tkind\tsale\tq\theld\tfreed\towedOut\towedIn\tpostedIn\tfloor\tmoved\tafter\tnet", sc.Text())
	num := func(s string) *uint256.Int {
		if s == "" {
			return nil
		}
		v, err := uint256.FromDecimal(s)
		require.NoError(t, err, s)
		return v
	}
	owed := func(s string) uint8 {
		switch s {
		case "":
			return owedUnknown
		case "0":
			return owedNone
		}
		return owedSome
	}
	type key struct {
		chain, kind int
		sale        bool
	}
	worst := map[key]int64{}
	at := map[key]string{}
	var rows, over, far, afterRows, afterFar int
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		require.Len(t, c, 15)
		chain, _ := strconv.Atoi(c[1])
		kind, _ := strconv.Atoi(c[3])
		net, _ := strconv.ParseInt(c[14], 10, 64)
		sale, moved, after := c[4] == "1", c[12] == "1", c[13] == "1"
		cells := cellsOf(c[2], valueobject.ChainID(chain), uint8(kind), sale)
		require.NotEmpty(t, cells, c[0])
		second, ok := secondOf(c[2], valueobject.ChainID(chain), uint8(kind), sale)
		require.True(t, ok, c[0])
		l := Ladder{Cells: cells, Second: second, Held: num(c[6]), Freed: num(c[7]), OwedOut: owed(c[8]),
			OwedIn: owed(c[9]), PostedIn: c[10] != "" && c[10] != "0"}
		if l.After = afterOf(c[2], valueobject.ChainID(chain), uint8(kind), sale); slices.Equal(l.After, cells) {
			l.After = nil // as the tracker leaves a ladder with no cell after another pool
		}
		// one venue a pool: no first borrow while owing; every measured venue takes repays, and a held-repay
		// venue only past its floor
		l.Repay = l.OwedIn == owedSome && (c[11] == "" || heldRepayOk(num(c[9]), num(c[11])))
		g := chargeAt(t, l, num(c[5]), moved, after)
		if g > net {
			over++
			t.Errorf("%s: charged %d over the net %d", c[0], g, net)
		}
		k := key{chain, kind, sale}
		if net-g > worst[k] {
			worst[k], at[k] = net-g, c[0]
		}
		if net-g > 100_000 {
			far++
		}
		if after {
			afterRows++
			if net-g > 100_000 {
				afterFar++
			}
		}
		rows++
	}
	require.NoError(t, sc.Err())
	require.Positive(t, rows)
	require.Zero(t, over)
	keys := make([]key, 0, len(worst))
	for k := range worst {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b key) int { return strings.Compare(fmt.Sprint(a), fmt.Sprint(b)) })
	for _, k := range keys {
		t.Logf("chain %d kind %d sale %v: worst under-charge %d at %s", k.chain, k.kind, k.sale, worst[k], at[k])
	}
	t.Logf("%d rows, %d over; %d charged more than 100,000 under their net", rows, over, far)
	t.Logf("%d of the rows after another pool; %d of them charged more than 100,000 under their net", afterRows,
		afterFar)
}
