package lotflow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"golang.org/x/sync/errgroup"
)

// Track reads through dex-lib's ethrpc multicall (tryAggregate at the pinned block), not one eth_call
// per word. The reads of one venue form a dependency chain (hook wiring -> constituents -> registry
// entries -> pools and feeds -> tick bitmaps -> ticks), so Track runs breadth-first PASSES of the
// RPCLoader's own read methods in collect mode: a read the cache lacks is recorded and answered with
// zeros, and everything a pass recorded is fetched in one multicall round. Then the real load runs on
// the warm cache through the same decode code, so the multicall path decodes exactly what the
// per-call path does. A read the passes did not predict is fetched on its own (counted in Stats).

const (
	mcChunk    = 400 // sub-calls per tryAggregate request
	mcParallel = 4   // concurrent requests in one round
	mcAttempts = 3   // per request, rate limiting only
	mcTimeout  = 20 * time.Second
	maxPasses  = 12
	// maxRingPrefetch: an oracle ring up to this size is fetched whole in one round, so the TWAP
	// binary search runs on the cache; a larger ring is read lazily (one round per probe).
	maxRingPrefetch = 1024
)

// MulticallStats counts one Track's reads.
type MulticallStats struct {
	Rounds   int // multicall rounds (each may span several requests)
	Requests int // tryAggregate HTTP requests
	Calls    int // sub-calls fetched
	Misses   int // reads the passes did not predict (fetched alone)
	Passes   int // collect passes run
	Rechecks int // failed sub-calls re-read on their own
}

type mcResult struct {
	ok  bool
	ret []byte
}

type multicaller struct {
	rc    *ethrpc.Client
	block *big.Int

	mu      sync.Mutex
	cache   map[string]mcResult
	collect map[string]*callReq // non-nil during a collect pass
	stats   MulticallStats
}

func newMulticaller(rc *ethrpc.Client, block *big.Int) *multicaller {
	return &multicaller{rc: rc, block: block, cache: map[string]mcResult{}}
}

func mcKey(to string, data []byte) string { return lc(to) + ":" + hexOf(data) }

var tryAggregateOut = func() abi.Arguments {
	a, err := abi.JSON(strings.NewReader(`[{"name":"tryAggregate","type":"function","inputs":[],"outputs":[
{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return a.Methods["tryAggregate"].Outputs
}()

// rawCall makes ethrpc pack data verbatim: a method whose ID is the whole calldata and that has no
// inputs or outputs. Results are decoded from the raw tryAggregate response instead.
func rawCall(to string, data []byte) *ethrpc.Call {
	return &ethrpc.Call{ABI: abi.ABI{Methods: map[string]abi.Method{"raw": {ID: data, Outputs: abi.Arguments{}}}},
		Target: to, Method: "raw"}
}

// get answers one read: from the cache, from the collect pass (zeros), or with a single fetch.
func (m *multicaller) get(ctx context.Context, to string, data []byte) ([]byte, error) {
	k := mcKey(to, data)
	m.mu.Lock()
	r, ok := m.cache[k]
	if !ok && m.collect != nil {
		if m.collect[k] == nil {
			m.collect[k] = &callReq{to: to, data: data}
		}
		m.mu.Unlock()
		return make([]byte, 32*8), nil
	}
	m.mu.Unlock()
	if !ok {
		m.mu.Lock()
		m.stats.Misses++
		m.mu.Unlock()
		if err := m.fetch(ctx, []*callReq{{to: to, data: data}}); err != nil {
			return nil, err
		}
		m.mu.Lock()
		r = m.cache[k]
		m.mu.Unlock()
	}
	if !r.ok {
		return nil, fmt.Errorf("%w: %s %s", ErrReverted, to, hexOf(data[:min(4, len(data))]))
	}
	return r.ret, nil
}

// fetch reads every request the cache lacks in one round of tryAggregate requests.
func (m *multicaller) fetch(ctx context.Context, reqs []*callReq) error {
	m.mu.Lock()
	seen := map[string]bool{}
	var todo []*callReq
	for _, r := range reqs {
		k := mcKey(r.to, r.data)
		if _, ok := m.cache[k]; ok || seen[k] {
			continue
		}
		seen[k] = true
		todo = append(todo, r)
	}
	m.mu.Unlock()
	if len(todo) == 0 {
		return nil
	}
	sort.Slice(todo, func(i, j int) bool { return mcKey(todo[i].to, todo[i].data) < mcKey(todo[j].to, todo[j].data) })
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(mcParallel)
	for i := 0; i < len(todo); i += mcChunk {
		chunk := todo[i:min(i+mcChunk, len(todo))]
		g.Go(func() error { return m.aggregate(gctx, chunk) })
	}
	err := g.Wait()
	m.mu.Lock()
	m.stats.Rounds++
	var failed []*callReq
	for _, r := range todo {
		if c, ok := m.cache[mcKey(r.to, r.data)]; ok && !c.ok {
			failed = append(failed, r)
		}
	}
	m.mu.Unlock()
	if err != nil || len(failed) == 0 {
		return err
	}
	return m.recheck(ctx, failed)
}

// recheck re-reads failed sub-calls one by one (one JSON-RPC batch of plain eth_calls at the pinned
// block). A sub-call can fail inside tryAggregate without reverting on its own (out of gas in a
// large batch), so only the node's "execution reverted" on the plain call stays a revert; a success
// replaces the cached failure and any other error is a transport failure.
func (m *multicaller) recheck(ctx context.Context, reqs []*callReq) error {
	block := hexutil.EncodeBig(m.block)
	rets := make([]hexutil.Bytes, len(reqs))
	elems := make([]rpc.BatchElem, len(reqs))
	for i, r := range reqs {
		elems[i] = rpc.BatchElem{Method: "eth_call", Result: &rets[i], Args: []any{map[string]any{
			"to": r.to, "data": hexutil.Bytes(r.data)}, block}}
	}
	var err error
	for attempt := 0; attempt < mcAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &TransportError{Err: ctx.Err()}
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		rctx, cancel := context.WithTimeout(ctx, mcTimeout)
		err = m.rc.GetETHClient().Client().BatchCallContext(rctx, elems)
		cancel()
		m.mu.Lock()
		m.stats.Requests++
		m.mu.Unlock()
		if !rateLimited(err) {
			break
		}
	}
	if err != nil {
		return &TransportError{Err: err}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.Rechecks += len(reqs)
	for i, r := range reqs {
		switch e := classifyRPC(elems[i].Error); {
		case e == nil:
			m.cache[mcKey(r.to, r.data)] = mcResult{ok: true, ret: rets[i]}
		case isTransport(e):
			return e
		}
	}
	return nil
}

func (m *multicaller) aggregate(ctx context.Context, chunk []*callReq) error {
	var res *ethrpc.Response
	var err error
	for attempt := 0; attempt < mcAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return &TransportError{Err: ctx.Err()} // no answer: a transport failure, not a revert
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		rctx, cancel := context.WithTimeout(ctx, mcTimeout)
		req := m.rc.NewRequest().SetContext(rctx).SetBlockNumber(m.block)
		for _, c := range chunk {
			req.AddCall(rawCall(c.to, c.data), []any{new(struct{})})
		}
		res, err = req.TryAggregate()
		cancel()
		m.mu.Lock()
		m.stats.Requests++
		m.mu.Unlock()
		if !rateLimited(err) {
			break
		}
	}
	if err != nil {
		return &TransportError{Err: err}
	}
	var out []struct {
		Success    bool
		ReturnData []byte
	}
	if res == nil {
		return &TransportError{Err: errors.New("empty multicall response")}
	}
	vals, err := tryAggregateOut.Unpack(res.RawResponse)
	if err == nil {
		err = tryAggregateOut.Copy(&out, vals)
	}
	if err != nil {
		return &TransportError{Err: fmt.Errorf("multicall response: %w", err)}
	}
	if len(out) != len(chunk) {
		return &TransportError{Err: fmt.Errorf("multicall returned %d results for %d calls", len(out), len(chunk))}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats.Calls += len(chunk)
	for i, c := range chunk {
		m.cache[mcKey(c.to, c.data)] = mcResult{ok: out[i].Success, ret: out[i].ReturnData}
	}
	return nil
}

// pass runs f in collect mode and fetches what it recorded; it reports whether anything was new.
func (m *multicaller) pass(ctx context.Context, f func()) (bool, error) {
	m.mu.Lock()
	m.collect = map[string]*callReq{}
	m.stats.Passes++
	m.mu.Unlock()
	func() {
		defer func() { _ = recover() }() // zero answers may trip a decoder; the real load re-runs it
		f()
	}()
	m.mu.Lock()
	reqs := make([]*callReq, 0, len(m.collect))
	for _, r := range m.collect {
		reqs = append(reqs, r)
	}
	m.collect = nil
	m.mu.Unlock()
	if len(reqs) == 0 {
		return false, nil
	}
	return true, m.fetch(ctx, reqs)
}

func (m *multicaller) collecting() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.collect != nil
}

func (m *multicaller) Stats() MulticallStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stats
}

// ---------------------------------------------------------------- prefetch

// prefetch warms the cache for one venue: passes over every read Tracker.Load can make, breadth
// first (each stock and each of its sources independently, so one stock's progress does not wait
// for another's), until a pass predicts nothing new.
func (l *RPCLoader) prefetch(ctx context.Context, tr *Tracker, lot string) error {
	if l.mc == nil {
		return nil
	}
	for i := 0; i < maxPasses; i++ {
		more, err := l.mc.pass(ctx, func() { l.probe(ctx, tr, lot) })
		if err != nil || !more {
			return err
		}
	}
	return nil
}

func (l *RPCLoader) probe(ctx context.Context, tr *Tracker, lot string) {
	try := func(f func()) {
		defer func() { _ = recover() }()
		f()
	}
	try(func() { _, _ = l.HookParams(ctx) })
	try(func() { _, _ = l.Venue(ctx, lot) })
	try(func() { _, _ = l.NavGuardParams(ctx) })
	try(func() { _, _ = l.PoolManagerLotBalance(ctx, lot) })
	var stocks []string
	try(func() { stocks, _, _ = l.Constituents(ctx, lot) })
	var q *MqMarket
	for _, stock := range stocks {
		try(func() {
			snap, err := l.Stock(ctx, stock)
			if err != nil {
				return
			}
			if !sameAddr(snap.Registry.Stable, tr.Cfg.Usdg) {
				if q == nil {
					if q, err = l.MqMarket(ctx); err != nil {
						q = nil
						return
					}
				}
				_, _, _ = l.MqToken(ctx, stock, q)
				return
			}
			_, _, _ = l.Feed(ctx, stock)
			srcs, _, _ := Sources(tr.Cfg, stock, snap.Registry, snap.VenuesOK, snap.Venues)
			for _, src := range srcs {
				try(func() { _, _ = l.Pool(ctx, stock, src) })
			}
		})
	}
}

// PrefetchTwap warms the oracle reads of the market's TWAP states before the checked NAV: a small
// ring is fetched whole; a large one by shadowing MqPricing's observe on a copy (so the published Obs
// map holds only the slots the real search reads) and fetching, per round, the next twapTreeDepth
// levels of every state's binary search.
func (l *RPCLoader) PrefetchTwap(ctx context.Context, m *Market) error {
	if l.mc == nil || m == nil || m.Mq == nil {
		return nil
	}
	type ring struct {
		pool string
		s    *TwapState
	}
	rings := []ring{{m.Mq.ConvPool, m.Mq.ConvTwap}}
	for _, c := range m.Constituents {
		if c.Mq != nil {
			rings = append(rings, ring{c.Mq.Pool, c.Mq.Twap})
		}
	}
	var small []*callReq
	var large []ring
	for _, r := range rings {
		switch {
		case r.s == nil || r.s.Cardinality <= 0:
		case r.s.Cardinality <= maxRingPrefetch:
			for i := 0; i < r.s.Cardinality; i++ {
				small = append(small, observationReq(r.pool, i))
			}
		default:
			large = append(large, r)
			// observe's first reads: the newest, the oldest and slot 0
			small = append(small, observationReq(r.pool, r.s.Index), observationReq(r.pool, (r.s.Index+1)%r.s.Cardinality),
				observationReq(r.pool, 0))
		}
	}
	if err := l.mc.fetch(ctx, small); err != nil {
		return err
	}
	for round := 0; round < maxPasses && len(large) > 0; round++ {
		var reqs []*callReq
		var next []ring
		for _, r := range large {
			miss := l.shadowObserve(ctx, r.pool, r.s, m.Timestamp)
			if miss == nil {
				continue
			}
			next = append(next, r)
			searchTree(r.s.Cardinality, miss.l, miss.r, twapTreeDepth, func(i int) {
				reqs = append(reqs, observationReq(r.pool, i))
			})
		}
		if err := l.mc.fetch(ctx, reqs); err != nil {
			return err
		}
		large = next
	}
	return nil
}

const twapTreeDepth = 8

// errNotCached ends a shadow search at the first slot the cache lacks.
var errNotCached = errors.New("lotflow: observation not cached")

// shadowObserve runs TwapTick on a copy of s that reads the cache only; it returns the binary-search
// range at the first uncached slot (nil: the search completes, or fails for a reason prefetching
// cannot change).
func (l *RPCLoader) shadowObserve(ctx context.Context, pool string, s *TwapState, ts uint64) *searchMiss {
	c := &TwapState{Tick: s.Tick, Index: s.Index, Cardinality: s.Cardinality, Obs: map[int]Observation{}}
	c.Fetch = func(i int) (Observation, error) {
		r := observationReq(pool, i)
		l.mc.mu.Lock()
		_, ok := l.mc.cache[mcKey(r.to, r.data)]
		l.mc.mu.Unlock()
		if !ok {
			return Observation{}, errNotCached
		}
		return s.Fetch(i)
	}
	_, err := c.TwapTick(ts, MqTwapWindow)
	var miss *searchMiss
	if errors.As(err, &miss) && errors.Is(err, errNotCached) {
		return miss
	}
	return nil
}

// searchTree visits the slots binarySearch can read in its next `depth` steps from range [lo, hi]
// (the probe i and its successor; an uninitialized probe continues at [i+1, hi], like a "later" one).
func searchTree(card, lo, hi, depth int, visit func(int)) {
	if depth == 0 || lo > hi {
		return
	}
	i := (lo + hi) / 2
	visit(i % card)
	visit((i + 1) % card)
	searchTree(card, lo, i-1, depth-1, visit)
	searchTree(card, i+1, hi, depth-1, visit)
}

func observationReq(pool string, i int) *callReq {
	return &callReq{to: pool, data: append(sel("observations(uint256)"), encWord(big.NewInt(int64(i)))...)}
}
