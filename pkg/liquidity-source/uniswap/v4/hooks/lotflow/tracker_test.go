package lotflow

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fixtureLoader serves a P2-FIX fixture's pre-state through the Loader interface, the way an RPC
// loader would serve chain state. It exists to prove the interface carries everything Quote needs.
type fixtureLoader struct{ c *Case }

func (f *fixtureLoader) constituent(stock string) (*Constituent, int) {
	for i := range f.c.Pre.Constituents {
		if strings.EqualFold(f.c.Pre.Constituents[i].Stock, stock) {
			return &f.c.Pre.Constituents[i], i
		}
	}
	return nil, -1
}

func (f *fixtureLoader) Constituents(_ context.Context, lot string) ([]string, []*big.Int, error) {
	var s []string
	for _, k := range f.c.Pre.Constituents {
		s = append(s, k.Stock)
	}
	return s, unitsOf(f.c), nil
}

func (f *fixtureLoader) HookParams(context.Context) (Params, error) { return venueOf(f.c).Params, nil }

func (f *fixtureLoader) Venue(_ context.Context, lot string) (*VenueSnapshot, error) {
	v := f.c.Pre.Venue
	return &VenueSnapshot{PoolID: v.PoolID, UsdgIs0: v.UsdgIsCurrency0, UsdgDecimals: f.c.Pre.Hook.UsdgDecimals,
		SqrtPriceX96: v.SqrtPriceX96.Copy(), Tick: v.Tick}, nil
}

func (f *fixtureLoader) NavGuardParams(context.Context) (NavGuardParams, error) {
	_, p, _ := navInputsOf(f.c)
	return p, nil
}

func (f *fixtureLoader) Feed(_ context.Context, stock string) (string, *FeedState, error) {
	k, i := f.constituent(stock)
	cs, _, _ := navInputsOf(f.c)
	return k.Nav.Feed, cs[i].Feed, nil
}

func (f *fixtureLoader) Stock(_ context.Context, stock string) (*StockSnapshot, error) {
	k, _ := f.constituent(stock)
	ok, venues := bookOf(k)
	return &StockSnapshot{Registry: registryOf(k), StockDecimals: k.Registry.StockDecimals,
		StableDecimals: k.Registry.StableDecimals, OraclePaused: k.Nav.OraclePaused == "true",
		VenuesOK: ok, Venues: venues}, nil
}

func (f *fixtureLoader) Pool(_ context.Context, stock string, src Source) (*LegPool, error) {
	k, _ := f.constituent(stock)
	for j := range k.Candidates {
		if strings.EqualFold(k.Candidates[j].SourceID, src.ID) {
			return legPoolOf(&k.Candidates[j]), nil
		}
	}
	return nil, fmt.Errorf("no candidate %s", src.ID)
}

func (f *fixtureLoader) PoolManagerLotBalance(context.Context, string) (*big.Int, error) {
	return f.c.Pre.PM.LotBalance.Copy(), nil
}

func (f *fixtureLoader) BlockTimestamp(context.Context) (uint64, error) {
	return f.c.Pin.BlockTimestamp.Uint64(), nil
}

func trackerOf(c *Case) *Tracker {
	return &Tracker{Cfg: fixtureConfig(c, FinalisedRules), Hook: c.Pre.Hook.Address,
		PoolManager: "0x00000000000000000000000000000000000000aa", NavGuard: c.Pre.Nav.NavGuard,
		Registry: c.Pre.Hook.Registry, VenueBook: c.Pre.Hook.VenueBook,
		StockBeacon: "0x00000000000000000000000000000000000000bb"}
}

// The Loader interface carries everything a quote needs: a state loaded through it from each
// fixture's pre-state quotes the fixture's outcome exactly (85/85; an Inexact case by one of its two
// enumerated outcomes, as in TestWholeSwap), and loading recomputes NAV from
// the registry price pools and feeds rather than reading it.
func TestTrackerLoadQuotesFixtures(t *testing.T) {
	_, cases := loadAll(t)
	for _, c := range cases {
		st, err := trackerOf(c).Load(context.Background(), &fixtureLoader{c}, c.Input.Lot)
		require.NoError(t, err, c.ID)
		res, err := Quote(st.Cfg, st.Market, inputOf(c))
		if mm := checkOutcome(c, res, err); mm != "" {
			require.True(t, res != nil && res.Inexact, "%s: %s", c.ID, mm)
			skip := *st.Cfg
			skip.NearStipend = NearStipendSkip
			alt, altErr := Quote(&skip, st.Market, inputOf(c))
			require.Empty(t, checkOutcome(c, alt, altErr), c.ID)
		}
	}
}

// KYBER-PACK.md §5 "Events that invalidate state" as an event -> invalidated-entity table.
func TestTrackerInvalidation(t *testing.T) {
	c, err := LoadCase(fixtureDir + "/navjit2-buy-1kusd.json")
	require.NoError(t, err)
	tr := trackerOf(c)
	st, err := tr.Load(context.Background(), &fixtureLoader{c}, c.Input.Lot)
	require.NoError(t, err)

	k0, k1 := &c.Pre.Constituents[0], &c.Pre.Constituents[1]
	s0, s1 := strings.ToLower(k0.Stock), strings.ToLower(k1.Stock)
	stranger := "0x00000000000000000000000000000000000000cc"
	pricePool := strings.ToLower(k0.Registry.Pools[k0.Registry.PriceIndex].PoolID)
	var v3cand, foreignVenue string
	for _, cand := range k0.Candidates {
		if cand.Kind == KindV3 {
			v3cand = cand.Pool
			break
		}
	}
	for _, v := range k0.Book.Venues {
		if v.Admission == string(AdmFactoryNotV3) {
			foreignVenue = v.Pool
			break
		}
	}
	require.NotEmpty(t, v3cand)
	require.NotEmpty(t, foreignVenue)
	venueID := strings.ToLower(c.Pre.Venue.PoolID)
	feed0 := k0.Nav.Feed
	allStocks := []Entity{{EntStock, s0}, {EntStock, s1}, {EntNav, ""}}
	allBooks := []Entity{{EntBook, s0}, {EntBook, s1}}

	cases := []struct {
		name string
		ev   Event
		want []Entity
	}{
		// VenueBook
		{"book VenueAdded own stock", NewEvent(tr.VenueBook, "VenueAdded", s0, stranger), []Entity{{EntBook, s0}}},
		{"book VenueRemoved other stock", NewEvent(tr.VenueBook, "VenueRemoved", stranger, stranger), nil},
		{"book FactorySet", NewEvent(tr.VenueBook, "FactorySet", stranger), allBooks},
		{"book ParamsSet", NewEvent(tr.VenueBook, "BookParamsSet"), allBooks},
		{"book KeeperSet", NewEvent(tr.VenueBook, "KeeperSet", stranger), nil},
		// StockRegistry
		{"registry StockVenuesSet own", NewEvent(tr.Registry, "StockVenuesSet", s1, stranger),
			[]Entity{{EntStock, s1}, {EntNav, ""}}},
		{"registry StockVenuesSet other", NewEvent(tr.Registry, "StockVenuesSet", stranger, stranger), nil},
		{"registry StockRemoved own", NewEvent(tr.Registry, "StockRemoved", s0), []Entity{{EntStock, s0}, {EntNav, ""}}},
		{"registry StableSet", NewEvent(tr.Registry, "StableSet", stranger), allStocks},
		{"registry StablePoolSet", NewEvent(tr.Registry, "StablePoolSet", stranger, stranger), allStocks},
		// NavGuard + feeds
		{"guard FeedSet own", NewEvent(tr.NavGuard, "FeedSet", s0), []Entity{{EntFeed, s0}, {EntNav, ""}}},
		{"guard FeedSet other", NewEvent(tr.NavGuard, "FeedSet", stranger), nil},
		{"guard ParamsSet", NewEvent(tr.NavGuard, "NavGuardParamsSet"), []Entity{{EntNav, ""}, {EntNavParams, ""}}},
		{"feed new round", NewEvent(feed0, "AnswerUpdated", "0x01", "0x02"), []Entity{{EntFeed, s0}, {EntNav, ""}}},
		// hook
		{"hook ParamsSet", NewEvent(tr.Hook, "HookParamsSet"), []Entity{{EntHookParams, ""}}},
		{"hook NavGuardSet", NewEvent(tr.Hook, "NavGuardSet"), []Entity{{EntFull, ""}}},
		{"hook TreasurySet", NewEvent(tr.Hook, "TreasurySet"), nil},
		{"hook VenueOpened own", NewEvent(tr.Hook, "VenueOpened", c.Input.Lot, venueID), []Entity{{EntVenue, ""}}},
		{"hook VenueOpened other", NewEvent(tr.Hook, "VenueOpened", stranger, stranger), nil},
		{"hook StandingSeeded own", NewEvent(tr.Hook, "StandingSeeded", venueID), []Entity{{EntStanding, ""}, {EntVenue, ""}}},
		{"hook StandingWithdrawn own", NewEvent(tr.Hook, "StandingWithdrawn", venueID), []Entity{{EntStanding, ""}, {EntVenue, ""}}},
		{"hook StandingSeeded other", NewEvent(tr.Hook, "StandingSeeded", stranger), nil},
		// PoolManager
		{"pm Swap price pool", NewEvent(tr.PoolManager, "Swap", pricePool, stranger),
			[]Entity{{EntPool, pricePool}, {EntAdmission, s0}, {EntNav, ""}}},
		{"pm ModifyLiquidity price pool", NewEvent(tr.PoolManager, "ModifyLiquidity", pricePool, stranger),
			[]Entity{{EntPool, pricePool}, {EntAdmission, s0}, {EntNav, ""}}},
		{"pm ProtocolFeeUpdated", NewEvent(tr.PoolManager, "ProtocolFeeUpdated", pricePool), []Entity{{EntPool, pricePool}}},
		{"pm Swap venue", NewEvent(tr.PoolManager, "Swap", venueID, stranger), []Entity{{EntVenue, ""}}},
		{"pm Swap unrelated", NewEvent(tr.PoolManager, "Swap", stranger, stranger), nil},
		// v3 venues
		{"v3 Swap candidate", NewEvent(v3cand, "V3Swap", stranger, stranger),
			[]Entity{{EntPool, v3SourceID(v3cand)}, {EntAdmission, s0}}},
		{"v3 Mint book venue not admitted", NewEvent(foreignVenue, "V3Mint", stranger), []Entity{{EntAdmission, s0}}},
		{"v3 Burn unknown pool", NewEvent(stranger, "V3Burn", stranger), nil},
		// stock tokens, beacon, LOT float
		{"stock Paused", NewEvent(k1.Stock, "Paused"), []Entity{{EntStock, s1}, {EntNav, ""}}},
		{"stock Unpaused", NewEvent(k1.Stock, "Unpaused"), []Entity{{EntStock, s1}, {EntNav, ""}}},
		{"beacon Upgraded", NewEvent(tr.StockBeacon, "Upgraded", stranger), allStocks},
		{"LOT into PM", NewEvent(c.Input.Lot, "Transfer", stranger, tr.PoolManager), []Entity{{EntPMBalance, ""}}},
		{"LOT out of PM", NewEvent(c.Input.Lot, "Transfer", tr.PoolManager, stranger), []Entity{{EntPMBalance, ""}}},
		{"LOT elsewhere", NewEvent(c.Input.Lot, "Transfer", stranger, stranger), nil},
	}
	for _, tc := range cases {
		require.Equal(t, fmtEntities(tc.want), fmtEntities(st.Invalidate(tc.ev)), tc.name)
	}
	// an event whose topic0 matches nothing on a known emitter invalidates nothing
	require.Empty(t, st.Invalidate(Event{Emitter: tr.Hook, Topic0: "0x" + strings.Repeat("ab", 32)}))
}

// Every emitter the invalidation table listens to is in the subscription list; what no event
// reports (the Fables price-pool fee, block.timestamp for feed staleness) is in the polled list.
func TestTrackerDependencies(t *testing.T) {
	c, err := LoadCase(fixtureDir + "/metal-buy-1kusd.json") // metal: GLD's hooked dynamic-fee price pool
	require.NoError(t, err)
	tr := trackerOf(c)
	st, err := tr.Load(context.Background(), &fixtureLoader{c}, c.Input.Lot)
	require.NoError(t, err)
	d := st.Dependencies()
	has := func(xs []string, x string) bool {
		for _, y := range xs {
			if strings.EqualFold(x, y) {
				return true
			}
		}
		return false
	}
	for _, a := range []string{tr.Hook, tr.PoolManager, tr.NavGuard, tr.Registry, tr.VenueBook, tr.StockBeacon, c.Input.Lot} {
		require.True(t, has(d.Emitters, a), a)
	}
	for _, k := range c.Pre.Constituents {
		require.True(t, has(d.Emitters, k.Stock), k.Symbol)
		require.True(t, has(d.Emitters, k.Nav.Feed), k.Symbol)
		for _, p := range k.Registry.Pools {
			require.True(t, has(d.PoolIDs, p.PoolID), k.Symbol)
		}
		for _, v := range k.Book.Venues {
			require.True(t, has(d.Emitters, v.Pool), k.Symbol)
		}
	}
	require.True(t, has(d.PoolIDs, c.Pre.Venue.PoolID))
	var dyn, ts, feeds int
	for _, p := range d.Polled {
		switch p.Kind {
		case PollDynamicFee:
			dyn++
		case PollBlockTimestamp:
			ts++
		case PollFeedRound:
			feeds++
		}
	}
	require.Equal(t, 0, dyn) // D-B: no hooked candidate is admitted any more
	require.Equal(t, 1, ts)
	require.Equal(t, len(c.Pre.Constituents), feeds)
}

func fmtEntities(es []Entity) string {
	var s []string
	for _, e := range es {
		s = append(s, fmt.Sprintf("%s:%s", e.Kind, e.Key))
	}
	sort.Strings(s)
	return strings.Join(s, ",")
}

// topic0 of well-known events against their published constants (positive control on the
// signature table and the hashing).
func TestTrackerKnownTopics(t *testing.T) {
	require.Equal(t, "0x40e9cecb9f5f1f1c5b9c97dec2917b7ee92e57ba5563708daca94dd84ad7112f", Topic0("Swap"))
	require.Equal(t, "0xc42079f94a6350d7e6235f29174924f928cc2ac818eb64fed8004e115fbcca67", Topic0("V3Swap"))
	require.Equal(t, "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", Topic0("Transfer"))
	require.Equal(t, "0x0559884fd3a460db3073b7fc896cc77986f16e378210ded43186175bf646fc5f", Topic0("AnswerUpdated"))
	require.Equal(t, "0xf208f4912782fd25c7f114ca3723a2d5dd6f3bcc3ac8db5af63baa85f711d5ec", Topic0("ModifyLiquidity"))
}
