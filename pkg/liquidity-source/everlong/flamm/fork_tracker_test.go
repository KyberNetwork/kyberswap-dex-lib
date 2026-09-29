package everlongflamm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The tracker on an anvil fork (same gating as fork_parity_test.go, default block 51330060) behind a JSON-RPC proxy:
//
//   - tamper: every word of every view the refresh aggregate returns, every success bit, and every storage word is
//     altered one at a time; a refresh whose reads change must never come back attested;
//   - pinning: a swap is mined between the refresh's first aggregate at "latest" and its later rounds; every later
//     read names the first round's block and the reads equal a refresh pinned to that block;
//   - drift: real storage changes on the fork (beacon implementation, spread hook slot, venue count) and code / state
//     overrides through GetNewPoolStateWithOverrides.

type proxyRPCReq struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
}

type proxyRPCResp struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// rpcProxy forwards JSON-RPC to anvil and lets a hook rewrite each answer.
type rpcProxy struct {
	upstream string
	srv      *httptest.Server
	mu       sync.Mutex
	hook     func(req *proxyRPCReq, resp *proxyRPCResp)
}

func newRPCProxy(t *testing.T, upstream string) *rpcProxy {
	p := &rpcProxy{upstream: upstream}
	p.srv = httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(p.srv.Close)
	return p
}

func (p *rpcProxy) client() *ethrpc.Client {
	return ethrpc.New(p.srv.URL).SetMulticallContract(multicall3)
}

func (p *rpcProxy) setHook(h func(req *proxyRPCReq, resp *proxyRPCResp)) {
	p.mu.Lock()
	p.hook = h
	p.mu.Unlock()
}

func (p *rpcProxy) serve(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	batch := bytes.HasPrefix(bytes.TrimSpace(raw), []byte("["))
	var reqs []proxyRPCReq
	if batch {
		err = json.Unmarshal(raw, &reqs)
	} else {
		var one proxyRPCReq
		err = json.Unmarshal(raw, &one)
		reqs = []proxyRPCReq{one}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := http.Post(p.upstream, "application/json", bytes.NewReader(raw))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	var resps []proxyRPCResp
	if batch {
		err = json.Unmarshal(body, &resps)
	} else {
		var one proxyRPCResp
		err = json.Unmarshal(body, &one)
		resps = []proxyRPCResp{one}
	}
	if err != nil {
		http.Error(w, string(body), http.StatusBadGateway)
		return
	}
	p.mu.Lock()
	hook := p.hook
	p.mu.Unlock()
	if hook != nil {
		byID := map[string]*proxyRPCReq{}
		for i := range reqs {
			byID[string(reqs[i].ID)] = &reqs[i]
		}
		for i := range resps {
			if q := byID[string(resps[i].ID)]; q != nil {
				hook(q, &resps[i])
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if batch {
		_ = json.NewEncoder(w).Encode(resps)
		return
	}
	_ = json.NewEncoder(w).Encode(resps[0])
}

type multicallIn struct {
	Target   common.Address
	CallData []byte
}

type multicallOut struct {
	Success    bool
	ReturnData []byte
}

// decodeMulticall decodes an eth_call request to Multicall3.tryBlockAndAggregate: its calls and block tag.
func decodeMulticall(req *proxyRPCReq) ([]multicallIn, string, bool) {
	if req.Method != "eth_call" || len(req.Params) < 2 {
		return nil, "", false
	}
	var msg map[string]any
	if json.Unmarshal(req.Params[0], &msg) != nil {
		return nil, "", false
	}
	if to, _ := msg["to"].(string); !strings.EqualFold(to, multicall3.Hex()) {
		return nil, "", false
	}
	data, _ := msg["input"].(string)
	if data == "" {
		data, _ = msg["data"].(string)
	}
	raw := common.FromHex(data)
	m := multicallABI.Methods["tryBlockAndAggregate"]
	if len(raw) < 4 || !bytes.Equal(raw[:4], m.ID) {
		return nil, "", false
	}
	vals, err := m.Inputs.Unpack(raw[4:])
	if err != nil || len(vals) != 2 {
		return nil, "", false
	}
	calls, err := tupleOf[[]multicallIn](vals[1])
	if err != nil {
		return nil, "", false
	}
	var tag string
	_ = json.Unmarshal(req.Params[1], &tag)
	return calls, tag, true
}

func selectorNames() map[[4]byte]string {
	out := map[[4]byte]string{}
	for name, a := range map[string]*gethabi.ABI{"pool": &flammABI, "hook": &hookABI, "levHook": &levHookABI,
		"spreadHook": &spreadHookABI, "feed": &priceFeedABI, "router": &routerABI, "account": &accountABI,
		"factory": &factoryABI, "morpho": &morphoABI, "irm": &irmABI, "aggregator": &aggregatorABI,
		"oracle": &oracleABI, "multicall": &multicallABI} {
		for _, m := range a.Methods {
			var sel [4]byte
			copy(sel[:], m.ID)
			if _, dup := out[sel]; !dup {
				out[sel] = name + "." + m.Name
			}
		}
	}
	return out
}

func readsJSON(t *testing.T, tracked entity.Pool) (Extra, string) {
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	raw, err := json.Marshal(extra.Reads)
	require.NoError(t, err)
	return extra, string(raw)
}

// publishedJSON is everything a refresh publishes that a quote can read, not only Extra.Reads: the venue set, the
// market oracle's end-of-window answers, the dependency set and the scheduled-change stamp, whose two source words
// (the factory's pending implementation, the pool's pending hook set) refuse every quote without moving a read.
// The verdict fields are classified before this is compared, and the policy is the harness's own.
func publishedJSON(t *testing.T, tracked entity.Pool) (Extra, string) {
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(tracked.Extra), &extra))
	c := extra
	c.Attested, c.AttestFailure, c.ProfileDrift, c.Policy = false, "", "", Policy{}
	raw, err := json.Marshal(&c)
	require.NoError(t, err)
	return extra, string(raw)
}

// poolRefusals are the simulator's own whole-pool refusals: a tamper that produces one is fail-closed, and the
// quote grid must not be run against the adapter for it (every amount refuses, which judgeFill would score as a
// misquote).
var poolRefusals = []error{ErrNotAttested, ErrProfileDrift, ErrPoolRefused, ErrInvalidProfile, ErrScheduledChange,
	ErrSnapshotStale}

func isPoolRefusal(err error) bool {
	for _, e := range poolRefusals {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

// jsonDiff lists the leaf paths where two JSON documents differ.
func jsonDiff(a, b string) []string {
	var x, y any
	_ = json.Unmarshal([]byte(a), &x)
	_ = json.Unmarshal([]byte(b), &y)
	var out []string
	var walk func(path string, p, q any)
	walk = func(path string, p, q any) {
		switch pv := p.(type) {
		case map[string]any:
			qv, ok := q.(map[string]any)
			if !ok {
				out = append(out, path)
				return
			}
			keys := make([]string, 0, len(pv)+len(qv))
			for k := range pv {
				keys = append(keys, k)
			}
			for k := range qv {
				if _, both := pv[k]; !both { // a field only one side has (an omitempty that became non-zero)
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(path+"."+k, pv[k], qv[k])
			}
		case []any:
			qv, ok := q.([]any)
			if !ok || len(qv) != len(pv) {
				out = append(out, path)
				return
			}
			for i := range pv {
				walk(fmt.Sprintf("%s[%d]", path, i), pv[i], qv[i])
			}
		default:
			if !reflect.DeepEqual(p, q) {
				out = append(out, fmt.Sprintf("%s: %v -> %v", path, p, q))
			}
		}
	}
	walk("", x, y)
	return out
}

func TestForkTrackerTamper(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	proxy := newRPCProxy(t, e.f.url)
	cfg := parityConfig(Policy{LeverRouting: true})
	e.armLeverage(t, 13_000, 3600) // leverage probes and the lever venue take part
	block := new(big.Int).SetUint64(e.f.head().Number.Uint64())
	tracker := NewPoolTracker(cfg, proxy.client())
	ctx := context.Background()

	// Baseline, capturing the view aggregate's calls.
	var viewCalls []multicallIn
	var storageSlots []string
	proxy.setHook(func(req *proxyRPCReq, _ *proxyRPCResp) {
		if calls, _, ok := decodeMulticall(req); ok && len(calls) > 0 && viewCalls == nil &&
			bytes.Equal(calls[0].CallData[:4], flammABI.Methods["asset"].ID) {
			viewCalls = calls
		}
		if req.Method == "eth_getStorageAt" {
			storageSlots = append(storageSlots, string(req.Params[0])+string(req.Params[1]))
		}
	})
	base, err := tracker.GetNewPoolStateAtBlock(ctx, e.listed, block)
	require.NoError(t, err)
	baseExtra, basePublished := publishedJSON(t, base)
	require.True(t, baseExtra.Attested, "baseline: %q %q", baseExtra.AttestFailure, baseExtra.ProfileDrift)
	require.NotEmpty(t, viewCalls)
	names := selectorNames()
	t.Logf("baseline at %d: %d view calls, %d storage words, %d probes", block, len(viewCalls), len(storageSlots),
		baseExtra.Probes)

	type outcome struct {
		what, kind, result string
	}
	var results []outcome
	var misquotes, unbound, refused []string
	baseSim, err := NewPoolSimulator(base)
	require.NoError(t, err)
	baseTs := baseSim.state.Timestamp
	baseSim.nowFn = func() uint64 { return baseTs }
	impactAmounts := map[string][]uint64{}
	for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
		for _, sell := range []bool{true, false} {
			amts := edgeAmounts(baseSim, venue, sell, 3)
			if len(amts) > 90 { // keep the windows, thin the dust sweep
				thin := amts[:0:0]
				for i, a := range amts {
					if i%3 == 0 || i < 12 || i > len(amts)-12 {
						thin = append(thin, a)
					}
				}
				amts = thin
			}
			impactAmounts[fmt.Sprint(venue, sell)] = amts
		}
	}
	// Every tamper hook below returns silently on a decode or pack failure, so one that stopped matching the answer
	// it edits would be filed as "changed nothing the refresh publishes" instead of failing. applied counts the
	// answers a hook really edited, and run requires it of every tamper it schedules.
	var applied int
	run := func(what string, hook func(req *proxyRPCReq, resp *proxyRPCResp)) {
		applied = 0
		proxy.setHook(hook)
		tracked, err := tracker.GetNewPoolStateAtBlock(ctx, e.listed, block)
		proxy.setHook(nil)
		require.Positive(t, applied, "%s: the tamper edited no answer", what)
		var kind, res string
		switch {
		case err != nil:
			kind, res = "error", "error"
		default:
			extra, published := publishedJSON(t, tracked)
			switch {
			case extra.ProfileDrift != "":
				kind, res = "drift", "drift "+extra.ProfileDrift
			case !extra.Attested:
				kind, res = "attest", "attest "+strings.SplitN(extra.AttestFailure, ":", 2)[0]
			case published == basePublished:
				kind, res = "unchanged", "attested, published entity unchanged"
			default:
				diff := strings.Join(jsonDiff(basePublished, published), "; ")
				// Impact: does the attested, changed entity quote differently from the adapter? A whole-pool
				// refusal is fail-closed and is recorded as such: running the grid against the adapter would
				// score every refused amount as a misquote.
				sim, serr := NewPoolSimulator(tracked)
				if serr != nil {
					kind, res = "refused", "attested, refused by the simulator: "+serr.Error()
					refused = append(refused, fmt.Sprintf("%s (%v; %s)", what, serr, diff))
					break
				}
				sim.nowFn = func() uint64 { return sim.state.Timestamp }
				if _, qerr := sim.calcAmountOut(amountIn(sim, true, probeSellGrid[1]), int(VenueSwap)); isPoolRefusal(qerr) {
					all := true
					for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
						for _, sell := range []bool{true, false} {
							for _, a := range impactAmounts[fmt.Sprint(venue, sell)] {
								if _, e2 := sim.calcAmountOut(amountIn(sim, sell, a), venue); !isPoolRefusal(e2) {
									all = false
								}
							}
						}
					}
					require.True(t, all, "%s: %v refuses some amounts and not others", what, qerr)
					kind, res = "refused", "attested, every quote refused: "+qerr.Error()+" ("+diff+")"
					refused = append(refused, fmt.Sprintf("%s (%v; %s)", what, qerr, diff))
					break
				}
				kind, res = "changed", "ATTESTED WITH CHANGED READS "+diff
				sub := &forkReport{t: t, counts: map[string]int{}, quiet: true}
				for _, venue := range []int{int(VenueSwap), int(VenueLever)} {
					for _, sell := range []bool{true, false} {
						e.compareFills(t, sub, what, sim, venue, sell, impactAmounts[fmt.Sprint(venue, sell)],
							sim.state.Timestamp)
					}
				}
				if len(sub.list) != 0 {
					m := sub.list[0]
					misquotes = append(misquotes, fmt.Sprintf("%s: %s -> MISQUOTES %d of the grid, e.g. %s: go %s, chain %s",
						what, res, len(sub.list), m.Input, m.Go, m.Chain))
				} else {
					unbound = append(unbound, fmt.Sprintf("%s (%s)", what, diff))
				}
			}
		}
		results = append(results, outcome{what, kind, res})
	}

	isView := func(req *proxyRPCReq) bool {
		calls, _, ok := decodeMulticall(req)
		return ok && len(calls) == len(viewCalls) && bytes.Equal(calls[0].CallData[:4], flammABI.Methods["asset"].ID)
	}
	rewrite := func(resp *proxyRPCResp, f func(outs []multicallOut)) {
		var hexResult string
		if json.Unmarshal(resp.Result, &hexResult) != nil {
			return
		}
		m := multicallABI.Methods["tryBlockAndAggregate"]
		vals, err := m.Outputs.Unpack(common.FromHex(hexResult))
		if err != nil {
			return
		}
		outs, err := tupleOf[[]multicallOut](vals[2])
		if err != nil {
			return
		}
		f(outs)
		packed, err := m.Outputs.Pack(vals[0], vals[1], outs)
		if err != nil {
			return
		}
		resp.Result, _ = json.Marshal(hexutil.Encode(packed))
		applied++
	}

	// Every word of every view answer.
	var baseOuts []multicallOut
	proxy.setHook(func(req *proxyRPCReq, resp *proxyRPCResp) {
		if isView(req) && baseOuts == nil {
			rewrite(resp, func(outs []multicallOut) { baseOuts = append([]multicallOut(nil), outs...) })
		}
	})
	_, err = tracker.GetNewPoolStateAtBlock(ctx, e.listed, block)
	require.NoError(t, err)
	require.Len(t, baseOuts, len(viewCalls))
	for k := range viewCalls {
		var sel [4]byte
		copy(sel[:], viewCalls[k].CallData[:4])
		name := fmt.Sprintf("%d:%s", k, names[sel])
		words := len(baseOuts[k].ReturnData) / 32
		for j := 0; j < words; j++ {
			k, j := k, j
			run(fmt.Sprintf("%s word %d", name, j), func(req *proxyRPCReq, resp *proxyRPCResp) {
				if !isView(req) {
					return
				}
				rewrite(resp, func(outs []multicallOut) {
					d := append([]byte(nil), outs[k].ReturnData...)
					var w uint256.Int
					w.SetBytes(d[j*32 : (j+1)*32])
					if w.IsZero() {
						w.SetOne()
					} else if w.Eq(uOne) {
						w.Clear()
					} else {
						w.AddUint64(&w, 1)
					}
					b := w.Bytes32()
					copy(d[j*32:], b[:])
					outs[k].ReturnData = d
				})
			})
		}
		// The removal direction: a +1 on a non-zero word is another value, but a word an answer *drops* is what
		// turns a sentinel off (a pending implementation, a pending hook), and the generic mutation above never
		// produces it on a non-zero word. Every word that is not already zero and fits an address is zeroed.
		hi := new(uint256.Int).Lsh(uOne, 160)
		for j := 0; j < words; j++ {
			var w uint256.Int
			w.SetBytes(baseOuts[k].ReturnData[j*32 : (j+1)*32])
			if w.IsZero() || !w.Lt(hi) {
				continue
			}
			k, j := k, j
			run(fmt.Sprintf("%s word %d -> 0", name, j), func(req *proxyRPCReq, resp *proxyRPCResp) {
				if !isView(req) {
					return
				}
				rewrite(resp, func(outs []multicallOut) {
					d := append([]byte(nil), outs[k].ReturnData...)
					copy(d[j*32:(j+1)*32], make([]byte, 32))
					outs[k].ReturnData = d
				})
			})
		}
		run(name+" success->revert", func(req *proxyRPCReq, resp *proxyRPCResp) {
			if isView(req) {
				rewrite(resp, func(outs []multicallOut) { outs[k] = multicallOut{Success: false, ReturnData: []byte{}} })
			}
		})
	}
	// Every storage word.
	for i := range storageSlots {
		slot := storageSlots[i]
		run("storage "+slot, func(req *proxyRPCReq, resp *proxyRPCResp) {
			if req.Method != "eth_getStorageAt" || string(req.Params[0])+string(req.Params[1]) != slot {
				return
			}
			var h common.Hash
			if json.Unmarshal(resp.Result, &h) != nil {
				return
			}
			var w uint256.Int
			w.SetBytes(h[:]).AddUint64(&w, 1)
			resp.Result, _ = json.Marshal(common.Hash(w.Bytes32()))
			applied++
		})
	}

	tally := map[string]int{}
	var unchanged []string
	for _, o := range results {
		tally[o.kind]++
		if o.kind == "unchanged" {
			unchanged = append(unchanged, o.what)
		}
	}
	sort.Strings(unchanged)
	sort.Strings(refused)
	t.Logf("tamper outcomes over %d refreshes: %v", len(results), tally)
	for _, r := range refused {
		t.Logf("FAIL-CLOSED (attested with a changed entity the simulator refuses every quote on): %s", r)
	}
	t.Logf("tampers that changed nothing the refresh publishes (words the state never reads): %s",
		strings.Join(unchanged, ", "))
	for _, u := range unbound {
		t.Logf("NOT BOUND BY ATTESTATION (quotes still identical to the adapter on the grid): %s", u)
	}
	for _, f := range misquotes {
		t.Errorf("MISQUOTE %s", f)
	}
}

// TestForkTrackerPinning: a swap lands between the first aggregate at "latest" and the later rounds.
func TestForkTrackerPinning(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	proxy := newRPCProxy(t, e.f.url)
	b0 := e.f.head().Number.Uint64()
	ts0 := e.f.head().Time
	f := e.f
	f.setBalance(e.cbBTC, forkAdapter, big.NewInt(20_000))
	calldata, err := forkABI.Pack("executeEverlongFlamm", adapterData(e.pool, VenueSwap), big.NewInt(20_000), e.cbBTC,
		e.usdc, forkRecipient)
	require.NoError(t, err)

	var mu sync.Mutex
	var mined bool
	var mineErr error
	var tags []string
	proxy.setHook(func(req *proxyRPCReq, _ *proxyRPCResp) {
		mu.Lock()
		defer mu.Unlock()
		var tag string
		switch req.Method {
		case "eth_call":
			_ = json.Unmarshal(req.Params[1], &tag)
		case "eth_getStorageAt":
			_ = json.Unmarshal(req.Params[2], &tag)
		default:
			tag = "(" + req.Method + ")"
		}
		if mined {
			tags = append(tags, req.Method+"@"+tag)
		}
		if calls, bt, ok := decodeMulticall(req); ok && !mined && bt == "latest" && len(calls) > 0 {
			// the chain moves on before the refresh's next round
			var h common.Hash
			if mineErr = f.rc.Call(nil, "evm_setNextBlockTimestamp", hexutil.Uint64(ts0+2)); mineErr == nil {
				mineErr = f.rc.Call(&h, "eth_sendTransaction", map[string]any{"from": forkDeployer, "to": forkAdapter,
					"data": hexutil.Encode(calldata), "gas": hexutil.Uint64(8_000_000)})
			}
			// wait until the block is committed (anvil serves the executing block's state under the old head)
			for i := 0; mineErr == nil && i < 600; i++ {
				var rcpt map[string]any
				if mineErr = f.rc.Call(&rcpt, "eth_getTransactionReceipt", h); rcpt != nil {
					var n hexutil.Uint64
					if mineErr = f.rc.Call(&n, "eth_blockNumber"); uint64(n) == b0+1 {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			mined = true
		}
	})
	tracked, err := NewPoolTracker(baseConfig(), proxy.client()).GetNewPoolState(context.Background(), e.listed,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.NoError(t, mineErr)
	proxy.setHook(nil)
	require.True(t, mined)
	require.Equal(t, b0+1, f.head().Number.Uint64(), "the swap was mined mid-refresh")
	require.Equal(t, b0, tracked.BlockNumber)
	want := hexutil.EncodeUint64(b0)
	for _, tg := range tags {
		require.True(t, strings.HasSuffix(tg, "@"+want), "a later read is not pinned to %s: %s", want, tg)
	}
	t.Logf("reads after the first round: %v", tags)
	_, reads := readsJSON(t, tracked)
	pinned, err := NewPoolTracker(baseConfig(), f.client).GetNewPoolStateAtBlock(context.Background(), e.listed,
		new(big.Int).SetUint64(b0))
	require.NoError(t, err)
	pinnedExtra, pinnedReads := readsJSON(t, pinned)
	require.True(t, pinnedExtra.Attested)
	require.Empty(t, jsonDiff(pinnedReads, reads), "the mid-refresh swap leaked into the refresh")
	latest, err := NewPoolTracker(baseConfig(), f.client).GetNewPoolState(context.Background(), e.listed,
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	_, latestReads := readsJSON(t, latest)
	require.NotEqual(t, reads, latestReads, "the mined swap did not move the state")
	extra, _ := readsJSON(t, tracked)
	require.True(t, extra.Attested, "%q %q", extra.AttestFailure, extra.ProfileDrift)
}

// TestForkTrackerDrift: wiring changes made on the fork, and overrides.
func TestForkTrackerDrift(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	f := e.f
	ctx := context.Background()
	prof := &c104
	tracker := NewPoolTracker(baseConfig(), f.client)
	track := func() Extra {
		p, err := tracker.GetNewPoolState(ctx, e.listed, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		extra, _ := readsJSON(t, p)
		return extra
	}
	require.True(t, track().Attested)
	storage := func(addr common.Address, slot common.Hash) common.Hash {
		var h common.Hash
		f.rpc(&h, "eth_getStorageAt", addr, slot, "latest")
		return h
	}
	var key [64]byte
	copy(key[12:32], prof.Pool[:])
	key[63] = 1
	recBase := new(big.Int).SetBytes(crypto.Keccak256(key[:]))
	venuesLen := common.BigToHash(recBase.Add(recBase, big.NewInt(3)))
	other := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	var implCode hexutil.Bytes
	f.rpc(&implCode, "eth_getCode", prof.Implementation, "latest")
	f.rpc(nil, "anvil_setCode", other, implCode)
	for _, c := range []struct {
		name  string
		addr  common.Address
		slot  common.Hash
		value common.Hash
		want  string
	}{
		{"beacon implementation (same code elsewhere)", prof.Factory, common.Hash{}, common.BytesToHash(other[:]),
			"implementation"},
		{"spread hook slot", prof.Pool, flammSlot(slotSpreadHook), common.Hash{}, "hook set"},
		{"venue count", prof.Router, venuesLen, common.BigToHash(big.NewInt(2)), "venue count"},
		{"leverage hook slot", prof.Pool, flammSlot(slotLevHook), common.BytesToHash(other[:]), "hook set"},
	} {
		old := storage(c.addr, c.slot)
		f.rpc(nil, "anvil_setStorageAt", c.addr, c.slot, c.value)
		// A wiring change whose views no longer answer (the Router's venue set) is drift too, published at the
		// block the chain answered at rather than returned as an error that would leave the last entity quoting.
		p, err := tracker.GetNewPoolState(ctx, e.listed, pool.GetNewPoolStateParams{})
		require.NoError(t, err, c.name)
		extra, _ := readsJSON(t, p)
		require.False(t, extra.Attested, c.name)
		if c.want == "venue count" && strings.HasPrefix(extra.ProfileDrift, "unreadable: ") {
			require.Nil(t, extra.Reads, c.name)
		} else {
			require.Equal(t, c.want, extra.ProfileDrift, c.name)
		}
		require.Equal(t, f.head().Number.Uint64(), p.BlockNumber, c.name)
		_, err = NewPoolSimulator(p)
		require.ErrorIs(t, err, ErrProfileDrift, c.name)
		t.Logf("%s: drift %q", c.name, extra.ProfileDrift)
		// The lister skips a pool that does not answer as the registry has it; the run itself does not fail.
		listed, _, err := NewPoolsListUpdater(baseConfig(), f.client).GetNewPools(ctx, nil)
		require.NoError(t, err, c.name)
		require.Empty(t, listed, "%s: a drifted pool was listed", c.name)
		f.rpc(nil, "anvil_setStorageAt", c.addr, c.slot, old)
		require.True(t, track().Attested, "restored after %s", c.name)
	}

	// A code override on a registered contract (its own code, so every view still answers) is drift.
	var code hexutil.Bytes
	f.rpc(&code, "eth_getCode", prof.Router, "latest")
	p, err := tracker.GetNewPoolStateWithOverrides(ctx, e.listed,
		pool.GetNewPoolStateWithOverridesParams{Overrides: map[common.Address]gethclient.OverrideAccount{prof.Router: {Code: code}}})
	require.NoError(t, err)
	extra, _ := readsJSON(t, p)
	require.Contains(t, extra.ProfileDrift, "code override")

	// A state override is honoured by the views, the storage words and the probes alike: the refresh attests the
	// overridden state, and its quotes are the pool's previews under the same override.
	var xw uint256.Int
	xOld := storage(prof.Hook, common.BigToHash(big.NewInt(17)))
	xw.SetBytes(xOld[:])
	xw.Add(&xw, new(uint256.Int).Div(&xw, uint256.NewInt(50)))
	phys := storage(prof.Pool, flammSlot(slotPhysical))
	var pw uint256.Int
	pw.SetBytes(phys[:]).AddUint64(&pw, 5_000)
	ov := map[common.Address]gethclient.OverrideAccount{
		prof.Hook: {StateDiff: map[common.Hash]common.Hash{common.BigToHash(big.NewInt(17)): xw.Bytes32()}},
		prof.Pool: {StateDiff: map[common.Hash]common.Hash{flammSlot(slotPhysical): pw.Bytes32()}},
	}
	p, err = tracker.GetNewPoolStateWithOverrides(ctx, e.listed,
		pool.GetNewPoolStateWithOverridesParams{Overrides: ov})
	require.NoError(t, err)
	extra, _ = readsJSON(t, p)
	require.True(t, extra.Attested, "%q %q", extra.AttestFailure, extra.ProfileDrift)
	require.Equal(t, xw.Dec(), extra.Reads.Hook.XWad.Dec())
	require.Equal(t, pw.Dec(), extra.Reads.Pool.Physical.Dec())
	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	ts := sim.state.Timestamp
	sim.nowFn = func() uint64 { return ts }
	gc := gethclient.New(f.client.GetETHClient().Client())
	matched := 0
	for _, sell := range []bool{true, false} {
		grid := liveGrid(1, 400_000, 24)
		if !sell {
			grid = liveGrid(1_000, 400_000_000, 24)
		}
		for _, a := range grid {
			fl, qerr := sim.quoteSwap(sell, uint256.NewInt(a), ts)
			data, _ := flammABI.Pack("previewSwap", sell, new(big.Int).SetUint64(a))
			ret, cerr := gc.CallContract(ctx, ethereum.CallMsg{To: &prof.Pool, Data: data},
				new(big.Int).SetUint64(p.BlockNumber), &ov)
			if cerr != nil {
				rd, ok := revertData(cerr)
				require.True(t, ok, "%v", cerr)
				want := revertError(rd)
				require.NotNil(t, want)
				if qerr == nil || (!strings.Contains(qerr.Error(), want.Error()) && !isSettlementOnly(qerr)) {
					require.ErrorIs(t, qerr, want, "override sell=%v a=%d", sell, a)
				}
				continue
			}
			vals, err := flammABI.Methods["previewSwap"].Outputs.Unpack(ret)
			require.NoError(t, err)
			if qerr != nil {
				require.True(t, isSettlementOnly(qerr), "override sell=%v a=%d: chain %v, sim %v", sell, a, vals, qerr)
				continue
			}
			used, _ := wordOf(vals[0])
			out, _ := wordOf(vals[1])
			require.Equal(t, used.Dec(), fl.used.Dec(), "override sell=%v a=%d used", sell, a)
			require.Equal(t, out.Dec(), fl.out.Dec(), "override sell=%v a=%d out", sell, a)
			matched++
		}
	}
	t.Logf("override refresh: %d quotes identical to previews under the override", matched)
	require.Positive(t, matched)
}

var trackerMorphoABI = func() gethabi.ABI {
	a, err := gethabi.JSON(strings.NewReader(`[
 {"type":"function","name":"supplyCollateral","stateMutability":"nonpayable","inputs":[{"name":"marketParams","type":"tuple","components":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]},{"name":"assets","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[]},
 {"type":"function","name":"supply","stateMutability":"nonpayable","inputs":[{"name":"marketParams","type":"tuple","components":[{"name":"loanToken","type":"address"},{"name":"collateralToken","type":"address"},{"name":"oracle","type":"address"},{"name":"irm","type":"address"},{"name":"lltv","type":"uint256"}]},{"name":"assets","type":"uint256"},{"name":"shares","type":"uint256"},{"name":"onBehalf","type":"address"},{"name":"data","type":"bytes"}],"outputs":[{"name":"","type":"uint256"},{"name":"","type":"uint256"}]},
 {"type":"function","name":"approve","stateMutability":"nonpayable","inputs":[{"name":"s","type":"address"},{"name":"a","type":"uint256"}],"outputs":[{"name":"","type":"bool"}]}
]`))
	if err != nil {
		panic(err)
	}
	return a
}()

// TestForkDonation: anyone can supply Morpho collateral (or loan-asset supply) on behalf of the pool's venue
// account. The refresh still attests; the production configuration refuses the donated pool (envelope), and with
// QuoteDonatedVenues, which prices the position exactly (recognized = min(actual, managed)), every quote on the
// donated state is the adapter's fill.
func TestForkDonation(t *testing.T) {
	e := newForkEnvAt(t, forkEdgesBlock)
	rep := newForkReport(t)
	f := e.f
	ctx := context.Background()
	se := StaticExtra{}
	require.NoError(t, json.Unmarshal([]byte(e.listed.StaticExtra), &se))
	v := entityVenues(t, e.listed)[0]
	type mp struct {
		LoanToken, CollateralToken, Oracle, Irm common.Address
		Lltv                                    *big.Int
	}
	params := mp{se.LoanAsset, se.PoolAsset, v.Oracle, v.Irm, v.Lltv.ToBig()}
	griefer := common.HexToAddress("0x00000000000000000000000000000000000beef1")
	f.impersonate(griefer)
	morpho := c104.Morpho
	for _, c := range []struct {
		name  string
		token common.Address
		call  func() []byte
	}{
		{"collateral", se.PoolAsset, func() []byte {
			d, err := trackerMorphoABI.Pack("supplyCollateral", params, big.NewInt(1), v.Account, []byte{})
			require.NoError(t, err)
			return d
		}},
		{"supply", se.LoanAsset, func() []byte {
			d, err := trackerMorphoABI.Pack("supply", params, big.NewInt(1), big.NewInt(0), v.Account, []byte{})
			require.NoError(t, err)
			return d
		}},
	} {
		f.setBalance(c.token, griefer, big.NewInt(1))
		approve, err := trackerMorphoABI.Pack("approve", morpho, big.NewInt(1))
		require.NoError(t, err)
		f.send(griefer, c.token, approve, f.head().Time+2)
		rcpt := f.send(griefer, morpho, c.call(), f.head().Time+2)
		require.Equal(t, uint64(1), rcpt.Status, "%s donation", c.name)

		tracked, err := NewPoolTracker(baseConfig(), f.client).GetNewPoolState(ctx, e.listed, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		extra, _ := readsJSON(t, tracked)
		t.Logf("%s donation of 1 unit: attested %v (%q %q)", c.name, extra.Attested, extra.AttestFailure, extra.ProfileDrift)
		require.True(t, extra.Attested, c.name)
		_, simErr := NewPoolSimulator(tracked)
		require.ErrorIs(t, simErr, ErrPoolRefused, "%s donation: refused by default", c.name)
		tracked, err = NewPoolTracker(parityConfig(Policy{QuoteDonatedVenues: true}), f.client).GetNewPoolState(ctx,
			e.listed, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		sim, simErr := NewPoolSimulator(tracked)
		require.NoError(t, simErr, "%s donation: quotable with QuoteDonatedVenues", c.name)
		st := sim.state
		dv := &st.Router.Venues[0]
		require.True(t, dv.Morpho.Position.Collateral.Gt(&dv.ManagedCollateral) ||
			dv.Morpho.Position.SupplyShares.Gt(&dv.ManagedSupplyShares), "%s: the donation is on the state", c.name)
		ts := st.Timestamp
		for _, sell := range []bool{true, false} {
			amounts := liveGrid(1, 400_000, 30)
			if !sell {
				amounts = liveGrid(100, 400_000_000, 30)
			}
			calls := make([]adapterCall, len(amounts))
			fills := make([]*fill, len(amounts))
			errs := make([]error, len(amounts))
			for i, a := range amounts {
				fills[i], errs[i] = sim.quoteSwap(sell, uint256.NewInt(a), ts)
				calls[i] = adapterCall{venue: VenueSwap, sell: sell, amount: new(big.Int).SetUint64(a)}
			}
			got := e.batchFills(t, calls, ts)
			for i := range amounts {
				var res *pool.CalcAmountOutResult
				if errs[i] == nil {
					var rem big.Int
					rem.Sub(calls[i].amount, fills[i].used.ToBig())
					res = &pool.CalcAmountOutResult{TokenAmountOut: &pool.TokenAmount{Amount: fills[i].out.ToBig()},
						RemainingTokenAmountIn: &pool.TokenAmount{Amount: &rem}, SwapInfo: SwapInfo{Venue: VenueSwap}}
				}
				judgeFill(rep, "donated-"+c.name, calls[i], res, errs[i], got[i], chainSpreadLive(sim, ts))
			}
		}
	}
	require.Empty(t, rep.list)
}
