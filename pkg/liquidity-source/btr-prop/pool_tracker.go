package btrprop

import (
	"context"
	"math/big"
	"slices"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

// probeExps are the decimal exponents (in whole tokenIn units) of the rate
// probes: 0.01, 1 and 100 units. The best rate of the three sizes the grid.
var probeExps = [...]int{-2, 0, 2}

// gridPermille are the grid amounts as a share of maxIn, the tokenIn amount
// worth the full tokenOut balance at the probed rate, clamped to both legs'
// coverage caps (clampToCaps). The curve is near-linear below the cap and
// steepens toward it (the coverage toll), so the samples are denser toward the
// top. The 17 sample trims the chord overquote between 10 and 30 (the pool's
// own +-50 sat output sawtooth on cbBTC pairs remains).
var gridPermille = [...]uint64{1, 3, 10, 17, 30, 60, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

// The coverage cap refuses a size past it, and the toll steepens as a leg nears
// it, so the grid's last segment (900..1000) would straddle the steep part and
// underquote; a few extra samples go around it. Cold start (no previous
// ladder): fixed permille of maxIn. Warm start: permille of the previous
// ladder's last In, which is the last size the pool still served.
var (
	coldKneePermille = [...]uint64{950, 975, 990}
	warmKneePermille = [...]uint64{950, 980, 995}
)

// PoolTracker refreshes a BTR pool every block: it caches the per-token risk
// flags and balances and probes `getSwapQuote` over a grid per ordered pair to
// build a ladder. The keeper mark heartbeat is independent of trades, so a lazy
// tracker would quote stale marks.
type PoolTracker struct {
	config *Config
	client *ethrpc.Client
}

func NewPoolTracker(cfg *Config, client *ethrpc.Client) *PoolTracker {
	if cfg == nil {
		cfg = NewConfig(ChainIDMonad)
	}
	return &PoolTracker{config: cfg, client: client}
}

// rpcSwapQuote mirrors IPool.SwapQuote field for field so ethrpc unpacks the
// tuple. Only AmountOut is consumed.
type rpcSwapQuote struct {
	AmountOut  *big.Int         `abi:"amountOut"`
	AmountIn   *big.Int         `abi:"amountIn"`
	SpreadPbps uint16           `abi:"spreadPbps"`
	ProtoFee   *big.Int         `abi:"protoFee"`
	LpFee      *big.Int         `abi:"lpFee"`
	SkewIn     int8             `abi:"skewIn"`
	SkewOut    int8             `abi:"skewOut"`
	MarkPrice  *big.Int         `abi:"markPrice"`
	MidPrice   *big.Int         `abi:"midPrice"`
	CovToll    *big.Int         `abi:"covToll"`
	RouteHops  []common.Address `abi:"routeHops"`
	HopAmounts []*big.Int       `abi:"hopAmounts"`
	HopPrices  []*big.Int       `abi:"hopPrices"`
}

// rpcSwapQuoteResult wraps the single unnamed tuple output: go-ethereum copies
// a lone return value into the first field of the target struct, so the tuple
// must sit one level down (same pattern as baseline's rpcGetQuoteStateResult).
type rpcSwapQuoteResult struct {
	Quote rpcSwapQuote
}

// rpcAsset mirrors the IPool.Asset tuple returned by getAsset, in ABI order. The tracker reads
// Reserves, Liabilities, Flags, CovBits and KneeBits: they size the coverage caps.
type rpcAsset struct {
	Reserves          *big.Int       `abi:"reserves"`
	Liabilities       *big.Int       `abi:"liabilities"`
	Anchor            common.Address `abi:"anchor"`
	MinLiquidity      *big.Int       `abi:"minLiquidity"`
	LiquidityIndexWad *big.Int       `abi:"liquidityIndexWad"`
	MinDispersionPbps uint32         `abi:"minDispersionPbps"`
	CurveID           uint16         `abi:"curveId"`
	MinFeePbps        uint16         `abi:"minFeePbps"`
	VegaBps           uint16         `abi:"vegaBps"`
	DepositCapCode    uint16         `abi:"depositCapCode"`
	Decimals          uint8          `abi:"decimals"`
	DeadSeedPow10     uint8          `abi:"deadSeedPow10"`
	Flags             uint16         `abi:"flags"`
	MaxLiabWeightBps  uint16         `abi:"maxLiabWeightBps"`
	OracleBits        uint8          `abi:"oracleBits"`
	CovBits           uint32         `abi:"covBits"`
	KneeBits          uint16         `abi:"kneeBits"`
}

// rpcAssetResult wraps the single unnamed tuple output, as rpcSwapQuoteResult does.
type rpcAssetResult struct {
	Asset rpcAsset
}

// pairProbe holds the rate probes of one ordered pair, then its grid.
type pairProbe struct {
	from, to int
	ins      []uint256.Int
	res      []*rpcSwapQuoteResult
	ok       []bool
	// probes is the pair's rate probes; set on grids only.
	probes *pairProbe
	// stop is the index of the sample that ended ladder() (len(ins) when none did).
	stop int
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	n := len(p.Tokens)
	if n < 2 || t.client == nil {
		return p, ErrBTRBadState
	}
	ctx, cancel := context.WithTimeout(ctx, RPCTimeout)
	defer cancel()

	addrs := make([]common.Address, n)
	for i, tok := range p.Tokens {
		if tok == nil || !common.IsHexAddress(tok.Address) {
			return p, ErrBTRBadState
		}
		addrs[i] = common.HexToAddress(tok.Address)
	}
	poolAddr := common.HexToAddress(p.Address)

	// The previous ladders locate the cap onset for the knee samples.
	var prev Extra
	_ = json.Unmarshal([]byte(p.Extra), &prev)

	// Phase 1: flags, balances, books, leg words, liquid reserves, the base token, the executor's
	// swap-gate membership and the rate probes, sent as one JSON-RPC batch of plain eth_calls
	// pinned to one block. Each call keeps its own gas budget; a single multicall would share one
	// budget across ~500 calls and run out.
	// Stamped before the reads: the simulator's MaxAge measures from this, so the grid's
	// duration must not hide a stale snapshot.
	fetchedAt := time.Now().Unix()
	blockNumber, err := t.client.GetBlockNumber(ctx)
	if err != nil {
		return p, err
	}
	bn := new(big.Int).SetUint64(blockNumber)
	var batch batchReads
	flags := make([]uint16, n)
	balances := make([]*big.Int, n)
	assets := make([]*rpcAssetResult, n)
	flagOK, balOK, assetOK := make([]bool, n), make([]bool, n), make([]bool, n)
	markOK, liquidOK := make([]bool, n), make([]bool, n)
	marks := make([]*rpcLegMark, n)
	liquids := make([]*big.Int, n)
	for i := range addrs {
		assets[i] = new(rpcAssetResult)
		batch.add(poolABI, p.Address, "getRiskFlags", []any{addrs[i]}, &flags[i], &flagOK[i])
		batch.add(utilabi.Erc20ABI, p.Tokens[i].Address, "balanceOf", []any{poolAddr}, &balances[i], &balOK[i])
		batch.add(poolABI, p.Address, "getAsset", []any{addrs[i]}, assets[i], &assetOK[i])
		marks[i] = new(rpcLegMark)
		batch.add(poolABI, p.Address, "getLegMark", []any{addrs[i]}, marks[i], &markOK[i])
		batch.add(poolABI, p.Address, "getLiquidReserves", []any{addrs[i]}, &liquids[i], &liquidOK[i])
	}
	var base common.Address
	var baseOK bool
	batch.add(poolABI, p.Address, "baseToken", nil, &base, &baseOK)
	// Whether the executor may swap SWAP_GATED legs. Read, never assumed: the owner admits the
	// executor to the pool's swap gate separately, and the answer flips the legs open.
	var inGate bool
	var gateRead bool
	batch.add(poolABI, p.Address, "inSwapGate", []any{t.config.executor()}, &inGate, &gateRead)
	probes := make([]*pairProbe, 0, n*(n-1))
	for i := range addrs {
		for j := range addrs {
			if i == j {
				continue
			}
			pp := &pairProbe{from: i, to: j}
			for _, exp := range probeExps {
				pp.ins = append(pp.ins, unitAmount(p.Tokens[i].Decimals, exp))
			}
			pp.addQuoteCalls(&batch, p.Address, addrs[i], addrs[j])
			probes = append(probes, pp)
		}
	}
	if err := batch.run(ctx, t.client, bn); err != nil {
		return p, err
	}
	legs := make([]*markLeg, n)
	rooms := make([]*uint256.Int, n)
	for i := range addrs {
		if !flagOK[i] {
			flags[i] = 0 // unreadable flags read as swap disabled
		}
		if !assetOK[i] {
			assets[i] = nil // unreadable book: no cap clamp, the first revert ends the ladder
		}
		if markOK[i] { // an unreadable word leaves legs[i] nil: every path through it is refused
			legs[i], _ = decodeMark(marks[i].Word)
		}
		if liquidOK[i] && assets[i] != nil {
			rooms[i] = liquidRoom(liquids[i], &assets[i].Asset)
		}
	}
	gateOK := gateRead && inGate
	if baseOK { // an unreadable base leaves every leg unexempted: stricter, never looser
		for i := range addrs {
			if legs[i] != nil {
				legs[i].base = addrs[i] == base
			}
		}
	}

	// The leg ages come from the block the ladder is sampled at, not the wall clock.
	now, err := t.client.NewRequest().SetContext(ctx).SetBlockNumber(bn).GetCurrentBlockTimestamp()
	if err != nil {
		return p, err
	}
	gd := &guard{tokens: addrs, legs: legs, assets: assets, base: base, rooms: rooms, now: now, window: uint64((MaxAge + InclusionAllowance) / time.Second)}

	// Phase 2: the grid, pinned to the block of phase 1.
	grids := make([]*pairProbe, 0, len(probes))
	var grid batchReads
	for _, pp := range probes {
		if balances[pp.to] == nil {
			continue
		}
		var balOut uint256.Int
		if balOut.SetFromBig(balances[pp.to]) {
			continue
		}
		// settle pays only liquid reserves above minLiquidity: size the grid inside that room.
		if room := rooms[pp.to]; room != nil && room.Lt(&balOut) {
			balOut.Set(room)
		}
		maxIn := pp.maxIn(&balOut)
		// Both coverage caps refuse a size past them (ThresholdViolation), so a grid sized
		// past either would spend its upper samples on reverts.
		maxIn = pp.clampToCaps(maxIn, &balOut, assets[pp.from], assets[pp.to])
		g := &pairProbe{from: pp.from, to: pp.to, probes: pp, ins: gridAmounts(maxIn, kneeAmounts(prev.ladder(pp.from, pp.to), &maxIn))}
		if len(g.ins) == 0 {
			continue
		}
		g.addQuoteCalls(&grid, p.Address, addrs[pp.from], addrs[pp.to])
		grids = append(grids, g)
	}
	extra := Extra{Flags: flags, GateOK: gateOK, Ladders: make([][]Ladder, n)}
	for i := range extra.Ladders {
		extra.Ladders[i] = make([]Ladder, n)
		for j := range extra.Ladders[i] {
			extra.Ladders[i][j] = Ladder{}
		}
	}
	if len(grids) > 0 {
		if err := grid.run(ctx, t.client, bn); err != nil {
			return p, err
		}
		built := make([][]Point, len(grids))
		// Phase 3: where a ladder ended on a revert, probe inside the gap so its top is the pool's
		// real cap. Per-pair limits the tracker does not model are found from the pool itself.
		var bisect batchReads
		bisects := make([]*pairProbe, len(grids))
		for k, g := range grids {
			built[k] = g.probes.mergeInto(g.ladder())
			if ins := g.bisectAmounts(built[k]); len(ins) > 0 {
				bisects[k] = &pairProbe{from: g.from, to: g.to, ins: ins}
				bisects[k].addQuoteCalls(&bisect, p.Address, addrs[g.from], addrs[g.to])
			}
		}
		if len(bisect.calls) > 0 {
			if err := bisect.run(ctx, t.client, bn); err != nil {
				return p, err
			}
			for k, bp := range bisects {
				if bp != nil {
					built[k] = bp.extend(built[k])
				}
			}
		}
		for k, g := range grids {
			spread := max(g.probes.maxSpread(), g.maxSpread())
			if bisects[k] != nil {
				spread = max(spread, bisects[k].maxSpread())
			}
			if pts := gd.refine(g.from, g.to, g.hops(), spread, built[k]); len(pts) > 0 {
				extra.Ladders[g.from][g.to] = newLadder(pts)
				extra.setHops(n, g.from, g.to, len(g.hops()))
			}
		}
	}

	b, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(b)
	p.Reserves = make(entity.PoolReserves, n)
	for i, bal := range balances {
		p.Reserves[i] = "0"
		// A leg the swap would revert on keeps its ladders but pays out nothing: zero reserve.
		if bal != nil && !gd.legClosed(i) {
			p.Reserves[i] = bal.String()
		}
	}
	p.BlockNumber = blockNumber
	p.Timestamp = fetchedAt
	return p, nil
}

// addQuoteCalls queues one getSwapQuote per amount in pp.ins.
func (pp *pairProbe) addQuoteCalls(b *batchReads, poolAddr string, tokenIn, tokenOut common.Address) {
	pp.res = make([]*rpcSwapQuoteResult, len(pp.ins))
	pp.ok = make([]bool, len(pp.ins))
	for k := range pp.ins {
		pp.res[k] = new(rpcSwapQuoteResult)
		b.add(poolABI, poolAddr, "getSwapQuote",
			[]any{tokenIn, tokenOut, pp.ins[k].ToBig()}, pp.res[k], &pp.ok[k])
	}
}

// maxSpread is the largest SpreadPbps over the filled quotes.
func (pp *pairProbe) maxSpread() uint64 {
	var m uint64
	for k := range pp.ins {
		if pp.ok[k] {
			m = max(m, uint64(pp.res[k].Quote.SpreadPbps))
		}
	}
	return m
}

// hops is the route of the largest filled quote, nil when none filled.
func (pp *pairProbe) hops() []common.Address {
	for k := len(pp.ins) - 1; k >= 0; k-- {
		if pp.ok[k] && len(pp.res[k].Quote.RouteHops) > 0 {
			return pp.res[k].Quote.RouteHops
		}
	}
	return nil
}

// maxIn is the tokenIn amount worth the whole tokenOut balance at the best
// probed rate, or zero when no probe filled.
func (pp *pairProbe) maxIn(balOut *uint256.Int) uint256.Int {
	best := -1
	var bestOut uint256.Int
	for k := range pp.ins {
		var out uint256.Int
		if !pp.ok[k] || pp.res[k].Quote.AmountOut == nil || out.SetFromBig(pp.res[k].Quote.AmountOut) || out.IsZero() {
			continue
		}
		// Compare out/in by cross-multiplication. Both factors are far below
		// 2^128 (a probe size and a token amount), so the products fit.
		var l, r uint256.Int
		if best < 0 || l.Mul(&out, &pp.ins[best]).Gt(r.Mul(&bestOut, &pp.ins[k])) {
			best, bestOut = k, out
		}
	}
	var res uint256.Int
	if best >= 0 {
		big256.MulDivDown(&res, balOut, &pp.ins[best], &bestOut)
	}
	return res
}

// clampToCaps limits maxIn, the tokenIn amount worth the whole tokenOut balance, to what both
// legs' coverage caps admit: the input room of tokenIn directly, and the output room of
// tokenOut converted at the probed rate. A nil book (unreadable) leaves its side unclamped.
func (pp *pairProbe) clampToCaps(maxIn uint256.Int, balOut *uint256.Int, in, out *rpcAssetResult) uint256.Int {
	if in != nil {
		if room := legRoom(&in.Asset, false); room != nil && maxIn.Gt(room) {
			maxIn = *room
		}
	}
	if out != nil {
		if room := legRoom(&out.Asset, true); room != nil && !balOut.IsZero() {
			// maxIn is worth balOut, so the output room is worth maxIn*room/balOut.
			var scaled uint256.Int
			big256.MulDivDown(&scaled, &maxIn, room, balOut)
			if scaled.Lt(&maxIn) {
				maxIn = scaled
			}
		}
	}
	return maxIn
}

// ladder collects the pair's grid into strictly increasing samples. It stops at
// the first revert, and at the first output that stops growing: past the pool's
// liquidity cap swap_qe still takes the whole input but pays only the capped
// output, so a saturated sample would make the simulator overquote.
func (pp *pairProbe) ladder() []Point {
	var pts []Point
	pp.stop = len(pp.ins)
	for k := range pp.ins {
		if !pp.ok[k] {
			pp.stop = k
			break
		}
		var out uint256.Int
		if q := pp.res[k].Quote.AmountOut; q == nil || out.SetFromBig(q) {
			pp.stop = k
			break
		}
		if out.IsZero() && len(pts) == 0 {
			continue // too small to fill: the simulator scales from the first sample
		}
		if len(pts) > 0 && !out.Gt(&pts[len(pts)-1].Out) {
			pp.stop = k
			break
		}
		pts = append(pts, Point{In: pp.ins[k], Out: out})
	}
	return pts
}

// bisectAmounts returns the sizes strictly between the last served sample and the first one the
// pool refused, when the ladder ended on a revert: the pool's real cap lies in that gap. They are quartiles, so one extra multicall narrows the gap to a
// quarter. Nil when the ladder ended any other way. The probes read the quote view, so the top of
// the ladder can still revert on a check only settlement runs.
func (pp *pairProbe) bisectAmounts(pts []Point) []uint256.Int {
	k := pp.stop
	if len(pts) == 0 || k == 0 || k >= len(pp.ins) || pp.ok[k] || !pts[len(pts)-1].In.Eq(&pp.ins[k-1]) {
		return nil
	}
	lo, hi := &pp.ins[k-1], &pp.ins[k]
	var gap, step, off uint256.Int
	gap.Sub(hi, lo)
	var res []uint256.Int
	for q := uint64(1); q <= 3; q++ {
		// gap is a token amount far below 2^192, so gap*q fits.
		step.Div(step.Mul(&gap, off.SetUint64(q)), uint256.NewInt(4))
		var amt uint256.Int
		if amt.Add(lo, &step); amt.Gt(lo) && amt.Lt(hi) && (len(res) == 0 || amt.Gt(&res[len(res)-1])) {
			res = append(res, amt)
		}
	}
	return res
}

// extend appends the served bisect samples to pts, stopping at the first revert or flat output,
// like ladder().
func (pp *pairProbe) extend(pts []Point) []Point {
	for k := range pp.ins {
		var out uint256.Int
		if !pp.ok[k] || pp.res[k].Quote.AmountOut == nil || out.SetFromBig(pp.res[k].Quote.AmountOut) || !out.Gt(&pts[len(pts)-1].Out) {
			break
		}
		pts = append(pts, Point{In: pp.ins[k], Out: out})
	}
	return pts
}

// mergeInto adds the filled rate probes to pts, the grid ladder. The probes come
// from the same block and cost no extra call; the small ones sit below the first
// grid sample and cut the dust extrapolation. A probe is dropped, never allowed
// to cut the ladder, unless it lies strictly inside it with an output strictly
// between its neighbours' (pts stays strictly increasing in In and Out).
func (pp *pairProbe) mergeInto(pts []Point) []Point {
	if len(pts) == 0 {
		return pts
	}
	last := pts[len(pts)-1].In // by value: slices.Insert may shift pts in place
	for k := range pp.ins {
		in := &pp.ins[k]
		var out uint256.Int
		if !pp.ok[k] || !in.Lt(&last) {
			continue
		}
		if q := pp.res[k].Quote.AmountOut; q == nil || out.SetFromBig(q) || out.IsZero() {
			continue
		}
		i, found := slices.BinarySearchFunc(pts, in, func(p Point, x *uint256.Int) int { return p.In.Cmp(x) })
		if found || (i > 0 && !out.Gt(&pts[i-1].Out)) || !out.Lt(&pts[i].Out) {
			continue
		}
		pts = slices.Insert(pts, i, Point{In: *in, Out: out})
	}
	return pts
}

// unitAmount returns 10^(decimals+exp) wei, at least 1.
func unitAmount(decimals uint8, exp int) uint256.Int {
	var res uint256.Int
	res.SetUint64(1)
	if e := int(decimals) + exp; e > 0 {
		res.Exp(uint256.NewInt(10), uint256.NewInt(uint64(e)))
	}
	return res
}

// kneeAmounts returns the extra samples around the cap onset: permille of the
// previous ladder's last In when it has one, else of maxIn.
func kneeAmounts(prev []Point, maxIn *uint256.Int) []uint256.Int {
	base, permille := maxIn, coldKneePermille[:]
	if len(prev) >= 2 {
		base, permille = &prev[len(prev)-1].In, warmKneePermille[:]
	}
	res := make([]uint256.Int, len(permille))
	var m, d uint256.Int
	d.SetUint64(1000)
	for k, pm := range permille {
		// base is a token amount far below 2^192, so base*pm fits.
		res[k].Div(res[k].Mul(base, m.SetUint64(pm)), &d)
	}
	return res
}

// gridAmounts returns maxIn scaled by gridPermille plus the knee amounts,
// sorted, dropping zeros and repeats so the amounts stay strictly increasing.
func gridAmounts(maxIn uint256.Int, knee []uint256.Int) []uint256.Int {
	all := make([]uint256.Int, len(gridPermille), len(gridPermille)+len(knee))
	var m, d uint256.Int
	d.SetUint64(1000)
	for k, pm := range gridPermille {
		// maxIn is a token amount far below 2^192, so maxIn*pm fits.
		all[k].Div(all[k].Mul(&maxIn, m.SetUint64(pm)), &d)
	}
	all = append(all, knee...)
	slices.SortFunc(all, func(a, b uint256.Int) int { return a.Cmp(&b) })
	amounts := all[:0]
	for k := range all {
		if !all[k].IsZero() && (len(amounts) == 0 || all[k].Gt(&amounts[len(amounts)-1])) {
			amounts = append(amounts, all[k])
		}
	}
	return amounts
}
