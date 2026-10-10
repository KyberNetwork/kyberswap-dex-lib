package lotflow

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

// Dependency tracker, part 1 (A7): the state one LOT venue's quote depends on, a Loader interface that
// fetches it (an RPC implementation comes with the live-parity step), and the event -> invalidated
// entity table of KYBER-PACK.md §5 "Events that invalidate state". dex-lib's v4 tracker does not
// implement IPoolTrackerWithDependencies today (K2-dexlib); this is the concrete dependency list a
// LOTFLOW venue needs from it.

// Loader fetches the on-chain state a quote needs. Every method is one view (or a small batch) at the
// block being tracked.
type Loader interface {
	// Constituents: Basket.constituents() — stocks in leg order and units per 1e18 LOT.
	Constituents(ctx context.Context, lot string) ([]string, []*big.Int, error)
	// HookParams: NavJitHookV17.getParams().
	HookParams(ctx context.Context) (Params, error)
	// Venue: venueOf(lot), its slot0, usdg side/decimals and standingOf(poolId).
	Venue(ctx context.Context, lot string) (*VenueSnapshot, error)
	// NavGuardParams: NavGuard.getParams().
	NavGuardParams(ctx context.Context) (NavGuardParams, error)
	// Feed: NavGuard.feedOf(stock) and its latestRoundData/decimals (nil state = no feed).
	Feed(ctx context.Context, stock string) (string, *FeedState, error)
	// Stock: StockRegistry.getStock(stock) with each registry pool's slot0/liquidity, the stock's
	// oraclePaused(), and VenueBook.venuesOf(stock) with the READ_GAS admission reads.
	Stock(ctx context.Context, stock string) (*StockSnapshot, error)
	// Pool: the leg state of one admitted source (slot0, liquidity, fee, ticks over the leg window).
	Pool(ctx context.Context, stock string, src Source) (*LegPool, error)
	// PoolManagerLotBalance: LOT.balanceOf(PoolManager), the pay-after sell float.
	PoolManagerLotBalance(ctx context.Context, lot string) (*big.Int, error)
	// BlockTimestamp: for NavGuard feed staleness.
	BlockTimestamp(ctx context.Context) (uint64, error)
}

type VenueSnapshot struct {
	PoolID       string
	UsdgIs0      bool
	UsdgDecimals int
	SqrtPriceX96 *big.Int
	Tick         int
	Standing     *StandingPosition // nil = no standing position
}

type StockSnapshot struct {
	Registry       RegistryEntry
	StockDecimals  int
	StableDecimals int
	OraclePaused   bool
	VenuesOK       bool
	Venues         []BookVenueView
}

// Tracker holds the addresses of one deployment.
type Tracker struct {
	Cfg         *Config
	Hook        string
	PoolManager string
	NavGuard    string
	Registry    string
	VenueBook   string
	StockBeacon string // the stock tokens' beacon (Upgraded)
}

// TrackerState is one venue's loaded state plus the indexes the invalidation table uses.
type TrackerState struct {
	*Tracker
	Lot         string
	VenuePoolID string
	Market      *Market

	stocks     map[string]bool
	feedStock  map[string]string // feed -> stock
	regPool    map[string]string // registry PoolId -> stock
	pricePool  map[string]bool   // registry price-pool PoolIds
	bookVenue  map[string]string // VenueBook pool -> stock
	candidate  map[string]bool   // admitted source ids
	dynamicFee map[string]bool   // dynamic-fee candidate PoolIds
}

func lc(s string) string { return strings.ToLower(s) }

// Load fetches everything one venue's quote needs and builds the Market.
func (t *Tracker) Load(ctx context.Context, l Loader, lot string) (*TrackerState, error) {
	stocks, units, err := l.Constituents(ctx, lot)
	if err != nil {
		return nil, fmt.Errorf("constituents: %w", err)
	}
	params, err := l.HookParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("hook params: %w", err)
	}
	vs, err := l.Venue(ctx, lot)
	if err != nil {
		return nil, fmt.Errorf("venue: %w", err)
	}
	navP, err := l.NavGuardParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("navguard params: %w", err)
	}
	ts, err := l.BlockTimestamp(ctx)
	if err != nil {
		return nil, err
	}
	pmLot, err := l.PoolManagerLotBalance(ctx, lot)
	if err != nil {
		return nil, err
	}
	st := &TrackerState{Tracker: t, Lot: lc(lot), VenuePoolID: lc(vs.PoolID),
		stocks: map[string]bool{}, feedStock: map[string]string{}, regPool: map[string]string{},
		pricePool: map[string]bool{}, bookVenue: map[string]string{}, candidate: map[string]bool{},
		dynamicFee: map[string]bool{}}
	m := &Market{
		Lot: lot,
		Venue: &Venue{UsdgIs0: vs.UsdgIs0, UsdgDecimals: vs.UsdgDecimals, Params: params,
			SqrtPriceX96: vs.SqrtPriceX96, Tick: vs.Tick, Standing: vs.Standing},
		NavParams: navP, Timestamp: ts, PmLotBalance: pmLot,
	}
	for i, stock := range stocks {
		s := lc(stock)
		st.stocks[s] = true
		snap, err := l.Stock(ctx, stock)
		if err != nil {
			return nil, fmt.Errorf("stock %s: %w", stock, err)
		}
		reg := snap.Registry
		if ml, ok := l.(MqLoader); ok && !sameAddr(reg.Stable, t.Cfg.Usdg) {
			// a WETH-quoted token on an MQ venue: priced by MqPricing in Quote, one kind-3 source
			if m.Mq == nil {
				if m.Mq, err = ml.MqMarket(ctx); err != nil {
					return nil, fmt.Errorf("mq route: %w", err)
				}
			}
			tok, pool, err := ml.MqToken(ctx, stock, m.Mq)
			if err != nil {
				return nil, fmt.Errorf("mq token %s: %w", stock, err)
			}
			id := v3SourceID(tok.Pool)
			st.candidate[id] = true
			st.bookVenue[lc(tok.Pool)] = s
			m.Constituents = append(m.Constituents, MarketConstituent{Stock: stock, Unit: units[i], Registry: reg,
				Pools: map[string]*LegPool{id: pool}, Mq: tok,
				Nav: NavConstituent{Unit: units[i], SpotUsd18: new(big.Int), StockDecimals: snap.StockDecimals}})
			continue
		}
		if reg.PriceIndex >= len(reg.Pools) {
			return nil, fmt.Errorf("stock %s: price index %d of %d pools", stock, reg.PriceIndex, len(reg.Pools))
		}
		spot, err := SpotUsd18(reg.Pools[reg.PriceIndex].SqrtPriceX96, reg.StockIsCurrency0, snap.StockDecimals,
			snap.StableDecimals)
		if err != nil {
			return nil, fmt.Errorf("stock %s spot: %w", stock, err)
		}
		feedAddr, feed, err := l.Feed(ctx, stock)
		if err != nil {
			return nil, fmt.Errorf("feed %s: %w", stock, err)
		}
		if feed != nil && !isZeroAddr(feedAddr) {
			st.feedStock[lc(feedAddr)] = s
		}
		for j, p := range reg.Pools {
			st.regPool[lc(p.PoolID)] = s
			if j == reg.PriceIndex {
				st.pricePool[lc(p.PoolID)] = true
			}
		}
		for _, v := range snap.Venues {
			st.bookVenue[lc(v.Pool)] = s
		}
		mc := MarketConstituent{Stock: stock, Unit: units[i], Registry: reg, VenuesOK: snap.VenuesOK,
			Venues: snap.Venues, Pools: map[string]*LegPool{}, StableDecimals: snap.StableDecimals,
			Nav: NavConstituent{Unit: units[i], SpotUsd18: spot, StockDecimals: snap.StockDecimals, Feed: feed,
				OraclePaused: snap.OraclePaused}}
		srcs, _, _ := Sources(t.Cfg, stock, reg, snap.VenuesOK, snap.Venues)
		for _, src := range srcs {
			p, err := l.Pool(ctx, stock, src)
			if err != nil {
				return nil, fmt.Errorf("pool %s: %w", src.ID, err)
			}
			mc.Pools[lc(src.ID)] = p
			st.candidate[lc(src.ID)] = true
			if p.Kind == KindV4 && p.DynamicFee {
				st.dynamicFee[lc(src.ID)] = true
			}
		}
		m.Constituents = append(m.Constituents, mc)
	}
	st.Market = m
	return st, nil
}

// ---------------------------------------------------------------- events

// eventSignatures: logical name -> canonical Solidity signature. Names that collide on chain
// (ParamsSet on three contracts, Swap/Mint/Burn on v3 vs the PoolManager) get distinct logical names;
// dispatch is by emitter role and topic0.
var eventSignatures = map[string]string{
	// VenueBook (src/interfaces/IVenueBook.sol)
	"VenueAdded":    "VenueAdded(address,address,uint8)",
	"VenueRemoved":  "VenueRemoved(address,address)",
	"KeeperSet":     "KeeperSet(address)",
	"FactorySet":    "FactorySet(address,uint8,bool)",
	"BookParamsSet": "ParamsSet(uint8,uint16)",
	// StockRegistry (src/StockRegistry.sol)
	"StockVenuesSet":   "StockVenuesSet(address,address,(address,address,uint24,int24,address)[],uint8,bool,uint8,uint8)",
	"StockRemoved":     "StockRemoved(address)",
	"StableSet":        "StableSet(address,bool)",
	"StablePoolSet":    "StablePoolSet(address,address,(address,address,uint24,int24,address))",
	"EthStablePoolSet": "EthStablePoolSet(address,(address,address,uint24,int24,address))",
	// NavGuard (src/NavGuard.sol)
	"FeedSet":           "FeedSet(address,address)",
	"NavGuardParamsSet": "ParamsSet((uint16,uint16,uint32,uint16))",
	// NavJitHookV17
	"HookParamsSet":     "ParamsSet((uint16,uint16,uint16,uint16,int24))",
	"TreasurySet":       "TreasurySet(address)",
	"NavGuardSet":       "NavGuardSet(address)",
	"VenueOpened":       "VenueOpened(address,bytes32,uint256)",
	"StandingSeeded":    "StandingSeeded(bytes32,uint128,int24,int24)",
	"StandingWithdrawn": "StandingWithdrawn(bytes32,address)",
	// v4 PoolManager
	"Swap":               "Swap(bytes32,address,int128,int128,uint160,uint128,int24,uint24)",
	"ModifyLiquidity":    "ModifyLiquidity(bytes32,address,int24,int24,int256,bytes32)",
	"ProtocolFeeUpdated": "ProtocolFeeUpdated(bytes32,uint24)",
	// Uniswap v3 pools
	"V3Swap": "Swap(address,address,int256,int256,uint160,uint128,int24)",
	"V3Mint": "Mint(address,address,int24,int24,uint128,uint256,uint256)",
	"V3Burn": "Burn(address,int24,int24,uint128,uint256,uint256)",
	// Chainlink aggregators
	"AnswerUpdated": "AnswerUpdated(int256,uint256,uint256)",
	"NewRound":      "NewRound(uint256,address,uint256)",
	// stock tokens (OZ Pausable), their beacon, ERC-20
	"Paused":   "Paused(address)",
	"Unpaused": "Unpaused(address)",
	"Upgraded": "Upgraded(address)",
	"Transfer": "Transfer(address,address,uint256)",
}

// Topic0 is keccak256 of the named event's signature.
func Topic0(name string) string {
	return fmt.Sprintf("0x%x", crypto.Keccak256([]byte(eventSignatures[name])))
}

// Event is one log: emitter, topic0 and the indexed arguments (topics[1:]).
type Event struct {
	Emitter string
	Topic0  string
	Topics  []string
}

// NewEvent builds an Event from a logical name and its indexed arguments (addresses or bytes32,
// padded to 32 bytes as on chain).
func NewEvent(emitter, name string, indexed ...string) Event {
	ev := Event{Emitter: emitter, Topic0: Topic0(name)}
	for _, a := range indexed {
		h := strings.TrimPrefix(lc(a), "0x")
		ev.Topics = append(ev.Topics, "0x"+strings.Repeat("0", 64-len(h))+h)
	}
	return ev
}

func (ev Event) is(name string) bool { return lc(ev.Topic0) == Topic0(name) }

// addr reads indexed topic i as an address; bytes32 reads it whole.
func (ev Event) addr(i int) string {
	if i >= len(ev.Topics) || len(ev.Topics[i]) < 40 {
		return ""
	}
	t := lc(ev.Topics[i])
	return "0x" + t[len(t)-40:]
}

func (ev Event) bytes32(i int) string {
	if i >= len(ev.Topics) {
		return ""
	}
	return lc(ev.Topics[i])
}

// ---------------------------------------------------------------- invalidation

type EntityKind string

const (
	EntFull       EntityKind = "full"         // reload everything
	EntHookParams EntityKind = "hookParams"   // spreads, band, size buffer, width
	EntVenue      EntityKind = "venue"        // venue slot0
	EntStanding   EntityKind = "standing"     // standingOf(venue)
	EntNavParams  EntityKind = "navParams"    // NavGuard params
	EntNav        EntityKind = "nav"          // checked NAV must be recomputed
	EntFeed       EntityKind = "feed"         // one stock's feed (Key = stock)
	EntStock      EntityKind = "stock"        // registry entry / paused state (Key = stock): re-run admission
	EntBook       EntityKind = "book"         // venuesOf(stock) (Key = stock): re-run admission
	EntAdmission  EntityKind = "admission"    // admission reads of a stock's pools moved (Key = stock)
	EntPool       EntityKind = "pool"         // one candidate's leg state (Key = source id)
	EntPMBalance  EntityKind = "pmLotBalance" // the PoolManager's LOT balance (sell float)
)

type Entity struct {
	Kind EntityKind
	Key  string
}

// Invalidate maps one log to the entities it invalidates (sorted, unique; empty = irrelevant).
func (s *TrackerState) Invalidate(ev Event) []Entity {
	set := map[Entity]bool{}
	add := func(k EntityKind, key string) { set[Entity{k, key}] = true }
	allStocks := func() {
		for st := range s.stocks {
			add(EntStock, st)
		}
		add(EntNav, "")
	}
	from := lc(ev.Emitter)
	switch {
	case from == lc(s.VenueBook):
		switch {
		case ev.is("VenueAdded"), ev.is("VenueRemoved"):
			if st := ev.addr(0); s.stocks[st] {
				add(EntBook, st)
			}
		case ev.is("FactorySet"), ev.is("BookParamsSet"):
			for st := range s.stocks {
				add(EntBook, st)
			}
		}
	case from == lc(s.Registry):
		switch {
		case ev.is("StockVenuesSet"), ev.is("StockRemoved"):
			if st := ev.addr(0); s.stocks[st] {
				add(EntStock, st)
				add(EntNav, "")
			}
		case ev.is("StableSet"), ev.is("StablePoolSet"), ev.is("EthStablePoolSet"):
			allStocks()
		}
	case from == lc(s.NavGuard):
		switch {
		case ev.is("FeedSet"):
			if st := ev.addr(0); s.stocks[st] {
				add(EntFeed, st)
				add(EntNav, "")
			}
		case ev.is("NavGuardParamsSet"):
			add(EntNavParams, "")
			add(EntNav, "")
		}
	case from == lc(s.Hook):
		switch {
		case ev.is("HookParamsSet"):
			add(EntHookParams, "")
		case ev.is("NavGuardSet"):
			add(EntFull, "")
		case ev.is("VenueOpened"):
			if ev.addr(0) == s.Lot {
				add(EntVenue, "")
			}
		case ev.is("StandingSeeded"), ev.is("StandingWithdrawn"):
			if ev.bytes32(0) == s.VenuePoolID {
				add(EntStanding, "")
				add(EntVenue, "")
			}
		}
	case from == lc(s.PoolManager):
		if !ev.is("Swap") && !ev.is("ModifyLiquidity") && !ev.is("ProtocolFeeUpdated") {
			break
		}
		id := ev.bytes32(0)
		if id == s.VenuePoolID && !ev.is("ProtocolFeeUpdated") {
			add(EntVenue, "")
		}
		if s.candidate[id] {
			add(EntPool, id)
		}
		if st, ok := s.regPool[id]; ok && !ev.is("ProtocolFeeUpdated") {
			add(EntAdmission, st) // slot0 / liquidity are admission reads
			if s.pricePool[id] {
				add(EntNav, "") // the price pool sets spot NAV
			}
		}
	case s.feedStock[from] != "":
		if ev.is("AnswerUpdated") || ev.is("NewRound") {
			add(EntFeed, s.feedStock[from])
			add(EntNav, "")
		}
	case s.bookVenue[from] != "":
		if ev.is("V3Swap") || ev.is("V3Mint") || ev.is("V3Burn") {
			if id := v3SourceID(from); s.candidate[id] {
				add(EntPool, id)
			}
			add(EntAdmission, s.bookVenue[from])
		}
	case s.stocks[from]:
		if ev.is("Paused") || ev.is("Unpaused") {
			add(EntStock, from)
			add(EntNav, "")
		}
	case from == lc(s.StockBeacon):
		if ev.is("Upgraded") {
			allStocks()
		}
	case from == s.Lot:
		if ev.is("Transfer") {
			pm := lc(s.PoolManager)
			if ev.addr(0) == pm || ev.addr(1) == pm {
				add(EntPMBalance, "")
			}
		}
	}
	out := make([]Entity, 0, len(set))
	for e := range set {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// ---------------------------------------------------------------- dependencies

type PollKind string

const (
	// PollDynamicFee: a hooked price pool's fee is set by its hook per swap and can change without an
	// event (Fables `currentFee(poolId, zeroForOne)`; dex-lib's fables handler tracks it).
	PollDynamicFee PollKind = "dynamicFee"
	// PollBlockTimestamp: NavGuard staleness compares feed updatedAt with block.timestamp.
	PollBlockTimestamp PollKind = "blockTimestamp"
	// PollFeedRound: NavGuard.feedOf is normally a Chainlink proxy, and AnswerUpdated/NewRound are
	// emitted by the aggregator behind it, not the proxy; poll latestRoundData (or subscribe to
	// proxy.aggregator(), re-resolved on the proxy's aggregator change). Key = stock.
	PollFeedRound PollKind = "feedRound"
)

type Polled struct {
	Kind PollKind
	Key  string
}

// Dependencies is what a tracker must watch: log emitters, PoolManager PoolIds, and polled values.
type Dependencies struct {
	Emitters []string
	PoolIDs  []string
	Polled   []Polled
}

func (s *TrackerState) Dependencies() Dependencies {
	em := map[string]bool{lc(s.Hook): true, lc(s.PoolManager): true, lc(s.NavGuard): true, lc(s.Registry): true,
		lc(s.VenueBook): true, lc(s.StockBeacon): true, s.Lot: true}
	for st := range s.stocks {
		em[st] = true
	}
	for f := range s.feedStock {
		em[f] = true
	}
	for v := range s.bookVenue {
		em[v] = true
	}
	ids := map[string]bool{s.VenuePoolID: true}
	for id := range s.regPool {
		ids[id] = true
	}
	d := Dependencies{Emitters: sortedKeys(em), PoolIDs: sortedKeys(ids)}
	for _, id := range sortedKeys(s.dynamicFee) {
		d.Polled = append(d.Polled, Polled{Kind: PollDynamicFee, Key: id})
	}
	feedStocks := map[string]bool{}
	for _, st := range s.feedStock {
		feedStocks[st] = true
	}
	for _, st := range sortedKeys(feedStocks) {
		d.Polled = append(d.Polled, Polled{Kind: PollFeedRound, Key: st})
	}
	d.Polled = append(d.Polled, Polled{Kind: PollBlockTimestamp})
	return d
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
