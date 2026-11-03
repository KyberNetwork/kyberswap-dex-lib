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
	test.SkipCI(t)
	url := os.Getenv("THOG_PROP_RPC_URL")
	tenderly := tenderlySettler{account: os.Getenv("THOG_PROP_TENDERLY_ACCOUNT"),
		project: os.Getenv("THOG_PROP_TENDERLY_PROJECT"), key: os.Getenv("THOG_PROP_TENDERLY_ACCESS_KEY")}
	if url == "" || tenderly.account == "" || tenderly.project == "" || tenderly.key == "" {
		t.Skip("THOG_PROP_RPC_URL and THOG_PROP_TENDERLY_{ACCOUNT,PROJECT,ACCESS_KEY} required")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	c, err := rpc.DialContext(ctx, url)
	require.NoError(t, err)
	t.Cleanup(c.Close)

	sim := trackedSimulator(ctx, t, url)
	next := waitHeader(ctx, t, c, sim.state.BlockNumber+1)
	tenderly.parent, tenderly.block = sim.state.BlockNumber, next
	execCtx := ExecutionContext{BlockNumber: uint64(next.Number)}
	execCtx.BaseFee.SetFromBig(next.BaseFee.ToInt())
	execCtx.GasPrice = execCtx.BaseFee

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
			rows[i].settle, rows[i].settleErr = tenderly.settle(gctx, rows[i].in, rows[i].out, rows[i].amountIn)
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

// quote is CalcAmountOut at zero elapsed time (quote block N+1).
func (s verifySim) quote(in, out verifyToken, amt *big.Int) (*big.Int, error) {
	r, err := s.calcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in.hex(), Amount: amt},
		TokenOut: out.hex()}, s.capturedAtMs)
	if err != nil {
		return nil, err
	}
	return r.TokenAmountOut.Amount, nil
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
	StateObjects   map[string]map[string]any `json:"state_objects"`
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

// settle funds amt of in through its balance slot, then runs [balanceOf(in),
// balanceOf(out), approve, swap, balanceOf(out)] and returns the tokenOut delta.
func (s tenderlySettler) settle(ctx context.Context, in, out verifyToken, amt *big.Int) (*big.Int, error) {
	sender, proxy := common.HexToAddress(verifySender), common.HexToAddress(verifyProxy)
	units := amt
	if in.ratio != nil { // share token: store floor(amt/ratio)+1 so balanceOf covers amt
		q := new(big.Rat).Quo(new(big.Rat).SetInt(amt), in.ratio)
		units = new(big.Int).Add(new(big.Int).Quo(q.Num(), q.Denom()), big.NewInt(1))
	}
	slot := crypto.Keccak256Hash(common.LeftPadBytes(sender[:], 32), common.LeftPadBytes(big.NewInt(in.slot).Bytes(), 32))
	overrides := map[string]map[string]any{
		verifySender: {"balance": "100000000000000000000"},
		in.hex():     {"storage": map[string]string{slot.Hex(): common.BigToHash(units).Hex()}},
	}
	expiry := new(big.Int).SetUint64(uint64(s.block.Time) + 3600)
	balanceOf := pack("balanceOf", sender)
	calls := []struct {
		to   common.Address
		data hexutil.Bytes
	}{
		{in.address, balanceOf},
		{out.address, balanceOf},
		{in.address, pack("approve", proxy, amt)},
		{proxy, pack("swapWithExtraFees", in.address, out.address, amt, new(big.Int), expiry, new(big.Int),
			common.HexToAddress(verifyFeeTo))},
		{out.address, balanceOf},
	}
	// block_number B runs on the state at the END of B, so execute on the tracker
	// block's state with the next block's header (number, time, base fee).
	header := map[string]any{"number": s.block.Number, "timestamp": s.block.Time, "baseFeePerGas": &s.block.BaseFee}
	sims := make([]tenderlySim, len(calls))
	for i, call := range calls {
		// Zero priority fee: gas price is the block base fee.
		sims[i] = tenderlySim{NetworkID: verifyChainID, From: verifySender, To: call.to, Input: call.data, Gas: 3_000_000,
			GasPrice: s.block.BaseFee.ToInt().String(), Value: "0", BlockNumber: s.parent, BlockHeader: header,
			SimulationType: "quick", StateObjects: overrides}
	}
	results, err := s.simulateBundle(ctx, sims)
	if err != nil {
		return nil, err
	}
	if len(results) != len(sims) {
		return nil, fmt.Errorf("tenderly: %d results for %d simulations", len(results), len(sims))
	}
	word := func(i int) *big.Int {
		return new(big.Int).SetBytes(results[i].Transaction.TransactionInfo.CallTrace.Output)
	}
	for i, r := range results {
		if !r.Transaction.Status {
			msg := r.Transaction.ErrorMessage
			if e := r.Transaction.TransactionInfo.CallTrace.Error; e != "" && e != msg {
				msg += ": " + e
			}
			return nil, fmt.Errorf("sim %d reverted: %s", i, msg)
		}
	}
	if got := word(0); got.Cmp(amt) < 0 {
		return nil, fmt.Errorf("funding %s: balance %s < %s", in.symbol, got, amt)
	}
	return new(big.Int).Sub(word(4), word(1)), nil
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
