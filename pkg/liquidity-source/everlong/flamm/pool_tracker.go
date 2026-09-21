package everlongflamm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// PoolTracker refreshes one pool at one block B:
//
//  1. one Multicall3 aggregate of every view the state is built from (tracker_reads.go) at the latest block,
//     which fixes B and its timestamp: the pool's own views and, for each hook the listing names, the reads its
//     registered kind declares (hook_kinds.go);
//  2. the storage words no view exposes, at B -- FLAMMStore.lastLeverSpreadPpm, the pending loan-asset ceremony,
//     and each venue's managed collateral and supply shares -- with the storage layout checked against the views:
//     the FLAMMStore words beside them, the pending venue ceremony, the factory's beacon and upgrade schedule, and
//     every MMRouter configuration word the port decodes from a view. That is 33 words on the one-venue pool,
//     read in JSON-RPC batches of at most ten (multicall.go rpcBatchSize);
//  3. the drift check against the listing, the state build (state_reads.go) and the view attestation;
//  4. a dependency-resolve round and two forward rounds, all at B. The resolve round reads, at the block's own
//     clock, the aggregators the price feed proxies currently resolve to (the dependency set below). The forward
//     rounds read at B with only block.timestamp overridden: what each venue's market oracle answers at the far
//     end of the snapshot window, which is the one input the clock alone moves; and, at each clock the port
//     believes a deadline word flips at inside that window, the deployed views that deadline decides -- the
//     PriceFeed's cross and quotes, the loan asset's peg and the keeper's spread post -- which is what binds words
//     whose only effect is in the future. Each forward round carries Multicall3's own block timestamp and fails in
//     transport unless the node answered at the clock it asked for (multicall.go clockProof), so a node that
//     accepts blockOverrides and ignores it cannot leave the window guard disarmed in silence
//     (tracker_reads.go readAggregators / readOracleAhead / readClock, attest.go attestDeadlines);
//  5. one aggregate at B of the pool's own previews and the Router's funding ceiling on amounts the port picks,
//     each of which the port must reproduce exactly (attest.go).
//
// Under the shipped defaults that is nine HTTP round trips: five eth_calls and four storage batches. A policy that
// declares no window (maxSnapshotAgeSec 0) sends neither forward round of step 4 and makes seven.
//
// The refresh publishes the addresses whose logs should trigger one besides the pool's own
// (pool.IPoolTrackerWithDependencies): the Chainlink aggregators behind the two token feeds, the sequencer feed
// and each venue's market oracle feeds, plus every hook whose kind emits events of its own that move a quote (the
// swap hook and the leverage spread hook) and the Router. Those are exactly the contracts that move a quote with an
// event of their own and none from the pool -- a feed round, a keeper's EverlongHook.setFeeRow / _setTuning
// (TuningChanged), a keeper's spread post, a guardian's MMRouter.setGlobalPaused (GlobalPauseSet), a curator's
// venue configuration. Morpho Blue and the IRM are left out on purpose: every borrow, repay and supply anywhere on
// Base would trigger a refresh, and the accrual they drive is recomputed at the quote clock anyway. The aggregators
// are resolved on every refresh because a Chainlink phase rotation moves proxy.aggregator() with no log at the
// proxy, and Extra.DependenciesStored is cleared whenever the resolved set changes, so pool-service re-registers it.
//
// The venue set the round reads is the one the previous refresh published (Extra.Venues; the listing's, on the
// first refresh). A curator may add or retire a venue on a live pool, so a round whose venue count or venue
// identity disagrees with that set -- or whose reads of a venue the pool no longer has were answered with a revert
// -- re-reads the set at B and runs the round again with it, and publishes it. A listed pool therefore heals
// itself without being relisted, which pool-service does not do for an existing pool.
//
// A refresh that drifts or fails attestation still publishes its reads with Attested false, so the simulator
// stops quoting the pool rather than keep quoting an older snapshot. So does a view round the chain answered at B
// with a required view reverting or not decoding (an upgraded implementation whose views changed, a Router whose
// venue set no longer answers): it is published as drift, without reads. Only a transport failure returns the
// error, leaving pool-service with the last entity until the node answers; the simulator stops quoting that entity
// once it is Config.MaxSnapshotAgeSec old. A refresh also stamps the earliest time a scheduled implementation,
// hook-set, venue or loan-asset change can execute (Extra.ScheduledChangeAt), which the simulator refuses to quote
// across.
type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var (
	_ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

	_ pool.IPoolTracker                 = (*PoolTracker)(nil)
	_ pool.IPoolTrackerWithOverrides    = (*PoolTracker)(nil)
	_ pool.IPoolTrackerWithDependencies = (*PoolTracker)(nil)
)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ pool.GetNewPoolStateParams) (entity.Pool, error) {
	return t.track(ctx, p, nil, nil)
}

// GetNewPoolStateWithOverrides runs the same refresh under state overrides: the aggregates carry them and the
// storage words apply them as eth_call would (a code override on a registered contract is drift).
func (t *PoolTracker) GetNewPoolStateWithOverrides(ctx context.Context, p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams) (entity.Pool, error) {
	return t.track(ctx, p, nil, params.Overrides)
}

// GetNewPoolStateAtBlock pins the whole refresh to a historical block (replays and fixtures).
func (t *PoolTracker) GetNewPoolStateAtBlock(ctx context.Context, p entity.Pool, block *big.Int) (entity.Pool, error) {
	return t.track(ctx, p, block, nil)
}

func (t *PoolTracker) track(ctx context.Context, p entity.Pool, block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	var se StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &se); err != nil {
		return p, err
	}
	w, err := validEntity(&p, &se)
	if err != nil {
		return p, err
	}
	if t.config == nil || t.config.ChainID != se.ChainID || (t.config.DexID != "" && t.config.DexID != p.Exchange) {
		return p, ErrInvalidProfile
	}
	address := common.HexToAddress(p.Address)
	rpc := &mcRPC{client: t.ethrpcClient, overrides: overrides}
	var prev Extra
	_ = json.Unmarshal([]byte(p.Extra), &prev) // an unreadable Extra only means there is no venue set to start from
	policy := t.config.policy()
	snap, venues, err := readRound(ctx, rpc, address, &se, &w.hooks, prev.Venues, block)
	if err != nil && (snap == nil || !chainAnswered(err)) {
		return p, err
	}
	if err != nil {
		return t.publish(p, snap.Block, &Extra{Venues: venues, Dependencies: prev.Dependencies,
			DependenciesStored: prev.DependenciesStored,
			ProfileDrift:       "unreadable: " + err.Error(), Policy: policy}, entity.PoolReserves{"0", "0"})
	}

	at := new(big.Int).SetUint64(snap.Block)
	extra := Extra{Reads: &snap.Reads, Venues: venues, ScheduledChangeAt: snap.scheduledChangeAt(), Policy: policy}
	extra.Dependencies, extra.DependenciesStored = resolveDependencies(ctx, rpc, &se, &w.hooks, venues, at, &prev)
	var state *flammState
	if extra.ProfileDrift = snap.drift(address, &se, venues, w, overrides); extra.ProfileDrift == "" {
		st, err := snap.Reads.state(&w.hooks)
		switch {
		case err != nil:
			extra.AttestFailure = "state: " + err.Error()
		default:
			if extra.AttestFailure = attestViews(snap, st); extra.AttestFailure == "" {
				if extra.OracleAhead, err = readWindow(ctx, rpc, venues, at, snap.Timestamp,
					policy.MaxSnapshotAgeSec); err != nil {
					if !chainAnswered(err) {
						return p, err
					}
					extra.AttestFailure = "oracleAhead: " + err.Error()
				}
				if extra.AttestFailure == "" {
					if extra.AttestFailure, err = attestDeadlines(ctx, rpc, &se, snap, st,
						policy.MaxSnapshotAgeSec); err != nil {
						return p, err
					}
				}
				if extra.AttestFailure == "" {
					if extra.Probes, extra.AttestFailure, err = attestProbes(ctx, rpc, address, &se, snap,
						st); err != nil {
						return p, err
					}
				}
				if extra.Attested = extra.AttestFailure == ""; extra.Attested {
					state = st
				}
			}
		}
	}
	return t.publish(p, snap.Block, &extra, reservesOf(&snap.Reads.Pool.Gross, state, snap.Timestamp))
}

// readWindow reads each venue's market oracle at the far end of the snapshot window, the clock the policy admits
// quoting at (Policy.MaxSnapshotAgeSec past the refresh block). A policy that declares no window -- an explicit
// maxSnapshotAgeSec of 0, which also lifts the staleness refusal -- has no clock to read at and publishes no ahead
// answer, which leaves the simulator quoting the snapshot's own answer alone, as it does every other margin the
// configuration turns off.
func readWindow(ctx context.Context, rpc *mcRPC, venues []StaticVenue, block *big.Int,
	snapshotTs, window uint64) ([]OracleAnswer, error) {
	if window == 0 {
		return nil, nil
	}
	end := snapshotTs + window
	if end < snapshotTs { // a window that overflows the clock binds nothing, as it binds no deadline (deadlineClocks)
		return nil, nil
	}
	return readOracleAhead(ctx, rpc, venues, block, end)
}

// resolveDependencies is the refresh's dependency set and whether pool-service still holds it: the aggregators the
// price feed proxies resolve to at this block, the hooks whose kind is a refresh trigger (hookKindSpec.dependency),
// in role order, and the Router. A resolve round the node did not answer keeps the previous set rather than failing
// an otherwise good refresh -- it triggers refreshes, it does not price anything -- and any change of the set clears
// the stored flag so pool-service registers it again.
func resolveDependencies(ctx context.Context, rpc *mcRPC, se *StaticExtra, hooks *poolHookSet, venues []StaticVenue,
	block *big.Int, prev *Extra) ([]common.Address, bool) {
	aggregators, err := readAggregators(ctx, rpc, se, venues, block)
	if err != nil {
		return prev.Dependencies, prev.DependenciesStored
	}
	deps := make([]common.Address, 0, len(aggregators)+3)
	seen := make(map[common.Address]bool, len(aggregators)+3)
	for _, a := range aggregators {
		if a != (common.Address{}) && !seen[a] {
			seen[a], deps = true, append(deps, a)
		}
	}
	triggers := make([]common.Address, 0, hookRoles+1)
	for role := roleSwap; role < hookRoles; role++ {
		if e := hooks.Entries[role]; e != nil {
			if spec := hookKindSpecOf(e.Kind); spec != nil && spec.dependency() {
				triggers = append(triggers, e.Address)
			}
		}
	}
	for _, a := range append(triggers, se.Router) {
		if a != (common.Address{}) && !seen[a] {
			seen[a], deps = true, append(deps, a)
		}
	}
	return deps, prev.DependenciesStored && sameAddresses(prev.Dependencies, deps)
}

func sameAddresses(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// GetDependencies is pool.IPoolTrackerWithDependencies: the non-pool addresses whose logs should also refresh this
// pool, as the last refresh resolved them (Extra.Dependencies), and whether pool-service has already stored them.
// A pool that has not been refreshed since it was listed has none yet.
func (t *PoolTracker) GetDependencies(_ context.Context, p entity.Pool) ([]string, bool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return nil, false, err
	}
	out := make([]string, len(extra.Dependencies))
	for i, a := range extra.Dependencies {
		out[i] = lowerHex(a)
	}
	return out, extra.DependenciesStored, nil
}

// SetDependenciesStored records that pool-service registered the set GetDependencies returned. The next refresh
// clears it again if the resolved set has moved.
func (t *PoolTracker) SetDependenciesStored(p *entity.Pool, isStored bool) error {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return err
	}
	extra.DependenciesStored = isStored
	raw, err := json.Marshal(&extra)
	if err != nil {
		return err
	}
	p.Extra = string(raw)
	return nil
}

// readRound runs the refresh's view round with the venue set it should read, and returns the set it ran with. It
// re-reads the set from the Router and runs the round again once, when the round has no set to start from, when
// the pool's venue count or venue identity disagrees with it, or when the chain answered the round with a revert
// or undecodable data -- which is what a venue the pool retired looks like. Everything stays pinned to the block
// the first round ran at, so the re-run sees the same state.
func readRound(ctx context.Context, rpc *mcRPC, address common.Address, se *StaticExtra, hooks *poolHookSet,
	venues []StaticVenue, block *big.Int) (*snapshot, []StaticVenue, error) {
	scope := se.venueScope(address)
	if len(venues) == 0 {
		read, at, err := readVenueSet(ctx, rpc, scope, block)
		if err != nil {
			if at == 0 || !chainAnswered(err) {
				return nil, nil, err
			}
			return &snapshot{Block: at}, nil, err
		}
		venues, block = read, new(big.Int).SetUint64(at)
	}
	snap, err := readSnapshot(ctx, rpc, address, se, hooks, venues, block)
	moved := snap != nil && (err == nil && snap.venueSetMoved(venues) || err != nil && chainAnswered(err))
	if !moved {
		return snap, venues, err
	}
	at := new(big.Int).SetUint64(snap.Block)
	read, _, rerr := readVenueSet(ctx, rpc, scope, at)
	if rerr != nil || sameVenues(read, venues) {
		return snap, venues, err // the set is the pool's: the round's answer stands, and drift reports it
	}
	snap, err = readSnapshot(ctx, rpc, address, se, hooks, read, at)
	return snap, read, err
}

// venueSetMoved reports whether the pool's venues at the round's block are not the ones the round read.
func (s *snapshot) venueSetMoved(venues []StaticVenue) bool {
	if !s.VenueCount.IsUint64() || s.VenueCount.Uint64() != uint64(len(venues)) {
		return true
	}
	for i := range venues {
		if s.VenueAccounts[i] != venues[i].Account || s.VenueIDs[i] != venues[i].MarketID {
			return true
		}
	}
	return false
}

func sameVenues(a, b []StaticVenue) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// publish stamps a refresh's verdict at block into the entity.
func (t *PoolTracker) publish(p entity.Pool, block uint64, extra *Extra, reserves entity.PoolReserves) (entity.Pool,
	error) {
	raw, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(raw)
	p.Reserves = reserves
	p.BlockNumber = block
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// reservesOf is what the entity publishes as its reserves: the gross poolAsset and the sell capacity below, or
// zero on both sides when the refresh produced nothing the simulator will quote (drift, a state that does not
// build, a failed attestation). pool-service turns reserves into TVL and a liquidity score, so an entity every
// quote refuses must not carry a number that ranks it.
func reservesOf(gross *uint256.Int, st *flammState, now uint64) entity.PoolReserves {
	if st == nil {
		return entity.PoolReserves{"0", "0"}
	}
	capacity := sellCapacity(st, now)
	return entity.PoolReserves{gross.Dec(), capacity.Dec()}
}

// sellCapacity is a lower bound on the loan asset one sell can be paid at the snapshot: the ceiling of a sell that
// adds no collateral of its own -- liquid plus the Router's funding against the pool's physical poolAsset, capped
// by the gate room and by the per-swap notional cap, which is exactly FLAMMSwapLib's own ceiling on its first
// funding pass (FLAMMSwapLib.sol:150-166) evaluated at collateralIn = 0. It is a bound and not the achievable
// maximum because the chain counts the incoming poolAsset as collateral ("The funding counts the incoming
// poolAsset as collateral", FLAMMSwapLib.sol:164) and re-plans on it, so the ceiling only rises with the input:
// accepted sells on the live pool have paid 35-58% above this number. Zero when unpriceable.
func sellCapacity(st *flammState, now uint64) uint256.Int {
	var zero uint256.Int
	pool := st.Pool
	b, err := st.priced(&pool, now)
	if err != nil || len(b.Legs) == 0 || b.Legs[0].PriceWad.IsZero() {
		return zero
	}
	u, err := gateExposurePW(&b)
	if err != nil {
		return zero
	}
	room, err := gateRoomNative(&pool, &b, 0, &u)
	if err != nil {
		return zero
	}
	if cap0 := &pool.Loans[0].MaxSwapNotional; !cap0.IsZero() && cap0.Lt(&room) {
		room = *cap0
	}
	funding, err := st.Router.fundingCeiling(0, &b.Physical, &b.Legs[0].PriceWad, now)
	if err != nil {
		return zero
	}
	capacity, err := gateAdd(&b.Legs[0].Liquid, &funding)
	if err != nil {
		return zero
	}
	return *minU(&room, &capacity)
}
