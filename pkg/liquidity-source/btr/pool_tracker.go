package btr

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/ethrpc"

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
// worth the full tokenOut balance at the probed rate. The core caps its output
// near 93% of the balance, and the curve is near-linear below that, so the
// samples are denser toward the top where the cap shows up.
var gridPermille = [...]uint64{1, 3, 10, 30, 60, 100, 200, 300, 400, 500, 600, 700, 800, 900, 1000}

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

// pairProbe holds the rate probes of one ordered pair, then its grid.
type pairProbe struct {
	from, to int
	ins      []uint256.Int
	res      []*rpcSwapQuoteResult
	ok       []bool
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

	// Phase 1: flags, balances and the rate probes, all in one multicall.
	flags := make([]uint16, n)
	balances := make([]*big.Int, n)
	req := t.client.NewRequest().SetContext(ctx)
	for i := range addrs {
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: p.Address, Method: "getRiskFlags", Params: []any{addrs[i]},
		}, []any{&flags[i]})
		req.AddCall(&ethrpc.Call{
			ABI: utilabi.Erc20ABI, Target: p.Tokens[i].Address, Method: "balanceOf", Params: []any{poolAddr},
		}, []any{&balances[i]})
	}
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
			pp.addQuoteCalls(req, p.Address, addrs[i], addrs[j])
			probes = append(probes, pp)
		}
	}
	// TryBlockAndAggregate tolerates a reverting leg; resp.Result has one
	// success flag per call in request order.
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for i := range addrs {
		if !resp.Result[2*i] {
			flags[i] = 0 // unreadable flags read as swap disabled
		}
	}
	probeOK := resp.Result[2*n:]
	for _, pp := range probes {
		pp.ok, probeOK = probeOK[:len(pp.ins)], probeOK[len(pp.ins):]
	}

	// Phase 2: the grid, pinned to the block of phase 1.
	grids := make([]*pairProbe, 0, len(probes))
	req = t.client.NewRequest().SetContext(ctx).SetBlockNumber(resp.BlockNumber)
	for _, pp := range probes {
		if balances[pp.to] == nil {
			continue
		}
		var balOut uint256.Int
		if balOut.SetFromBig(balances[pp.to]) {
			continue
		}
		g := &pairProbe{from: pp.from, to: pp.to, ins: gridAmounts(pp.maxIn(&balOut))}
		if len(g.ins) == 0 {
			continue
		}
		g.addQuoteCalls(req, p.Address, addrs[pp.from], addrs[pp.to])
		grids = append(grids, g)
	}
	extra := Extra{Flags: flags}
	if len(grids) > 0 {
		gridResp, err := req.TryBlockAndAggregate()
		if err != nil {
			return p, err
		}
		okAll := gridResp.Result
		for _, g := range grids {
			g.ok, okAll = okAll[:len(g.ins)], okAll[len(g.ins):]
			if pts := g.ladder(); len(pts) > 0 {
				extra.Ladders = append(extra.Ladders, Ladder{From: g.from, To: g.to, Points: pts})
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
		if bal != nil {
			p.Reserves[i] = bal.String()
		}
	}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// addQuoteCalls queues one getSwapQuote per amount in pp.ins.
func (pp *pairProbe) addQuoteCalls(req *ethrpc.Request, poolAddr string, tokenIn, tokenOut common.Address) {
	pp.res = make([]*rpcSwapQuoteResult, len(pp.ins))
	for k := range pp.ins {
		pp.res[k] = new(rpcSwapQuoteResult)
		req.AddCall(&ethrpc.Call{
			ABI: poolABI, Target: poolAddr, Method: "getSwapQuote",
			Params: []any{tokenIn, tokenOut, pp.ins[k].ToBig()},
		}, []any{pp.res[k]})
	}
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

// ladder collects the pair's grid into strictly increasing samples. It stops at
// the first revert, and at the first output that stops growing: past the pool's
// liquidity cap swap_qe still takes the whole input but pays only the capped
// output, so a saturated sample would make the simulator overquote.
func (pp *pairProbe) ladder() []Point {
	var pts []Point
	for k := range pp.ins {
		if !pp.ok[k] {
			break
		}
		var out uint256.Int
		if q := pp.res[k].Quote.AmountOut; q == nil || out.SetFromBig(q) {
			break
		}
		if out.IsZero() && len(pts) == 0 {
			continue // too small to fill: the simulator scales from the first sample
		}
		if len(pts) > 0 && !out.Gt(&pts[len(pts)-1].Out) {
			break
		}
		pts = append(pts, Point{In: pp.ins[k], Out: out})
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

// gridAmounts returns maxIn scaled by gridPermille, dropping zeros and repeats
// so the amounts stay strictly increasing.
func gridAmounts(maxIn uint256.Int) []uint256.Int {
	amounts := make([]uint256.Int, 0, len(gridPermille))
	var a, m, d uint256.Int
	d.SetUint64(1000)
	for _, pm := range gridPermille {
		// maxIn is a token amount far below 2^192, so maxIn*pm fits.
		a.Div(a.Mul(&maxIn, m.SetUint64(pm)), &d)
		if !a.IsZero() && (len(amounts) == 0 || a.Gt(&amounts[len(amounts)-1])) {
			amounts = append(amounts, a)
		}
	}
	return amounts
}
