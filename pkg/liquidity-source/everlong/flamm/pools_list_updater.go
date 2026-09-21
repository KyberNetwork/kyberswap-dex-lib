package everlongflamm

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// PoolsListUpdater lists the pools the registry names (hook_registry.go) that the configured FLAMMFactory owns. It
// walks the registered swap hooks of the configured chain to the pools they are bound to, and never reads the
// factory's `pools()` array, which createPool appends to permissionlessly (FLAMMFactory.sol:216-254) and which
// therefore grows without bound: an operator's eth_call gas cap could otherwise stop every listing run. One
// aggregate at the latest block reads, for every such pool, FLAMMFactory.isPool, its hooks() and its PriceFeed; a
// pool the factory owns is listed only if every hook it names is registered, of the kind its slot requires and bound
// to that pool, and its PriceFeed is registered. Every later read is pinned to that block: the pool's pair and
// bindings, what each hook reports it is bound to (by its kind), the runtime codehash of every contract on the swap
// path, the Router venues with their Morpho market params and financing accounts, and the PriceFeed wiring. A pool
// whose wiring is not exactly the registries', or whose views the chain answers with a revert or undecodable data,
// is skipped with a warning; only a transport failure fails the run. The cursor keeps each listed pool's StaticExtra
// digest, so a change of the immutable wiring (a new hook set, a new implementation) is relisted once the registries
// admit it; the mutable venue set lives in Extra and is refreshed by the tracker (pool_tracker.go). A run that finds
// a pool already listed with the wiring the view round read stops there, so a steady-state poll is two round trips
// rather than the six a new or changed listing costs.
type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

// Metadata is the listing cursor: pool -> digest of the StaticExtra it was listed with.
type Metadata struct {
	Listed map[string]string `json:"listed"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{config: config, ethrpcClient: ethrpcClient}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	md := Metadata{Listed: map[string]string{}}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &md); err != nil {
			return nil, metadataBytes, err
		}
		if md.Listed == nil {
			md.Listed = map[string]string{}
		}
	}
	if u.config == nil || u.config.DexID == "" || !common.IsHexAddress(u.config.Factory) {
		return nil, metadataBytes, ErrInvalidProfile
	}
	allowed := map[common.Address]bool{}
	for _, a := range u.config.Pools {
		if !common.IsHexAddress(a) {
			return nil, metadataBytes, ErrInvalidProfile
		}
		allowed[common.HexToAddress(a)] = true
	}

	var out []entity.Pool
	live := map[string]bool{}
	d := deploymentFor(u.config.ChainID, common.HexToAddress(u.config.Factory))
	if d == nil {
		// The same misconfiguration the tracker fails loudly on: a factory, or a chain, the registries do not know.
		log.Ctx(ctx).Warn().Str("dex", DexType).Str("dexID", u.config.DexID).Str("chain", u.config.ChainID.String()).
			Str("factory", u.config.Factory).Msg("factory is not a registered deployment; nothing to list")
	}
	candidates := listingCandidates(u.config.ChainID, d, allowed)
	if len(candidates) == 0 {
		// Nothing to look at. The cursor is what a run that looked prunes -- a listed pool the factory no longer
		// owns (isPool false) -- so a run that read nothing returns it as it was rather than forgetting every pool
		// it holds and relisting them all on the next run.
		return nil, metadataBytes, nil
	}
	rpcc := &mcRPC{client: u.ethrpcClient}
	heads := make([]poolHead, len(candidates))
	plan := &readPlan{}
	for i := range candidates {
		heads[i].reads(plan, d, candidates[i])
	}
	blockNumber, err := plan.run(ctx, rpcc, nil)
	if err != nil {
		return nil, metadataBytes, err
	}
	block := new(big.Int).SetUint64(blockNumber)
	for i, pool := range candidates {
		h := &heads[i]
		if h.ownedRead && !h.owned {
			continue // the factory does not own this pool at this block
		}
		key := lowerHex(pool)
		live[key] = true
		se, venues, digest, err := u.listPool(ctx, rpcc, d, pool, h, block, md.Listed[key])
		if err != nil && !errors.Is(err, ErrInvalidProfile) && !chainAnswered(err) {
			return nil, metadataBytes, err // transport: retry the whole run
		} else if err != nil {
			log.Ctx(ctx).Warn().Str("dex", DexType).Str("dexID", u.config.DexID).Str("pool", key).Err(err).
				Msg("pool does not match the registry; not listed")
			continue
		}
		if se == nil {
			continue // already listed with this wiring
		}
		raw, err := json.Marshal(se)
		if err != nil {
			return nil, metadataBytes, err
		}
		// The venue set is published in Extra, which every refresh re-reads and replaces.
		extra, err := json.Marshal(&Extra{Venues: venues})
		if err != nil {
			return nil, metadataBytes, err
		}
		out = append(out, entity.Pool{
			Address:   key,
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Tokens: []*entity.PoolToken{
				{Address: lowerHex(se.PoolAsset), Swappable: true},
				{Address: lowerHex(se.LoanAsset), Swappable: true},
			},
			Reserves:    entity.PoolReserves{"0", "0"},
			StaticExtra: string(raw),
			Extra:       string(extra),
			BlockNumber: blockNumber,
		})
		md.Listed[key] = digest
	}
	for key := range md.Listed {
		if !live[key] {
			delete(md.Listed, key)
		}
	}
	raw, err := json.Marshal(&md)
	if err != nil {
		return nil, metadataBytes, err
	}
	return out, raw, nil
}

// listingCandidates is every pool a registered swap hook of chainID names d's factory as the creator of, in registry
// order, once each, and within the allow-list when one is configured. A factory the registries do not know has none.
func listingCandidates(chainID valueobject.ChainID, d *flammDeployment,
	allowed map[common.Address]bool) []common.Address {
	if d == nil {
		return nil
	}
	var out []common.Address
	seen := map[common.Address]bool{}
	for i := range hookRegistry {
		e := &hookRegistry[i]
		spec := hookKindSpecOf(e.Kind)
		if e.ChainID != chainID || e.Factory != d.Factory.Address || spec == nil || spec.role() != roleSwap ||
			seen[e.Pool] || (len(allowed) != 0 && !allowed[e.Pool]) {
			continue
		}
		seen[e.Pool] = true
		out = append(out, e.Pool)
	}
	return out
}

// poolHead is the first listing round's answer for one candidate pool: whether the factory owns it, the hook set it
// lists and the PriceFeed it prices through. The round is one aggregate over every candidate, so a read the chain
// answers with a revert or with data that does not decode leaves that pool unread instead of failing the run.
type poolHead struct {
	ownedRead, hooksRead, feedRead bool
	owned                          bool
	hooks                          [7]common.Address
	priceFeed                      common.Address
}

func (h *poolHead) reads(p *readPlan, d *flammDeployment, pool common.Address) {
	p.optional(fmt.Sprintf("factory.isPool(%s)", lowerHex(pool)), d.Factory.Address, &factoryABI, "isPool",
		[]any{pool}, func(vals []any) {
			if v, err := valAt[bool](vals, 0); err == nil {
				h.owned, h.ownedRead = v, true
			}
		})
	p.optional(fmt.Sprintf("pool(%s).hooks", lowerHex(pool)), pool, &flammABI, "hooks", nil, func(vals []any) {
		if len(vals) == 1 {
			if hs, err := tupleOf[abiHookSet](vals[0]); err == nil {
				h.hooks, h.hooksRead = hs.array(), true
			}
		}
	})
	p.optional(fmt.Sprintf("pool(%s).priceFeed", lowerHex(pool)), pool, &flammABI, "priceFeed", nil,
		func(vals []any) {
			if a, err := valAt[common.Address](vals, 0); err == nil {
				h.priceFeed, h.feedRead = a, true
			}
		})
}

// resolve is the head's hook set against chainID's registry for a pool of d, once the PriceFeed is one d registers.
func (h *poolHead) resolve(chainID valueobject.ChainID, d *flammDeployment, pool common.Address) (poolHookSet, error) {
	if !h.ownedRead || !h.hooksRead || !h.feedRead {
		return poolHookSet{}, fmt.Errorf("%w: isPool, hooks or priceFeed", errReadDecode)
	}
	if _, ok := sharedIn(d.PriceFeeds, h.priceFeed); !ok {
		return poolHookSet{}, fmt.Errorf("%w: price feed %s is not registered", ErrInvalidProfile, h.priceFeed.Hex())
	}
	return resolveHooks(chainID, d.Factory.Address, pool, h.hooks)
}

// staticDigest identifies a listing's wiring (the listing block excluded).
func staticDigest(se *StaticExtra) (string, error) {
	c := *se
	c.ListedBlock = 0
	raw, err := json.Marshal(&c)
	if err != nil {
		return "", err
	}
	return crypto.Keccak256Hash(raw).Hex(), nil
}

// listPool reads and verifies one candidate pool at block, whose first-round answer is head, and returns its pinned
// identity, the digest that identifies it, and the venue set the pool has at that block (Extra.Venues, which every
// refresh re-reads).
//
// listed is the digest the pool is already listed with. A pool whose wiring still hashes to it is returned as a nil
// identity and nothing is re-emitted, before the rounds that follow the view round: those verify only what an
// unchanged StaticExtra cannot have changed. Every address whose runtime code they hash is either registered by
// address (the factory, the Router, the implementation, the PriceFeed, the hooks) and part of the digest, or pinned
// by codehash to the registry's (the pool proxy) -- so a beacon upgrade or a new hook set relists the pool and
// re-runs them -- or a financing account, which validVenues binds, on the listing and on every refresh, to a
// registered account of the pool whose code the registry pins. Re-reading all of it on every poll would re-download
// some 196 KB of runtime code for a pool that has not moved, so a steady-state poll stops at the view round and a new
// or changed listing runs everything.
func (u *PoolsListUpdater) listPool(ctx context.Context, rpcc *mcRPC, d *flammDeployment, pool common.Address,
	head *poolHead, block *big.Int, listed string) (*StaticExtra, []StaticVenue, string, error) {
	chainID := u.config.ChainID
	hooks, err := head.resolve(chainID, d, pool)
	if err != nil {
		return nil, nil, "", err
	}
	swap := hooks.Entries[roleSwap]
	se := &StaticExtra{ProfileVersion: profileVersion, ChainID: chainID, Morpho: d.Morpho, ListedBlock: block.Uint64()}
	var (
		loanCount, routerLoans, venueCount uint256.Int
		bindings                           [hookRoles]hookBinding
		configs                            [2]abiFeedToken
	)
	p := &readPlan{}
	p.add("pool.asset", pool, &flammABI, "asset", nil, readAddr(&se.PoolAsset))
	p.add("pool.loanAsset", pool, &flammABI, "loanAsset", nil, readAddr(&se.LoanAsset))
	p.add("pool.loanCount", pool, &flammABI, "loanCount", nil, readWord(&loanCount))
	p.add("pool.router", pool, &flammABI, "router", nil, readAddr(&se.Router))
	p.add("pool.priceFeed", pool, &flammABI, "priceFeed", nil, readAddr(&se.PriceFeed))
	p.add("pool.factory", pool, &flammABI, "factory", nil, readAddr(&se.Factory))
	p.add("pool.hooks", pool, &flammABI, "hooks", nil, func(vals []any) error {
		h, err := tupleOf[abiHookSet](vals[0])
		se.Hooks = h.array()
		return err
	})
	p.add("factory.implementation", d.Factory.Address, &factoryABI, "implementation", nil,
		readAddr(&se.Implementation))
	for role := roleSwap; role < hookRoles; role++ {
		if e := hooks.Entries[role]; e != nil {
			hookKindSpecOf(e.Kind).listingReads(p, e.Address, &bindings[role])
		}
	}
	p.add("router.loanCount", d.Router.Address, &routerABI, "loanCount", []any{pool}, readWord(&routerLoans))
	p.add("router.venueCount", d.Router.Address, &routerABI, "venueCount", []any{pool}, readWord(&venueCount))
	p.add("feed.SEQUENCER_FEED", head.priceFeed, &priceFeedABI, "SEQUENCER_FEED", nil, readAddr(&se.SequencerFeed))
	for i, token := range []common.Address{swap.PoolAsset, swap.LoanAsset} {
		p.add(fmt.Sprintf("feed.config(%d)", i), head.priceFeed, &priceFeedABI, "config", []any{token},
			func(vals []any) (err error) {
				configs[i], err = tupleOf[abiFeedToken](vals[0])
				return err
			})
	}
	if _, err := p.run(ctx, rpcc, block); err != nil {
		return nil, nil, "", err
	}
	implementation, implKnown := sharedIn(d.Implementations, se.Implementation)
	switch {
	case se.PoolAsset != swap.PoolAsset || se.LoanAsset != swap.LoanAsset || !loanCount.Eq(uOne) ||
		!routerLoans.Eq(uOne):
		return nil, nil, "", fmt.Errorf("%w: pair or loan set", ErrInvalidProfile)
	case se.Router != d.Router.Address || se.PriceFeed != head.priceFeed || se.Factory != d.Factory.Address ||
		!implKnown:
		return nil, nil, "", fmt.Errorf("%w: bindings", ErrInvalidProfile)
	case se.Hooks != hooks.Addrs:
		return nil, nil, "", fmt.Errorf("%w: hook set", ErrInvalidProfile)
	}
	for role := roleSwap; role < hookRoles; role++ {
		if !hooks.bound(role, &bindings[role], pool) {
			return nil, nil, "", fmt.Errorf("%w: %s binding", ErrInvalidProfile, roleName[role])
		}
	}
	if !venueCount.IsUint64() || venueCount.IsZero() || venueCount.Uint64() > maxVenues {
		return nil, nil, "", fmt.Errorf("%w: venues", ErrInvalidProfile)
	}
	se.Aggregators = [2]common.Address{configs[0].Aggregator, configs[1].Aggregator}
	se.Heartbeats = [2]uint64{uint64(configs[0].Heartbeat), uint64(configs[1].Heartbeat)}

	// The identity is complete here; the rounds below verify it, which a pool already listed with it needs no more.
	digest, err := staticDigest(se)
	if err != nil {
		return nil, nil, "", err
	}
	if digest == listed {
		return nil, nil, digest, nil
	}

	venues, err := readVenues(ctx, rpcc, se.venueScope(pool), int(venueCount.Uint64()), block)
	if err != nil {
		return nil, nil, "", err
	}

	// Runtime code of everything the port mirrors, at the same block.
	type codeCheck struct {
		name string
		addr common.Address
		hash common.Hash
	}
	feed, _ := sharedIn(d.PriceFeeds, head.priceFeed)
	codes := []codeCheck{{"pool", pool, d.PoolCodeHash}, {"factory", d.Factory.Address, d.Factory.CodeHash},
		{"implementation", implementation.Address, implementation.CodeHash}}
	for role := roleSwap; role < hookRoles; role++ {
		if e := hooks.Entries[role]; e != nil {
			codes = append(codes, codeCheck{roleCodeName[role], e.Address, e.CodeHash})
		}
	}
	codes = append(codes, codeCheck{"router", d.Router.Address, d.Router.CodeHash},
		codeCheck{"priceFeed", feed.Address, feed.CodeHash})
	for i := range venues {
		var hash common.Hash // an account the registry does not know has no code to match
		if a := financingAccount(chainID, venues[i].Account); a != nil {
			hash = a.CodeHash
		}
		codes = append(codes, codeCheck{fmt.Sprintf("account(%d)", i), venues[i].Account, hash})
	}
	got := make([]hexutil.Bytes, len(codes))
	batch := make([]rpc.BatchElem, len(codes))
	for i := range codes {
		batch[i] = rpc.BatchElem{Method: "eth_getCode", Args: []any{codes[i].addr, hexutil.EncodeBig(block)},
			Result: &got[i]}
	}
	if err := batchCall(ctx, rpcc.client, batch); err != nil {
		return nil, nil, "", err
	}
	for i := range codes {
		if batch[i].Error != nil {
			return nil, nil, "", batch[i].Error
		}
		if crypto.Keccak256Hash(got[i]) != codes[i].hash {
			return nil, nil, "", fmt.Errorf("%w: %s codehash", ErrInvalidProfile, codes[i].name)
		}
	}
	w, err := validStatic(se, lowerHex(pool), DexType, []string{lowerHex(se.PoolAsset), lowerHex(se.LoanAsset)})
	if err != nil {
		return nil, nil, "", err
	}
	if err := validVenues(w, venues); err != nil {
		return nil, nil, "", fmt.Errorf("%w: venue set", err)
	}
	return se, venues, digest, nil
}

// venueScope addresses one pool's venue reads: the pool, its Router, the Morpho singleton and the pair every venue
// market must price.
type venueScope struct {
	Pool, Router, Morpho, PoolAsset, LoanAsset common.Address
}

func (se *StaticExtra) venueScope(pool common.Address) venueScope {
	return venueScope{Pool: pool, Router: se.Router, Morpho: se.Morpho, PoolAsset: se.PoolAsset,
		LoanAsset: se.LoanAsset}
}

// readVenues reads the pool's n Router venues at block: each venue's financing account and Morpho market id, the
// immutable market params behind that id, whose pair must be the pool's, and the price feeds the market oracle is
// wired to. Three rounds, because each one's targets come from the one before.
func readVenues(ctx context.Context, rpcc *mcRPC, scope venueScope, n int, block *big.Int) ([]StaticVenue, error) {
	venues := make([]StaticVenue, n)
	p := &readPlan{}
	for i := 0; i < n; i++ {
		p.add(fmt.Sprintf("router.venue(%d)", i), scope.Router, &routerABI, "venue", []any{scope.Pool, uint16(i)},
			func(vals []any) error {
				v, err := tupleOf[abiVenueView](vals[0])
				venues[i].Account, venues[i].MarketID = v.Account, v.Id
				return err
			})
	}
	if _, err := p.run(ctx, rpcc, block); err != nil {
		return nil, err
	}
	p = &readPlan{}
	for i := 0; i < n; i++ {
		v := &venues[i]
		p.add(fmt.Sprintf("morpho.idToMarketParams(%d)", i), scope.Morpho, &morphoABI, "idToMarketParams",
			[]any{v.MarketID}, func(vals []any) error {
				mp, err := tupleOf[abiMarketParams](vals[0])
				if err != nil {
					return err
				}
				if mp.LoanToken != scope.LoanAsset || mp.CollateralToken != scope.PoolAsset {
					return fmt.Errorf("%w: venue %d market pair", ErrInvalidProfile, i)
				}
				v.Oracle, v.Irm = mp.Oracle, mp.Irm
				v.Lltv, err = wordOf(mp.Lltv)
				return err
			})
	}
	if _, err := p.run(ctx, rpcc, block); err != nil {
		return nil, err
	}
	if err := readOracleFeeds(ctx, rpcc, venues, block); err != nil {
		return nil, err
	}
	return venues, nil
}

// readVenueSet is readVenues preceded by the venue count, for a refresh that has no venue set to start from or
// whose set the pool no longer agrees with (pool_tracker.go). It returns the block the reads ran at, which pins
// the count and the venues together; a nil block reads the latest and pins the rest of the refresh to it.
func readVenueSet(ctx context.Context, rpcc *mcRPC, scope venueScope, block *big.Int) ([]StaticVenue, uint64,
	error) {
	var count uint256.Int
	p := &readPlan{}
	p.add("router.venueCount", scope.Router, &routerABI, "venueCount", []any{scope.Pool}, func(vals []any) error {
		return wordsInto(vals, &count)
	})
	at, err := p.run(ctx, rpcc, block)
	if err != nil {
		return nil, at, err
	}
	if !count.IsUint64() || count.IsZero() || count.Uint64() > maxVenues {
		return nil, at, fmt.Errorf("%w: venues", errReadDecode)
	}
	venues, err := readVenues(ctx, rpcc, scope, int(count.Uint64()), new(big.Int).SetUint64(at))
	return venues, at, err
}
