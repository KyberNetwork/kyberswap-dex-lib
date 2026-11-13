package everlongflamm

import (
	"cmp"
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

// PoolTracker probes the pool's own quote, `previewSwap(tokenIn, tokenOut, amountIn)`, at a ladder of sizes in each
// direction the pool trades and this source routes, every read pinned to one block and every probe made at the
// taker, and settles each direction's probes by its pricer's rule (settle). The probes run in one eth_call, the
// sampler's code (FlammSampler) set at the taker by a state override; a node that does not run it gets them one
// eth_call each (the plain path). The quote runs the pool's swap plan, its pricing hook, its price feed and its
// lending venues, so no hook, feed or venue math lives here: of a hook it reads only its fixed parameters, the
// curve's fee and, on a buy, its leverage state's top bid.
type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client

	// samplerRefusedUntilMs: until then, unix milliseconds, every refresh takes the plain path, the node having
	// refused the sampler samplerFailuresToCooldown times in a row. samplerFailures counts those refusals since the
	// last good answer, from any pool.
	samplerRefusedUntilMs atomic.Int64
	samplerFailures       atomic.Int32

	mu sync.RWMutex
	// family is the configured family of the code each contract seen runs (-1: none). A contract's code never
	// changes in place, so it is kept.
	family map[common.Address]int
	// rows is each hook's leverage row as its params() decoded, fixed at the hook's construction, so kept; a hook
	// whose read failed has none here.
	rows map[common.Address]hookRow
	// caps are the sampler's per-probe caps raised past the default, by pool (lower case): a pool whose probe
	// starved is asked at twice its cap, up to maxCap, until an answer spends under 7/8 of the default on every
	// probe.
	caps map[string]uint64
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

var (
	_ pool.IPoolTrackerWithSchedule  = (*PoolTracker)(nil)
	_ pool.IPoolTrackerWithOverrides = (*PoolTracker)(nil)
)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, ethrpcClient: ethrpcClient, family: map[common.Address]int{},
		rows: map[common.Address]hookRow{}, caps: map[string]uint64{}}
}

// poolState is the refresh's first read, at one block.
type poolState struct {
	block *big.Int
	// closed: a gate read refuses the pool whatever the other reads gave: no pool of its Factory, paused, or an
	// implementation or a price feed the config does not name.
	closed   bool
	isPool   bool
	paused   bool
	version  uint32
	tradable tradableResult
	hooks    hooksResult
	pairHook pairHookResult
	lane     laneResult
	router   common.Address
	feed     common.Address
	tokens   []tokenConfigResult
	// venues are the pool's live lending venues (neither retired nor fenced), for the kind gate and the charge;
	// idleDebt has bit t set where a venue that is not live lends token t; venueUnread: a record did not read, or
	// the count passed maxVenues, so no pair is routed.
	venues      []liveVenue
	idleDebt    uint16
	venueUnread bool
}

// liveVenue is a live venue's record and the actions it allows now (venueActions); actionsOk false: that read
// failed, which counts as a refusal of every action. held: its account holds a minimum position (capabilities() has
// capHeldRepay), floor that minimum (minimumDebt(id)), nil where it did not read or answered not ok.
type liveVenue struct {
	venueView
	actions   uint8
	actionsOk bool
	held      bool
	floor     *big.Int
}

// direction is one ordered pair the pool trades now and this source routes, with what settles its ladder: its
// pricer, the pricer's rule and leverage row, and whether that row is trusted, passing rowOk at the live band on a
// pool of exactly two tokens.
type direction struct {
	in, out int
	hook    common.Address
	rule    settleRule
	row     hookRow
	trusted bool
	// the charge's (gas.go): the cells of the pair's live kinds, their least second-fill charge and their cells after
	// another pool of the source filled, whether a first borrow of the output may happen while it owes, and whether a
	// repay of the input is certain where it owes; and the live venue holding a minimum position that lends the input
	// or the output, where one does (guard.go)
	cells, after        []Cell
	second              int64
	firstOwing, repayOk bool
	heldIn, heldOut     *heldRepay
}

// pricerVerdict is what a refresh finds of one pricer: whether the Factory lists it as a front, whether its pairs
// pass the code gates, the build its code is of (its family's CodeHash, which keys the gas table), and the rule and
// row its ladders settle by.
type pricerVerdict struct {
	front, routed bool
	family        string
	rule          settleRule
	row           hookRow
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.track(ctx, p, nil)
}

// GetNewPoolStateWithOverrides is a refresh on the state the overrides set: they reach every read, and the
// sampler's code takes the taker's place over any override there.
func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.track(ctx, p, params.Overrides)
}

// track is one refresh: the gate reads at one block, the directions routed, and each direction's ladder.
func (t *PoolTracker) track(ctx context.Context, p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	n := len(p.Tokens)
	if n < 2 || n > maxTokens {
		return p, ErrInvalidState
	}
	var se StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &se); err != nil {
		return p, err
	}
	if !t.cfg.hasFactory(se.Factory) {
		return dark(p, nil), nil
	}

	st, err := t.read(ctx, p.Address, se.Factory, n, overrides)
	if err != nil {
		return p, err
	}
	if st.closed {
		return dark(p, st.block), nil
	}
	for k, tc := range st.tokens {
		if !strings.EqualFold(hexutil.Encode(tc.Config.Token[:]), p.Tokens[k].Address) {
			return p, ErrInvalidToken
		}
	}

	if err := t.readVenues(ctx, p.Address, st, overrides); err != nil {
		return p, err
	}
	dirs, err := t.directions(ctx, se.Factory, n, st, overrides)
	if err != nil {
		return p, err
	}
	var prev Extra
	_ = json.Unmarshal([]byte(p.Extra), &prev) // an earlier refresh's ladders guide the sizes; none is fine
	ladders, err := t.ladders(ctx, p.Address, st, dirs, &prev, overrides)
	if err != nil {
		return p, err
	}

	now := time.Now().Unix()
	e := Extra{Ladders: ladders, HaircutBps: t.cfg.haircutBps()}
	if len(ladders) != 0 {
		e.LiveAt = now
	}
	extra, err := json.Marshal(e)
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = rooms(n, ladders)
	p.BlockNumber = st.block.Uint64()
	p.Timestamp = now
	return p, nil
}

// NextUpdateAt asks for the next refresh refreshSeconds after the last, event or none.
func (t *PoolTracker) NextUpdateAt(p entity.Pool) int64 { return p.Timestamp + refreshSeconds }

// read is the refresh's first multicall: the Factory's provenance, the switches, the trade mask, both pricers, the
// cheap lane's state, the Router, the price feed and every token's config.
func (t *PoolTracker) read(ctx context.Context, poolAddr, factory string, n int,
	overrides map[common.Address]gethclient.OverrideAccount) (*poolState, error) {
	st := &poolState{tokens: make([]tokenConfigResult, n)}
	pl := common.HexToAddress(poolAddr)
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: factory, Method: "isPool", Params: []any{pl}},
			[]any{&st.isPool}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "paused"}, []any{&st.paused}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "poolCodeVersion"}, []any{&st.version}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "tradablePairs"}, []any{&st.tradable}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "hooks"}, []any{&st.hooks}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "pairHook"}, []any{&st.pairHook}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "lane"}, []any{&st.lane}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "router"}, []any{&st.router}).
		AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "priceFeed"}, []any{&st.feed})
	for k := range n {
		req.AddCall(&ethrpc.Call{ABI: poolABI, Target: poolAddr, Method: "tokenConfig", Params: []any{uint8(k)}},
			[]any{&st.tokens[k]})
	}
	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, err
	}
	if res.BlockNumber == nil || len(res.Result) != 9+n {
		return nil, ErrRead
	}
	st.block = res.BlockNumber
	if r := res.Result; r[0] && !st.isPool || r[1] && st.paused || r[2] && !t.cfg.codeVersionOk(st.version) ||
		r[8] && !t.cfg.hasFeed(st.feed) {
		st.closed = true
		return st, nil
	}
	if slices.Contains(res.Result, false) ||
		st.tradable.Tradable == nil || st.pairHook.Mask == nil || st.lane.LaneConfig == nil {
		return nil, ErrRead
	}
	for _, tc := range st.tokens {
		if tc.Config.MaxSwapNotional == nil {
			return nil, ErrRead
		}
	}
	return st, nil
}

// directions applies the gates every routed pair passes: the trade mask, the front check, the pricer's code family
// and the shape it states, and the kind gate (charges): the pair's live venues are of kinds the chain routes, with
// gas cells for the pricer's build. The pricer of a pair is the pool's pair hook where its mask names the pair, else
// the swap hook. A pricer of the everlong shape prices its hook pair only, tokens 0 and 1, which come first.
func (t *PoolTracker) directions(ctx context.Context, factory string, n int, st *poolState,
	overrides map[common.Address]gethclient.OverrideAccount) ([]direction, error) {
	swapHook, laneOn := st.hooks.Set.Hook, laneVersion(st.lane.LaneConfig) != 0
	liveBand := new(big.Int).SetUint64(st.tokens[1].Config.SwapPriceBandWad) // the hook pair's band: token 0 has none
	verdicts := map[common.Address]pricerVerdict{}
	var dirs []direction
	for i := range n {
		for j := range n {
			if i == j || st.tradable.Tradable.Bit(16*i+j) == 0 {
				continue
			}
			pricer := swapHook
			if i < pairHookTokens && j < pairHookTokens && st.pairHook.Hook != (common.Address{}) &&
				st.pairHook.Mask.Bit(pairBit(i, j)) == 1 {
				pricer = st.pairHook.Hook
			}
			if pricer == (common.Address{}) {
				continue
			}
			// A front prices the swap hook's pairs of a pool whose cheap lane is on, for good, though the Factory
			// may delist it: this build routes no pair a front prices, whatever SkipFrontPriced says.
			if laneOn && pricer == swapHook {
				continue
			}
			v, seen := verdicts[pricer]
			if !seen {
				var err error
				if v, err = t.judge(ctx, factory, pricer, st.block, overrides); err != nil {
					return nil, err
				}
				verdicts[pricer] = v
			}
			if !v.routed || v.rule == ruleEverlong && (i > 1 || j > 1) {
				continue
			}
			c, ok := t.charges(st, v.family, i, j)
			if !ok {
				continue
			}
			dirs = append(dirs, direction{in: i, out: j, hook: pricer, rule: v.rule, row: v.row,
				trusted: v.rule == ruleEverlong && n == 2 && rowOk(v.row, liveBand), cells: c.cells,
				after: c.after, second: c.second, firstOwing: c.firstOwing, repayOk: c.repayOk, heldIn: c.heldIn,
				heldOut: c.heldOut})
		}
	}
	rest := func(d direction) int { return min(max(d.in, d.out)/2, 1) } // 0 on the hook pair, tokens 0 and 1
	slices.SortStableFunc(dirs, func(a, b direction) int { return cmp.Compare(rest(a), rest(b)) })
	return dirs, nil
}

// judge reads what the refresh needs of a pricer, once. A pricer the Factory lists is a front, not routed by this
// build whatever SkipFrontPriced or AnyHook say: that check comes first, before the family and params() reads. A
// family's pricer is routed under the shape the family states, with the leverage row its params() gives; a pricer
// of no family has no gas cells and is not routed.
func (t *PoolTracker) judge(ctx context.Context, factory string, pricer common.Address, block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) (pricerVerdict, error) {
	var v pricerVerdict
	var err error
	if v.front, err = t.isListedFront(ctx, factory, pricer, block, overrides); err != nil || v.front {
		return v, err
	}
	var family int
	if family, v.routed, err = t.familyOfHook(ctx, pricer, false, block); err != nil || !v.routed {
		return v, err
	}
	f := t.cfg.families()[family]
	if f.Shape != shapeEverlong {
		logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "hook": hexutil.Encode(pricer[:])}).
			Debug("a hook of a family that states no shape this source settles: its pairs are not routed")
		v.routed = false
		return v, nil
	}
	row, decoded, err := t.row(ctx, pricer, block)
	if err != nil {
		return v, err
	}
	if !decoded {
		// the family's code answers params(): a revert or a short answer is the node's, never "no row", which would
		// skip the room test on the tier points
		return v, fmt.Errorf("%w: a listed hook's params()", ErrRead)
	}
	v.rule, v.row, v.family = ruleEverlong, row, f.CodeHash
	return v, nil
}

// row is the leverage row of `hook`'s params(), words 27-34, read pinned to the block and kept once decoded, as the
// row is fixed at the hook's construction. A revert, or an answer short of its 35 words, is asked again once;
// decoded false: the second failed too, and nothing is kept, so one spurious revert cannot settle a hook for good.
// Any other error is the node's.
func (t *PoolTracker) row(ctx context.Context, hook common.Address, block *big.Int) (hookRow, bool, error) {
	t.mu.RLock()
	r, seen := t.rows[hook]
	t.mu.RUnlock()
	if seen {
		return r, true, nil
	}
	data, err := hookABI.Pack("params")
	if err != nil {
		return hookRow{}, false, err
	}
	for range 2 {
		ret, err := t.ethrpcClient.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &hook, Data: data}, block)
		if err != nil {
			if !reverted(err) {
				return hookRow{}, false, err
			}
			continue
		}
		out, err := hookABI.Unpack("params", ret)
		if err != nil {
			continue
		}
		w := *abi.ConvertType(out[0], new([paramsWords]*big.Int)).(*[paramsWords]*big.Int)
		r = hookRow{spread: w[27], spreadDn: w[28], upCap: w[29], dnCap: w[30], ceiling: w[31], band: w[32],
			feedGate: w[33], upFloor: w[34]}
		t.mu.Lock()
		t.rows[hook] = r
		t.mu.Unlock()
		return r, true, nil
	}
	return hookRow{}, false, nil
}

func (t *PoolTracker) isListedFront(ctx context.Context, factory string, hook common.Address, block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) (bool, error) {
	var listed bool
	_, err := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(block).SetOverrides(overrides).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: factory, Method: "isListedFront", Params: []any{hook}},
			[]any{&listed}).Call()
	return listed, err
}

// familyOfHook is the configured family `hook`'s code is of (-1: none), and whether it is routed: a configured
// family's, not a front's. Only a front the Factory lists would be told by the implementation in its ERC-1967 slot,
// a branch this build never takes, as it routes no front: any pool creator's contract can hold any word there and
// any bytes in its code, so every other hook is judged by its own code, where a front build does not pass.
func (t *PoolTracker) familyOfHook(ctx context.Context, hook common.Address, front bool,
	block *big.Int) (int, bool, error) {
	code := hook
	if front {
		slot, err := t.ethrpcClient.GetETHClient().StorageAt(ctx, hook, erc1967Slot, block)
		if err != nil {
			return -1, false, err
		}
		if impl := common.BytesToAddress(slot); impl != (common.Address{}) {
			code = impl
		}
	}
	f, err := t.familyAt(ctx, code, block)
	if err != nil {
		return -1, false, err
	}
	if f >= 0 && (front || !t.cfg.families()[f].Front) {
		return f, true, nil
	}
	logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "hook": hexutil.Encode(hook[:])}).
		Debug("a hook of no configured family: its pairs are not routed")
	return -1, false, nil
}

// readVenues reads the Router's record of the pool's venues at the refresh block, each with the actions it allows
// now (venueActions): the count and the first records in one multicall, the rest in one more. A record that does
// not read, or a count past maxVenues, leaves every pair unrouted (venueUnread).
func (t *PoolTracker) readVenues(ctx context.Context, poolAddr string, st *poolState,
	overrides map[common.Address]gethclient.OverrideAccount) error {
	if st.router == (common.Address{}) {
		return nil
	}
	pl, router := common.HexToAddress(poolAddr), hexutil.Encode(st.router[:])
	var count *big.Int
	views := make([]venueResult, maxVenues)
	acts := make([]uint8, maxVenues)
	read := func(lo, hi int, withCount bool) ([]bool, error) {
		req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(st.block).SetOverrides(overrides)
		if withCount {
			req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "venueCount", Params: []any{pl}},
				[]any{&count})
		}
		for id := lo; id < hi; id++ {
			req.AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "venue", Params: []any{pl, uint16(id)}},
				[]any{&views[id]}).
				AddCall(&ethrpc.Call{ABI: routerABI, Target: router, Method: "venueActions",
					Params: []any{pl, uint16(id)}}, []any{&acts[id]})
		}
		res, err := req.TryAggregate()
		if err != nil {
			return nil, err
		}
		return res.Result, nil
	}
	ok, err := read(0, venueWindow, true)
	if err != nil {
		return err
	}
	if !ok[0] || count == nil {
		st.venueUnread = true
		return nil
	}
	n := maxVenues
	if count.IsUint64() && count.Uint64() <= maxVenues {
		n = int(count.Uint64())
	} else {
		st.venueUnread = true
	}
	ok = ok[1:]
	if n > venueWindow {
		more, err := read(venueWindow, n, false)
		if err != nil {
			return err
		}
		ok = append(ok, more...)
	}
	for id := range n {
		switch v := views[id].View; {
		case !ok[2*id]:
			st.venueUnread = true
		case v.Status == 0:
			st.venues = append(st.venues, liveVenue{venueView: v, actions: acts[id], actionsOk: ok[2*id+1]})
		case v.Debt < maxTokens:
			st.idleDebt |= 1 << v.Debt
		}
	}
	if st.venueUnread || len(st.venues) == 0 {
		return nil
	}
	return t.readHeld(ctx, st, overrides)
}

// readHeld reads, at the refresh block in one multicall, each live venue's account's capabilities() and its
// minimumDebt(id), which only an account holding a minimum position answers. A capabilities() that does not read
// leaves every pair unrouted, as an unread record does: whether that venue needs the guard is unknown.
func (t *PoolTracker) readHeld(ctx context.Context, st *poolState,
	overrides map[common.Address]gethclient.OverrideAccount) error {
	type minimum struct {
		Ok    bool
		Floor *big.Int
	}
	caps := make([]*big.Int, len(st.venues))
	mins := make([]minimum, len(st.venues))
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(st.block).SetOverrides(overrides)
	for k, v := range st.venues {
		account := hexutil.Encode(v.Account[:])
		req.AddCall(&ethrpc.Call{ABI: accountABI, Target: account, Method: "capabilities"}, []any{&caps[k]}).
			AddCall(&ethrpc.Call{ABI: accountABI, Target: account, Method: "minimumDebt", Params: []any{v.Id}},
				[]any{&mins[k]})
	}
	res, err := req.TryAggregate()
	if err != nil {
		return err
	}
	for k := range st.venues {
		if !res.Result[2*k] || caps[k] == nil {
			st.venueUnread = true
			return nil
		}
		v := &st.venues[k]
		v.held = new(big.Int).And(caps[k], big.NewInt(capHeldRepay)).Sign() != 0
		if v.held && res.Result[2*k+1] && mins[k].Ok && mins[k].Floor != nil {
			v.floor = mins[k].Floor
		}
	}
	return nil
}

// familyAt is the configured family of the code at `a` (-1: none), read once.
func (t *PoolTracker) familyAt(ctx context.Context, a common.Address, block *big.Int) (int, error) {
	t.mu.RLock()
	f, seen := t.family[a]
	t.mu.RUnlock()
	if seen {
		return f, nil
	}
	code, err := t.ethrpcClient.GetETHClient().CodeAt(ctx, a, block)
	if err != nil {
		return -1, err
	}
	f = familyOf(t.cfg.families(), code)
	if len(code) != 0 {
		t.mu.Lock()
		t.family[a] = f
		t.mu.Unlock()
	}
	return f, nil
}

// familyOf is the index of the first family whose hash the code has with that family's immutables zeroed, or -1.
func familyOf(families []HookFamily, code []byte) int {
	if len(code) == 0 {
		return -1
	}
	for k, f := range families {
		masked := slices.Clone(code)
		ok := true
		for _, r := range f.Immutables {
			if r[0] < 0 || r[1] < 0 || r[0]+r[1] > len(masked) {
				ok = false
				break
			}
			clear(masked[r[0] : r[0]+r[1]])
		}
		if ok && strings.EqualFold(crypto.Keccak256Hash(masked).Hex(), f.CodeHash) {
			return k
		}
	}
	return -1
}

// dark is the pool with nothing routed: no ladder, no room.
func dark(p entity.Pool, block *big.Int) entity.Pool {
	extra, _ := json.Marshal(Extra{Ladders: []Ladder{}})
	p.Extra = string(extra)
	p.Reserves = rooms(len(p.Tokens), nil)
	if block != nil {
		p.BlockNumber = block.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p
}

// rooms is each token's payout room: the most any routed direction's ladder pays of it.
func rooms(n int, ladders []Ladder) entity.PoolReserves {
	top := make([]*big.Int, n)
	for _, l := range ladders {
		if k := len(l.Points); k != 0 {
			y := l.Points[k-1][1].ToBig()
			if top[l.Out] == nil || y.Cmp(top[l.Out]) > 0 {
				top[l.Out] = y
			}
		}
	}
	r := make(entity.PoolReserves, n)
	for k := range r {
		r[k] = "0"
		if top[k] != nil {
			r[k] = top[k].String()
		}
	}
	return r
}

func laneVersion(laneConfig *big.Int) uint64 {
	return new(big.Int).Rsh(laneConfig, laneVersionShift).Uint64() & laneVersionMask
}

// pairBit is the pair-hook mask's bit of the unordered pair {i, j} (FLAMMStore.pairHookBit): from bit 160, the pair
// lo < hi at 160 + hi*(hi-1)/2 + lo.
func pairBit(i, j int) int {
	lo, hi := min(i, j), max(i, j)
	return 160 + hi*(hi-1)/2 + lo
}
