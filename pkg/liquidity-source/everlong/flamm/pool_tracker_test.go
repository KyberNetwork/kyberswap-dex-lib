package everlongflamm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The lister and tracker replayed from Base's recorded answers (testdata/tracker_rpc_51302915.json.gz, recorded with
// EVERLONG_FLAMM_RECORD=1 BASE_RPC_URL=https://mainnet.base.org): the listing at the recording head, the refresh
// pinned to block 51302915 (the parent of the settled swap 0x46c3cd72...), and the tracked entity it produced
// (testdata/tracked_51302915.json).

const (
	trackerTape   = "testdata/tracker_rpc_51302915.json.gz"
	trackedEntity = "testdata/tracked_51302915.json"
	trackerBlock  = 51302915
)

// TestIntegrationFixtureDigests pins the integration fixtures (testdata/README.md section 5). Re-recording the tape
// moves the listing's head block, so the tape's and the tracked entity's digests change together (extending it with
// EVERLONG_FLAMM_RECORD=missing keeps the listing).
func TestIntegrationFixtureDigests(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]string{
		"tracker_rpc_51302915.json.gz":       "1622fd4df79c7e027fe93bea579ce47bee641ad0aa54159ed4d51a36d159f8ad",
		"tracked_51302915.json":              "4281d97c4c7af0c159265b4ab4c0d35644e06f711eda5b3e89d065938d5e4cb8",
		"tracker_rpc_armed_51313004.json.gz": "42413a226967a887e070b8fe42cc5ee5acad8db84f0c3231495fdfb690e1c1cf",
		"fork_sequence_51330064.json":        "9b1ef76bc47701646c7f2721cf573ede806ff48a9260c883e03699d0614621f5",
	} {
		raw, err := os.ReadFile("testdata/" + name)
		require.NoError(t, err)
		sum := sha256.Sum256(raw)
		require.Equal(t, want, hex.EncodeToString(sum[:]), name)
	}
}

func replayListing(t *testing.T, tp *rpcTape) entity.Pool {
	t.Helper()
	pools, _, err := NewPoolsListUpdater(tapeConfig(), tp.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	return pools[0]
}

func replayTracked(t *testing.T, tp *rpcTape, listed entity.Pool) entity.Pool {
	t.Helper()
	tracked, err := NewPoolTracker(baseConfig(), tp.client()).GetNewPoolStateAtBlock(context.Background(), listed,
		big.NewInt(trackerBlock))
	require.NoError(t, err)
	return tracked
}

// gridReads is a scenario state of the core end-to-end grid fixture at block (a fresh copy on every call).
func gridReads(t testing.TB, block, tag string) *flammReads {
	t.Helper()
	raw, ok := gridStateJSON(t, block, tag)
	if !ok {
		t.Fatalf("no %s state at %s", tag, block)
	}
	var reads flammReads
	require.NoError(t, json.Unmarshal(raw, &reads))
	return &reads
}

func TestTrackerReplay(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracked := replayTracked(t, tp, listed)
	tp.save()

	require.Equal(t, uint64(trackerBlock), tracked.BlockNumber)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	require.GreaterOrEqual(t, extra.Probes, 12)
	require.Equal(t, "219423", tracked.Reserves[0], "gross cbBTC")
	// The refresh republishes the venue set it read; the listing's identity carries none.
	require.Equal(t, entityVenues(t, listed), extra.Venues)
	require.NotContains(t, tracked.StaticExtra, "venues")

	// The tracker reads exactly the state the core end-to-end fixture dumped at the same block (the fork's
	// unmodified "live" scenario): same views, same storage words, same decoding.
	want := gridReads(t, "51302915", "live")
	if !reflect.DeepEqual(*want, *extra.Reads) {
		gotJ, _ := json.Marshal(extra.Reads)
		wantJ, _ := json.Marshal(want)
		t.Fatalf("tracker reads differ from the fixture dump:\n got %s\nwant %s", gotJ, wantJ)
	}

	// The persisted entity is what this replay produces (the refresh wall-clock stamp aside); it is rewritten when
	// recording or with EVERLONG_FLAMM_WRITE_TRACKED=1.
	if r := os.Getenv("EVERLONG_FLAMM_RECORD"); r == "1" || r == "missing" || r == "tracked" {
		tracked.Timestamp = 0
		raw, err := json.MarshalIndent(tracked, "", " ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(trackedEntity, append(raw, '\n'), 0o644))
	}
	stored := loadTracked(t)
	require.Equal(t, stored.Extra, tracked.Extra)
	require.Equal(t, stored.StaticExtra, tracked.StaticExtra)
	require.Equal(t, stored.Reserves, tracked.Reserves)

	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(15000)},
		TokenOut:      sim.Info.Tokens[1]})
	require.NoError(t, err)
	require.Equal(t, "11301759", res.TokenAmountOut.Amount.String())
}

// entityVenues is the Router venue set an entity publishes (Extra.Venues): the listing's at listing, and the one
// the last refresh read afterwards.
// mustState builds the state a set of reads describes.
func (r *flammReads) mustState(t testing.TB) *flammState {
	t.Helper()
	st, err := r.baseState()
	require.NoError(t, err)
	return st
}

func entityVenues(t testing.TB, p entity.Pool) []StaticVenue {
	t.Helper()
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.NotEmpty(t, extra.Venues)
	return extra.Venues
}

func loadTracked(t testing.TB) entity.Pool {
	t.Helper()
	raw, err := os.ReadFile(trackedEntity)
	require.NoError(t, err)
	var p entity.Pool
	require.NoError(t, json.Unmarshal(raw, &p))
	return p
}

// replaySnapshot reads the refresh's snapshot and state from the tape, with the venue set the tracked entity
// published (Extra.Venues).
func replaySnapshot(t *testing.T, tp *rpcTape) (*StaticExtra, []StaticVenue, *snapshot, *flammState) {
	t.Helper()
	stored := loadTracked(t)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(stored.StaticExtra), &se))
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(stored.Extra), &extra))
	require.NotEmpty(t, extra.Venues)
	rpc := &mcRPC{client: tp.client()}
	snap, err := readSnapshot(context.Background(), rpc, common.HexToAddress(stored.Address), &se,
		&c104.wiring(t).hooks, extra.Venues, big.NewInt(trackerBlock))
	require.NoError(t, err)
	st, err := snap.Reads.baseState()
	require.NoError(t, err)
	return &se, extra.Venues, snap, st
}

// TestTrackerAttestationBinds: a state that differs from the chain in any priced word fails the attestation.
func TestTrackerAttestationBinds(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	se, _, snap, st := replaySnapshot(t, tp)
	pool := c104.Pool
	require.Empty(t, attestViews(snap, st))
	n, failure, err := attestProbes(context.Background(), &mcRPC{client: tp.client()}, pool, se, snap, st)
	require.NoError(t, err)
	require.Empty(t, failure)

	probes := probesFor(pool, se, st, snap.PeekCrossOk, &snap.PeekCrossPrice, snap.Timestamp)
	require.Len(t, probes, n)
	calls := make([]mcCall, len(probes))
	for i := range probes {
		calls[i] = probes[i].call
	}
	_, results, err := (&mcRPC{client: tp.client()}).aggregate(context.Background(), big.NewInt(trackerBlock),
		callOpts{gas: probeAggregateGas}, calls)
	require.NoError(t, err)

	// reprobe re-evaluates the same calls on a tampered state.
	reprobe := func(tampered *flammState) string {
		for i := range probes {
			pr := probes[i]
			pr.words, pr.err = nil, nil
			args := pr.call.Args
			switch pr.call.Method {
			case "previewSwap":
				a, _ := uint256.FromBig(args[1].(*big.Int))
				pr = probeSwap(pool, tampered, args[0].(bool), a, snap.Timestamp)
			case "previewLever":
				a, _ := uint256.FromBig(args[1].(*big.Int))
				pr = probeLever(pool, tampered, args[0].(bool), a, snap.Timestamp)
			case "fundingCeiling":
				c, _ := uint256.FromBig(args[2].(*big.Int))
				p, _ := uint256.FromBig(args[3].(*big.Int))
				w, err := tampered.Router.fundingCeiling(0, c, p, snap.Timestamp)
				pr.words, pr.err = []uint256.Int{w}, err
				if err != nil {
					pr.words = nil
				}
			}
			if f := pr.check(results[i]); f != "" {
				return f
			}
		}
		return ""
	}
	require.Empty(t, reprobe(st))
	for name, tamper := range map[string]func(s *flammState){
		"hook x": func(s *flammState) { s.Hooks.Swap.EverlongSwap.XWad.AddUint64(&s.Hooks.Swap.EverlongSwap.XWad, 1e14) },
		"hook rp": func(s *flammState) {
			rp := &s.Hooks.Swap.EverlongSwap.ReservationPriceWad
			rp.Add(rp, new(uint256.Int).Div(rp, uint256.NewInt(100)))
		},
		"fee cap":  func(s *flammState) { s.FeeCapWad.SetUint64(1e15) },
		"band":     func(s *flammState) { s.Pool.Loans[0].SwapPriceBandWad.SetUint64(1e16) },
		"physical": func(s *flammState) { s.Pool.Physical.AddUint64(&s.Pool.Physical, 1000) },
		"borrow shares": func(s *flammState) {
			s.Router.Venues[0].Morpho.Position.BorrowShares.SetUint64(1_000_000_000_000)
		},
		"rate at target": func(s *flammState) {
			m := &s.Router.Venues[0].Morpho
			m.RateAtTarget.Mul(&m.RateAtTarget, uint256.NewInt(3))
		},
		"feed answer": func(s *flammState) {
			a := &s.Feed.Asset.Round.Answer
			a.Add(a, uint256.NewInt(1_000_000))
		},
	} {
		c := st.clone()
		tamper(c)
		views := attestViews(snap, c)
		probed := reprobe(c)
		require.True(t, views != "" || probed != "", "%s: tampered state still attests", name)
		t.Logf("%s: views %q probes %q", name, views, probed)
	}

	// The other side of the same comparison: every view attestViews reproduces, one answer at a time. Each is the
	// only thing that binds the words behind it, and several of them (the peg, a venue position, the account's
	// share of the Morpho position, whether the IRM answers, the totalAssets revert class) price states no probe
	// of this snapshot reaches.
	copySnapshot := func() *snapshot {
		c := *snap
		c.VenuePositions = append([][5]uint256.Int(nil), snap.VenuePositions...)
		c.TryPositions = append([]mmVenueRead(nil), snap.TryPositions...)
		c.BorrowRates = append([]uint256.Int(nil), snap.BorrowRates...)
		if snap.Positions != nil {
			pos := *snap.Positions
			pos.Coll = append([]uint256.Int(nil), snap.Positions.Coll...)
			pos.Sup = append([]uint256.Int(nil), snap.Positions.Sup...)
			pos.Debt = append([]uint256.Int(nil), snap.Positions.Debt...)
			c.Positions = &pos
		}
		if snap.TotalAssets != nil {
			c.TotalAssets = new(uint256.Int).Set(snap.TotalAssets)
		}
		if snap.PegOk != nil {
			peg := *snap.PegOk
			c.PegOk = &peg
		}
		return &c
	}
	require.NotNil(t, snap.PegOk, "the recorded refresh reads pegOk")
	require.NotNil(t, snap.Positions)
	require.NotNil(t, snap.TotalAssets)
	require.True(t, snap.TryPositions[0].Readable)
	for name, c := range map[string]struct {
		tamper func(s *snapshot)
		prefix string
	}{
		"peekCross ok":        {func(s *snapshot) { s.PeekCrossOk = !s.PeekCrossOk }, "peekCross:"},
		"peekCross price":     {func(s *snapshot) { s.PeekCrossPrice.AddUint64(&s.PeekCrossPrice, 1) }, "peekCross:"},
		"peekCross ts":        {func(s *snapshot) { s.PeekCrossTs++ }, "peekCross:"},
		"pegOk":               {func(s *snapshot) { *s.PegOk = !*s.PegOk }, "pegOk:"},
		"pegOk reverts":       {func(s *snapshot) { s.PegOk = nil }, "pegOk:"},
		"positions revert":    {func(s *snapshot) { s.Positions = nil }, "positions:"},
		"positions coll":      {func(s *snapshot) { s.Positions.Coll[0].AddUint64(&s.Positions.Coll[0], 1) }, "positions:"},
		"positions supplied":  {func(s *snapshot) { s.Positions.Sup[0].AddUint64(&s.Positions.Sup[0], 1) }, "positions:"},
		"positions debt":      {func(s *snapshot) { s.Positions.Debt[0].AddUint64(&s.Positions.Debt[0], 1) }, "positions:"},
		"positions total":     {func(s *snapshot) { s.Positions.TotalColl.AddUint64(&s.Positions.TotalColl, 1) }, "positions:"},
		"gross identity":      {func(s *snapshot) { s.Reads.Pool.Gross.AddUint64(&s.Reads.Pool.Gross, 1) }, "gross:"},
		"loanPosition liquid": {func(s *snapshot) { s.LoanPosition[0].AddUint64(&s.LoanPosition[0], 1) }, "loanPosition:"},
		"loanPosition supply": {func(s *snapshot) { s.LoanPosition[1].AddUint64(&s.LoanPosition[1], 1) }, "loanPosition:"},
		"loanPosition debt":   {func(s *snapshot) { s.LoanPosition[2].AddUint64(&s.LoanPosition[2], 1) }, "loanPosition:"},
		"venuePosition": {func(s *snapshot) {
			vp := &s.VenuePositions[0]
			vp[3].AddUint64(&vp[3], 1)
		}, "venuePosition(0):"},
		"tryPosition readable": {func(s *snapshot) { s.TryPositions[0].Readable = false }, "tryPosition(0):"},
		"tryPosition collateral": {func(s *snapshot) {
			tp := &s.TryPositions[0]
			tp.Collateral.AddUint64(&tp.Collateral, 1)
		}, "tryPosition(0):"},
		"tryPosition supply shares": {func(s *snapshot) {
			tp := &s.TryPositions[0]
			tp.SupplyShares.AddUint64(&tp.SupplyShares, 1)
		}, "tryPosition(0):"},
		"tryPosition supplied": {func(s *snapshot) {
			tp := &s.TryPositions[0]
			tp.Supplied.AddUint64(&tp.Supplied, 1)
		}, "tryPosition(0):"},
		"tryPosition debt": {func(s *snapshot) {
			tp := &s.TryPositions[0]
			tp.Debt.AddUint64(&tp.Debt, 1)
		}, "tryPosition(0):"},
		"borrowRateAfter":     {func(s *snapshot) { s.BorrowRates[0].AddUint64(&s.BorrowRates[0], 1) }, "borrowRateAfter(0):"},
		"totalAssets":         {func(s *snapshot) { s.TotalAssets.AddUint64(s.TotalAssets, 1) }, "totalAssets:"},
		"totalAssets reverts": {func(s *snapshot) { s.TotalAssets, s.TotalAssetsErr = nil, selectorOf(t, ErrPriceBand) }, "totalAssets:"},
	} {
		tampered := copySnapshot()
		c.tamper(tampered)
		failure := attestViews(tampered, st)
		require.True(t, strings.HasPrefix(failure, c.prefix), "%s: %q", name, failure)
	}

	// The one comparison in borrowRateAfter that the rate beside it does not already make: whether the IRM
	// answers at all. It separates the two only on a market whose stored lastUpdate is ahead of the block, where
	// the port's own rate view stops answering and returns zero -- so the chain is given that market's position,
	// a zero rate, and the readable flag it reports today.
	ahead := st.clone()
	m := &ahead.Router.Venues[0].Morpho
	m.Market.LastUpdate.SetUint64(snap.Timestamp + 1_000_000)
	rateOk, rate, err := m.tryBorrowRate(uZero, uZero, snap.Timestamp)
	require.NoError(t, err)
	require.False(t, rateOk, "the port's rate view stops answering")
	require.True(t, rate.IsZero())
	require.True(t, m.IrmReadable, "while the chain reports the IRM readable")
	consistent := copySnapshot()
	tp0, err := m.tryPosition(snap.Timestamp)
	require.NoError(t, err)
	consistent.TryPositions[0] = tp0
	consistent.BorrowRates[0].Clear()
	require.True(t, strings.HasPrefix(attestViews(consistent, ahead), "borrowRateAfter(0):"),
		"%q", attestViews(consistent, ahead))
}

// TestTrackerDrift: the refresh refuses a wiring that is not the listing.
func TestTrackerDrift(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	se, venues, snap, _ := replaySnapshot(t, tp)
	pool := c104.Pool
	prof := &c104
	w := prof.wiring(t)
	require.Empty(t, snap.drift(pool, se, venues, w, nil))
	other := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	for name, tamper := range map[string]func(s *snapshot){
		"pair":           func(s *snapshot) { s.LoanAsset = other },
		"loan count":     func(s *snapshot) { s.LoanCount.SetUint64(2) },
		"implementation": func(s *snapshot) { s.Implementation = other },
		"hooks":          func(s *snapshot) { s.Hooks[5] = other },
		"router":         func(s *snapshot) { s.Router = other },
		"genesis":        func(s *snapshot) { s.Bindings[roleSwap].Genesis[0] ^= 1 },
		"lev binding":    func(s *snapshot) { s.Bindings[roleLeverage].Pool = other },
		"venue count":    func(s *snapshot) { s.VenueCount.SetUint64(2) },
		"venue id":       func(s *snapshot) { s.VenueIDs[0][31] ^= 1 },
		"aggregator":     func(s *snapshot) { s.FeedAggregators[1] = other },
		"heartbeat": func(s *snapshot) {
			s.Reads.Feed.Tokens = append([]feedTokenReads(nil), s.Reads.Feed.Tokens...)
			s.Reads.Feed.Tokens[0].Heartbeat.SetUint64(7200)
		},
		"layout": func(s *snapshot) { s.LayoutOk = false },
		// One case per remaining predicate: nothing else binds these words, and a refresh that published them
		// would quote a pool wired to something the port does not mirror.
		"pool asset": func(s *snapshot) { s.Asset = other },
		"loan config pair": func(s *snapshot) {
			s.Reads.Pool.Loans = append([]loanConfigReads(nil), s.Reads.Pool.Loans...)
			s.Reads.Pool.Loans[0].Token = other
		},
		"router loan count":  func(s *snapshot) { s.RouterLoanCount.SetUint64(2) },
		"router loan token":  func(s *snapshot) { s.RouterLoanToken = other },
		"price feed binding": func(s *snapshot) { s.PriceFeed = other },
		"factory binding":    func(s *snapshot) { s.Factory = other },
		"hook set hash":      func(s *snapshot) { s.Hooks[0] = other },
		"swap hook pool":     func(s *snapshot) { s.Bindings[roleSwap].Pool = other },
		"swap hook loan scale": func(s *snapshot) {
			s.Bindings[roleSwap].LoanScale.AddUint64(&s.Bindings[roleSwap].LoanScale, 1)
		},
		"leverage hook binding": func(s *snapshot) { s.Bindings[roleLeverage].Hook = other },
		"leverage hook loan scale": func(s *snapshot) {
			s.Bindings[roleLeverage].LoanScale.AddUint64(&s.Bindings[roleLeverage].LoanScale, 1)
		},
		"spread hook pool": func(s *snapshot) { s.Bindings[roleSpread].Pool = other },
		"sequencer feed":   func(s *snapshot) { s.SequencerFeed = other },
		"loan heartbeat": func(s *snapshot) {
			s.Reads.Feed.Tokens = append([]feedTokenReads(nil), s.Reads.Feed.Tokens...)
			s.Reads.Feed.Tokens[1].Heartbeat.SetUint64(7200)
		},
		"token not registered": func(s *snapshot) {
			s.Reads.Feed.Tokens = append([]feedTokenReads(nil), s.Reads.Feed.Tokens...)
			s.Reads.Feed.Tokens[1].Known = false
		},
	} {
		c := *snap
		c.Hooks = snap.Hooks
		c.VenueIDs = append([]common.Hash(nil), snap.VenueIDs...)
		tamper(&c)
		require.NotEmpty(t, c.drift(pool, se, venues, w, nil), name)
	}
	for _, a := range []common.Address{pool, prof.Factory, prof.Implementation, prof.Hook, prof.LeverageHook,
		prof.SpreadHook, prof.Router, prof.PriceFeed, prof.Account, prof.Morpho, adaptiveCurveIrm} {
		code := map[common.Address]gethclient.OverrideAccount{a: {Code: []byte{0x00}}}
		require.Equal(t, "code override "+a.Hex(), snap.drift(pool, se, venues, w, code), a.Hex())
	}
	// The venue set the refresh publishes is checked against the pool and against the registry.
	require.Equal(t, "venue count", snap.drift(pool, se, append(append([]StaticVenue(nil), venues...), venues[0]),
		w, nil))
	foreign := append([]StaticVenue(nil), venues...)
	foreign[0].Account = other
	require.Equal(t, "venue profile", snap.drift(pool, se, foreign, w, nil))
	foreign[0] = venues[0]
	foreign[0].Irm = other
	require.Equal(t, "venue profile", snap.drift(pool, se, foreign, w, nil))
	require.False(t, snap.venueSetMoved(venues))
	require.True(t, snap.venueSetMoved(nil))
	require.True(t, sameVenues(venues, append([]StaticVenue(nil), venues...)))
	require.False(t, sameVenues(venues, foreign))

	// Overrides of state alone, or of code elsewhere, are not drift.
	require.Empty(t, snap.drift(pool, se, venues, w, map[common.Address]gethclient.OverrideAccount{
		prof.Account: {StateDiff: map[common.Hash]common.Hash{{}: {}}}, prof.PoolAsset: {Code: []byte{0x00}}}))
}

// TestTrackerVenueSetReread: a refresh with no venue set to start from reads it from the Router before the view
// round, at the same block. That is the path a pool whose curator added a venue heals through -- the round's venue
// count then disagrees with the set it read, the set is read again and republished in Extra, and the pool attests
// without being relisted (TestForkVenuesAndFlows runs it end to end on a fork).
func TestTrackerVenueSetReread(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	prof := &c104
	stored := loadTracked(t)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(stored.StaticExtra), &se))

	// The venue set the tracked entity carries needs no re-read: the round agrees with it.
	rpc := &mcRPC{client: tp.client()}
	snap, venues, err := readRound(context.Background(), rpc, prof.Pool, &se, &prof.wiring(t).hooks,
		entityVenues(t, stored), big.NewInt(trackerBlock))
	require.NoError(t, err)
	require.Equal(t, entityVenues(t, stored), venues)
	require.False(t, snap.venueSetMoved(venues))

	// With no set, the refresh reads router.venueCount(pool) on its own first (the tape, which holds only the
	// requests the recorded refresh made, has no answer for it).
	count, err := routerABI.Pack("venueCount", prof.Pool)
	require.NoError(t, err)
	round, err := multicallABI.Pack("tryBlockAndAggregate", false, []struct {
		Target   common.Address
		CallData []byte
	}{{Target: prof.Router, CallData: count}})
	require.NoError(t, err)
	_, _, err = readRound(context.Background(), rpc, prof.Pool, &se, &prof.wiring(t).hooks, nil,
		big.NewInt(trackerBlock))
	require.Error(t, err)
	require.True(t, tapeAsked(tp, round), "the refresh reads the venue set itself")

	// A curator's addVenue on the live pool: the round's venue count no longer agrees with the set the entity
	// carries, so the refresh re-reads the set from the Router at the same block, runs the round again with it and
	// publishes the set it read. The re-read's own rounds are ones the recorded refresh never made, so they are
	// assembled call by call from the answers the tape did record (aggregateFromTape), with the market's lltv moved
	// in the assembled answer alone -- a word no call is addressed with and no registry check reads, so the Router
	// answers a set that differs from the entity's and the round it drives is the one already on the tape.
	lltv := uint256.NewInt(815_000_000_000_000_000)
	marketParams := callTo(t, prof.Morpho, &morphoABI, "idToMarketParams", entityVenues(t, stored)[0].MarketID)
	for _, c := range []struct {
		name  string
		count uint64
	}{{"agrees", 1}, {"addVenue", 2}} {
		tp := openTape(t, trackerTape)
		listed := replayListing(t, tp)
		answers := tapeSubcallAnswers(t, tp)
		var countMoved func(string, json.RawMessage, *tapeEntry)
		if c.count != uint64(len(entityVenues(t, stored))) {
			countMoved = aggregateCallMutation(t, callTo(t, prof.Router, &routerABI, "venueCount", prof.Pool),
				setResultWord(0, new(big.Int).SetUint64(c.count).Bytes()))
		}
		tp.mutate = chainMutations(countMoved, aggregateFromTape(t, answers,
			func(target common.Address, data []byte, r *tapeResult) {
				if marketParams(target, data) {
					setResultWord(4, lltv.Bytes())(r) // MarketParams.lltv, the tuple's fifth word
				}
			}))
		tracked, err := NewPoolTracker(baseConfig(), tp.client()).GetNewPoolStateAtBlock(context.Background(),
			listed, big.NewInt(trackerBlock))
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.Equal(t, c.count != 1, tapeAsked(tp, round), "%s: standalone venueCount round", c.name)
		if c.count == 1 {
			require.Empty(t, extra.ProfileDrift, c.name)
			require.True(t, extra.Attested, "%s: %q", c.name, extra.AttestFailure)
			require.Equal(t, entityVenues(t, stored), extra.Venues, c.name)
			continue
		}
		// The published set is the one the Router answered the re-read with, not the one the entity carried, and
		// the round the refresh publishes ran on it.
		require.Len(t, extra.Venues, 1, c.name)
		require.Equal(t, lltv.Dec(), extra.Venues[0].Lltv.Dec(), c.name)
		require.Equal(t, lltv.Dec(), extra.Reads.Router.Venues[0].MarketLltv.Dec(), c.name)
		require.Equal(t, entityVenues(t, stored)[0].Account, extra.Venues[0].Account, c.name)
		// The round still answers the higher count, so the refresh publishes the drift rather than a quote.
		require.Equal(t, "venue count", extra.ProfileDrift, c.name)
		require.Equal(t, entity.PoolReserves{"0", "0"}, tracked.Reserves, c.name)
	}
}

// tapeAsked reports whether the tape was asked for an eth_call carrying data.
func tapeAsked(tp *rpcTape, data []byte) bool {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	for k := range tp.used {
		if strings.Contains(k, hexutil.Encode(data)[2:]) {
			return true
		}
	}
	return false
}

// tapeSubcallAnswers is every (target, calldata) -> answer the tape recorded inside a tryBlockAndAggregate result.
// A round sent at an overridden clock is left out: the same call answers differently there, and the rounds these
// answers reassemble are all read at the block's own clock.
func tapeSubcallAnswers(t *testing.T, tp *rpcTape) map[string]tapeResult {
	t.Helper()
	out := map[string]tapeResult{}
	tp.mu.Lock()
	defer tp.mu.Unlock()
	for key, e := range tp.entries {
		method, params, ok := strings.Cut(key, "|")
		if !ok || method != "eth_call" || e.Result == nil || strings.Contains(params, `"time"`) {
			continue
		}
		calls, results := aggregateOf(t, json.RawMessage(params), e.Result)
		for i := range results {
			out[subcallKey(calls[i].Target, calls[i].CallData)] = results[i]
		}
	}
	return out
}

func subcallKey(target common.Address, data []byte) string { return string(target[:]) + string(data) }

// aggregateCall is one call of a recorded tryBlockAndAggregate request.
type aggregateCall struct {
	Target   common.Address
	CallData []byte
}

// aggregateOf decodes a recorded tryBlockAndAggregate eth_call into its calls and their answers; a request that is
// not one has neither, and an answer that is not one leaves the answers nil.
func aggregateOf(t *testing.T, params, result json.RawMessage) ([]aggregateCall, []tapeResult) {
	t.Helper()
	m := multicallABI.Methods["tryBlockAndAggregate"]
	in := aggregateInput(params)
	if len(in) < 4 || !bytes.Equal(in[:4], m.ID) {
		return nil, nil
	}
	ins, err := m.Inputs.Unpack(in[4:])
	if err != nil {
		return nil, nil
	}
	calls := *abiConvert[[]aggregateCall](t, ins[1])
	var hexResult string
	if result == nil || json.Unmarshal(result, &hexResult) != nil {
		return calls, nil
	}
	vals, err := m.Outputs.Unpack(common.FromHex(hexResult))
	if err != nil || len(vals) != 3 {
		return calls, nil
	}
	results := *abiConvert[[]tapeResult](t, vals[2])
	if len(results) != len(calls) {
		return calls, nil
	}
	return calls, results
}

// aggregateInput is the calldata an eth_call request's params carry (nil: not an eth_call at a block).
func aggregateInput(params json.RawMessage) []byte {
	var args []json.RawMessage
	if json.Unmarshal(params, &args) != nil || len(args) < 2 {
		return nil
	}
	var msg struct {
		Input hexutil.Bytes `json:"input"`
		Data  hexutil.Bytes `json:"data"`
	}
	if json.Unmarshal(args[0], &msg) != nil {
		return nil
	}
	if len(msg.Input) != 0 {
		return msg.Input
	}
	return msg.Data
}

// aggregateFromTape answers an aggregate the tape has no entry for -- a round the recorded run had no reason to
// make -- by assembling the answer the tape recorded for each of its calls elsewhere, at the block the request
// names. edit rewrites an assembled answer; a call with no recorded answer leaves the tape miss alone.
func aggregateFromTape(t *testing.T, answers map[string]tapeResult,
	edit func(target common.Address, data []byte, r *tapeResult)) func(string, json.RawMessage, *tapeEntry) {
	m := multicallABI.Methods["tryBlockAndAggregate"]
	return func(method string, params json.RawMessage, e *tapeEntry) {
		if method != "eth_call" || e.Result != nil {
			return
		}
		calls, _ := aggregateOf(t, params, nil)
		if len(calls) == 0 {
			return
		}
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal(params, &args))
		var block string
		if json.Unmarshal(args[1], &block) != nil || !strings.HasPrefix(block, "0x") {
			return
		}
		if len(args) > 3 && strings.Contains(string(args[3]), `"time"`) {
			return // a round sent at an overridden clock: the recorded answers are the snapshot's own
		}
		results := make([]tapeResult, len(calls))
		for i := range calls {
			r, ok := answers[subcallKey(calls[i].Target, calls[i].CallData)]
			if !ok {
				return
			}
			results[i] = tapeResult{Success: r.Success, ReturnData: append([]byte(nil), r.ReturnData...)}
			if edit != nil {
				edit(calls[i].Target, calls[i].CallData, &results[i])
			}
		}
		packed, err := m.Outputs.Pack(new(big.Int).SetBytes(common.FromHex(block)), [32]byte{}, results)
		require.NoError(t, err)
		e.Error = nil
		e.Result, _ = json.Marshal(hexutil.Encode(packed))
	}
}

// TestTrackerFailsClosed: a storage word that disagrees with the views, a required view that reverts and one whose
// answer does not decode each publish a refusing entity at the block the chain answered at; a transport failure
// returns the error.
func TestTrackerFailsClosed(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	physical := strings.ToLower(flammSlot(slotPhysical).Hex())
	tp.mutate = func(method string, params json.RawMessage, e *tapeEntry) {
		if method == "eth_getStorageAt" && strings.Contains(strings.ToLower(string(params)), physical) {
			e.Result = json.RawMessage(`"0x0000000000000000000000000000000000000000000000000000000000000001"`)
		}
	}
	tracked := replayTracked(t, tp, listed)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.False(t, extra.Attested)
	require.Equal(t, "storage layout", extra.ProfileDrift)
	_, err := NewPoolSimulator(tracked)
	require.ErrorIs(t, err, ErrProfileDrift)

	tracker := NewPoolTracker(baseConfig(), tp.client())
	for name, mutate := range map[string]func(e *tapeEntry){
		"eth_call transport": func(e *tapeEntry) { e.Result, e.Error = nil, &tapeError{Code: -32000, Message: "unavailable"} },
		"multicall envelope": func(e *tapeEntry) { e.Result = json.RawMessage(`"0x00"`) },
	} {
		tp.mutate = func(method string, _ json.RawMessage, e *tapeEntry) {
			if method == "eth_call" {
				mutate(e)
			}
		}
		_, err = tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.Error(t, err, name)
		require.False(t, chainAnswered(err), name)
	}
	tp.mutate = func(method string, _ json.RawMessage, e *tapeEntry) {
		if method == "eth_getStorageAt" {
			e.Result, e.Error = nil, &tapeError{Code: -32000, Message: "header not found"}
		}
	}
	_, err = tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
	require.ErrorIs(t, err, errRPC, "storage transport")

	// The view round's first call (pool.asset) answered by the chain with a revert, or with data that does not decode.
	for name, c := range map[string]struct {
		edit func(r *tapeResult)
		want error
	}{
		"reverted":    {func(r *tapeResult) { r.Success = false }, errReadFailed},
		"undecodable": {func(r *tapeResult) { r.ReturnData = nil }, errReadDecode},
	} {
		tp.mutate = aggregateMutation(t, 30, func(results []tapeResult) { c.edit(&results[0]) })
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.False(t, extra.Attested, name)
		require.Nil(t, extra.Reads, name)
		require.True(t, strings.HasPrefix(extra.ProfileDrift, "unreadable: "), "%s: %q", name, extra.ProfileDrift)
		require.Contains(t, extra.ProfileDrift, "pool.asset", name)
		require.Contains(t, extra.ProfileDrift, c.want.Error(), name)
		require.Equal(t, uint64(trackerBlock), tracked.BlockNumber, name)
		require.Equal(t, entity.PoolReserves{"0", "0"}, tracked.Reserves, name)
		_, err = NewPoolSimulator(tracked)
		require.ErrorIs(t, err, ErrProfileDrift, name)
	}
}

// TestTrackerLayoutBindsRouterConfig: every Router configuration word the port decodes from a view is compared with
// its storage slot, so a one-bit difference in either source -- a flag no probe prices at this state (a retired
// loan, a supply-disabled venue, the drawn-asset limit) included -- publishes drift instead of attesting.
func TestTrackerLayoutBindsRouterConfig(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	prof := &c104
	record := routerRecordSlot(prof.Pool)
	at := func(base *big.Int, off int64) common.Hash {
		return common.BigToHash(new(big.Int).Add(base, big.NewInt(off)))
	}
	loan, venue := routerLoanSlot(prof.Pool, 0), routerVenueSlot(prof.Pool, 0)
	cases := map[string]struct {
		account common.Address
		slot    common.Hash
		bit     uint
	}{
		"globalPaused":           {prof.Router, common.Hash{}, 0},
		"record poolAsset":       {prof.Router, at(record, 0), 0},
		"record pinLtvWad":       {prof.Router, at(record, 0), 160},
		"record maxDrawnAssets":  {prof.Router, at(record, 0), 224},
		"record safetyGapWad":    {prof.Router, at(record, 1), 0},
		"record oracleBandWad":   {prof.Router, at(record, 1), 64},
		"loans length":           {prof.Router, at(record, 2), 0},
		"venues length":          {prof.Router, at(record, 3), 0},
		"borrowOrder length":     {prof.Router, at(record, 4), 0},
		"repayOrder length":      {prof.Router, at(record, 7), 0},
		"supplyOrder[0]":         {prof.Router, common.BigToHash(routerArraySlot(new(big.Int).Add(record, big.NewInt(5)), 0)), 0},
		"loan token":             {prof.Router, at(loan, 0), 0},
		"loan decimals":          {prof.Router, at(loan, 0), 160},
		"loan borrowEnabled":     {prof.Router, at(loan, 0), 168},
		"loan retired":           {prof.Router, at(loan, 0), 176},
		"loan loanScale":         {prof.Router, at(loan, 1), 0},
		"loan debtCap":           {prof.Router, at(loan, 2), 0},
		"loan supplyCap":         {prof.Router, at(loan, 2), 128},
		"venue kind":             {prof.Router, at(venue, 2), 0},
		"venue loanIndex":        {prof.Router, at(venue, 2), 8},
		"venue lltvWad":          {prof.Router, at(venue, 2), 16},
		"venue borrowEnabled":    {prof.Router, at(venue, 2), 80},
		"venue supplyEnabled":    {prof.Router, at(venue, 2), 88},
		"venue retired":          {prof.Router, at(venue, 2), 96},
		"venue debtCap":          {prof.Router, at(venue, 2), 104},
		"venue supplyCap":        {prof.Router, at(venue, 3), 0},
		"venue maxBorrowRateWad": {prof.Router, at(venue, 3), 128},
		"venue account":          {prof.Router, at(venue, 0), 0},
		"venue id":               {prof.Router, at(venue, 1), 0},
		// FLAMMStore's own words beside the views that report them, and the factory's beacon and upgrade
		// schedule: the pending pair drives Extra.ScheduledChangeAt, which refuses every quote, and nothing else
		// binds it.
		"pool physical":                 {prof.Pool, flammSlot(slotPhysical), 0},
		"pool pendingControllerHook":    {prof.Pool, flammSlot(slotSpreadWord), 0},
		"pool hookSetExecutableAt":      {prof.Pool, flammSlot(slotSpreadWord), 192},
		"pool features":                 {prof.Pool, flammSlot(slotFeatures), 0},
		"pool leverageHook":             {prof.Pool, flammSlot(slotLevHook), 0},
		"pool spreadHook":               {prof.Pool, flammSlot(slotSpreadHook), 0},
		"factory implementation":        {prof.Factory, common.BigToHash(big.NewInt(factorySlotImplementation)), 0},
		"factory pendingImplementation": {prof.Factory, common.BigToHash(big.NewInt(factorySlotPending)), 0},
		"factory implementationExecutableAt": {prof.Factory,
			common.BigToHash(big.NewInt(factorySlotPending)), 160},
	}
	tracker := NewPoolTracker(baseConfig(), tp.client())
	for name, c := range cases {
		want := strings.ToLower(c.slot.Hex())
		account := strings.ToLower(c.account.Hex())
		hit := false
		tp.mutate = func(method string, params json.RawMessage, e *tapeEntry) {
			p := strings.ToLower(string(params))
			if method != "eth_getStorageAt" || !strings.Contains(p, want) || !strings.Contains(p, account) {
				return
			}
			var hexWord string
			require.NoError(t, json.Unmarshal(e.Result, &hexWord))
			var w uint256.Int
			w.SetBytes(common.FromHex(hexWord))
			var flip uint256.Int
			w.Xor(&w, flip.Lsh(uOne, c.bit))
			e.Result, _ = json.Marshal(hexutil.Encode(w.PaddedBytes(32)))
			hit = true
		}
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, name)
		require.True(t, hit, "%s: slot %s never read", name, want)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.Equal(t, "storage layout", extra.ProfileDrift, name)
	}
}

const (
	armedTape      = "testdata/tracker_rpc_armed_51313004.json.gz"
	armedTapeBlock = 51313004
)

// TestTrackerReplayArmed replays a refresh of the pool with the leverage venue armed (recorded on an anvil fork by
// TestForkRecordArmedTape): it attests with previewLever probes in both directions and their band edges, and a
// previewLever answer or revert the port does not reproduce fails it closed.
func TestTrackerReplayArmed(t *testing.T) {
	t.Parallel()
	tp := openTape(t, armedTape)
	pools, _, err := NewPoolsListUpdater(tapeConfig(), tp.client()).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	listed := pools[0]
	tracker := NewPoolTracker(parityConfig(Policy{LeverRouting: true}), tp.client())
	block := big.NewInt(armedTapeBlock)
	tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, block)
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	require.False(t, extra.Reads.Pool.LevPaused)

	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(listed.StaticExtra), &se))
	snap, err := readSnapshot(context.Background(), &mcRPC{client: tp.client()}, common.HexToAddress(listed.Address),
		&se, &c104.wiring(t).hooks, extra.Venues, block)
	require.NoError(t, err)
	st, err := snap.Reads.baseState()
	require.NoError(t, err)
	probes := probesFor(common.HexToAddress(listed.Address), &se, st, snap.PeekCrossOk, &snap.PeekCrossPrice,
		snap.Timestamp)
	lever := map[bool]int{}
	for i := range probes {
		if probes[i].call.Method == "previewLever" {
			lever[probes[i].call.Args[0].(bool)]++
		}
	}
	require.Len(t, probes, extra.Probes)
	require.GreaterOrEqual(t, lever[true], 3, "lever-up grid and band edge")
	require.GreaterOrEqual(t, lever[false], 3, "lever-down grid and band edge")
	t.Logf("armed refresh at %d: %d probes, previewLever up %d down %d", armedTapeBlock, extra.Probes, lever[true],
		lever[false])

	sim, err := NewPoolSimulator(tracked)
	require.NoError(t, err)
	sim.nowFn = func() uint64 { return snap.Timestamp }
	for _, up := range []bool{true, false} {
		amount := probeLeverUpGrid[0]
		if !up {
			amount = probeLeverDownGrid[0]
		}
		_, err := sim.calcAmountOut(amountIn(sim, up, amount), int(VenueLever))
		require.NoError(t, err, "lever up=%v", up)
	}

	pool := common.HexToAddress(listed.Address)
	for _, c := range []struct {
		name   string
		match  func(common.Address, []byte) bool
		edit   func(r *tapeResult)
		prefix string
	}{
		{"previewLever up word", callTo(t, pool, &flammABI, "previewLever", true, new(big.Int).SetUint64(probeLeverUpGrid[0])),
			addWord(1, 1), fmt.Sprintf("previewLever(true,%d): word 1", probeLeverUpGrid[0])},
		// A revert with data: an empty one is a class of its own, confirmed on its own call
		// (TestTrackerProbeEmptyRevert).
		{"previewLever down revert", callTo(t, pool, &flammABI, "previewLever", false,
			new(big.Int).SetUint64(probeLeverDownGrid[0])),
			func(r *tapeResult) { r.Success, r.ReturnData = false, selectorOf(t, ErrPriceBand) },
			fmt.Sprintf("previewLever(false,%d): chain reverted", probeLeverDownGrid[0])},
	} {
		tp.mutate = aggregateCallMutation(t, c.match, c.edit)
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, block)
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.False(t, extra.Attested, c.name)
		require.True(t, strings.HasPrefix(extra.AttestFailure, c.prefix), "%s: %q", c.name, extra.AttestFailure)
		_, err = NewPoolSimulator(tracked)
		require.ErrorIs(t, err, ErrNotAttested, c.name)
	}
}

// TestConfigPolicy: an unset margin takes its default, an explicit zero (in Go or in the JSON configuration) turns it
// off, and the tracker stamps the result into Extra.
func TestConfigPolicy(t *testing.T) {
	t.Parallel()
	defaults := Policy{LeverMinEdgeBps: defaultLeverMinEdgeBps, PriceBandMarginBps: defaultPriceBandMarginBps,
		PriceAgeMarginSec: defaultPriceAgeMarginSec, SpreadAgeMarginSec: defaultSpreadAgeMarginSec,
		DebtDriftSec: defaultDebtDriftSec, MaxSnapshotAgeSec: defaultMaxSnapshotAgeSec}
	require.Equal(t, defaults, baseConfig().policy())
	require.Equal(t, Policy{LeverRouting: true}, parityConfig(Policy{LeverRouting: true}).policy())

	var cfg Config
	require.NoError(t, json.Unmarshal([]byte(`{"priceAgeMarginSec":0,"debtDriftSec":5,"quoteDonatedVenues":true,
		"gas":{"swapSell":1400000,"swapSellCapEval":11000,"swapSellPass":220000,"swapBuy":1600000,
		"leverUp":3700000,"leverDown":4200000}}`), &cfg))
	want := defaults
	want.PriceAgeMarginSec, want.DebtDriftSec, want.QuoteDonatedVenues = 0, 5, true
	want.GasSwapSell, want.GasSwapSellCapEval, want.GasSwapSellPass = 1_400_000, 11_000, 220_000
	want.GasSwapBuy, want.GasLeverUp, want.GasLeverDown = 1_600_000, 3_700_000, 4_200_000
	require.Equal(t, want, cfg.policy())

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(loadTracked(t).Extra), &extra))
	require.Equal(t, defaults, extra.Policy, "the recorded refresh stamps the production defaults")
}

// aggregateCallMutation edits, in every recorded tryBlockAndAggregate answer, the result of each call match selects
// by its target and calldata.
func aggregateCallMutation(t *testing.T, match func(target common.Address, data []byte) bool,
	edit func(r *tapeResult)) func(string, json.RawMessage, *tapeEntry) {
	method := multicallABI.Methods["tryBlockAndAggregate"]
	return func(rpcMethod string, params json.RawMessage, e *tapeEntry) {
		if rpcMethod != "eth_call" || e.Result == nil {
			return
		}
		var args []json.RawMessage
		require.NoError(t, json.Unmarshal(params, &args))
		var msg struct {
			Input hexutil.Bytes `json:"input"`
			Data  hexutil.Bytes `json:"data"`
		}
		require.NoError(t, json.Unmarshal(args[0], &msg))
		in := msg.Input
		if len(in) == 0 {
			in = msg.Data
		}
		if len(in) < 4 || !bytes.Equal(in[:4], method.ID) {
			return
		}
		ins, err := method.Inputs.Unpack(in[4:])
		require.NoError(t, err)
		calls := *abiConvert[[]struct {
			Target   common.Address
			CallData []byte
		}](t, ins[1])
		var hexResult string
		require.NoError(t, json.Unmarshal(e.Result, &hexResult))
		vals, err := method.Outputs.Unpack(common.FromHex(hexResult))
		require.NoError(t, err)
		results := *abiConvert[[]tapeResult](t, vals[2])
		require.Len(t, results, len(calls))
		for i := range calls {
			if match(calls[i].Target, calls[i].CallData) {
				edit(&results[i])
			}
		}
		packed, err := method.Outputs.Pack(vals[0], vals[1], results)
		require.NoError(t, err)
		e.Result, _ = json.Marshal(hexutil.Encode(packed))
	}
}

// callTo matches a call of method on target whose arguments, when args is not nil, equal args.
func callTo(t *testing.T, target common.Address, a *abi.ABI, method string, args ...any) func(common.Address,
	[]byte) bool {
	m := a.Methods[method]
	return func(to common.Address, data []byte) bool {
		if to != target || len(data) < 4 || !bytes.Equal(data[:4], m.ID) {
			return false
		}
		if args == nil {
			return true
		}
		want, err := m.Inputs.Pack(args...)
		require.NoError(t, err)
		return bytes.Equal(data[4:], want)
	}
}

// addWord adds delta to word i of an ABI answer.
func addWord(i int, delta uint64) func(r *tapeResult) {
	return func(r *tapeResult) {
		var w uint256.Int
		w.SetBytes(r.ReturnData[32*i : 32*(i+1)])
		w.AddUint64(&w, delta)
		r.ReturnData = append([]byte(nil), r.ReturnData...)
		copy(r.ReturnData[32*i:32*(i+1)], w.PaddedBytes(32))
	}
}

// TestTrackerAttestationFailsClosed: a refresh whose probe answers or attested views the port does not reproduce is
// published unattested, and the simulator refuses it. Each case changes one word of the recorded answers: a probe
// word, a probe whose answer becomes a revert, a probe the port already refuses whose revert is another class of
// the pool's own, a funding ceiling, and views only the attestation binds.
func TestTrackerAttestationFailsClosed(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	prof := &c104
	tracker := NewPoolTracker(baseConfig(), tp.client())
	priceBand := selectorOf(t, ErrPriceBand)
	// The probes the recorded refresh reverts on: previewSwap(true,1000000) and previewSwap(false,1000) revert
	// PriceBand on both sides, previewLever LevPaused. Swapping one of those for another of the pool's own errors
	// is the only case in which both sides revert and the classes differ -- the comparison attest.go probe.check
	// makes with errors.Is.
	notionalCap := selectorOf(t, ErrNotionalCap)
	levPaused := selectorOf(t, ErrLevPaused)
	for _, c := range []struct {
		name   string
		match  func(common.Address, []byte) bool
		edit   func(r *tapeResult)
		hits   int
		probes int
		prefix string
	}{
		{"previewSwap word", callTo(t, prof.Pool, &flammABI, "previewSwap", true, big.NewInt(1_000)), addWord(1, 1), 1,
			17, "previewSwap(true,1000): word 1"},
		{"previewSwap answer becomes revert", callTo(t, prof.Pool, &flammABI, "previewSwap", false,
			big.NewInt(1_000_000)), func(r *tapeResult) { r.Success, r.ReturnData = false, priceBand }, 1, 17,
			"previewSwap(false,1000000): chain reverted"},
		{"previewSwap revert class", callTo(t, prof.Pool, &flammABI, "previewSwap", true, big.NewInt(1_000_000)),
			func(r *tapeResult) {
				require.False(t, r.Success, "the recorded probe already reverts")
				require.Equal(t, priceBand, []byte(r.ReturnData), "with PriceBand, which the port reproduces")
				r.ReturnData = notionalCap
			}, 1, 17, "previewSwap(true,1000000): chain reverted"},
		{"previewLever revert class", callTo(t, prof.Pool, &flammABI, "previewLever", true, big.NewInt(10_000)),
			func(r *tapeResult) {
				require.False(t, r.Success)
				require.Equal(t, levPaused, []byte(r.ReturnData))
				r.ReturnData = priceBand
			}, 1, 17, "previewLever(true,10000): chain reverted"},
		{"fundingCeiling", callTo(t, prof.Router, &routerABI, "fundingCeiling"), addWord(0, 1), 2, 17,
			"fundingCeiling("},
		{"totalAssets view", callTo(t, prof.Pool, &flammABI, "totalAssets"), addWord(0, 1), 1, 0, "totalAssets: "},
		{"positions view", callTo(t, prof.Router, &routerABI, "positions", prof.Pool), addWord(3, 1), 1, 0,
			"positions: "},
		{"peekCross view", callTo(t, prof.PriceFeed, &priceFeedABI, "peekCross"), addWord(1, 1), 1, 0, "peekCross: "},
	} {
		hit := 0
		mutate := aggregateCallMutation(t, c.match, func(r *tapeResult) {
			hit++
			c.edit(r)
		})
		tp.mutate = mutate
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, c.name)
		require.Equal(t, c.hits, hit, "%s: calls edited", c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.False(t, extra.Attested, c.name)
		require.Empty(t, extra.ProfileDrift, c.name)
		require.True(t, strings.HasPrefix(extra.AttestFailure, c.prefix), "%s: %q", c.name, extra.AttestFailure)
		require.Equal(t, c.probes, extra.Probes, c.name)
		require.NotNil(t, extra.Reads, c.name)
		_, err = NewPoolSimulator(tracked)
		require.ErrorIs(t, err, ErrNotAttested, c.name)
	}
}

// TestTrackerScheduledChange: each of the four delayed changes -- a pending implementation (the factory's, which
// anyone may execute), a pending hook set, a scheduled venue admission and a scheduled loan asset -- is stamped as
// the earliest time any of them can execute, and the simulator stops quoting that long ahead of it. Either word of
// a pair marks it pending, and an answer that drops one from a single side (the views alone, or the storage word
// alone) is drift.
func TestTrackerScheduledChange(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	prof := &c104
	tracker := NewPoolTracker(parityConfig(Policy{}), tp.client())
	stored := loadTracked(t)
	var storedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(stored.Extra), &storedExtra))
	require.Zero(t, storedExtra.ScheduledChangeAt, "nothing is scheduled at the recorded block")
	ts := storedExtra.Reads.Timestamp
	other := common.HexToAddress("0x0000000000000000000000000000000000000bad")
	setWord := func(i int, v []byte) func(r *tapeResult) {
		return func(r *tapeResult) {
			r.ReturnData = append([]byte(nil), r.ReturnData...)
			copy(r.ReturnData[32*i:32*(i+1)], common.LeftPadBytes(v, 32))
		}
	}
	u64 := func(v uint64) []byte { return new(big.Int).SetUint64(v).Bytes() }
	// The factory's pending pair is bound to slot 1 of the factory (pendingImplementation | executableAt << 160),
	// so a scheduled upgrade has to move the storage word with the views (tracker_reads.go factorySlotPending).
	factoryPending := strings.ToLower(common.BigToHash(big.NewInt(factorySlotPending)).Hex())
	factoryWord := func(pending common.Address, at uint64) func(method string, params json.RawMessage, e *tapeEntry) {
		return func(method string, params json.RawMessage, e *tapeEntry) {
			p := strings.ToLower(string(params))
			if method != "eth_getStorageAt" || !strings.Contains(p, strings.ToLower(prof.Factory.Hex()[2:])) ||
				!strings.Contains(p, factoryPending) {
				return
			}
			var w, v uint256.Int
			w.SetBytes(pending.Bytes())
			w.Or(&w, v.Lsh(uint256.NewInt(at), 160))
			e.Result, _ = json.Marshal(hexutil.Encode(w.PaddedBytes(32)))
		}
	}
	// A FLAMMStore word of the pool, by slot offset: the two delayed admission ceremonies live in the struct's
	// last four slots (pendingLoanHash +30, pendingLoanAt +31, pendingVenueHash +32, pendingVenueAt +33), and only
	// the venue pair has a view.
	poolWord := func(off int64, word common.Hash) func(method string, params json.RawMessage, e *tapeEntry) {
		slot := strings.ToLower(flammSlot(off).Hex())
		return func(method string, params json.RawMessage, e *tapeEntry) {
			p := strings.ToLower(string(params))
			if method != "eth_getStorageAt" || !strings.Contains(p, strings.ToLower(prof.Pool.Hex()[2:])) ||
				!strings.Contains(p, slot) {
				return
			}
			e.Result, _ = json.Marshal(hexutil.Encode(word[:]))
		}
	}
	venueHash := common.HexToHash("0x56570de287d73cd1cb6092bb8fdee6173974955fdef345ae579ee9f475ea7432")
	// A scheduled venue: pendingHookSet answers it as (venueHash, venueAt) in words 8 and 9, and the layout check
	// binds both to slots +32 and +33.
	venue := func(at uint64) func(tp *rpcTape) {
		return func(tp *rpcTape) {
			tp.mutate = chainMutations(tp.mutate,
				aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "pendingHookSet"), func(r *tapeResult) {
					setWord(8, venueHash.Bytes())(r)
					setWord(9, u64(at))(r)
				}),
				poolWord(slotPendingVenue, venueHash),
				poolWord(slotPendingVenue+1, common.BigToHash(new(big.Int).SetUint64(at))))
		}
	}
	// A scheduled loan asset: no view reports it, so only its two storage words move.
	loanHash := common.HexToHash("0x340f5a8a3b67a7d00cf1eaf37678ccbcee89c036e107a15dc3eeaae4dd97cff8")
	loanAsset := func(hash common.Hash, at uint64) func(tp *rpcTape) {
		return func(tp *rpcTape) {
			tp.mutate = chainMutations(tp.mutate, poolWord(slotPendingLoan, hash),
				poolWord(slotPendingLoan+1, common.BigToHash(new(big.Int).SetUint64(at))))
		}
	}
	implementation := func(at uint64) []func(*rpcTape) {
		return []func(*rpcTape){
			func(tp *rpcTape) {
				tp.mutate = chainMutations(
					aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "pendingImplementation"),
						setWord(0, other.Bytes())),
					aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "implementationExecutableAt"),
						setWord(0, u64(at))),
					factoryWord(other, at))
			}}
	}
	// pendingHookSet answers (HookSet invariantHook .. loanSwapHook, executableAt, venueHash, venueAt); the layout
	// check binds executableAt to its FLAMMStore word (bits 192..240), which moves with it.
	spreadWord := strings.ToLower(flammSlot(slotSpreadWord).Hex())
	hookSet := func(at uint64) func(tp *rpcTape) {
		return func(tp *rpcTape) {
			tp.mutate = chainMutations(tp.mutate,
				aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "pendingHookSet"), func(r *tapeResult) {
					setWord(0, other.Bytes())(r)
					setWord(7, u64(at))(r)
				}),
				func(method string, params json.RawMessage, e *tapeEntry) {
					if method != "eth_getStorageAt" || !strings.Contains(strings.ToLower(string(params)), spreadWord) {
						return
					}
					var hexWord string
					require.NoError(t, json.Unmarshal(e.Result, &hexWord))
					var w, mask, v uint256.Int
					w.SetBytes(common.FromHex(hexWord))
					mask.Lsh(uint256.NewInt(1<<48-1), 192)
					w.And(&w, mask.Not(&mask))
					w.Or(&w, v.Lsh(uint256.NewInt(at), 192))
					e.Result, _ = json.Marshal(hexutil.Encode(w.PaddedBytes(32)))
				})
		}
	}
	for _, c := range []struct {
		name  string
		setup []func(*rpcTape)
		want  uint64
	}{
		{"implementation", implementation(ts + 7_200), ts + 7_200},
		{"hook set", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil }, hookSet(ts + 5_000)}, ts + 5_000},
		{"both, the earlier", append(implementation(ts+9_000), hookSet(ts+4_000)), ts + 4_000},
		{"executable at zero", implementation(0), 1},
		// Either word of a pair marks a change pending: an answer that zeroes only the pending address, or only the
		// executableAt, still stamps it (tracker_reads.go scheduledChangeAt).
		{"implementation address only", []func(*rpcTape){func(tp *rpcTape) {
			tp.mutate = chainMutations(
				aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "pendingImplementation"),
					setWord(0, other.Bytes())),
				factoryWord(other, 0))
		}}, 1},
		{"implementation executableAt only", []func(*rpcTape){func(tp *rpcTape) {
			tp.mutate = chainMutations(
				aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "implementationExecutableAt"),
					setWord(0, u64(ts+6_000))),
				factoryWord(common.Address{}, ts+6_000))
		}}, ts + 6_000},
		{"hook set pending hook only", []func(*rpcTape){func(tp *rpcTape) {
			tp.mutate = aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "pendingHookSet"),
				setWord(0, other.Bytes()))
		}}, 1},
		// The pool's other two delayed admissions (FLAMMOpsLib.sol:384-392, :523-529). A venue adds a funding leg
		// to every settlement; a second loan asset puts the pool outside the quoted envelope altogether.
		{"venue admission", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil }, venue(ts + 8_000)}, ts + 8_000},
		{"venue hash only", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil }, venue(0)}, 1},
		{"loan asset admission", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil },
			loanAsset(loanHash, ts+2_500)}, ts + 2_500},
		{"loan asset executableAt only", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil },
			loanAsset(common.Hash{}, ts+2_600)}, ts + 2_600},
		{"venue and loan, the earlier", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil },
			venue(ts + 8_000), loanAsset(loanHash, ts+2_500)}, ts + 2_500},
		{"hook set executableAt only", []func(*rpcTape){func(tp *rpcTape) { tp.mutate = nil },
			func(tp *rpcTape) {
				tp.mutate = chainMutations(tp.mutate,
					aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "pendingHookSet"), setWord(7, u64(ts+3_000))),
					func(method string, params json.RawMessage, e *tapeEntry) {
						if method != "eth_getStorageAt" || !strings.Contains(strings.ToLower(string(params)), spreadWord) {
							return
						}
						var hexWord string
						require.NoError(t, json.Unmarshal(e.Result, &hexWord))
						var w, mask, v uint256.Int
						w.SetBytes(common.FromHex(hexWord))
						mask.Lsh(uint256.NewInt(1<<48-1), 192)
						w.And(&w, mask.Not(&mask))
						w.Or(&w, v.Lsh(uint256.NewInt(ts+3_000), 192))
						e.Result, _ = json.Marshal(hexutil.Encode(w.PaddedBytes(32)))
					})
			}}, ts + 3_000},
	} {
		tp.mutate = nil
		for _, s := range c.setup {
			s(tp)
		}
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.True(t, extra.Attested, "%s: %q %q", c.name, extra.AttestFailure, extra.ProfileDrift)
		require.Equal(t, c.want, extra.ScheduledChangeAt, c.name)
		sim, err := NewPoolSimulator(tracked)
		require.NoError(t, err, c.name)
		for _, clock := range []uint64{ts, c.want - min(c.want, scheduledChangeLeadSec) - 1,
			c.want - min(c.want, scheduledChangeLeadSec)} {
			if clock < ts {
				continue
			}
			sim.nowFn = func() uint64 { return clock }
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(15_000)},
				TokenOut:      sim.Info.Tokens[1]})
			if clock+scheduledChangeLeadSec >= c.want {
				require.ErrorIs(t, err, ErrScheduledChange, "%s at %d", c.name, clock)
			} else {
				require.NotErrorIs(t, err, ErrScheduledChange, "%s at %d", c.name, clock)
			}
		}
	}

	// The removal direction: with an upgrade really scheduled, an answer that drops it from one side only -- the
	// views alone, or the storage word alone -- no longer publishes an attested entity with the guard off. It is
	// the layout check that catches it, and it is the only thing that does.
	for _, c := range []struct {
		name  string
		setup func(tp *rpcTape)
	}{
		{"views zero a scheduled upgrade", func(tp *rpcTape) { tp.mutate = factoryWord(other, ts+7_200) }},
		{"storage zeroes a scheduled upgrade", func(tp *rpcTape) {
			tp.mutate = chainMutations(
				aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "pendingImplementation"),
					setWord(0, other.Bytes())),
				aggregateCallMutation(t, callTo(t, prof.Factory, &factoryABI, "implementationExecutableAt"),
					setWord(0, u64(ts+7_200))))
		}},
		{"views zero a scheduled venue", func(tp *rpcTape) {
			tp.mutate = chainMutations(poolWord(slotPendingVenue, venueHash),
				poolWord(slotPendingVenue+1, common.BigToHash(new(big.Int).SetUint64(ts+8_000))))
		}},
		{"storage zeroes a scheduled venue", func(tp *rpcTape) {
			tp.mutate = aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "pendingHookSet"),
				func(r *tapeResult) {
					setWord(8, venueHash.Bytes())(r)
					setWord(9, u64(ts+8_000))(r)
				})
		}},
	} {
		tp.mutate = nil
		c.setup(tp)
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.Equal(t, "storage layout", extra.ProfileDrift, c.name)
		_, err = NewPoolSimulator(tracked)
		require.ErrorIs(t, err, ErrProfileDrift, c.name)
	}
	tp.mutate = nil
}

// selectorOf is the four-byte selector the deployed contracts revert want with.
func selectorOf(t *testing.T, want error) []byte {
	t.Helper()
	for sel, e := range revertSelectors {
		if errors.Is(e, want) {
			return append([]byte(nil), sel[:]...)
		}
	}
	t.Fatalf("no selector for %v", want)
	return nil
}

// chainMutations applies each non-nil mutation in order.
func chainMutations(ms ...func(string, json.RawMessage, *tapeEntry)) func(string, json.RawMessage, *tapeEntry) {
	return func(method string, params json.RawMessage, e *tapeEntry) {
		for _, m := range ms {
			if m != nil {
				m(method, params, e)
			}
		}
	}
}

type tapeResult struct {
	Success    bool
	ReturnData []byte
}

// aggregateMutation edits the results of every recorded tryBlockAndAggregate answer with at least minCalls calls.
func aggregateMutation(t *testing.T, minCalls int, edit func(results []tapeResult)) func(string, json.RawMessage,
	*tapeEntry) {
	return func(method string, _ json.RawMessage, e *tapeEntry) {
		if method != "eth_call" || e.Result == nil {
			return
		}
		var hexResult string
		require.NoError(t, json.Unmarshal(e.Result, &hexResult))
		out := multicallABI.Methods["tryBlockAndAggregate"].Outputs
		vals, err := out.Unpack(common.FromHex(hexResult))
		if err != nil {
			return
		}
		results := *abiConvert[[]tapeResult](t, vals[2])
		if len(results) < minCalls {
			return
		}
		edit(results)
		packed, err := out.Pack(vals[0], vals[1], results)
		require.NoError(t, err)
		e.Result, _ = json.Marshal(hexutil.Encode(packed))
	}
}

func abiConvert[T any](t *testing.T, v any) *T {
	t.Helper()
	out, err := tupleOf[T](v)
	require.NoError(t, err)
	return &out
}

// TestStorageOverrides: storage words honor state overrides the way eth_call does, without asking the node.
func TestStorageOverrides(t *testing.T) {
	t.Parallel()
	a := common.HexToAddress("0x01")
	b := common.HexToAddress("0x02")
	slot := common.HexToHash("0x05")
	val := common.HexToHash("0x07")
	rpc := &mcRPC{overrides: map[common.Address]gethclient.OverrideAccount{
		a: {State: map[common.Hash]common.Hash{}},
		b: {StateDiff: map[common.Hash]common.Hash{slot: val}},
	}}
	words, err := rpc.storageAt(context.Background(), big.NewInt(1), []storageRead{{a, slot}, {b, slot}})
	require.NoError(t, err)
	require.Equal(t, common.Hash{}, words[0], "a full state override clears unlisted slots")
	require.Equal(t, val, words[1])
}

// setResultWord overwrites word i of an ABI answer.
func setResultWord(i int, v []byte) func(r *tapeResult) {
	return func(r *tapeResult) {
		r.ReturnData = append([]byte(nil), r.ReturnData...)
		copy(r.ReturnData[32*i:32*(i+1)], common.LeftPadBytes(v, 32))
	}
}

// forwardRound restricts a mutation to one of a refresh's forward rounds: the eth_calls it sends with a block
// timestamp override (the market oracle's end-of-window answer, and the deadline round at each clock the port
// believes a deadline flips at), told apart by a selector their aggregate carries.
func forwardRound(selector []byte, m func(string, json.RawMessage,
	*tapeEntry)) func(string, json.RawMessage, *tapeEntry) {
	return func(method string, params json.RawMessage, e *tapeEntry) {
		var args []json.RawMessage
		if err := json.Unmarshal(params, &args); err != nil || len(args) < 4 ||
			!strings.Contains(string(args[3]), `"time"`) {
			return
		}
		var msg struct {
			Input hexutil.Bytes `json:"input"`
			Data  hexutil.Bytes `json:"data"`
		}
		if json.Unmarshal(args[0], &msg) != nil {
			return
		}
		in := msg.Input
		if len(in) == 0 {
			in = msg.Data
		}
		if !bytes.Contains(in, selector) {
			return
		}
		m(method, params, e)
	}
}

// aheadMutation edits the answers of the forward round that reads each venue's market oracle later in the snapshot
// window.
func aheadMutation(t *testing.T, edit func(results []tapeResult)) func(string, json.RawMessage, *tapeEntry) {
	return forwardRound(accountABI.Methods["oraclePrice"].ID, aggregateMutation(t, 1, edit))
}

// clockMutation edits one call of the deadline round at every clock it is sent at.
func clockMutation(t *testing.T, match func(common.Address, []byte) bool,
	edit func(r *tapeResult)) func(string, json.RawMessage, *tapeEntry) {
	return forwardRound(priceFeedABI.Methods["peekCross"].ID, aggregateCallMutation(t, match, edit))
}

// TestTrackerDependencies: the refresh publishes the non-pool addresses whose logs should also trigger one
// (pool.IPoolTrackerWithDependencies). They are the aggregators the price feed proxies resolve to -- not the
// proxies themselves, which emit nothing -- plus the two hooks that move a quote with an event of their own
// (EverlongHook's TuningChanged, the spread hook's post) and the Router (GlobalPauseSet, venue configuration).
// Morpho Blue and the IRM are left out: every borrow anywhere on Base would trigger a refresh.
func TestTrackerDependencies(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	ctx := context.Background()
	tracked, err := tracker.GetNewPoolStateAtBlock(ctx, listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(listed.StaticExtra), &se))
	prof := &c104

	deps, stored, err := tracker.GetDependencies(ctx, tracked)
	require.NoError(t, err)
	require.False(t, stored, "a set pool-service has not been given yet")
	require.Equal(t, []string{
		"0x51ce3091cf646587e02cad83b580992f8723e718", // cbBTC / USD, behind proxy 0x07da0e54
		"0x68be4c50235205ede361ac8244b1ee221cdda5e2", // USDC / USD, behind proxy 0x7e860098
		"0x606c6ecbd272e2174f6710b5974f23fe9899602e", // sequencer uptime, behind proxy 0xbcf85224
		"0xe5ec87a39445b8d5b751b116802a53c5ae7e9df1", // BTC / USD, the venue market oracle's only feed
		lowerHex(prof.Hook), lowerHex(prof.SpreadHook), lowerHex(prof.Router),
	}, deps)
	for _, a := range []common.Address{se.Aggregators[0], se.Aggregators[1], se.SequencerFeed,
		entityVenues(t, tracked)[0].Oracle, entityVenues(t, tracked)[0].OracleFeeds[0], prof.Morpho,
		adaptiveCurveIrm, prof.Pool, prof.LeverageHook, prof.Factory} {
		require.NotContains(t, deps, lowerHex(a), a.Hex())
	}

	// pool-service's own flag round trips through the entity, and a refresh that resolves the same set keeps it.
	require.NoError(t, tracker.SetDependenciesStored(&tracked, true))
	_, stored, err = tracker.GetDependencies(ctx, tracked)
	require.NoError(t, err)
	require.True(t, stored)
	again, err := tracker.GetNewPoolStateAtBlock(ctx, tracked, big.NewInt(trackerBlock))
	require.NoError(t, err)
	_, stored, err = tracker.GetDependencies(ctx, again)
	require.NoError(t, err)
	require.True(t, stored, "an unchanged set stays registered")

	// A Chainlink phase rotation moves proxy.aggregator() with no log at the proxy, so the set is resolved every
	// refresh and a change clears the flag for pool-service to register it again.
	rotated := common.HexToAddress("0x00000000000000000000000000000000000000ad")
	tp.mutate = aggregateCallMutation(t, callTo(t, se.Aggregators[0], &aggregatorABI, "aggregator"),
		setResultWord(0, rotated.Bytes()))
	moved, err := tracker.GetNewPoolStateAtBlock(ctx, tracked, big.NewInt(trackerBlock))
	require.NoError(t, err)
	deps, stored, err = tracker.GetDependencies(ctx, moved)
	require.NoError(t, err)
	require.False(t, stored, "a rotated aggregator has to be registered again")
	require.Equal(t, lowerHex(rotated), deps[0])

	// A proxy that does not answer `aggregator()` resolves to nothing and simply contributes no dependency; the
	// refresh is otherwise unaffected.
	tp.mutate = aggregateCallMutation(t, callTo(t, se.Aggregators[0], &aggregatorABI, "aggregator"),
		func(r *tapeResult) { r.Success, r.ReturnData = false, nil })
	unresolved, err := tracker.GetNewPoolStateAtBlock(ctx, tracked, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var unresolvedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(unresolved.Extra), &unresolvedExtra))
	require.True(t, unresolvedExtra.Attested, "%q %q", unresolvedExtra.AttestFailure, unresolvedExtra.ProfileDrift)
	deps, _, err = tracker.GetDependencies(ctx, unresolved)
	require.NoError(t, err)
	require.Len(t, deps, 6)
	require.NotContains(t, deps, lowerHex(se.Aggregators[0]))

	// A resolve round the node does not answer at all keeps the previous set rather than failing an otherwise good
	// refresh: the set triggers refreshes, it prices nothing.
	resolveSelector := hexutil.Encode(aggregatorABI.Methods["aggregator"].ID)[2:]
	tp.mutate = func(method string, params json.RawMessage, e *tapeEntry) {
		if method == "eth_call" && strings.Contains(string(params), resolveSelector) {
			e.Result, e.Error = nil, &tapeError{Code: -32000, Message: "unavailable"}
		}
	}
	kept, err := tracker.GetNewPoolStateAtBlock(ctx, tracked, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var keptExtra Extra
	require.NoError(t, json.Unmarshal([]byte(kept.Extra), &keptExtra))
	require.True(t, keptExtra.Attested, "%q %q", keptExtra.AttestFailure, keptExtra.ProfileDrift)
	deps, stored, err = tracker.GetDependencies(ctx, kept)
	require.NoError(t, err)
	require.True(t, stored, "the set pool-service already holds stands")
	require.Len(t, deps, 7)
}

// TestTrackerOracleWindow: the refresh reads what each venue's Morpho market oracle answers at the far end of the
// snapshot window, at the refresh block with only the block timestamp overridden, and publishes it. That answer is
// the one input of a snapshot the clock alone moves: the feed behind the c104 market oracle is a Chainlink SVR
// DualAggregator, which withholds each primary round for a fixed delay.
func TestTrackerOracleWindow(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	ctx := context.Background()
	tracked, err := tracker.GetNewPoolStateAtBlock(ctx, listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.Len(t, extra.OracleAhead, 1)
	snapshot := extra.Reads.Router.Venues[0].OraclePrice
	require.True(t, extra.OracleAhead[0].Ok)
	require.True(t, extra.OracleAhead[0].Price.Eq(&snapshot), "no round is withheld at the recorded block")

	// The round is sent at the snapshot's own block, with block.timestamp moved to the end of the window.
	at := hexutil.EncodeUint64(extra.Reads.Timestamp + defaultMaxSnapshotAgeSec)
	sent := false
	for k := range tp.used {
		sent = sent || strings.Contains(k, `{"time":"`+at+`"}`) && strings.Contains(k,
			hexutil.EncodeUint64(trackerBlock))
	}
	require.True(t, sent, "the forward round is pinned to the refresh block at the end of the window")

	// A withheld round the window reveals moves the answer, and the simulator refuses every fill the move changes
	// (TestSimulatorOracleWindow); the refresh itself still attests, because nothing at its own clock moved.
	var moved uint256.Int
	moved.Div(&snapshot, uint256.NewInt(2))
	tp.mutate = aheadMutation(t, func(results []tapeResult) {
		setResultWord(1, moved.Bytes())(&results[0])
	})
	shifted, err := tracker.GetNewPoolStateAtBlock(ctx, listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(shifted.Extra), &extra))
	require.True(t, extra.Attested, "%q %q", extra.AttestFailure, extra.ProfileDrift)
	require.True(t, extra.OracleAhead[0].Price.Eq(&moved))
	sim, err := NewPoolSimulator(shifted)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	_, err = sim.CalcAmountOut(amountIn(sim, true, 15_000))
	require.ErrorIs(t, err, ErrOracleDrift)

	// A policy that declares no snapshot window has no clock to read at: the round is not sent and the simulator
	// quotes the snapshot's own answer, as it does with every other margin the configuration turns off.
	tp.mutate = nil
	unbounded, err := NewPoolTracker(parityConfig(Policy{}), tp.client()).GetNewPoolStateAtBlock(ctx, listed,
		big.NewInt(trackerBlock))
	require.NoError(t, err)
	var open Extra
	require.NoError(t, json.Unmarshal([]byte(unbounded.Extra), &open))
	require.True(t, open.Attested, "%q %q", open.AttestFailure, open.ProfileDrift)
	require.Empty(t, open.OracleAhead)
	sim, err = NewPoolSimulator(unbounded)
	require.NoError(t, err)
	require.Nil(t, sim.oracleShifted())
}

// TestTrackerForwardRoundProvesTheClock: every round sent with a block timestamp override reads the clock it ran at
// back from Multicall3 and fails the refresh in transport unless it is the one it asked for. A node that rejects
// `blockOverrides` fails the eth_call on its own; one that accepts the parameter and ignores it would otherwise
// answer the snapshot's own state, which is a silent no-op -- the market oracle would "not move", the simulator
// would find nothing to refuse, and the refresh would attest with a window guard that can never fire.
func TestTrackerForwardRoundProvesTheClock(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	ctx := context.Background()
	tracked, err := tracker.GetNewPoolStateAtBlock(ctx, listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "%q %q", extra.AttestFailure, extra.ProfileDrift)

	// The proof rides on the aggregate's own target, so it is asked for at the overridden clock and nowhere else.
	proof, err := multicallABI.Pack("getCurrentBlockTimestamp")
	require.NoError(t, err)
	at := hexutil.EncodeUint64(extra.Reads.Timestamp + defaultMaxSnapshotAgeSec)
	proved, plain := 0, 0
	tp.mu.Lock()
	for k := range tp.used {
		if !strings.Contains(k, hexutil.Encode(proof)[2:]) {
			continue
		}
		if strings.Contains(k, `{"time":"`+at+`"}`) {
			proved++
		} else {
			plain++
		}
	}
	tp.mu.Unlock()
	require.Equal(t, 2, proved, "the oracle window and the deadline round each carry the proof")
	require.Equal(t, 1, plain, "the view round reads the snapshot's own clock and sends no override")

	// A node that answers the override with the block's own timestamp: transport, so pool-service keeps the entity
	// it has rather than being handed one that claims to have looked ahead.
	tp.mutate = forwardRound(multicallABI.Methods["getCurrentBlockTimestamp"].ID, aggregateMutation(t, 1,
		func(results []tapeResult) {
			setResultWord(0, new(big.Int).SetUint64(extra.Reads.Timestamp).Bytes())(&results[len(results)-1])
		}))
	ignored, err := tracker.GetNewPoolStateAtBlock(ctx, listed, big.NewInt(trackerBlock))
	require.ErrorIs(t, err, errRPC)
	require.False(t, chainAnswered(err))
	require.Contains(t, err.Error(), "does not apply blockOverrides")
	require.Equal(t, listed.Extra, ignored.Extra, "the entity is left as it was")
	tp.mutate = nil
}

// TestTrackerProbeEmptyRevert: Multicall3 reports a subcall that ran out of gas exactly as it reports a revert with
// no data, which is the shape OpenZeppelin's Math.mulDiv reverts with, so an empty revert is never read as the
// port's own errMulDivOverflow until the same call, run on its own with an explicit gas limit, reverts too.
func TestTrackerProbeEmptyRevert(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	prof := &c104
	data, err := flammABI.Pack("previewSwap", true, big.NewInt(1_000))
	require.NoError(t, err)
	empty := aggregateCallMutation(t, callTo(t, prof.Pool, &flammABI, "previewSwap", true, big.NewInt(1_000)),
		func(r *tapeResult) { r.Success, r.ReturnData = false, nil })
	// alone answers the guard's own eth_call of the probe, the only one addressed to the pool itself.
	alone := func(answer func(e *tapeEntry)) func(string, json.RawMessage, *tapeEntry) {
		return func(method string, params json.RawMessage, e *tapeEntry) {
			if method == "eth_call" && strings.Contains(string(params), `"to":"`+lowerHex(prof.Pool)+`"`) {
				answer(e)
			}
		}
	}
	for _, c := range []struct {
		name   string
		answer func(e *tapeEntry)
		prefix string // the failure the refresh publishes; empty: the refresh returns the error instead
	}{
		{"reverted", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: 3, Message: "execution reverted", Data: json.RawMessage(`"0x"`)}
		}, "previewSwap(true,1000): chain reverted"},
		{"answered", func(e *tapeEntry) {
			e.Result, e.Error = json.RawMessage(`"0x`+strings.Repeat("00", 96)+`"`), nil
		}, "previewSwap(true,1000): empty revert unconfirmed"},
		// Base answers an out-of-gas eth_call with -32003 (multicall.go execErrorCodes); geth's generic -32000 is
		// the same class and is read the same way.
		{"out of gas", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: -32003, Message: "out of gas: gas required exceeds: 30000"}
		}, "previewSwap(true,1000): empty revert unconfirmed"},
		{"out of gas, generic code", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: -32000, Message: "out of gas"}
		}, "previewSwap(true,1000): empty revert unconfirmed"},
		// The aggregate answered empty returndata; a standalone revert that carries data is a different answer, so
		// it is the chain saying the port's refusal is wrong rather than a confirmation of it.
		{"reverted with data", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: 3, Message: "execution reverted",
				Data: json.RawMessage(`"0x7939f4246c6a7ba2"`)}
		}, "previewSwap(true,1000): empty revert unconfirmed"},
		// A confirmation the node never ran is transport, exactly as the probe aggregate's own failure is: the
		// refresh returns it and pool-service keeps the last entity, rather than publishing a pool with no reserves
		// because one eth_call was rate limited.
		{"rate limited", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: -32016, Message: "over rate limit"}
		}, ""},
		{"node internal", func(e *tapeEntry) {
			e.Result, e.Error = nil, &tapeError{Code: -32603, Message: "internal error"}
		}, ""},
	} {
		tp.mutate = chainMutations(empty, alone(c.answer))
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		if c.prefix == "" {
			require.ErrorIs(t, err, errRPC, c.name)
			require.False(t, chainAnswered(err), c.name)
			require.Equal(t, listed.Extra, tracked.Extra, "%s: the entity is left as it was", c.name)
			continue
		}
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.False(t, extra.Attested, c.name)
		require.True(t, strings.HasPrefix(extra.AttestFailure, c.prefix), "%s: %q", c.name, extra.AttestFailure)
	}

	// The probe aggregate names its own gas, so its cost does not depend on what a node applies to a call that
	// names none (attest.go probeAggregateGas).
	gassed := false
	for k := range tp.used {
		gassed = gassed || strings.Contains(k, `"gas":"`+hexutil.EncodeUint64(probeAggregateGas)+`"`) &&
			strings.Contains(k, hexutil.Encode(data)[2:])
	}
	require.True(t, gassed, "the probe aggregate is sent with an explicit gas limit")
}

// TestRevertedClassifies: a node answers a revert with JSON-RPC code 3 and carries the revert data where there is
// any; anything else is not the contract's own answer (multicall.go reverted).
func TestRevertedClassifies(t *testing.T) {
	t.Parallel()
	require.False(t, reverted(nil))
	require.False(t, reverted(errReadFailed))
	require.True(t, reverted(&jsonRPCError{code: 3, message: "execution reverted"}))
	require.True(t, reverted(&jsonRPCError{code: -32000, message: "execution reverted", data: "0x1234"}))
	require.False(t, reverted(&jsonRPCError{code: -32000, message: "out of gas"}))
	require.False(t, reverted(fmt.Errorf("wrapped: %w", &jsonRPCError{code: -32000, message: "header not found"})))
	require.True(t, reverted(fmt.Errorf("wrapped: %w", &jsonRPCError{code: 3, message: "execution reverted"})))

	// executed separates a call the node ran and could not finish from a request it never ran (multicall.go).
	for _, code := range execErrorCodes {
		require.True(t, executed(&jsonRPCError{code: code, message: "out of gas"}), code)
		require.True(t, executed(fmt.Errorf("wrapped: %w", &jsonRPCError{code: code, message: "out of gas"})), code)
	}
	require.Contains(t, execErrorCodes, -32003, "the code Base answers an out-of-gas eth_call with")

	// emptyRevert is the half of reverted that confirms an aggregate's empty returndata: a revert carrying data is
	// a different answer than the one being confirmed (multicall.go confirmRevert).
	require.True(t, emptyRevert(&jsonRPCError{code: 3, message: "execution reverted"}))
	require.True(t, emptyRevert(&jsonRPCError{code: 3, message: "execution reverted", data: "0x"}))
	require.True(t, emptyRevert(fmt.Errorf("wrapped: %w", &jsonRPCError{code: 3, message: "execution reverted"})))
	require.False(t, emptyRevert(&jsonRPCError{code: 3, message: "execution reverted", data: "0x1234"}))
	require.False(t, emptyRevert(&jsonRPCError{code: -32000, message: "execution reverted", data: "0x1234"}))
	require.False(t, emptyRevert(&jsonRPCError{code: -32003, message: "out of gas"}))
	require.False(t, emptyRevert(errReadFailed))
	require.False(t, emptyRevert(nil))
	require.False(t, executed(&jsonRPCError{code: -32016, message: "over rate limit"}))
	require.False(t, executed(&jsonRPCError{code: -32603, message: "internal error"}))
	require.False(t, executed(context.DeadlineExceeded))
	require.False(t, executed(errors.New("connection reset")))
	require.False(t, executed(nil))
}

// jsonRPCError is a node's error object as the rpc package hands it over (rpc.Error / rpc.DataError).
type jsonRPCError struct {
	code    int
	message string
	data    any
}

func (e *jsonRPCError) Error() string  { return e.message }
func (e *jsonRPCError) ErrorCode() int { return e.code }
func (e *jsonRPCError) ErrorData() any { return e.data }

// TestTrackerReserves: what the entity's reserves mean and when it carries any.
//
// Reserves[1] is the ceiling of a sell that brings no collateral of its own -- liquid plus the Router's funding
// against the physical poolAsset, capped by the gate room and the notional cap, which is FLAMMSwapLib's own
// ceiling on its first funding pass (FLAMMSwapLib.sol:150-166) at collateralIn = 0. The chain counts the incoming
// poolAsset as collateral (:164) and re-plans on it, so the ceiling only rises with the input and the published
// number is a lower bound, not the sell capacity. pool-service turns reserves into TVL and a liquidity score, so
// an entity no quote will be served from carries none at all.
func TestTrackerReserves(t *testing.T) {
	t.Parallel()
	stored := loadTracked(t)
	require.Equal(t, entity.PoolReserves{"219423", "94577910"}, stored.Reserves)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(stored.Extra), &extra))
	st, err := extra.Reads.baseState()
	require.NoError(t, err)
	published := sellCapacity(st, extra.Reads.Timestamp)
	require.Equal(t, stored.Reserves[1], published.Dec(), "the published number is what the refresh computes")

	// The pool's own ceiling, with every margin off: a sell 35% above the published number is accepted whole.
	extra.Policy = Policy{}
	raw, err := json.Marshal(&extra)
	require.NoError(t, err)
	free := stored
	free.Extra = string(raw)
	sim, err := NewPoolSimulator(free)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	big177 := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(177_385)},
		TokenOut:      sim.Info.Tokens[1]}
	paid, err := sim.CalcAmountOut(big177)
	require.NoError(t, err)
	require.Equal(t, "128065084", paid.TokenAmountOut.Amount.String())
	require.Zero(t, paid.RemainingTokenAmountIn.Amount.Sign(), "the whole input is used")
	require.Positive(t, paid.TokenAmountOut.Amount.Cmp(published.ToBig()),
		"an accepted sell pays above the published lower bound")

	// A fill moves both reserves: the capacity is recomputed from the adopted state, not carried through.
	small := pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: sim.Info.Tokens[0], Amount: big.NewInt(50_000)},
		TokenOut:      sim.Info.Tokens[1]}
	res, err := sim.CalcAmountOut(small)
	require.NoError(t, err)
	after := sim.CloneState().(*PoolSimulator)
	after.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
	require.Equal(t, "269423", after.Info.Reserves[0].String())
	require.Equal(t, "78629652", after.Info.Reserves[1].String(), "recomputed from the state the fill left")
	recomputed := sellCapacity(after.state, after.now())
	require.Equal(t, recomputed.Dec(), after.Info.Reserves[1].String())

	// The per-swap notional cap is part of the ceiling (FLAMMSwapLib.sol:158), so it bounds the capacity too: on
	// the capped scenario the room is 210,152,119 but no single sell can be paid past maxSwapNotional.
	capped := gridReads(t, "51302915", "capped")
	cappedState, err := capped.baseState()
	require.NoError(t, err)
	notional := cappedState.Pool.Loans[0].MaxSwapNotional
	require.Equal(t, "5000000", notional.Dec())
	cappedCapacity := sellCapacity(cappedState, capped.Timestamp)
	require.Equal(t, notional.Dec(), cappedCapacity.Dec())
	cappedSim := simFor(t, capped, Policy{})
	sold, err := cappedSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: cappedSim.Info.Tokens[0], Amount: big.NewInt(177_385)},
		TokenOut:      cappedSim.Info.Tokens[1]})
	require.NoError(t, err)
	require.LessOrEqual(t, sold.TokenAmountOut.Amount.Cmp(notional.ToBig()), 0,
		"a sell on a notional-capped state cannot be paid past the cap")

	// An entity the simulator refuses carries no reserves: a probe the port does not reproduce fails the refresh,
	// and the reserves it would have published are dropped with the attestation.
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	tp.mutate = aggregateCallMutation(t, callTo(t, common.HexToAddress(listed.Address), &flammABI, "previewSwap",
		true, big.NewInt(1_000)), addWord(1, 1))
	broken, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var brokenExtra Extra
	require.NoError(t, json.Unmarshal([]byte(broken.Extra), &brokenExtra))
	require.False(t, brokenExtra.Attested)
	require.NotEmpty(t, brokenExtra.AttestFailure)
	require.Equal(t, entity.PoolReserves{"0", "0"}, broken.Reserves)
}

// TestTrackerDeadlines: the words whose only effect is at a later clock -- the spread hook's lastSetTs and
// maxSpreadAge, each aggregator round's updatedAt, the sequencer round's startedAt -- are bound by the forward
// round the refresh sends at the clocks the port believes each predicate flips at, inside the window the policy
// admits quoting in (attest.go deadlineClocks / attestDeadlines). At the recorded block the spread post lapsed
// long ago, the sequencer came up 6.9M seconds earlier and both feed rounds outlive the 600 s window, so the
// round is a single aggregate at the window's far end.
func TestTrackerDeadlines(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	prof := &c104
	stored := loadTracked(t)
	var storedExtra Extra
	require.NoError(t, json.Unmarshal([]byte(stored.Extra), &storedExtra))
	ts := storedExtra.Reads.Timestamp
	end := ts + defaultMaxSnapshotAgeSec

	// The round is sent at the refresh block with the window's far end as the block timestamp, and nowhere else.
	tracker := NewPoolTracker(baseConfig(), tp.client())
	tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	clockKeys := map[string]bool{}
	peekCross := hexutil.Encode(priceFeedABI.Methods["peekCross"].ID)[2:]
	pinned := false
	for k := range tp.used {
		i := strings.Index(k, `{"time":"`)
		if i < 0 {
			continue
		}
		at := k[i+9 : i+9+strings.Index(k[i+9:], `"`)]
		clockKeys[at] = true
		if at == hexutil.EncodeUint64(end) && strings.Contains(k, peekCross) {
			// the round runs at the refresh block, not at the clock it names
			require.Contains(t, k, `"`+hexutil.EncodeUint64(trackerBlock)+`"`)
			pinned = true
		}
	}
	require.True(t, pinned, "one deadline round, at the refresh block and the window's far end: %v", clockKeys)
	storedState, err := storedExtra.Reads.baseState()
	require.NoError(t, err)
	require.Equal(t, []uint64{end}, deadlineClocks(storedState, ts, defaultMaxSnapshotAgeSec))

	// The chain's own deadline: peekCross answers through the cbBTC round's updatedAt + heartbeat and not one
	// second longer, which is what the forward round at a flip second compares against.
	cb := &storedExtra.Reads.Feed.Tokens[0]
	require.Equal(t, uint64(3_600), cb.Heartbeat.Uint64())
	require.Equal(t, ts+3_066, cb.Round.UpdatedAt.Uint64()+cb.Heartbeat.Uint64())

	spreadCall := func(method string) func(common.Address, []byte) bool {
		return callTo(t, prof.SpreadHook, &spreadHookABI, method)
	}
	var se StaticExtra
	require.NoError(t, json.Unmarshal([]byte(stored.StaticExtra), &se))
	for _, c := range []struct {
		name   string
		mutate func(string, json.RawMessage, *tapeEntry)
		prefix string
	}{
		// A keeper post the port believes is live through the window, where the chain's has lapsed: the fail-open
		// direction (a lastSetTs or a maxSpreadAge too large moves no answer at the snapshot clock).
		{"lastSetTs is live through the window",
			aggregateCallMutation(t, spreadCall("lastSetTs"), setResultWord(0, new(big.Int).SetUint64(ts).Bytes())),
			fmt.Sprintf("spreadPpm at %d: chain (false 0) port (true 17500)", end)},
		{"maxSpreadAge never expires",
			aggregateCallMutation(t, spreadCall("maxSpreadAge"), setResultWord(0, big.NewInt(1<<31).Bytes())),
			fmt.Sprintf("spreadPpm at %d: chain (false 0) port (true 17500)", end)},
		// The other direction on the feed: a round the port believes expires inside the window, where the chain's
		// does not.
		{"the pool asset round expires inside the window",
			aggregateCallMutation(t, callTo(t, se.Aggregators[0], &aggregatorABI, "latestRoundData"),
				func(r *tapeResult) {
					var w uint256.Int
					w.SetBytes(r.ReturnData[3*32 : 4*32])
					w.SubUint64(&w, 3_000)
					r.ReturnData = append([]byte(nil), r.ReturnData...)
					copy(r.ReturnData[3*32:4*32], w.PaddedBytes(32))
				}),
			fmt.Sprintf("peekCross at %d:", end)},
		// The chain's own answer at the far end, with the snapshot's round left alone: a cross that stops answering
		// inside the window is not a state the port may keep quoting through.
		{"the chain's cross stops answering inside the window",
			clockMutation(t, callTo(t, prof.PriceFeed, &priceFeedABI, "peekCross"), func(r *tapeResult) {
				r.ReturnData = append([]byte(nil), r.ReturnData...)
				copy(r.ReturnData[0:32], make([]byte, 32))
			}),
			fmt.Sprintf("peekCross at %d:", end)},
		// Each token's own quote at the far end, which is what separates the two rounds' deadlines when the
		// cross is decided by the other one.
		{"the pool asset's quote stops answering inside the window",
			clockMutation(t, callTo(t, prof.PriceFeed, &priceFeedABI, "peekUsd", prof.PoolAsset),
				setResultWord(0, nil)), fmt.Sprintf("peekUsd(0) at %d:", end)},
		{"the loan asset's quote stops answering inside the window",
			clockMutation(t, callTo(t, prof.PriceFeed, &priceFeedABI, "peekUsd", prof.LoanAsset),
				setResultWord(0, nil)), fmt.Sprintf("peekUsd(1) at %d:", end)},
		{"the loan asset's peg breaks inside the window",
			clockMutation(t, callTo(t, prof.PriceFeed, &priceFeedABI, "pegOk", prof.LoanAsset),
				setResultWord(0, nil)), fmt.Sprintf("pegOk at %d:", end)},
		{"the loan asset's peg reverts inside the window",
			clockMutation(t, callTo(t, prof.PriceFeed, &priceFeedABI, "pegOk", prof.LoanAsset),
				func(r *tapeResult) { r.Success, r.ReturnData = false, selectorOf(t, ErrUnknownToken) }),
			fmt.Sprintf("pegOk at %d:", end)},
		// And the spread hook's, which nothing else in the refresh reads.
		{"the chain's spread post lapses inside the window",
			clockMutation(t, callTo(t, prof.SpreadHook, &spreadHookABI, "spreadPpm"), func(r *tapeResult) {
				r.ReturnData = append([]byte(nil), r.ReturnData...)
				copy(r.ReturnData[0:32], common.LeftPadBytes([]byte{1}, 32))
				copy(r.ReturnData[32:64], common.LeftPadBytes(big.NewInt(17_500).Bytes(), 32))
			}),
			fmt.Sprintf("spreadPpm at %d: chain (true 17500) port (false 0)", end)},
	} {
		tp.mutate = c.mutate
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err, c.name)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.False(t, extra.Attested, c.name)
		require.Empty(t, extra.ProfileDrift, c.name)
		require.True(t, strings.HasPrefix(extra.AttestFailure, c.prefix), "%s: %q", c.name, extra.AttestFailure)
		require.Zero(t, extra.Probes, "%s: the deadline round runs before the probes", c.name)
		_, err = NewPoolSimulator(tracked)
		require.ErrorIs(t, err, ErrNotAttested, c.name)
	}
	tp.mutate = nil

	// A policy that declares no window has no end to bind against and sends no round at all.
	parity := openTape(t, trackerTape)
	parityListed := replayListing(t, parity)
	tracked, err = NewPoolTracker(parityConfig(Policy{}), parity.client()).GetNewPoolStateAtBlock(context.Background(),
		parityListed, big.NewInt(trackerBlock))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	for k := range parity.used {
		require.NotContains(t, k, `{"time":`, "no forward clock round under a policy with no window")
	}
}

// TestDeadlineClocks pins which clocks the deadline round is sent at. Each predicate is monotone in the clock, so
// given that the chain and the port agree at the snapshot, agreeing at min(deadline, end) and at the second after
// it pins the flip exactly; a predicate already false at the snapshot, or one whose deadline is past the window,
// needs no clock of its own.
func TestDeadlineClocks(t *testing.T) {
	t.Parallel()
	const now, window = 1_000_000, 600
	base := func() *flammState {
		st := gridReads(t, "51302915", "live").mustState(t)
		st.Timestamp = now
		// both rounds fresh and outliving the window, no spread post
		for _, tk := range []*feedToken{&st.Feed.Asset, &st.Feed.Loans[0]} {
			tk.Round.Ok = true
			tk.Round.UpdatedAt.SetUint64(now - 10)
			tk.Heartbeat.SetUint64(3_600)
		}
		st.Feed.Sequencer.StartedAt.SetUint64(now - 100_000)
		st.Hooks.Spread.EverlongSpread.MaxSpreadAge.SetUint64(600)
		st.Hooks.Spread.EverlongSpread.LastSetTs.SetUint64(now - 601) // lapsed
		return st
	}
	for _, c := range []struct {
		name   string
		window uint64
		mutate func(*flammState)
		want   []uint64
	}{
		{"nothing flips inside the window", window, nil, []uint64{now + window}},
		{"no window declared", 0, nil, nil},
		{"the pool asset round expires inside it", window, func(s *flammState) {
			s.Feed.Asset.Round.UpdatedAt.SetUint64(now - 3_400) // deadline now+200
		}, []uint64{now + window, now + 200, now + 201}},
		{"both rounds expire inside it", window, func(s *flammState) {
			s.Feed.Asset.Round.UpdatedAt.SetUint64(now - 3_400)    // now+200
			s.Feed.Loans[0].Round.UpdatedAt.SetUint64(now - 3_300) // now+300
		}, []uint64{now + window, now + 200, now + 201, now + 300, now + 301}},
		{"a live spread post expires inside it", window, func(s *flammState) {
			s.Hooks.Spread.EverlongSpread.LastSetTs.SetUint64(now - 100) // deadline now+500
		}, []uint64{now + window, now + 500, now + 501}},
		{"a spread post with no staleness window", window, func(s *flammState) {
			s.Hooks.Spread.EverlongSpread.MaxSpreadAge.Clear()
		}, []uint64{now + window}},
		{"a deadline on the window's last second", window, func(s *flammState) {
			s.Feed.Asset.Round.UpdatedAt.SetUint64(now - 3_000) // deadline now+600, the end itself
		}, []uint64{now + window}},
		{"a deadline one second inside it", window, func(s *flammState) {
			s.Feed.Asset.Round.UpdatedAt.SetUint64(now - 3_001) // deadline now+599
		}, []uint64{now + window, now + 599}},
		{"a round already stale at the snapshot", window, func(s *flammState) {
			s.Feed.Asset.Round.UpdatedAt.SetUint64(now - 4_000)
		}, []uint64{now + window}},
		// The sequencer's grace is the mirror of a deadline: the feed answers nothing until it ends.
		{"a sequencer grace that ends inside the window", window, func(s *flammState) {
			s.Feed.Sequencer.StartedAt.SetUint64(now - 3_550) // usable from now+51
		}, []uint64{now + window, now + 50, now + 51}},
		{"a sequencer grace that outlasts the window", window, func(s *flammState) {
			s.Feed.Sequencer.StartedAt.SetUint64(now - 100)
		}, []uint64{now + window}},
		{"a sequencer that is down", window, func(s *flammState) {
			s.Feed.Sequencer.StartedAt.SetUint64(now - 3_550)
			s.Feed.Sequencer.Answer.SetOne() // down: no clock of its own, the snapshot refuses the pool
		}, []uint64{now + window}},
		{"a window that overflows the clock", ^uint64(0), nil, nil},
	} {
		st := base()
		if c.mutate != nil {
			c.mutate(st)
		}
		require.Equal(t, c.want, deadlineClocks(st, now, c.window), c.name)
	}
	require.Nil(t, deadlineClocks(nil, now, window))

	// readWindow reads at the same end and takes the same branch on a window that overflows the clock: it reads at
	// no clock rather than at a wrapped one, which would be a clock before the snapshot's own. A nil client proves
	// it answers before it sends anything.
	for _, c := range []struct {
		name               string
		snapshotTs, window uint64
	}{
		{"no window declared", now, 0},
		{"a window that overflows the clock", now, ^uint64(0)},
		{"a window that overflows it by one", ^uint64(0) - 9, 10},
	} {
		ahead, err := readWindow(context.Background(), nil, []StaticVenue{{Account: c104.Account}}, nil,
			c.snapshotTs, c.window)
		require.NoError(t, err, c.name)
		require.Nil(t, ahead, c.name)
	}
}

// TestTrackerDecodesSpreadWord: FLAMMStore's lastLeverSpreadPpm has no view, so the only thing that fixes where it
// is read from is this test. It is zero on the deployed pool, which is what a shifted mask also reads, so the word
// is given a value here: the degrade value a lever-down is priced at once the keeper's post lapses
// (FLAMMLeverLib._spread) comes out of bits 160..191 of the slot that also holds pendingControllerHook and
// hookSetExecutableAt, and neither of those moves with it.
func TestTrackerDecodesSpreadWord(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	tracker := NewPoolTracker(baseConfig(), tp.client())
	spreadWord := strings.ToLower(flammSlot(slotSpreadWord).Hex())
	for _, want := range []uint64{1, 13_000, 17_500, 1<<32 - 1} {
		tp.mutate = func(method string, params json.RawMessage, e *tapeEntry) {
			if method != "eth_getStorageAt" || !strings.Contains(strings.ToLower(string(params)), spreadWord) {
				return
			}
			var hexWord string
			require.NoError(t, json.Unmarshal(e.Result, &hexWord))
			var w, mask, v uint256.Int
			w.SetBytes(common.FromHex(hexWord))
			mask.Lsh(uint256.NewInt(1<<32-1), 160)
			w.And(&w, mask.Not(&mask))
			w.Or(&w, v.Lsh(uint256.NewInt(want), 160))
			e.Result, _ = json.Marshal(hexutil.Encode(w.PaddedBytes(32)))
		}
		tracked, err := tracker.GetNewPoolStateAtBlock(context.Background(), listed, big.NewInt(trackerBlock))
		require.NoError(t, err)
		var extra Extra
		require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
		require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
		require.Equal(t, want, extra.Reads.Pool.LastLeverSpreadPpm.Uint64(), "lastLeverSpreadPpm")
		require.Zero(t, extra.ScheduledChangeAt, "the executableAt beside it does not move")
	}
	tp.mutate = nil
}

// TestTrackerConfigGuards: a tracker whose configuration is not the listing's refuses the pool instead of
// refreshing it against another chain's registry.
func TestTrackerConfigGuards(t *testing.T) {
	t.Parallel()
	tp := openTape(t, trackerTape)
	listed := replayListing(t, tp)
	block := big.NewInt(trackerBlock)
	ctx := context.Background()
	for _, c := range []struct {
		name string
		cfg  *Config
	}{
		{"no configuration", nil},
		{"another chain", &Config{DexID: DexType, ChainID: valueobject.ChainIDEthereum,
			Factory: c104.Factory.Hex()}},
		{"another dex id", &Config{DexID: "other", ChainID: valueobject.ChainIDBase,
			Factory: c104.Factory.Hex()}},
	} {
		_, err := NewPoolTracker(c.cfg, tp.client()).GetNewPoolStateAtBlock(ctx, listed, block)
		require.ErrorIs(t, err, ErrInvalidProfile, c.name)
	}
	// An empty DexID matches any exchange, which is how pool-service configures a single-dex source.
	tracked, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDBase,
		Factory: c104.Factory.Hex()}, tp.client()).GetNewPoolStateAtBlock(ctx, listed, block)
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	require.True(t, extra.Attested, "attest %q drift %q", extra.AttestFailure, extra.ProfileDrift)
	// And an entity the registry does not admit at all.
	foreign := listed
	foreign.Address = "0x0000000000000000000000000000000000000bad"
	_, err = NewPoolTracker(baseConfig(), tp.client()).GetNewPoolStateAtBlock(ctx, foreign, block)
	require.ErrorIs(t, err, ErrInvalidProfile)
}

// TestAttestClockPegRevertClass: attestClock compares the class of a pegOk revert, not only that both sides
// reverted. The only token the deployed feed reverts on is one it never registered, which the drift check already
// refuses, so the comparison is pinned here against a state the port itself refuses that way.
func TestAttestClockPegRevertClass(t *testing.T) {
	t.Parallel()
	st := gridReads(t, "51302915", "live").mustState(t)
	st.Feed.Loans[0].Known = false // PriceFeed.pegOk on an unregistered token: UnknownToken, outside the try
	at := st.Timestamp
	_, err := st.pegOk(0, at)
	require.ErrorIs(t, err, ErrUnknownToken)

	// The honest answer: what the port itself says at this clock, with pegOk reverting the way the pool does.
	f := &st.Feed
	ok, price, ts := f.peekCross(&f.Asset, &f.Loans[0], at)
	usdOk0, usd0, usdTs0 := f.peekUsd(&f.Asset, at)
	usdOk1, usd1, usdTs1 := f.peekUsd(&f.Loans[0], at)
	spreadOk, spreadPpm := st.Hooks.Spread.EverlongSpread.spreadPpm(at)
	honest := clockAnswer{At: at, CrossOk: ok, CrossPrice: price, CrossTs: ts,
		UsdOk: [2]bool{usdOk0, usdOk1}, Usd: [2]uint256.Int{usd0, usd1}, UsdTs: [2]uint64{usdTs0, usdTs1},
		PegErr: selectorOf(t, ErrUnknownToken), HasSpread: st.Hooks.hasSpread(), SpreadOk: spreadOk,
		SpreadPpm: spreadPpm}
	require.Empty(t, attestClock(st, &honest))

	// The same revert with another of the pool's own errors is not the same answer.
	other := honest
	other.PegErr = selectorOf(t, ErrPriceBand)
	require.Contains(t, attestClock(st, &other), "pegOk at ")
	// Nor is a revert the port cannot map at all.
	unmapped := honest
	unmapped.PegErr = []byte{0xde, 0xad, 0xbe, 0xef}
	require.Contains(t, attestClock(st, &unmapped), "pegOk at ")
}
