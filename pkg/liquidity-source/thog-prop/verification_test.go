package thogprop

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// TestVerification runs the real tracker against an anvil fork whose head is the
// pinned block N (started with --no-mining), quotes with the simulator at the
// routing block N+1 (zero elapsed time), and settles each case with ThogAMM's own
// swapWithExtraFees mined in block N+1 at zero priority fee. THOG_PROP_RPC_URL
// must point at that anvil fork.
func TestVerification(t *testing.T) {
	test.SkipCI(t)
	url := os.Getenv("THOG_PROP_RPC_URL")
	if url == "" {
		t.Skip("THOG_PROP_RPC_URL not set")
	}
	const (
		proxy  = "0x80c74517bcc2d67ffe02d3ed886796272f647210"
		poolID = "0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"
		sender = "0x00000000000000000000000000000000c0de0001"
		feeTo  = "0x1f55fFAc09C8Cd7aAa817EE07Ed43bEE3B424F47"
	)
	tok := map[string]string{
		"USDC": "0x754704bc059f8c67012fed69bc8a327a5aafb603", "USDT0": "0xe7cd86e13ac4309349f30b3435a9d337750fc82d",
		"WMON": "0x3bd359c1119da7da1d913d1c4d2b7c461115433a", "WETH": "0xee8c0e9f1bffb4eb878d8f15f368a02a35481242",
		"SPYx": "0x90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48",
	}
	ctx := context.Background()
	c, err := rpc.Dial(url)
	require.NoError(t, err)
	var head hexutil.Uint64
	require.NoError(t, c.CallContext(ctx, &head, "eth_blockNumber"))

	tracker, err := NewPoolTracker(&Config{Proxy: proxy, PoolId: poolID, RequestTimeoutMs: 60_000} /* cold fork */, ethrpc.New(url))
	require.NoError(t, err)
	static, _ := json.Marshal(StaticExtra{PoolId: poolID})
	ep, err := tracker.GetNewPoolState(ctx, entity.Pool{Address: proxy, Exchange: string(DexType), Type: DexType, StaticExtra: string(static)}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.Equal(t, uint64(head), ep.BlockNumber, "tracker must read the pinned head")
	sim, err := NewPoolSimulatorFromParams(pool.FactoryParams{EntityPool: ep, Opts: pool.FactoryOpts{StaleCheck: true}})
	require.NoError(t, err)
	quote := func(in, out string, amt *big.Int) (*big.Int, error) {
		r, err := sim.calcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: in, Amount: amt}, TokenOut: out}, sim.capturedAtMs)
		if err != nil {
			return nil, err
		}
		return r.TokenAmountOut.Amount, nil
	}

	// Input sizes in USD, converted through the simulator's own USDC quote.
	usd := func(sym string, dollars float64) *big.Int {
		u := new(big.Int).SetInt64(int64(dollars * 1e6))
		if sym == "USDC" {
			return u
		}
		// Scale the $1 rate so sizes beyond inventory still produce an amount.
		out, err := quote(tok["USDC"], tok[sym], big.NewInt(1e6))
		require.NoError(t, err, "sizing %s", sym)
		return out.Div(out.Mul(out, u), big.NewInt(1e6))
	}
	pairs := [][2]string{{"USDC", "WMON"}, {"WMON", "USDC"}, {"USDC", "USDT0"}, {"USDT0", "USDC"},
		{"USDC", "SPYx"}, {"SPYx", "USDC"}, {"WMON", "WETH"}, {"WETH", "WMON"}}
	sizes := []float64{1, 100, 777.123457, 2500, 5e7}

	var md strings.Builder
	fmt.Fprintf(&md, "Block N=%d (tracker, lens). Settlement: swapWithExtraFees mined in N+1, extraFeePpm=0, priority fee 0.\n\n", head)
	md.WriteString("simOut = router CalcAmountOut (read-only, no execution defense). execOut = State.QuoteExecution at N+1 with a warm AuctionHandler: anvil's Ethereum gas schedule makes the contract's gas-0 STATICCALL warmth probe read warm, so settlement carries the 12 bp defense. Gate: execOut == settleOut.\n\n")
	md.WriteString("| pair | usd | amountIn | simOut | execOut | settleOut | exec−settle | sim−settle (bp) | verdict |\n|---|---|---|---|---|---|---|---|---|\n")
	byAddr := map[string]int{}
	for i, tk := range sim.state.Tokens {
		byAddr[tk.Address] = i
	}
	exec := func(in, out string, amt *big.Int) (*big.Int, error) {
		var a uint256.Int
		if a.SetFromBig(amt) {
			return nil, ErrOverflow
		}
		q, err := sim.state.QuoteExecution(byAddr[in], byAddr[out], a, ExecutionContext{BlockNumber: uint64(head) + 1, AuctionHandlerWarm: true})
		if err != nil {
			return nil, err
		}
		return q.AmountOut.ToBig(), nil
	}
	fails := 0
	for _, p := range pairs {
		for _, d := range sizes {
			in := usd(p[0], d)
			simOut, simErr := quote(tok[p[0]], tok[p[1]], in)
			execOut, execErr := exec(tok[p[0]], tok[p[1]], in)
			settle, setErr := settleSwap(ctx, c, tok[p[0]], tok[p[1]], in, proxy, sender, feeTo)
			verdict, diff, bp := "", "-", "-"
			switch {
			case simErr != nil && execErr != nil && setErr != nil:
				verdict = "both reject"
			case simErr != nil || execErr != nil:
				verdict, fails = "FAIL sim rejects, chain pays", fails+1
			case setErr != nil:
				verdict, fails = "FAIL sim quotes, chain reverts", fails+1
			default:
				dd := new(big.Int).Sub(execOut, settle)
				diff = dd.String()
				bp = fmt.Sprintf("%.2f", 1e4*float64(new(big.Int).Sub(simOut, settle).Int64())/float64(settle.Int64()))
				if !settle.IsInt64() || !simOut.IsInt64() {
					f := new(big.Float).Quo(new(big.Float).SetInt(new(big.Int).Sub(simOut, settle)), new(big.Float).SetInt(settle))
					v, _ := f.Float64()
					bp = fmt.Sprintf("%.2f", 1e4*v)
				}
				if dd.Sign() == 0 {
					verdict = "exact"
				} else if p[1] == "SPYx" && dd.Sign() < 0 && dd.CmpAbs(big.NewInt(2)) <= 0 {
					// Guide: xStock settlement floors stock output plus dust to shares,
					// so receipt may exceed the guaranteed amount; compare received >= out.
					verdict = "≥ (xStock share rounding)"
				} else {
					verdict, fails = "FAIL mismatch", fails+1
				}
			}
			t.Logf("%s→%s $%g in=%s sim=%s exec=%s settle=%s %s", p[0], p[1], d, in, str(simOut, simErr), str(execOut, execErr), str(settle, setErr), verdict)
			fmt.Fprintf(&md, "| %s→%s | %g | %s | %s | %s | %s | %s | %s | %s |\n", p[0], p[1], d, in, str(simOut, simErr), str(execOut, execErr), str(settle, setErr), diff, bp, verdict)
		}
	}
	t.Log("\n" + md.String())
	if out := os.Getenv("THOG_PROP_PARITY_OUT"); out != "" {
		require.NoError(t, os.WriteFile(out, []byte(md.String()), 0o644))
	}
	require.Zero(t, fails)
}

// Detections from token-balance-slot (bin/tbs on the live Monad RPC, 2026-11-03).
var balanceSlots = map[string]struct {
	slot  int64
	ratio *big.Float
}{
	"0x754704bc059f8c67012fed69bc8a327a5aafb603": {slot: 9},                                                // USDC, bits 255
	"0xe7cd86e13ac4309349f30b3435a9d337750fc82d": {slot: 0x33},                                             // USDT0
	"0x3bd359c1119da7da1d913d1c4d2b7c461115433a": {slot: 3},                                                // WMON
	"0xee8c0e9f1bffb4eb878d8f15f368a02a35481242": {slot: 0x33},                                             // WETH
	"0x90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48": {slot: 0x107, ratio: big.NewFloat(1.007474126658765999)}, // SPYx shares, size 24
}

func str(v *big.Int, err error) string {
	if err != nil {
		s := err.Error()
		if len(s) > 60 {
			s = s[:60]
		}
		return "err: " + strings.ReplaceAll(s, "|", "/")
	}
	return v.String()
}

// settleSwap funds sender, approves the proxy and calls swapWithExtraFees in the
// next block, then reverts the fork. It returns the sender's tokenOut delta.
func settleSwap(ctx context.Context, c *rpc.Client, in, out string, amt *big.Int, proxy, sender, feeTo string) (*big.Int, error) {
	var snap hexutil.Big
	if err := c.CallContext(ctx, &snap, "evm_snapshot"); err != nil {
		return nil, err
	}
	defer func() { var ok bool; _ = c.CallContext(ctx, &ok, "evm_revert", &snap) }()
	for _, call := range [][]any{
		{"anvil_setBalance", sender, "0x56bc75e2d63100000"},
		{"anvil_impersonateAccount", sender},
	} {
		if err := c.CallContext(ctx, nil, call[0].(string), call[1:]...); err != nil {
			return nil, fmt.Errorf("%s: %w", call[0], err)
		}
	}
	balanceOf := func(token string) (*big.Int, error) {
		var r hexutil.Bytes
		err := c.CallContext(ctx, &r, "eth_call", map[string]any{"to": token, "data": hexutil.Bytes(append(common.FromHex("0x70a08231"), common.LeftPadBytes(common.FromHex(sender), 32)...))}, "latest")
		return new(big.Int).SetBytes(r), err
	}
	balance := func() (*big.Int, error) { return balanceOf(out) }
	// Fund through the token-balance-slot detection (solidity layout; bits/size/ratio).
	det, ok := balanceSlots[strings.ToLower(in)]
	if !ok {
		return nil, fmt.Errorf("no balance-slot detection for %s", in)
	}
	units := new(big.Int).Set(amt)
	if det.ratio != nil {
		f := new(big.Float).Quo(new(big.Float).SetInt(units), det.ratio)
		units, _ = f.Int(nil)
		units.Add(units, big.NewInt(1))
	}
	key := crypto.Keccak256Hash(common.LeftPadBytes(common.FromHex(sender), 32), common.LeftPadBytes(big.NewInt(det.slot).Bytes(), 32))
	if err := c.CallContext(ctx, nil, "anvil_setStorageAt", in, key, common.BigToHash(units)); err != nil {
		return nil, fmt.Errorf("setStorageAt: %w", err)
	}
	if got, err := balanceOf(in); err != nil || got.Cmp(amt) < 0 {
		return nil, fmt.Errorf("funding %s: balance %v < %s (%v)", in, got, amt, err)
	}
	before, err := balance()
	if err != nil {
		return nil, err
	}
	var block struct {
		BaseFee hexutil.Big    `json:"baseFeePerGas"`
		Time    hexutil.Uint64 `json:"timestamp"`
	}
	if err = c.CallContext(ctx, &block, "eth_getBlockByNumber", "pending", false); err != nil {
		return nil, err
	}
	word := func(b []byte) []byte { return common.LeftPadBytes(b, 32) }
	approve := append(common.FromHex("0x095ea7b3"), append(word(common.FromHex(proxy)), word(amt.Bytes())...)...)
	swap := common.FromHex("0xee744696")
	for _, w := range [][]byte{common.FromHex(in), common.FromHex(out), amt.Bytes(), nil, big.NewInt(time.Now().Unix() + 3600).Bytes(), nil, common.FromHex(feeTo)} {
		swap = append(swap, word(w)...)
	}
	fee := (*big.Int)(&block.BaseFee)
	tx := func(to string, data []byte) (common.Hash, error) {
		var h common.Hash
		err := c.CallContext(ctx, &h, "eth_sendTransaction", map[string]any{"from": sender, "to": to, "data": hexutil.Bytes(data),
			"gas": "0x1c9c380", "maxFeePerGas": (*hexutil.Big)(fee), "maxPriorityFeePerGas": "0x0"})
		return h, err
	}
	if _, err = tx(in, approve); err != nil {
		return nil, fmt.Errorf("approve: %w", err)
	}
	h, err := tx(proxy, swap)
	if err != nil {
		return nil, fmt.Errorf("swap send: %w", err)
	}
	if err = c.CallContext(ctx, nil, "evm_mine"); err != nil {
		return nil, err
	}
	var receipt struct {
		Status hexutil.Uint64 `json:"status"`
	}
	if err = c.CallContext(ctx, &receipt, "eth_getTransactionReceipt", h); err != nil {
		return nil, err
	}
	if receipt.Status != 1 {
		var reason hexutil.Bytes
		callErr := c.CallContext(ctx, &reason, "eth_call", map[string]any{"from": sender, "to": proxy, "data": hexutil.Bytes(swap)}, "latest")
		return nil, fmt.Errorf("swap reverted: %v", callErr)
	}
	after, err := balance()
	if err != nil {
		return nil, err
	}
	return after.Sub(after, before), nil
}
