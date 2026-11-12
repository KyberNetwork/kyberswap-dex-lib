package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// probesPerBatch bounds one JSON-RPC batch of the plain path's calls.
const probesPerBatch = 50

var maxAsk = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), maxPointBits), big.NewInt(1))

// sampleAsk, sampleLimits, sampleProbe and sampleWalk are FlammSampler's Ask, Limits, Probe and Walk, and
// sampleAnswer what its `sample` returns, their fields in the ABI's order.
type (
	sampleAsk struct {
		TokenIn, TokenOut, Hook common.Address
		TierA, TierB, MaxProbes *big.Int
		Seeds                   []*big.Int
	}
	sampleLimits struct{ ProbeGas, ReserveGas, FlipShift, EdgeProbes, GapPpm *big.Int }
	sampleProbe  struct {
		Ask, Used, Out, FeeWad, GasUsed, Status *big.Int
		Reason                                  [4]byte
	}
	sampleWalk struct {
		Probes        []sampleProbe
		CurveFeeWad   *big.Int
		BandWad       uint64
		UsdIn, UsdOut *big.Int
		Legs          [8]*big.Int
		BidWad        *big.Int
		Truncated     bool
	}
	sampleAnswer struct {
		Tag         [4]byte
		BlockNumber *big.Int
		Walks       []sampleWalk
	}
)

// walk is one direction's answer, the sampler's or the plain path's: its probes ascending by ask, what settle reads
// of the pool's request and, on a buy, the hook's top bid, and the curve's fee (zero: not read).
type walk struct {
	probes   []sampled
	words    pairWords
	curveFee *big.Int
}

var (
	// errNotSampled is an answer the sampler does not give: one that does not decode, an empty one, another tag, a
	// walk count other than the asks', or no probe in any walk with one stopped by the gas gate at the default cap
	// (a node giving the call less gas than one probe needs). The sampler did not run as sent, which no pool causes.
	errNotSampled = errors.New("the sampler did not run")
	// errNullAnswer is a null result in the answer's place, which a provider gives under load.
	errNullAnswer = errors.New("a null answer")
)

// ladders prices the directions in one sampler call, or on the plain path where the node does not run the sampler,
// and settles each (settle). Past maxSampledDirections the rest are not routed this refresh.
func (t *PoolTracker) ladders(ctx context.Context, poolAddr string, st *poolState, dirs []direction, prev *Extra,
	overrides map[common.Address]gethclient.OverrideAccount) ([]Ladder, error) {
	ladders := make([]Ladder, 0, len(dirs))
	if len(dirs) > maxSampledDirections {
		logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": poolAddr}).
			Debugf("%d directions past the first %d are not routed this refresh", len(dirs)-maxSampledDirections,
				maxSampledDirections)
		dirs = dirs[:maxSampledDirections]
	}
	if len(dirs) == 0 {
		return ladders, nil
	}
	last := map[[2]uint8]Ladder{}
	for _, l := range prev.Ladders {
		last[[2]uint8{l.In, l.Out}] = l
	}
	walks, err := t.walks(ctx, poolAddr, st, dirs, last, overrides)
	if err != nil {
		return nil, err
	}
	held, err := t.readHeldLegs(ctx, poolAddr, st, dirs, walks, overrides)
	if err != nil {
		return nil, err
	}
	for k, d := range dirs {
		// the pool's positions at the block serve both directions of a pair: a direction whose request did not
		// read takes the other's, its input's legs the other's output's
		legs := walks[k].words
		if !legsRead(legs) {
			if m := reverseRead(dirs, walks, k); m >= 0 {
				legs = swapLegs(walks[m].words)
			} else if legsRead(held[k]) {
				legs = held[k]
			}
		}
		if l, ok := build(d, st, walks[k], legs); ok {
			ladders = append(ladders, l)
		}
	}
	return ladders, nil
}

// reverseRead is the direction of dirs reversing dirs[k] whose walk read the pool's request, or -1.
func reverseRead(dirs []direction, walks []walk, k int) int {
	for m, e := range dirs {
		if e.in == dirs[k].out && e.out == dirs[k].in && legsRead(walks[m].words) {
			return m
		}
	}
	return -1
}

// readHeldLegs reads the pool's request (pairRequest) for each direction whose output a live venue holding a
// minimum position lends where no walk of its pair read it, as a walk reads it only past a tier-labelled probe: the
// legs and the band its guard (keepHeld) and its charge take. One batch at the refresh block, each at the ask of
// the direction's first probe priced or clipped paying at least dustOut. A read the pool refuses, or answers in
// another size, leaves the legs unread, and the guard then publishes nothing of the direction.
func (t *PoolTracker) readHeldLegs(ctx context.Context, poolAddr string, st *poolState, dirs []direction,
	walks []walk, overrides map[common.Address]gethclient.OverrideAccount) ([]pairWords, error) {
	legs := make([]pairWords, len(dirs))
	var batch []eth.BatchCall
	var keep []func(callAnswer)
	for k, d := range dirs {
		if d.heldOut == nil || legsRead(walks[k].words) || reverseRead(dirs, walks, k) >= 0 {
			continue
		}
		at := slices.IndexFunc(walks[k].probes, func(p sampled) bool { return live(p) && p.out.Cmp(dust) >= 0 })
		if at < 0 {
			continue
		}
		data, err := poolABI.Pack("pairRequest", st.tokens[d.in].Config.Token, st.tokens[d.out].Config.Token,
			walks[k].probes[at].ask)
		if err != nil {
			return nil, err
		}
		batch = append(batch, eth.BatchCall{To: poolAddr, Data: data})
		keep = append(keep, func(a callAnswer) {
			if len(a.ret) == 32*requestWords {
				legs[k] = requestOf(a.ret)
			}
		})
	}
	return legs, t.keep(ctx, batch, keep, st.block, overrides)
}

// walks is the sampler's answer, or the plain path's where the config disables the sampler, the node has refused it
// samplerFailuresToCooldown times in a row (until samplerRefusedUntilMs), or this call shows the node does not run
// it (samplerUnsupported). Any other failure fails the refresh: the last state stands until it ages out.
func (t *PoolTracker) walks(ctx context.Context, poolAddr string, st *poolState, dirs []direction,
	last map[[2]uint8]Ladder, overrides map[common.Address]gethclient.OverrideAccount) ([]walk, error) {
	until := t.samplerRefusedUntilMs.Load()
	cooling := time.Now().UnixMilli() < until
	if !t.cfg.DisableSampler && !cooling {
		w, err := t.sample(ctx, poolAddr, st, dirs, last, overrides)
		if err == nil || !samplerUnsupported(err) {
			return w, err
		}
		log := logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": poolAddr})
		if t.samplerFailures.Add(1) >= samplerFailuresToCooldown {
			until, cooling = time.Now().Add(samplerRetryAfter).UnixMilli(), true
			t.samplerRefusedUntilMs.Store(until)
			log.Warnf("the node refused the sampler %d times in a row, the last: %v; the plain path for %s",
				samplerFailuresToCooldown, err, samplerRetryAfter)
		} else {
			log.Debugf("the node did not run the sampler: %v; the plain path this refresh", err)
		}
	}
	w, err := t.plainWalks(ctx, poolAddr, st, dirs, last, overrides)
	if err != nil && cooling {
		// a node that refuses batches too: the sampler again from the next refresh
		t.samplerRefusedUntilMs.CompareAndSwap(until, 0)
	}
	return w, err
}

// sample asks the sampler once for every direction. It asks again once where the answer is a revert or not the
// sampler's, and once more at the default cap where a raised one got no probe, as a node's own gas cap can stop a
// probe at the raised cap: neither a refusal nor the pool's doing. A good answer, the sampler's tag and some probe,
// resets the count of refusals.
func (t *PoolTracker) sample(ctx context.Context, poolAddr string, st *poolState, dirs []direction,
	last map[[2]uint8]Ladder, overrides map[common.Address]gethclient.OverrideAccount) ([]walk, error) {
	asks := t.asks(st, dirs, last)
	pl := common.HexToAddress(poolAddr)
	base := t.defaultCap()
	capGas := t.capOf(poolAddr, base)
	var retried, reset bool
	for {
		a, err := t.askSampler(ctx, pl, asks, capGas, st.block, overrides)
		if err == nil && a.noProbes() {
			if capGas > base && !reset {
				capGas, reset = base, true
				t.setCap(poolAddr, base)
				continue
			}
			err = fmt.Errorf("%w: no probe, the gas gate stopped a walk at a cap of %d", errNotSampled, capGas)
		}
		if err != nil {
			if !retried && retryable(err) {
				retried = true
				continue
			}
			return nil, err
		}
		t.samplerFailures.Store(0)
		if a.BlockNumber.Cmp(st.block) < 0 {
			return nil, fmt.Errorf("%w: the sampler read block %s, before the refresh's %s", ErrRead, a.BlockNumber,
				st.block)
		}
		return t.walksOf(poolAddr, capGas, base, a)
	}
}

// asks are the directions as the sampler takes them: each with its pricer, its row's tier labels (zero: none, so
// neither the request nor the probe above is asked), an even share of the pool's budget and its seeds.
func (t *PoolTracker) asks(st *poolState, dirs []direction, last map[[2]uint8]Ladder) []sampleAsk {
	budget := min(maxProbesPerDirection, maxProbesPerRefresh/len(dirs))
	asks := make([]sampleAsk, len(dirs))
	for k, d := range dirs {
		in, out := st.tokens[d.in].Config, st.tokens[d.out].Config
		a := sampleAsk{TokenIn: in.Token, TokenOut: out.Token, Hook: d.hook, TierA: new(big.Int), TierB: new(big.Int),
			MaxProbes: big.NewInt(int64(budget)), Seeds: seedsOf(int(in.Decimals), capOf(in), lastOf(last, d),
				min(firstRefreshSeeds, budget), min(t.cfg.rungs(), budget-1))}
		if tiers := d.row.tiers(); d.rule == ruleEverlong && len(tiers) != 0 {
			a.TierA, a.TierB = tiers[0], tiers[1]
		}
		asks[k] = a
	}
	return asks
}

// askSampler runs the sampler once with every probe capped at capGas, and decodes its answer.
func (t *PoolTracker) askSampler(ctx context.Context, pl common.Address, asks []sampleAsk, capGas uint64,
	block *big.Int, overrides map[common.Address]gethclient.OverrideAccount) (*sampleAnswer, error) {
	data, err := samplerABI.Pack("sample", pl, asks, samplerLimits(capGas))
	if err != nil {
		return nil, err
	}
	ret, err := t.callSampler(ctx, data, block, overrides)
	if err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, fmt.Errorf("%w: an empty answer", errNotSampled)
	}
	var a sampleAnswer
	if err := samplerABI.UnpackIntoInterface(&a, "sample", ret); err != nil {
		return nil, fmt.Errorf("%w: %w", errNotSampled, err)
	}
	if a.Tag != samplerTag || len(a.Walks) != len(asks) {
		return nil, fmt.Errorf("%w: tag %x, %d walks for %d asks", errNotSampled, a.Tag, len(a.Walks), len(asks))
	}
	return &a, nil
}

// callSampler sends the sampler's call: one eth_call to the taker, with the sampler's code there by state override
// over the caller's own (ours wins at the taker), an explicit gas, `from` unset, pinned to the block, under
// samplerTimeout. It is one plain call, not a batch, so a provider that refuses batches still runs it.
func (t *PoolTracker) callSampler(ctx context.Context, data []byte, block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) ([]byte, error) {
	taker := t.cfg.taker()
	ov := maps.Clone(overrides)
	if ov == nil {
		ov = map[common.Address]gethclient.OverrideAccount{}
	}
	ov[taker] = gethclient.OverrideAccount{Code: samplerRuntime} // ours wins at the taker
	arg := map[string]any{"to": taker, "data": hexutil.Bytes(data), "gas": hexutil.Uint64(t.cfg.sampleGas())}
	ctx, cancel := context.WithTimeout(ctx, samplerTimeout)
	defer cancel()
	var res json.RawMessage // a null result is the provider's, not an answer that does not decode
	err := t.ethrpcClient.GetETHClient().Client().CallContext(ctx, &res, "eth_call", arg, hexutil.EncodeBig(block), ov)
	switch {
	case errors.Is(err, rpc.ErrNoResult):
		return nil, errNullAnswer
	case err != nil:
		return nil, err
	case bytes.Equal(bytes.TrimSpace(res), []byte("null")):
		return nil, errNullAnswer
	}
	var ret hexutil.Bytes
	if err := json.Unmarshal(res, &ret); err != nil {
		return nil, fmt.Errorf("%w: %w", errNotSampled, err)
	}
	return ret, nil
}

// noProbes reports an answer with no probe in any walk and a walk the gas gate stopped: the node gave the call less
// gas than one probe at the cap needs.
func (a *sampleAnswer) noProbes() bool {
	var truncated bool
	for _, w := range a.Walks {
		if len(w.Probes) != 0 {
			return false
		}
		truncated = truncated || w.Truncated
	}
	return truncated
}

// walksOf reads a good answer's walks and moves the pool's cap: a probe that starved doubles it, up to maxCap, and an
// answer whose every probe spent under 7/8 of the default cap brings it back to the default. A direction whose first
// probe the pool did not refuse starved has no point, and the refresh fails: below maxCap the next is asked at the
// raised cap, at it the pool misses this refresh as a node's error would.
func (t *PoolTracker) walksOf(poolAddr string, capGas, base uint64, a *sampleAnswer) ([]walk, error) {
	walks := make([]walk, len(a.Walks))
	var starved, firstStarved bool
	cool := true
	for k, sw := range a.Walks {
		w := walk{curveFee: sw.CurveFeeWad, words: pairWords{band: new(big.Int).SetUint64(sw.BandWad),
			usdIn: sw.UsdIn, usdOut: sw.UsdOut, legs: sw.Legs, bid: sw.BidWad}}
		first := true
		for _, sp := range sw.Probes {
			p := sampled{ask: sp.Ask, used: sp.Used, out: sp.Out, fee: sp.FeeWad, status: statusOf(sp)}
			if p.status == statusStarved {
				starved = true
				firstStarved = firstStarved || first
			}
			first = first && p.status == statusRefused
			if g := sp.GasUsed; !g.IsUint64() || g.Uint64() >= math.MaxUint64/8 || g.Uint64()*8 >= base*7 {
				cool = false
			}
			w.probes = append(w.probes, p)
		}
		walks[k] = w
	}
	top := t.maxCap()
	switch {
	case starved && capGas < top:
		t.setCap(poolAddr, min(2*capGas, top))
	case !starved && cool:
		t.setCap(poolAddr, base)
	}
	if firstStarved {
		if capGas < top {
			return nil, fmt.Errorf("%w: a direction's first probe starved at a cap of %d, raised", ErrRead, capGas)
		}
		return nil, fmt.Errorf("%w: a direction's first probe starved at the most cap, %d", ErrRead, capGas)
	}
	return walks, nil
}

// statusOf is a sampler probe's status: an answer the sampler priced or clipped that cannot be a fill, or a status
// it does not give, is malformed.
func statusOf(p sampleProbe) probeStatus {
	s := probeStatus(min(p.Status.Uint64(), math.MaxUint8))
	switch {
	case !p.Status.IsUint64() || s < statusPriced || s > statusMalformed:
		return statusMalformed
	case s == statusPriced || s == statusClipped:
		if fillStatus(p.Ask, p.Used, p.Out) == statusMalformed {
			return statusMalformed
		}
	}
	return s
}

// fillStatus is a previewSwap answer's status, read as the sampler does: nothing used or paid, more used than asked,
// or either at or past 2^maxPointBits, malformed; less used than asked by more than 1 part in 10,000, clipped; else
// priced.
func fillStatus(ask, used, out *big.Int) probeStatus {
	switch {
	case used.Sign() <= 0 || out.Sign() <= 0 || used.Cmp(ask) > 0 || used.BitLen() > maxPointBits ||
		out.BitLen() > maxPointBits:
		return statusMalformed
	case new(big.Int).Mul(used, big.NewInt(10_000)).Cmp(new(big.Int).Mul(ask, big.NewInt(9_999))) < 0:
		return statusClipped
	}
	return statusPriced
}

// defaultCap is the cap on each probe a pool's sampler call starts at; maxCap the most it is raised to, within what
// the call's own gas can give one probe.
func (t *PoolTracker) defaultCap() uint64 { return min(t.cfg.probeGas(), t.maxCap()) }

func (t *PoolTracker) maxCap() uint64 {
	return min(maxProbeGas, (t.cfg.sampleGas()-samplerReserveGas)*63/64)
}

func (t *PoolTracker) capOf(poolAddr string, base uint64) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if c, ok := t.caps[strings.ToLower(poolAddr)]; ok {
		return c
	}
	return base
}

// setCap keeps a pool's raised cap, or forgets it at the default.
func (t *PoolTracker) setCap(poolAddr string, c uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c == t.defaultCap() {
		delete(t.caps, strings.ToLower(poolAddr))
		return
	}
	t.caps[strings.ToLower(poolAddr)] = c
}

// samplerLimits are the sampler's settings at a cap of capGas on each probe.
func samplerLimits(capGas uint64) sampleLimits {
	return sampleLimits{ProbeGas: new(big.Int).SetUint64(capGas),
		ReserveGas: new(big.Int).SetUint64(samplerReserveGas), FlipShift: big.NewInt(samplerFlipShift),
		EdgeProbes: big.NewInt(samplerEdgeProbes), GapPpm: big.NewInt(samplerGapPpm)}
}

// seedsOf is a direction's sizes, ascending, none past the input's notional cap or maxAsk. With no last ladder, or
// one whose top paid under dustOut: `first` sizes at a constant ratio from a thousandth of a unit of the input (one
// base unit under three decimals) to a billion units. Otherwise one size at an eighth of the last ladder's first
// point, so a first point a refresh raised can fall again, then `rungs` sizes from that point to eight times the
// last top, at least twice the first point.
func seedsOf(decimals int, capIn *big.Int, last *Ladder, fresh, rungs int) []*big.Int {
	ceil := maxAsk
	if capIn != nil {
		ceil = minBig(capIn, maxAsk)
	}
	floor := minBig(pow10(decimals-3), ceil)
	from, to, n := floor, minBig(pow10(decimals+9), ceil), fresh
	var below *big.Int
	if last != nil && len(last.Points) != 0 {
		x0, top := last.Points[0][0], last.Points[len(last.Points)-1]
		if x0 != nil && top[0] != nil && top[1] != nil && top[1].CmpUint64(dustOut) >= 0 {
			first, end := x0.ToBig(), top[0].ToBig()
			below = maxBig(new(big.Int).Rsh(first, 3), floor)
			from = minBig(first, ceil)
			to = minBig(maxBig(new(big.Int).Lsh(end, 3), new(big.Int).Lsh(first, 1)), ceil)
			n = rungs
		}
	}
	sizes := geometric(from, to, n)
	if below != nil && below.Cmp(sizes[0]) < 0 {
		sizes = append([]*big.Int{below}, sizes...)
	}
	return sizes
}

// lastOf is the last refresh's ladder of the direction; nil: none.
func lastOf(last map[[2]uint8]Ladder, d direction) *Ladder {
	if l, ok := last[[2]uint8{uint8(d.in), uint8(d.out)}]; ok {
		return &l
	}
	return nil
}

// capOf is a token's swap notional cap; nil: none.
func capOf(c tokenConfig) *big.Int {
	if c.MaxSwapNotional != nil && c.MaxSwapNotional.Sign() > 0 {
		return c.MaxSwapNotional
	}
	return nil
}

// build is the direction's ladder, its probes settled by its pricer's rule (settle), with the charge's cells and
// reads (gas.go); false: nothing stands. legs are the pool's positions a walk of the pair read, the direction's own
// or the other direction's, swapped, or else the tracker's own read (readHeldLegs). A direction whose output a venue
// holding a minimum position lends keeps the piece of its ladder that venue's guard allows (keepHeld). A ladder past
// maxLadderPoints keeps its lowest points, already lowered, and is not capped, so quotes stop at its new top.
func build(d direction, st *poolState, w walk, legs pairWords) (Ladder, bool) {
	in := settleIn{probes: w.probes, words: w.words, rule: d.rule, tiers: d.row.tiers(), rowBand: d.row.band,
		sale: d.in == 0, decIn: st.tokens[d.in].Config.Decimals, decOut: st.tokens[d.out].Config.Decimals,
		trusted: d.trusted, curveFee: w.curveFee}
	s := settle(in)
	if d.heldOut != nil {
		var read *[8]*big.Int
		if legsRead(legs) {
			read = &legs.legs
		}
		lo, hi := keepHeld(s.x, s.y, w.probes, read, legs.band, *d.heldOut)
		s = cutHeld(s, lo, hi)
	}
	l := Ladder{In: uint8(d.in), Out: uint8(d.out), Capped: s.capped, Cells: d.cells, Second: d.second,
		After: d.after, FirstOwing: d.firstOwing}
	if h := st.tokens[d.out].Config.Held; h != nil {
		if held, overflow := uint256.FromBig(h); !overflow {
			l.Held = held
		}
	}
	l.readLegs(legs, d.repayOk, d.heldIn)
	if len(s.x) > maxLadderPoints {
		s.x, s.y, s.flat, l.Capped = s.x[:maxLadderPoints], s.y[:maxLadderPoints], s.flat[:maxLadderPoints], false
	}
	for k := range s.x {
		x, _ := uint256.FromBig(s.x[k]) // under 2^maxPointBits: a probe past it is malformed
		y, _ := uint256.FromBig(s.y[k])
		l.Points = append(l.Points, Point{x, y})
		if s.flat[k] {
			l.Flat |= 1 << k
		}
	}
	return l, len(l.Points) != 0
}

// nodeClass is what a failed sampler call says of the node, by the first list its error matches.
type nodeClass uint8

const (
	// classMissed is any other failure: a transport error, the deadline, the call itself out of gas.
	classMissed nodeClass = iota
	// classNotRun: the node did not run the sampler's code as sent. A revert at the taker (the sampler never
	// reverts), a method, parameter or override refused, a gas cap below the call's, a request too large, or an
	// answer that is not the sampler's.
	classNotRun
	// classSkipped: a rate limit, or a null answer.
	classSkipped
	// classBlockMissed: the node lacks the pinned block's state.
	classBlockMissed
)

var (
	rateWords = []string{"rate limit", "rate exceeded", "too many requests", "credits limited", "request limit",
		"limit reached", "maximum number of concurrent requests"}
	blockWords = []string{"header not found", "unknown block", "block not found", "missing trie"}
	rateCodes  = []int{429, -32005, -32007, -32012, -32016, -32055}
)

// classify sorts a sampler call's error by the first of these that matches, on its lower-cased message: not run (a
// JSON-RPC code 3, revert data or any word of a revert; -32601, -32602, -32604, or -32600 without a rate limit's
// words; an override; gas with "exceed", "limit" or "allowance" but not "out of gas"; HTTP 413; an answer not the
// sampler's), skipped (a rate limit's codes 429, -32005, -32007, -32012, -32016, -32055, HTTP 429 or 503, or its
// words; a null answer), the block missed, and missed for anything else.
func classify(err error) nodeClass {
	switch {
	case errors.Is(err, errNotSampled):
		return classNotRun
	case errors.Is(err, errNullAnswer):
		return classSkipped
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return classMissed
	}
	msg := strings.ToLower(err.Error())
	has := func(words ...string) bool {
		return slices.ContainsFunc(words, func(w string) bool { return strings.Contains(msg, w) })
	}
	var code, status int
	var re rpc.Error
	if errors.As(err, &re) {
		code = re.ErrorCode()
	}
	var he rpc.HTTPError
	if errors.As(err, &he) {
		status = he.StatusCode
	}
	rate := has(rateWords...)
	switch {
	case revertForm(err) || code == -32601 || code == -32602 || code == -32604 || code == -32600 && !rate ||
		has("override") || has("gas") && has("exceed", "limit", "allowance") && !has("out of gas") || status == 413:
		return classNotRun
	case slices.Contains(rateCodes, code) || status == 429 || status == 503 || rate:
		return classSkipped
	case has(blockWords...):
		return classBlockMissed
	}
	return classMissed
}

// samplerUnsupported reports a sampler call the node did not run as sent (classNotRun): the plain path serves that
// refresh, and samplerFailuresToCooldown of them in a row, with no good answer between, the next samplerRetryAfter.
// A rate limit, a timeout, a cancelled refresh and a missed block are not: they fail the refresh.
func samplerUnsupported(err error) bool { return classify(err) == classNotRun }

// retryable reports what a sampler call is asked again for, once: a revert or an answer that is not the sampler's,
// which a balancer mixing backends gives now and then.
func retryable(err error) bool { return errors.Is(err, errNotSampled) || revertForm(err) }

// revertForm reports an error that carries a revert: JSON-RPC code 3, revert data, or a revert's words.
func revertForm(err error) bool {
	return reverted(err) || len(revertData(err)) != 0 || strings.Contains(strings.ToLower(err.Error()), "revert")
}

// reverted reports whether an eth_call's error is an execution revert: JSON-RPC code 3, or a client's own code with
// the same words.
func reverted(err error) bool {
	var re rpc.Error
	return errors.As(err, &re) && re.ErrorCode() == 3 || strings.Contains(err.Error(), "execution reverted")
}

// revertData is the data a node gave with an error, decoded; nil: none.
func revertData(err error) []byte {
	var de rpc.DataError
	if !errors.As(err, &de) {
		return nil
	}
	s, ok := de.ErrorData().(string)
	if !ok {
		return nil
	}
	b, _ := hexutil.Decode(s)
	return b
}

// plainWalks prices the directions without the sampler: each size a previewSwap of its own, from the taker at
// plainProbeGas, the caller's overrides on every call, in JSON-RPC batches (batch), at up to plainSeedsPerDirection
// sizes a direction with no bisection; then the reads settle needs (readRequests). A direction whose first probe the
// pool did not refuse starved has no point, and with no cap to raise here the refresh fails.
func (t *PoolTracker) plainWalks(ctx context.Context, poolAddr string, st *poolState, dirs []direction,
	last map[[2]uint8]Ladder, overrides map[common.Address]gethclient.OverrideAccount) ([]walk, error) {
	walks := make([]walk, len(dirs))
	var calls []eth.BatchCall
	var of []int // each call's direction
	var asks []*big.Int
	for k, d := range dirs {
		in, out := st.tokens[d.in].Config, st.tokens[d.out].Config
		for _, a := range seedsOf(int(in.Decimals), capOf(in), lastOf(last, d), plainSeedsPerDirection,
			plainSeedsPerDirection-1) {
			data, err := poolABI.Pack("previewSwap", in.Token, out.Token, a)
			if err != nil {
				return nil, err
			}
			calls, of, asks = append(calls, eth.BatchCall{To: poolAddr, Data: data}), append(of, k), append(asks, a)
		}
	}
	answers, err := t.batch(ctx, calls, st.block, overrides)
	if err != nil {
		return nil, err
	}
	for i, a := range answers {
		walks[of[i]].probes = append(walks[of[i]].probes, previewOf(asks[i], a))
	}
	for _, w := range walks {
		if k := slices.IndexFunc(w.probes, func(p sampled) bool { return p.status != statusRefused }); k >= 0 &&
			w.probes[k].status == statusStarved {
			return nil, fmt.Errorf("%w: a direction's first probe starved", ErrRead)
		}
	}
	if err := t.readRequests(ctx, poolAddr, st, dirs, walks, overrides); err != nil {
		return nil, err
	}
	return walks, nil
}

// previewOf is a plain-path probe: a previewSwap's answer read as the sampler reads it, but for the gas it spent,
// which a node does not report.
func previewOf(ask *big.Int, a callAnswer) sampled {
	p := sampled{ask: ask, used: new(big.Int), out: new(big.Int), fee: new(big.Int)}
	switch {
	case a.starved:
		p.status = statusStarved
	case a.refused:
		p.status = statusRefused
	case len(a.ret) != 96:
		p.status = statusMalformed
	default:
		p.used, p.out, p.fee = word(a.ret, 0), word(a.ret, 1), word(a.ret, 2)
		p.status = fillStatus(ask, p.used, p.out)
	}
	return p
}

// readRequests reads what settle needs of the plain path's tier-labelled probes, for each direction whose pricer
// has a leverage row, in one batch and then, on buys, one more:
//   - the pool's request (pairRequest) at the ask of the first probe priced or clipped, paying at least dustOut,
//     with a tier label: its band, USD words and legs, for the room test;
//   - the probe above, where the priced probe of highest ask paying at least dustOut is tier-labelled: one
//     previewSwap at used + (used >> aboveShift) + 1, unless that ask is past maxAsk or already probed;
//   - on a buy (word 3 of the request, the input's index, is 1), the hook's levState on the request's 26 words: word
//     6 is the lever-up's top bid.
//
// A read the pool or the hook refuses, starves or answers in another size leaves its words at zero, and settle then
// cuts the ladder before its first tier-labelled point. The curve's fee is not read here.
func (t *PoolTracker) readRequests(ctx context.Context, poolAddr string, st *poolState, dirs []direction,
	walks []walk, overrides map[common.Address]gethclient.OverrideAccount) error {
	requests := make([][]byte, len(dirs)) // each direction's request, its 26 words as answered; nil: not read
	var batch []eth.BatchCall
	var keep []func(callAnswer) // each call's answer, by its place in the batch
	for k, d := range dirs {
		tiers := d.row.tiers()
		if d.rule != ruleEverlong || len(tiers) == 0 {
			continue
		}
		isTier := func(f *big.Int) bool {
			return slices.ContainsFunc(tiers, func(t *big.Int) bool { return f.Cmp(t) == 0 })
		}
		w := &walks[k]
		var first, top *sampled
		for i := range w.probes {
			p := &w.probes[i]
			if !live(*p) || p.out.Cmp(dust) < 0 {
				continue
			}
			if first == nil && isTier(p.fee) {
				first = p
			}
			if p.status == statusPriced {
				top = p
			}
		}
		in, out := st.tokens[d.in].Config.Token, st.tokens[d.out].Config.Token
		if first != nil {
			data, err := poolABI.Pack("pairRequest", in, out, first.ask)
			if err != nil {
				return err
			}
			batch = append(batch, eth.BatchCall{To: poolAddr, Data: data})
			keep = append(keep, func(a callAnswer) {
				if len(a.ret) == 32*requestWords {
					requests[k], w.words = a.ret, requestOf(a.ret)
				}
			})
		}
		if top == nil || !isTier(top.fee) {
			continue
		}
		ask := new(big.Int).Rsh(top.used, aboveShift)
		ask.Add(ask, top.used).Add(ask, one)
		if ask.Cmp(maxAsk) > 0 || slices.ContainsFunc(w.probes, func(p sampled) bool { return p.ask.Cmp(ask) == 0 }) {
			continue
		}
		data, err := poolABI.Pack("previewSwap", in, out, ask)
		if err != nil {
			return err
		}
		batch = append(batch, eth.BatchCall{To: poolAddr, Data: data})
		keep = append(keep, func(a callAnswer) { w.probes = append(w.probes, previewOf(ask, a)) })
	}
	if err := t.keep(ctx, batch, keep, st.block, overrides); err != nil {
		return err
	}
	for k := range walks {
		slices.SortFunc(walks[k].probes, func(a, b sampled) int { return a.ask.Cmp(b.ask) })
	}

	batch, keep = batch[:0], keep[:0]
	for k, d := range dirs {
		if requests[k] == nil || word(requests[k], 3).Cmp(one) != 0 { // the input's index: 1 on a buy
			continue
		}
		batch = append(batch, eth.BatchCall{To: hexutil.Encode(d.hook[:]),
			Data: slices.Concat(levStateSelector, requests[k])})
		keep = append(keep, func(a callAnswer) {
			if len(a.ret) == 32*levStateWords {
				walks[k].words.bid = word(a.ret, 6)
			}
		})
	}
	return t.keep(ctx, batch, keep, st.block, overrides)
}

// keep sends the calls (batch) and hands each answer to its keeper.
func (t *PoolTracker) keep(ctx context.Context, calls []eth.BatchCall, keep []func(callAnswer), block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) error {
	answers, err := t.batch(ctx, calls, block, overrides)
	if err != nil {
		return err
	}
	for k, a := range answers {
		keep[k](a)
	}
	return nil
}

// requestOf is what settle reads of the pool's request: the band (word 10), the USD words (11, 12) and the legs
// (15-22).
func requestOf(ret []byte) pairWords {
	p := pairWords{band: word(ret, 10), usdIn: word(ret, 11), usdOut: word(ret, 12)}
	for k := range p.legs {
		p.legs[k] = word(ret, 15+k)
	}
	return p
}

// word is an answer's k-th 32-byte word.
func word(ret []byte, k int) *big.Int { return new(big.Int).SetBytes(ret[32*k : 32*(k+1)]) }

// callAnswer is one eth_call's of the plain path: what came back, or refused where the callee reverted, or starved
// where it reverted ReadStarved() or GasShort(), or the node ran it out of gas.
type callAnswer struct {
	ret              []byte
	refused, starved bool
}

// batch sends the calls as eth_calls from the taker, each at plainProbeGas with the caller's overrides, at the block,
// in JSON-RPC batches. A revert is the callee's own word. Anything else that fails says nothing of the pool, a
// node's timeout or a block it has not reached, and fails the refresh: the last state stands until it ages out.
func (t *PoolTracker) batch(ctx context.Context, calls []eth.BatchCall, block *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount) ([]callAnswer, error) {
	out := make([]callAnswer, len(calls))
	taker := t.cfg.taker()
	for from := 0; from < len(calls); from += probesPerBatch {
		chunk := slices.Clone(calls[from:min(from+probesPerBatch, len(calls))])
		for k := range chunk {
			chunk[k].From, chunk[k].Gas, chunk[k].Overrides = taker, plainProbeGas, overrides
		}
		results, callErrs, err := eth.BatchEthCall(ctx, t.ethrpcClient.GetETHClient().Client(), chunk, block)
		if err != nil {
			return nil, err
		}
		for k := range chunk {
			switch err := callErrs[k]; {
			case err == nil:
				out[from+k].ret = results[k]
			case starvedRevert(err) || strings.Contains(strings.ToLower(err.Error()), "out of gas"):
				out[from+k].starved = true
			case reverted(err):
				out[from+k].refused = true
			default:
				return nil, fmt.Errorf("%w: a probe: %w", ErrRead, err)
			}
		}
	}
	return out, nil
}

// starvedRevert reports a revert whose data starts with ReadStarved() or GasShort().
func starvedRevert(err error) bool {
	d := revertData(err)
	return len(d) >= 4 && slices.ContainsFunc(starvedReverts, func(s []byte) bool { return bytes.Equal(d[:4], s) })
}

// geometric is n sizes from lo to hi, both included, at a constant ratio, ascending and distinct.
func geometric(lo, hi *big.Int, n int) []*big.Int {
	if hi.Cmp(lo) <= 0 || n < 2 {
		return []*big.Int{new(big.Int).Set(lo)}
	}
	lf, _ := new(big.Float).SetInt(lo).Float64()
	hf, _ := new(big.Float).SetInt(hi).Float64()
	step := math.Log(hf/lf) / float64(n-1)
	out := []*big.Int{new(big.Int).Set(lo)}
	for k := 1; k < n; k++ {
		v := new(big.Int).Set(hi)
		if k < n-1 {
			v, _ = new(big.Float).SetFloat64(math.Round(lf * math.Exp(step*float64(k)))).Int(nil)
		}
		if v.Cmp(out[len(out)-1]) > 0 && v.Cmp(hi) <= 0 {
			out = append(out, v)
		}
	}
	return out
}

func pow10(e int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(max(e, 0))), nil)
}

func minBig(a, b *big.Int) *big.Int {
	if a.Cmp(b) <= 0 {
		return a
	}
	return b
}

func maxBig(a, b *big.Int) *big.Int {
	if a.Cmp(b) >= 0 {
		return a
	}
	return b
}
