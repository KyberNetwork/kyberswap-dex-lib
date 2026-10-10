package lotflow

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// An RPC tape is the chain state one Track reads at one block, recorded live (read-only) through a
// proxy: every eth_call, multicall sub-calls included, as (to, calldata) -> (success, return data).
// The replay server answers both read paths from it: plain eth_call (a failed entry answers
// "execution reverted") and Multicall3 tryAggregate (decoded and answered per sub-call). A read the
// tape lacks fails the test, so a replayed Track can only see recorded chain state.

type tapeEntry struct {
	OK  bool          `json:"ok"`
	Ret hexutil.Bytes `json:"ret"`
}

type rpcTape struct {
	Block     string               `json:"block"`     // hex block number
	BlockJSON json.RawMessage      `json:"blockJson"` // eth_getBlockByNumber(block, false)
	Calls     map[string]tapeEntry `json:"calls"`     // mcKey(to, data)
}

var tryAggregateSel = sel("tryAggregate(bool,(address,bytes)[])")

var tryAggregateABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(`[{"name":"tryAggregate","type":"function","inputs":[
{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		panic(err)
	}
	return a
}()

type subCall struct {
	Target   common.Address
	CallData []byte
}

type subResult struct {
	Success    bool
	ReturnData []byte
}

func decodeTryAggregate(data []byte) ([]subCall, error) {
	vals, err := tryAggregateABI.Methods["tryAggregate"].Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	var calls []subCall
	raw, _ := json.Marshal(vals[1])
	if err := json.Unmarshal(raw, &calls); err != nil {
		return nil, err
	}
	return calls, nil
}

func loadTape(t testing.TB, path string) *rpcTape {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	var tape rpcTape
	require.NoError(t, json.NewDecoder(zr).Decode(&tape))
	return &tape
}

func (tape *rpcTape) save(t testing.TB, path string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	enc := json.NewEncoder(zw)
	require.NoError(t, enc.Encode(tape)) // map keys are sorted: deterministic
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
}

type rpcMsg struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      json.RawMessage   `json:"id"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	Result  json.RawMessage   `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type callArgs struct {
	To    string        `json:"to"`
	Data  hexutil.Bytes `json:"data"`
	Input hexutil.Bytes `json:"input"`
}

func (c callArgs) calldata() []byte {
	if len(c.Input) > 0 {
		return c.Input
	}
	return c.Data
}

// rpcCounter counts what one client sent.
type rpcCounter struct {
	mu       sync.Mutex
	Requests int            // HTTP requests
	Methods  map[string]int // JSON-RPC calls by method
	SubCalls int            // multicall sub-calls
	Tags     map[string]int // eth_call block tags
}

func (c *rpcCounter) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Requests, c.Methods, c.SubCalls, c.Tags = 0, map[string]int{}, 0, map[string]int{}
}

func (c *rpcCounter) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.Methods))
	for k := range c.Methods {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, c.Methods[k]))
	}
	return fmt.Sprintf("%d HTTP requests, %s, %d multicall sub-calls", c.Requests, strings.Join(parts, " "), c.SubCalls)
}

func readMsgs(body []byte) ([]rpcMsg, bool, error) {
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		var many []rpcMsg
		err := json.Unmarshal(body, &many)
		return many, true, err
	}
	var one rpcMsg
	err := json.Unmarshal(body, &one)
	return []rpcMsg{one}, false, err
}

func (c *rpcCounter) count(msgs []rpcMsg) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Methods == nil {
		c.Methods, c.Tags = map[string]int{}, map[string]int{}
	}
	c.Requests++
	for _, m := range msgs {
		c.Methods[m.Method]++
		if m.Method != "eth_call" || len(m.Params) == 0 {
			continue
		}
		var tag string
		if len(m.Params) > 1 && json.Unmarshal(m.Params[1], &tag) == nil {
			c.Tags[tag]++
		}
		var a callArgs
		if json.Unmarshal(m.Params[0], &a) == nil && bytes.HasPrefix(a.calldata(), tryAggregateSel) {
			if calls, err := decodeTryAggregate(a.calldata()); err == nil {
				c.SubCalls += len(calls)
			}
		}
	}
}

func isRevertMsg(code int, msg string) bool {
	return code == 3 || strings.Contains(strings.ToLower(msg), "revert")
}

// recordingProxy forwards to upstream and records every successful or reverted eth_call into tape.
func recordingProxy(t *testing.T, upstream string, tape *rpcTape, counter *rpcCounter) *httptest.Server {
	var mu sync.Mutex
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqs, _, err := readMsgs(body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		counter.count(reqs)
		resp, err := http.Post(upstream, "application/json", bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		out, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(out)
		res, _, err := readMsgs(out)
		if err != nil {
			return
		}
		byID := map[string]rpcMsg{}
		for _, m := range res {
			byID[string(m.ID)] = m
		}
		mu.Lock()
		defer mu.Unlock()
		for _, q := range reqs {
			a, ok := byID[string(q.ID)]
			if !ok {
				continue
			}
			switch q.Method {
			case "eth_getBlockByNumber":
				if a.Error == nil {
					tape.BlockJSON = a.Result
				}
			case "eth_call":
				var args callArgs
				if json.Unmarshal(q.Params[0], &args) != nil {
					continue
				}
				data := args.calldata()
				if a.Error != nil {
					if isRevertMsg(a.Error.Code, a.Error.Message) && !bytes.HasPrefix(data, tryAggregateSel) {
						tape.Calls[mcKey(args.To, data)] = tapeEntry{OK: false}
					}
					continue
				}
				var ret hexutil.Bytes
				if json.Unmarshal(a.Result, &ret) != nil {
					continue
				}
				if !bytes.HasPrefix(data, tryAggregateSel) {
					tape.Calls[mcKey(args.To, data)] = tapeEntry{OK: true, Ret: ret}
					continue
				}
				calls, err := decodeTryAggregate(data)
				if err != nil {
					t.Errorf("record: tryAggregate request: %v", err)
					continue
				}
				var results []subResult
				vals, err := tryAggregateABI.Methods["tryAggregate"].Outputs.Unpack(ret)
				if err == nil {
					err = tryAggregateABI.Methods["tryAggregate"].Outputs.Copy(&results, vals)
				}
				if err != nil || len(results) != len(calls) {
					t.Errorf("record: tryAggregate response: %v", err)
					continue
				}
				for i, c := range calls {
					tape.Calls[mcKey(c.Target.Hex(), c.CallData)] = tapeEntry{OK: results[i].Success,
						Ret: results[i].ReturnData}
				}
			}
		}
	}))
}

// replayFaults injects failures into a replay: revert answers a recorded read as reverted (a failed
// sub-call, or "execution reverted"); nodeError answers a plain eth_call with a non-revert node error;
// httpFail fails the n-th HTTP request (1-based) with a 503.
type replayFaults struct {
	revert     func(to string, data []byte) bool
	nodeError  func(to string, data []byte) bool
	httpFail   func(n int) bool
	rateLimit  func(n int) bool // answers the n-th HTTP request (1-based) with 429 "Rate Limit"
	delay      time.Duration    // added to every request
	headAhead  uint64           // the "latest" block's timestamp is this many seconds after the tape's
	headBehind uint64           // ... or this many seconds before it (a "latest" node lagging the pinned one)
	// batchOOG fails matching sub-calls inside tryAggregate only (as a sub-call starved of gas in a
	// large batch would); the same read as a plain eth_call answers from the tape
	batchOOG func(to string, data []byte) bool
}

// replayServer answers from tape and counts requests; unknown reads fail t.
func replayServer(t *testing.T, tape *rpcTape, counter *rpcCounter) *httptest.Server {
	return replayServerWith(t, tape, counter, replayFaults{})
}

func replayServerWith(t *testing.T, tape *rpcTape, counter *rpcCounter, f replayFaults) *httptest.Server {
	lookup := func(to string, data []byte) (tapeEntry, bool) {
		e, ok := tape.Calls[mcKey(to, data)]
		if ok && f.revert != nil && f.revert(to, data) {
			e = tapeEntry{OK: false}
		}
		return e, ok
	}
	var mu sync.Mutex
	n := 0
	answer := func(q rpcMsg) rpcMsg {
		a := rpcMsg{JSONRPC: "2.0", ID: q.ID}
		fail := func(code int, msg string) rpcMsg {
			a.Error = &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{code, msg}
			return a
		}
		switch q.Method {
		case "eth_blockNumber":
			a.Result, _ = json.Marshal(tape.Block)
		case "eth_getBlockByNumber":
			a.Result = tape.BlockJSON
			var tag string
			if f.headAhead+f.headBehind > 0 && len(q.Params) > 0 && json.Unmarshal(q.Params[0], &tag) == nil &&
				tag == "latest" {
				var b map[string]any
				require.NoError(t, json.Unmarshal(tape.BlockJSON, &b))
				ts, err := hexutil.DecodeUint64(b["timestamp"].(string))
				require.NoError(t, err)
				b["timestamp"] = hexutil.EncodeUint64(ts + f.headAhead - f.headBehind)
				a.Result, _ = json.Marshal(b)
			}
		case "eth_call":
			var args callArgs
			var tag string
			if json.Unmarshal(q.Params[0], &args) != nil || json.Unmarshal(q.Params[1], &tag) != nil {
				return fail(-32602, "bad params")
			}
			if tag != tape.Block {
				t.Errorf("replay: read at %s, tape is block %s", tag, tape.Block)
				return fail(-32000, "wrong block")
			}
			data := args.calldata()
			if !bytes.HasPrefix(data, tryAggregateSel) {
				if f.nodeError != nil && f.nodeError(args.To, data) {
					return fail(-32000, "upstream request timeout")
				}
				e, ok := lookup(args.To, data)
				if !ok {
					t.Errorf("replay: unrecorded eth_call %s %s", args.To, hexOf(data))
					return fail(-32000, "unrecorded")
				}
				if !e.OK {
					return fail(3, "execution reverted")
				}
				a.Result, _ = json.Marshal(e.Ret)
				return a
			}
			calls, err := decodeTryAggregate(data)
			if err != nil {
				return fail(-32602, err.Error())
			}
			results := make([]subResult, len(calls))
			for i, c := range calls {
				e, ok := lookup(c.Target.Hex(), c.CallData)
				if !ok {
					t.Errorf("replay: unrecorded sub-call %s %s", c.Target.Hex(), hexOf(c.CallData))
					return fail(-32000, "unrecorded")
				}
				results[i] = subResult{Success: e.OK, ReturnData: e.Ret}
				if f.batchOOG != nil && f.batchOOG(c.Target.Hex(), c.CallData) {
					results[i] = subResult{}
				}
			}
			ret, err := tryAggregateABI.Methods["tryAggregate"].Outputs.Pack(results)
			if err != nil {
				return fail(-32603, err.Error())
			}
			a.Result, _ = json.Marshal(hexutil.Bytes(ret))
		default:
			t.Errorf("replay: unexpected method %s", q.Method)
			return fail(-32601, "method not found")
		}
		return a
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqs, batch, err := readMsgs(body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		counter.count(reqs)
		mu.Lock()
		n++
		k := n
		mu.Unlock()
		time.Sleep(f.delay)
		if f.httpFail != nil && f.httpFail(k) {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		if f.rateLimit != nil && f.rateLimit(k) {
			http.Error(w, "Rate Limit", http.StatusTooManyRequests)
			return
		}
		out := make([]rpcMsg, len(reqs))
		for i, q := range reqs {
			out[i] = answer(q)
		}
		w.Header().Set("Content-Type", "application/json")
		if batch {
			_ = json.NewEncoder(w).Encode(out)
		} else {
			_ = json.NewEncoder(w).Encode(out[0])
		}
	}))
}
