package thogprop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const (
	verifyProxy   = "0x80c74517bcc2d67ffe02d3ed886796272f647210"
	verifyPoolID  = "0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"
	verifySender  = "0x00000000000000000000000000000000c0de0001"
	verifyFunder  = "0x00000000000000000000000000000000c0de0002"
	verifyFeeTo   = "0x1f55fFAc09C8Cd7aAa817EE07Ed43bEE3B424F47"
	verifyChainID = "143"
)

// verifyToken carries the token-balance-slot detection used to fund the sender
// (bin/tbs on the live Monad RPC, 2026-11-03; all solidity layouts).
type verifyToken struct {
	symbol  string
	address common.Address
	slot    int64
	ratio   *big.Rat // balance per storage unit for share tokens; nil for plain balances
}

func (v verifyToken) hex() string { return hexutil.Encode(v.address[:]) }

var verifyTokens = map[string]verifyToken{
	"USDC":  {"USDC", common.HexToAddress("0x754704bc059f8c67012fed69bc8a327a5aafb603"), 9, nil}, // bits 255
	"USDT0": {"USDT0", common.HexToAddress("0xe7cd86e13ac4309349f30b3435a9d337750fc82d"), 0x33, nil},
	"WMON":  {"WMON", common.HexToAddress("0x3bd359c1119da7da1d913d1c4d2b7c461115433a"), 3, nil},
	"WETH":  {"WETH", common.HexToAddress("0xee8c0e9f1bffb4eb878d8f15f368a02a35481242"), 0x33, nil},
	"SPYx": {"SPYx", common.HexToAddress("0x90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48"), 0x107,
		mustRat("1.007474126658765999")}, // size 24
}

// verifyABI packs the settlement calls; swapWithExtraFees is ThogAMM's 0xee744696.
var verifyABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[
	{"type":"function","name":"balanceOf","inputs":[{"type":"address"}],"outputs":[{"type":"uint256"}]},
	{"type":"function","name":"approve","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"bool"}]},
	{"type":"function","name":"transfer","inputs":[{"type":"address"},{"type":"uint256"}],"outputs":[{"type":"bool"}]},
	{"type":"function","name":"swapWithExtraFees","inputs":[{"type":"address"},{"type":"address"},{"type":"uint256"},
		{"type":"uint256"},{"type":"uint256"},{"type":"uint256"},{"type":"address"}],"outputs":[{"type":"uint256"}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

func mustRat(s string) *big.Rat {
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		panic("bad rational " + s)
	}
	return r
}

// TestVerification reads the pool with the real tracker at the live head N, quotes
// with the simulator at the routing block N+1 (zero elapsed time), and settles each
// case as a Tenderly bundle on the state after N with block N+1's header: approve plus ThogAMM's own
// swapWithExtraFees, priced at the block base fee (zero priority fee), with the
// sender funded through a storage override. Tenderly runs Monad's gas schedule, so
// the AuctionHandler warmth probe reads cold and no execution defense applies; the
// gate is State.QuoteExecution with that context, CalcAmountOut is shown beside it.
//
// Env: THOG_PROP_RPC_URL (Monad RPC), THOG_PROP_TENDERLY_{ACCOUNT,PROJECT,ACCESS_KEY};
// THOG_PROP_PARITY_OUT optionally receives the markdown table.
func TestVerification(t *testing.T) {
	ctx, sim, tenderly, execCtx := setupVerification(t)
	next := tenderly.block
	var err error

	pairs := [][2]string{{"USDC", "WMON"}, {"WMON", "USDC"}, {"USDC", "USDT0"}, {"USDT0", "USDC"},
		{"USDC", "SPYx"}, {"SPYx", "USDC"}, {"WMON", "WETH"}, {"WETH", "WMON"}}
	sizes := []string{"1", "100", "777.123457", "2500", "50000000"} // USD, exact decimals

	// Quote locally, then settle every case concurrently: bundles are independent.
	usdc := verifyTokens["USDC"]
	rows := make([]verifyRow, 0, len(pairs)*len(sizes))
	for _, p := range pairs {
		in, out := verifyTokens[p[0]], verifyTokens[p[1]]
		perDollar := big.NewInt(1e6)
		if in != usdc { // scale the $1 rate so sizes past inventory still get an amount
			perDollar, err = sim.quote(usdc, in, perDollar)
			require.NoError(t, err, "sizing %s", in.symbol)
		}
		for _, usd := range sizes {
			r := verifyRow{in: in, out: out, usd: usd, amountIn: scale(perDollar, usd)}
			r.sim, r.simErr = sim.quote(in, out, r.amountIn)
			r.exec, r.execErr = sim.quoteExecution(in, out, r.amountIn, execCtx)
			rows = append(rows, r)
		}
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for i := range rows {
		g.Go(func() error {
			hop := []routeHop{{in: rows[i].in, out: rows[i].out, amount: rows[i].amountIn}}
			rows[i].settle, rows[i].settleErr = tenderly.settleRoute(gctx, hop)[0].get()
			return nil
		})
	}
	require.NoError(t, g.Wait())

	var md strings.Builder
	fmt.Fprintf(&md, "Tracker block N=%d (lens). Settlement: Tenderly bundle on state N, header N+1 (base fee %s wei, priority fee 0), "+
		"extraFeePpm=0.\n\n", sim.state.BlockNumber, next.BaseFee.ToInt())
	md.WriteString("simOut = router CalcAmountOut (read-only). execOut = State.QuoteExecution at N+1, gas price = base fee, " +
		"cold AuctionHandler. Gate: execOut == settleOut; xStock output may exceed it by share rounding.\n\n")
	md.WriteString("| pair | usd | amountIn | simOut | execOut | settleOut | exec−settle | sim−settle (bp) | verdict |\n" +
		"|---|---|---|---|---|---|---|---|---|\n")
	for i := range rows {
		r := &rows[i]
		r.classify()
		md.WriteString(r.markdown())
	}
	for _, p := range pairs {
		t.Run(p[0]+"->"+p[1], func(t *testing.T) {
			for _, r := range rows {
				if r.in.symbol != p[0] || r.out.symbol != p[1] {
					continue
				}
				t.Logf("$%s in=%s sim=%s exec=%s settle=%s → %s", r.usd, r.amountIn,
					short(r.sim, r.simErr), short(r.exec, r.execErr), short(r.settle, r.settleErr), r.verdict)
				if !r.ok {
					t.Errorf("$%s: %s", r.usd, r.verdict)
				}
			}
		})
	}
	t.Log("\n" + md.String())
	if path := os.Getenv("THOG_PROP_PARITY_OUT"); path != "" {
		require.NoError(t, os.WriteFile(path, []byte(md.String()), 0o644))
	}
}

// setupVerification reads the live pool with the real tracker at head N and waits
// for block N+1, whose header prices both the execution quote and the settlement.
func setupVerification(t *testing.T) (context.Context, verifySim, tenderlySettler, ExecutionContext) {
	t.Helper()
	test.SkipCI(t)
	url := os.Getenv("THOG_PROP_RPC_URL")
	tenderly := tenderlySettler{account: os.Getenv("THOG_PROP_TENDERLY_ACCOUNT"),
		project: os.Getenv("THOG_PROP_TENDERLY_PROJECT"), key: os.Getenv("THOG_PROP_TENDERLY_ACCESS_KEY")}
	if url == "" || tenderly.account == "" || tenderly.project == "" || tenderly.key == "" {
		t.Skip("THOG_PROP_RPC_URL and THOG_PROP_TENDERLY_{ACCOUNT,PROJECT,ACCESS_KEY} required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	t.Cleanup(cancel)
	c, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	t.Cleanup(c.Close)

	sim := trackedSimulator(ctx, t, url)
	next := waitHeader(ctx, t, c, sim.state.BlockNumber+1)
	tenderly.parent, tenderly.block = sim.state.BlockNumber, next
	execCtx := ExecutionContext{BlockNumber: uint64(next.Number)}
	execCtx.BaseFee.SetFromBig(next.BaseFee.ToInt())
	execCtx.GasPrice = execCtx.BaseFee
	return ctx, sim, tenderly, execCtx
}

// TestVerificationMultiHop settles routes that hit the pool several times in one
// transaction sequence (same block): round trips, repeats and chains. The simulator
// quotes each hop on a clone after UpdateBalance with the previous hop's SwapInfo;
// the chain runs the hops back to back in one bundle. Each hop is gated like a
// single swap, so a stale or partial UpdateBalance shows as a hop-2 mismatch.
func TestVerificationMultiHop(t *testing.T) {
	ctx, sim, tenderly, execCtx := setupVerification(t)
	type step struct {
		in, out string
		chained bool // amount = previous hop's simulated output; else the route size
	}
	routes := [][]step{
		{{"USDC", "WMON", false}, {"WMON", "USDC", true}},
		{{"USDC", "WMON", false}, {"USDC", "WMON", false}},
		{{"USDC", "WMON", false}, {"WMON", "WETH", true}},
		{{"WMON", "USDC", false}, {"USDC", "SPYx", true}},
		{{"USDC", "SPYx", false}, {"SPYx", "USDC", true}},
		{{"SPYx", "USDC", false}, {"USDC", "WMON", true}, {"WMON", "USDC", true}},
		{{"WETH", "WMON", false}, {"WETH", "WMON", false}, {"WMON", "WETH", true}},
	}
	sizes := []string{"1", "250", "2500"}

	type routeCase struct {
		name string
		rows []verifyRow
	}
	cases := make([]routeCase, 0, len(routes)*len(sizes))
	usdc := verifyTokens["USDC"]
	for _, route := range routes {
		first := verifyTokens[route[0].in]
		perDollar := big.NewInt(1e6)
		if first != usdc {
			var err error
			perDollar, err = sim.quote(usdc, first, perDollar)
			require.NoError(t, err, "sizing %s", first.symbol)
		}
		names := []string{route[0].in}
		for _, st := range route {
			names = append(names, st.out)
		}
		for _, usd := range sizes {
			size, clone := scale(perDollar, usd), sim.clone()
			rc := routeCase{name: strings.Join(names, "→") + " $" + usd}
			var prev *big.Int
			for k, st := range route {
				r := verifyRow{in: verifyTokens[st.in], out: verifyTokens[st.out], usd: fmt.Sprintf("%s #%d", usd, k+1),
					amountIn: size}
				if st.chained {
					r.amountIn = prev
				}
				if r.amountIn == nil { // an earlier hop was rejected: the route stops there
					break
				}
				res, err := clone.quoteResult(r.in, r.out, r.amountIn)
				r.exec, r.execErr = clone.quoteExecution(r.in, r.out, r.amountIn, execCtx)
				if r.simErr = err; err == nil {
					r.sim = res.TokenAmountOut.Amount
					clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: res.SwapInfo})
				}
				prev = r.sim
				rc.rows = append(rc.rows, r)
			}
			cases = append(cases, rc)
		}
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)
	for i := range cases {
		g.Go(func() error {
			hops := make([]routeHop, len(cases[i].rows))
			for k, r := range cases[i].rows {
				hops[k] = routeHop{in: r.in, out: r.out, amount: r.amountIn}
			}
			for k, h := range tenderly.settleRoute(gctx, hops) {
				cases[i].rows[k].settle, cases[i].rows[k].settleErr = h.get()
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	var md strings.Builder
	fmt.Fprintf(&md, "Tracker block N=%d. Each route settles as one Tenderly bundle on state N with header N+1; "+
		"hop k+1 is quoted on a clone after UpdateBalance(hop k).\n\n", tenderly.parent)
	md.WriteString("| hop | usd #hop | amountIn | simOut | execOut | settleOut | exec−settle | sim−settle (bp) | verdict |\n" +
		"|---|---|---|---|---|---|---|---|---|\n")
	for i := range cases {
		t.Run(cases[i].name, func(t *testing.T) {
			for k := range cases[i].rows {
				r := &cases[i].rows[k]
				r.classify()
				md.WriteString(r.markdown())
				t.Logf("#%d %s→%s in=%s sim=%s exec=%s settle=%s → %s", k+1, r.in.symbol, r.out.symbol, r.amountIn,
					short(r.sim, r.simErr), short(r.exec, r.execErr), short(r.settle, r.settleErr), r.verdict)
				if !r.ok {
					t.Errorf("hop %d: %s", k+1, r.verdict)
				}
			}
		})
	}
	t.Log("\n" + md.String())
	if path := os.Getenv("THOG_PROP_MULTIHOP_OUT"); path != "" {
		require.NoError(t, os.WriteFile(path, []byte(md.String()), 0o644))
	}
}

type blockHeader struct {
	Number  hexutil.Uint64 `json:"number"`
	Time    hexutil.Uint64 `json:"timestamp"`
	BaseFee hexutil.Big    `json:"baseFeePerGas"`
}

// waitHeader polls until block n exists (Monad produces one every ~0.3 s).
func waitHeader(ctx context.Context, t *testing.T, c *rpc.Client, n uint64) blockHeader {
	t.Helper()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); <-tick.C {
		var h *blockHeader
		require.NoError(t, c.CallContext(ctx, &h, "eth_getBlockByNumber", hexutil.Uint64(n), false))
		if h != nil {
			return *h
		}
	}
	t.Fatalf("block %d not produced", n)
	return blockHeader{}
}

// verifySim quotes against the tracker's own output, never a hand-built state.
type verifySim struct {
	*PoolSimulator
	index map[string]int
}

func trackedSimulator(ctx context.Context, t *testing.T, url string) verifySim {
	t.Helper()
	// A cold fork fetches every slot upstream: allow far more than the 2 s default.
	tracker, err := NewPoolTracker(&Config{Proxy: verifyProxy, PoolId: verifyPoolID, RequestTimeoutMs: 60_000}, ethrpc.New(url))
	require.NoError(t, err)
	static, err := json.Marshal(StaticExtra{PoolId: verifyPoolID})
	require.NoError(t, err)
	ep, err := tracker.GetNewPoolState(ctx, entity.Pool{Address: verifyProxy, Exchange: string(DexType), Type: DexType,
		StaticExtra: string(static)}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NotZero(t, ep.BlockNumber, "tracker must report its snapshot block")
	sim, err := NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	index := make(map[string]int, len(sim.state.Tokens))
	for i, token := range sim.state.Tokens {
		index[token.Address] = i
	}
	return verifySim{sim, index}
}

// quoteResult is CalcAmountOut at zero elapsed time (quote block N+1).
func (s verifySim) quoteResult(in, out verifyToken, amt *big.Int) (*pool.CalcAmountOutResult, error) {
	return s.calcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in.hex(), Amount: amt},
		TokenOut: out.hex()}, s.capturedAtMs)
}

func (s verifySim) quote(in, out verifyToken, amt *big.Int) (*big.Int, error) {
	r, err := s.quoteResult(in, out, amt)
	if err != nil {
		return nil, err
	}
	return r.TokenAmountOut.Amount, nil
}

// clone returns an independent simulator for applying UpdateBalance along a route.
func (s verifySim) clone() verifySim {
	return verifySim{s.CloneState().(*PoolSimulator), s.index}
}

func (s verifySim) quoteExecution(in, out verifyToken, amt *big.Int, ctx ExecutionContext) (*big.Int, error) {
	var a uint256.Int
	if a.SetFromBig(amt) {
		return nil, ErrOverflow
	}
	q, err := s.state.QuoteExecution(s.index[in.hex()], s.index[out.hex()], a, ctx)
	if err != nil {
		return nil, err
	}
	return q.AmountOut.ToBig(), nil
}

// scale returns floor(perDollar × usd) exactly.
func scale(perDollar *big.Int, usd string) *big.Int {
	r := new(big.Rat).Mul(new(big.Rat).SetInt(perDollar), mustRat(usd))
	return new(big.Int).Quo(r.Num(), r.Denom())
}

type verifyRow struct {
	in, out                     verifyToken
	usd                         string
	amountIn, sim, exec, settle *big.Int
	simErr, execErr, settleErr  error
	verdict, diff, simBps       string
	ok                          bool
}

// classify gates execOut against settlement. xStock settlement floors stock output
// plus dust to shares, so receipt may exceed the guarantee (the guide's received >= out).
func (r *verifyRow) classify() {
	xStockOut := r.out.ratio != nil
	r.diff, r.simBps = "-", "-"
	switch {
	case r.simErr != nil && r.execErr != nil && r.settleErr != nil:
		r.verdict, r.ok = "both reject", true
	case r.simErr != nil || r.execErr != nil:
		r.verdict = "FAIL sim rejects, chain pays"
	case r.settleErr != nil:
		r.verdict = "FAIL sim quotes, chain reverts"
	default:
		d := new(big.Int).Sub(r.exec, r.settle)
		r.diff = d.String()
		if r.settle.Sign() > 0 {
			bp := new(big.Int).Mul(new(big.Int).Sub(r.sim, r.settle), big.NewInt(10_000))
			r.simBps = new(big.Rat).SetFrac(bp, r.settle).FloatString(2)
		}
		switch {
		case d.Sign() == 0:
			r.verdict, r.ok = "exact", true
		case xStockOut && d.Sign() < 0 && d.CmpAbs(big.NewInt(2)) <= 0:
			r.verdict, r.ok = "≥ (xStock share rounding)", true
		default:
			r.verdict = "FAIL mismatch"
		}
	}
}

func (r *verifyRow) markdown() string {
	return fmt.Sprintf("| %s→%s | %s | %s | %s | %s | %s | %s | %s | %s |\n", r.in.symbol, r.out.symbol, r.usd, r.amountIn,
		short(r.sim, r.simErr), short(r.exec, r.execErr), short(r.settle, r.settleErr), r.diff, r.simBps, r.verdict)
}

func short(v *big.Int, err error) string {
	if err == nil {
		return v.String()
	}
	s := err.Error()
	if len(s) > 60 {
		s = s[:60]
	}
	return "err: " + strings.ReplaceAll(s, "|", "/")
}

func pack(method string, args ...any) hexutil.Bytes {
	data, err := verifyABI.Pack(method, args...)
	if err != nil {
		panic(err)
	}
	return data
}

// tenderlySettler settles one case per simulate-bundle call: state after parent,
// executed in block (parent+1).
type tenderlySettler struct {
	account, project, key string
	parent                uint64
	block                 blockHeader
}

type tenderlySim struct {
	NetworkID      string                    `json:"network_id"`
	From           string                    `json:"from"`
	To             common.Address            `json:"to"`
	Input          hexutil.Bytes             `json:"input"`
	Gas            uint64                    `json:"gas"`
	GasPrice       string                    `json:"gas_price"`
	Value          string                    `json:"value"`
	BlockNumber    uint64                    `json:"block_number"`
	BlockHeader    map[string]any            `json:"block_header"`
	SimulationType string                    `json:"simulation_type"`
	StateObjects   map[string]map[string]any `json:"state_objects,omitempty"`
}

type tenderlyResult struct {
	Transaction struct {
		Status          bool   `json:"status"`
		ErrorMessage    string `json:"error_message"`
		TransactionInfo struct {
			CallTrace struct {
				Output hexutil.Bytes `json:"output"`
				Error  string        `json:"error"`
			} `json:"call_trace"`
		} `json:"transaction_info"`
	} `json:"transaction"`
}

// routeHop is one swapWithExtraFees call of a route on the same pool.
type routeHop struct {
	in, out verifyToken
	amount  *big.Int
}

type hopResult struct {
	delta *big.Int
	err   error
}

func (h hopResult) get() (*big.Int, error) { return h.delta, h.err }

// settleRoute funds the first hop's input (the sum of every hop spending it), then
// runs one bundle: [transfer(sender) from the funder, balanceOf(in)] and per hop
// [balanceOf(out), approve, swap, balanceOf(out)]. Each hop's result is the sender's
// tokenOut delta across its swap; hops after a revert report it.
//
// The storage override sits on a separate funder: Tenderly keeps reporting an
// overridden slot's value through later simulations of the bundle, which would hide
// the sender's real balance whenever a hop pays out the funded token.
func (s tenderlySettler) settleRoute(ctx context.Context, hops []routeHop) []hopResult {
	results := make([]hopResult, len(hops))
	fail := func(err error) []hopResult {
		for i := range results {
			results[i].err = err
		}
		return results
	}
	sender, funder := common.HexToAddress(verifySender), common.HexToAddress(verifyFunder)
	proxy, first := common.HexToAddress(verifyProxy), hops[0].in
	funded := new(big.Int)
	for _, h := range hops {
		if h.in == first {
			funded.Add(funded, h.amount)
		}
	}
	sent, units := funded, funded
	if first.ratio != nil { // share token: transfer rounds shares down, so send 2 wei extra
		sent = new(big.Int).Add(funded, big.NewInt(2))
		q := new(big.Rat).Quo(new(big.Rat).SetInt(sent), first.ratio)
		units = new(big.Int).Add(new(big.Int).Quo(q.Num(), q.Denom()), big.NewInt(1))
	}
	slot := crypto.Keccak256Hash(common.LeftPadBytes(funder[:], 32), common.LeftPadBytes(big.NewInt(first.slot).Bytes(), 32))
	overrides := map[string]map[string]any{
		verifySender: {"balance": "100000000000000000000"},
		verifyFunder: {"balance": "100000000000000000000"},
		first.hex():  {"storage": map[string]string{slot.Hex(): common.BigToHash(units).Hex()}},
	}
	// block_number B runs on the state at the END of B, so execute on the tracker
	// block's state with the next block's header (number, time, base fee).
	header := map[string]any{"number": s.block.Number, "timestamp": s.block.Time, "baseFeePerGas": &s.block.BaseFee}
	expiry := new(big.Int).SetUint64(uint64(s.block.Time) + 3600)
	balanceOf := pack("balanceOf", sender)
	sim := func(to common.Address, data hexutil.Bytes) tenderlySim {
		// Zero priority fee: gas price is the block base fee.
		return tenderlySim{NetworkID: verifyChainID, From: verifySender, To: to, Input: data, Gas: 3_000_000,
			GasPrice: s.block.BaseFee.ToInt().String(), Value: "0", BlockNumber: s.parent, BlockHeader: header,
			SimulationType: "quick"}
	}
	// Overrides go on the first simulation only: later ones inherit the bundle's
	// state, and repeating them would reset the funded balance between hops.
	fund := sim(first.address, pack("transfer", sender, sent))
	fund.From, fund.StateObjects = verifyFunder, overrides
	sims := []tenderlySim{fund, sim(first.address, balanceOf)}
	for _, h := range hops {
		sims = append(sims,
			sim(h.out.address, balanceOf),
			sim(h.in.address, pack("approve", proxy, h.amount)),
			sim(proxy, pack("swapWithExtraFees", h.in.address, h.out.address, h.amount, new(big.Int), expiry,
				new(big.Int), common.HexToAddress(verifyFeeTo))),
			sim(h.out.address, balanceOf))
	}
	out, err := s.simulateBundle(ctx, sims)
	if err != nil {
		return fail(err)
	}
	if len(out) != len(sims) {
		return fail(fmt.Errorf("tenderly: %d results for %d simulations", len(out), len(sims)))
	}
	word := func(i int) *big.Int {
		return new(big.Int).SetBytes(out[i].Transaction.TransactionInfo.CallTrace.Output)
	}
	reverted := func(i int) error {
		tx := out[i].Transaction
		if tx.Status {
			return nil
		}
		msg := tx.ErrorMessage
		if e := tx.TransactionInfo.CallTrace.Error; e != "" && e != msg {
			msg += ": " + e
		}
		return fmt.Errorf("sim %d reverted: %s", i, msg)
	}
	for i := range 2 {
		if err = reverted(i); err != nil {
			return fail(err)
		}
	}
	if got := word(1); got.Cmp(funded) < 0 {
		return fail(fmt.Errorf("funding %s: balance %s < %s", first.symbol, got, funded))
	}
	for k := range hops {
		base := 2 + 4*k
		for i := base; i < base+4; i++ {
			if err = reverted(i); err != nil {
				for j := k; j < len(hops); j++ {
					results[j].err = err
				}
				return results
			}
		}
		results[k].delta = new(big.Int).Sub(word(base+3), word(base))
	}
	return results
}

func (s tenderlySettler) simulateBundle(ctx context.Context, sims []tenderlySim) ([]tenderlyResult, error) {
	body, err := json.Marshal(map[string]any{"simulations": sims})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.tenderly.co/api/v1/account/%s/project/%s/simulate-bundle", s.account, s.project)
	for attempt := 1; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Access-Key", s.key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		// Rate limits and transient upstream errors: back off and retry.
		if (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < 5 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("tenderly %d: %.200s", resp.StatusCode, raw)
		}
		var out struct {
			Results []tenderlyResult `json:"simulation_results"`
		}
		if err = json.Unmarshal(raw, &out); err != nil {
			return nil, fmt.Errorf("tenderly response: %w", err)
		}
		return out.Results, nil
	}
}
