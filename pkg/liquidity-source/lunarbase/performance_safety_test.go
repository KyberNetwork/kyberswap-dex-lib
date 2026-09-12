package lunarbase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// This transport deliberately retains orphan blocks: eth_call by hash alone
// does not establish canonicality. A response can be held after selecting its
// state, allowing deterministic reorg and cancellation tests without servers.
type performanceBlock struct {
	header  *types.Header
	reserve int64
}

type performanceFixture struct {
	mu           sync.Mutex
	head         common.Hash
	canonical    map[uint64]common.Hash
	blocks       map[common.Hash]performanceBlock
	poolReserves map[common.Address]int64
	requests     []verifiedRequest
	failNext     string
	malformed    bool
	hook         func(context.Context, verifiedRequest) error
	outer        abi.ABI
}

func newPerformanceFixture(t *testing.T) *performanceFixture {
	t.Helper()
	outer, err := abi.JSON(strings.NewReader(`[{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	if err != nil {
		t.Fatal(err)
	}
	f := &performanceFixture{canonical: make(map[uint64]common.Hash), blocks: make(map[common.Hash]performanceBlock), poolReserves: make(map[common.Address]int64), outer: outer}
	f.setHead(900, 1, 1230000000)
	return f
}

func (f *performanceFixture) setHead(number uint64, fork byte, reserve int64) *types.Header {
	h := &types.Header{Number: new(big.Int).SetUint64(number), Difficulty: big.NewInt(1), GasLimit: 30000000, Time: 1700000000 + number, Extra: []byte{fork}}
	f.mu.Lock()
	if number > 0 {
		h.ParentHash = f.canonical[number-1]
	}
	f.head = h.Hash()
	f.canonical[number] = f.head
	f.blocks[f.head] = performanceBlock{types.CopyHeader(h), reserve}
	f.mu.Unlock()
	return h
}

func (f *performanceFixture) headHeader() *types.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	return types.CopyHeader(f.blocks[f.head].header)
}

func (f *performanceFixture) setHook(hook func(context.Context, verifiedRequest) error) {
	f.mu.Lock()
	f.hook = hook
	f.mu.Unlock()
}

func (f *performanceFixture) takeRequests() []verifiedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	requests := append([]verifiedRequest(nil), f.requests...)
	f.requests = nil
	return requests
}

func (f *performanceFixture) client(t *testing.T) *ethrpc.Client {
	t.Helper()
	transport := verifiedTransport(func(r *http.Request) (*http.Response, error) {
		if err := r.Context().Err(); err != nil {
			return nil, err
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var request struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err = json.Unmarshal(body, &request); err != nil {
			return nil, err
		}
		req := verifiedRequest{request.Method, request.Params}
		f.mu.Lock()
		f.requests = append(f.requests, req)
		hook := f.hook
		response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
		if f.failNext == req.Method {
			f.failNext = ""
			response["error"] = map[string]any{"code": -32000, "message": "intentional transient failure"}
		} else {
			result, resultErr := f.responseLocked(req)
			if resultErr != nil {
				f.mu.Unlock()
				return nil, resultErr
			}
			response["result"] = result
		}
		f.mu.Unlock()
		// The selected response is immutable before the hook runs.
		if hook != nil {
			if err = hook(r.Context(), req); err != nil {
				return nil, err
			}
		}
		encoded, err := json.Marshal(response)
		if err != nil {
			return nil, err
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(encoded)), Request: r}, nil
	})
	rpcClient, err := rpc.DialOptions(context.Background(), "http://performance-safety.invalid", rpc.WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(rpcClient.Close)
	return ethrpc.NewWithClient(ethclient.NewClient(rpcClient)).SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))
}

func (f *performanceFixture) responseLocked(req verifiedRequest) (any, error) {
	switch req.Method {
	case "eth_getBlockByNumber":
		var selector string
		if err := json.Unmarshal(req.Params[0], &selector); err != nil {
			return nil, err
		}
		hash := f.head
		if selector != "latest" {
			number, err := hexutil.DecodeUint64(selector)
			if err != nil {
				return nil, err
			}
			hash = f.canonical[number]
		}
		block, ok := f.blocks[hash]
		if !ok {
			return nil, nil
		}
		return types.CopyHeader(block.header), nil
	case "eth_call":
		var call map[string]json.RawMessage
		if err := json.Unmarshal(req.Params[0], &call); err != nil {
			return nil, err
		}
		var data hexutil.Bytes
		field := call["input"]
		if field == nil {
			field = call["data"]
		}
		if err := json.Unmarshal(field, &data); err != nil {
			return nil, err
		}
		method := f.outer.Methods["tryBlockAndAggregate"]
		if len(data) < 4 || !bytes.Equal(data[:4], method.ID) {
			return nil, fmt.Errorf("unexpected aggregate selector")
		}
		decoded, err := method.Inputs.Unpack(data[4:])
		if err != nil {
			return nil, err
		}
		calls := *abi.ConvertType(decoded[1], new([]ethrpc.MultiCallParam)).(*[]ethrpc.MultiCallParam)
		if len(calls) != 10 {
			return nil, fmt.Errorf("want ten getters, got %d", len(calls))
		}
		for _, getter := range calls {
			if getter.Target != calls[0].Target {
				return nil, fmt.Errorf("cross-pool aggregate")
			}
		}
		var hash common.Hash
		if len(req.Params[1]) > 0 && req.Params[1][0] == '{' {
			var pin struct {
				BlockHash        common.Hash `json:"blockHash"`
				RequireCanonical bool        `json:"requireCanonical"`
			}
			if err = json.Unmarshal(req.Params[1], &pin); err != nil {
				return nil, err
			}
			hash = pin.BlockHash
			block, exists := f.blocks[hash]
			if !exists {
				return nil, fmt.Errorf("unknown block hash %s", hash)
			}
			if pin.RequireCanonical && f.canonical[block.header.Number.Uint64()] != hash {
				return nil, fmt.Errorf("noncanonical hash")
			}
		} else {
			var selector string
			if err = json.Unmarshal(req.Params[1], &selector); err != nil {
				return nil, err
			}
			if selector == "latest" {
				return nil, fmt.Errorf("un-pinned aggregate is forbidden")
			}
			number, err := hexutil.DecodeUint64(selector)
			if err != nil {
				return nil, err
			}
			hash = f.canonical[number]
		}
		block, ok := f.blocks[hash]
		if !ok {
			return nil, fmt.Errorf("unknown block hash %s", hash)
		}
		items := newVerifiedFixture().items
		reserve := block.reserve
		if own, ok := f.poolReserves[calls[0].Target]; ok {
			reserve = own
		}
		if len(req.Params) == 3 {
			var overrides map[common.Address]struct {
				Balance *hexutil.Big `json:"balance"`
			}
			if err = json.Unmarshal(req.Params[2], &overrides); err != nil {
				return nil, err
			}
			if override, ok := overrides[calls[0].Target]; ok && override.Balance != nil {
				reserve = (*big.Int)(override.Balance).Int64()
			}
		}
		items[5].ReturnData = verifiedWord(big.NewInt(reserve))
		if f.malformed {
			items[2].ReturnData = []byte{1}
		}
		items[8].ReturnData = verifiedWords(new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(15), big.NewInt(25), new(big.Int).Sub(block.header.Number, big.NewInt(1)))
		encoded, err := method.Outputs.Pack(block.header.Number, [32]byte{}, items)
		return hexutil.Encode(encoded), err
	default:
		return nil, fmt.Errorf("unexpected RPC method %s", req.Method)
	}
}

func performanceHint(h *types.Header) pool.GetNewPoolStateParams {
	return pool.GetNewPoolStateParams{BlockHeaders: map[uint64]entity.BlockHeader{h.Number.Uint64(): {Number: new(big.Int).Set(h.Number), Hash: h.Hash().Hex(), Timestamp: h.Time}}}
}

func performanceMethods(t *testing.T, f *performanceFixture, want ...string) []verifiedRequest {
	t.Helper()
	got := f.takeRequests()
	if len(got) != len(want) {
		t.Fatalf("RPC count got %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Method != want[i] {
			t.Fatalf("RPC %d got %s, want %s", i, got[i].Method, want[i])
		}
	}
	return got
}

func TestPerformanceSnapshotCacheBudgetsAndIdentity(t *testing.T) {
	for _, hinted := range []bool{false, true} {
		t.Run(fmt.Sprintf("hint-%t", hinted), func(t *testing.T) {
			f := newPerformanceFixture(t)
			tracker := publicTracker(f.client(t))
			h := f.headHeader()
			params := pool.GetNewPoolStateParams{}
			if hinted {
				params = performanceHint(h)
			}
			first, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), params)
			if err != nil {
				t.Fatal(err)
			}
			requests := performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
			for _, req := range requests {
				if req.Method == "eth_call" {
					var pin struct {
						BlockHash common.Hash `json:"blockHash"`
					}
					if err = json.Unmarshal(req.Params[1], &pin); err != nil || pin.BlockHash != h.Hash() {
						t.Fatalf("aggregate not pinned to expected hash: %s", req.Params[1])
					}
				}
			}
			second, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), params)
			if err != nil {
				t.Fatal(err)
			}
			requests = performanceMethods(t, f, "eth_getBlockByNumber")
			if string(requests[0].Params[0]) != `"latest"` {
				t.Fatalf("wrong cache verification selector: %s", requests[0].Params[0])
			}
			if first.Extra != second.Extra || first.Reserves[0] != second.Reserves[0] || second.BlockNumber != 900 || publicExtra(t, second).BlockHash != h.Hash().Hex() {
				t.Fatal("cached snapshot lost its financial state or identity")
			}
		})
	}
}

func TestPerformanceSnapshotInvalidHintsFallBack(t *testing.T) {
	for _, name := range []string{"nil_number", "number_mismatch", "negative_number", "overflow_number", "empty_hash", "short_hash", "nonhex_hash", "zero_hash"} {
		t.Run(name, func(t *testing.T) {
			f := newPerformanceFixture(t)
			h := f.headHeader()
			params := performanceHint(h)
			hint := params.BlockHeaders[900]
			switch name {
			case "nil_number":
				hint.Number = nil
			case "number_mismatch":
				hint.Number = big.NewInt(899)
			case "negative_number":
				hint.Number = big.NewInt(-1)
			case "overflow_number":
				hint.Number = new(big.Int).Lsh(big.NewInt(1), 65)
			case "empty_hash":
				hint.Hash = ""
			case "short_hash":
				hint.Hash = "0x01"
			case "nonhex_hash":
				hint.Hash = "0x" + strings.Repeat("z", 64)
			case "zero_hash":
				hint.Hash = (common.Hash{}).Hex()
			}
			params.BlockHeaders[900] = hint
			got, err := publicTracker(f.client(t)).GetNewPoolState(context.Background(), verifiedEntity(), params)
			if err != nil || got.BlockNumber != 900 {
				t.Fatalf("invalid optional hint broke verified fallback: %v", err)
			}
			performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
		})
	}
}

func TestPerformanceSnapshotReorgsNeverPublishOrphans(t *testing.T) {
	for _, scenario := range []string{"cached_hint", "hint_miss_during_call", "latest_miss_during_header"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPerformanceFixture(t)
			tracker := publicTracker(f.client(t))
			params := performanceHint(f.headHeader())
			if scenario == "cached_hint" {
				if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), params); err != nil {
					t.Fatal(err)
				}
				f.takeRequests()
				f.setHead(900, 2, 777000000)
			} else {
				var once sync.Once
				trigger := "eth_call"
				if scenario == "latest_miss_during_header" {
					trigger = "eth_getBlockByNumber"
					params = pool.GetNewPoolStateParams{}
				}
				f.setHook(func(_ context.Context, req verifiedRequest) error {
					if req.Method == trigger {
						once.Do(func() { f.setHead(900, 2, 777000000) })
					}
					return nil
				})
			}
			p := verifiedEntity()
			before := publicJSON(t, p)
			got, err := tracker.GetNewPoolState(context.Background(), p, params)
			if !bytes.Equal(before, publicJSON(t, p)) {
				t.Fatal("refresh mutated caller")
			}
			if scenario != "cached_hint" {
				if err == nil || !bytes.Equal(before, publicJSON(t, got)) {
					t.Fatalf("normal miss published a midflight orphan: %v", err)
				}
			} else {
				if err != nil || got.Reserves[0] != "777000000" || publicExtra(t, got).BlockHash != f.headHeader().Hash().Hex() {
					t.Fatalf("hint published an orphan or failed fresh fallback: %v", err)
				}
			}
			f.setHook(nil)
			if scenario == "cached_hint" {
				performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
			} else {
				f.takeRequests()
			}
			got, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			if err != nil || got.Reserves[0] != "777000000" || publicExtra(t, got).BlockHash != f.headHeader().Hash().Hex() {
				t.Fatalf("did not recover on replacement fork: %v", err)
			}
		})
	}
}

func TestPerformanceSnapshotAdvancedHeadAndPerCallerValidation(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{}); err != nil {
		t.Fatal(err)
	}
	f.takeRequests()
	f.setHead(901, 1, 888000000)
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || got.BlockNumber != 901 || got.Reserves[0] != "888000000" {
		t.Fatalf("completed cache hid advancing head: %v", err)
	}
	performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
	for _, kind := range []string{"minimum", "token_identity", "native_metadata"} {
		t.Run(kind, func(t *testing.T) {
			p := verifiedEntity()
			switch kind {
			case "minimum":
				p.BlockNumber = 902
			case "token_identity":
				p.Tokens[0].Address = "0x0000000000000000000000000000000000000001"
			case "native_metadata":
				p.StaticExtra = `{"n":true}`
			}
			before := publicJSON(t, p)
			out, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			if err == nil || !bytes.Equal(before, publicJSON(t, out)) || !bytes.Equal(before, publicJSON(t, p)) {
				t.Fatal("cache bypassed caller-specific validation")
			}
		})
	}
}

func TestPerformanceSnapshotOldHintsAlwaysRefreshToTip(t *testing.T) {
	for _, cached := range []bool{false, true} {
		for _, advance := range []uint64{1, 100} {
			t.Run(fmt.Sprintf("cached-%t-advance-%d", cached, advance), func(t *testing.T) {
				f := newPerformanceFixture(t)
				tracker := publicTracker(f.client(t))
				params := performanceHint(f.headHeader())
				if cached {
					if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), params); err != nil {
						t.Fatal(err)
					}
				}
				f.takeRequests()
				h := f.setHead(900+advance, 1, 444000000)
				got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), params)
				if err != nil || got.BlockNumber != h.Number.Uint64() || got.Reserves[0] != "444000000" || publicExtra(t, got).BlockHash != h.Hash().Hex() {
					t.Fatalf("canonical old hint hid fresh price/reserves: block=%d reserve=%v err=%v", got.BlockNumber, got.Reserves, err)
				}
				performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
			})
		}
	}
}

func TestPerformanceSnapshotFalseHashAndBehindTipNeverPublish(t *testing.T) {
	for _, scenario := range []string{"unknown_hash", "hint_newer_than_tip", "entity_newer_than_tip"} {
		t.Run(scenario, func(t *testing.T) {
			f := newPerformanceFixture(t)
			tracker := publicTracker(f.client(t))
			params := performanceHint(f.headHeader())
			p := verifiedEntity()
			switch scenario {
			case "unknown_hash":
				hint := params.BlockHeaders[900]
				hint.Hash = common.HexToHash("0x1234").Hex()
				params.BlockHeaders[900] = hint
			case "hint_newer_than_tip":
				h := f.setHead(901, 1, 555000000)
				params = performanceHint(h)
				f.setHead(900, 1, 1230000000)
			case "entity_newer_than_tip":
				p.BlockNumber = 901
			}
			before := publicJSON(t, p)
			got, err := tracker.GetNewPoolState(context.Background(), p, params)
			if scenario == "unknown_hash" {
				if err != nil || got.BlockNumber != 900 || publicExtra(t, got).BlockHash != f.headHeader().Hash().Hex() || got.Reserves[0] != "1230000000" {
					t.Fatalf("irrelevant caller hash broke verified latest refresh: %v", err)
				}
			} else if err == nil || !bytes.Equal(before, publicJSON(t, got)) {
				t.Fatalf("invalid snapshot bypassed freshness check: %v", err)
			}
			if !bytes.Equal(before, publicJSON(t, p)) {
				t.Fatal("bad hint mutated caller")
			}
			performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
			got, err = tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
			if err != nil || got.BlockNumber != 900 {
				t.Fatalf("invalid hint poisoned later refresh: %v", err)
			}
			performanceMethods(t, f, "eth_getBlockByNumber")
		})
	}
}

func TestPerformanceSnapshotOverridesBypassCache(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{}); err != nil {
		t.Fatal(err)
	}
	f.takeRequests()
	for _, reserve := range []int64{222000000, 333000000} {
		overrides := map[common.Address]gethclient.OverrideAccount{common.HexToAddress(verifiedPoolAddress): {Balance: big.NewInt(reserve)}}
		got, err := tracker.GetNewPoolStateWithOverrides(context.Background(), verifiedEntity(), pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
		if err != nil || got.Reserves[0] != fmt.Sprint(reserve) {
			t.Fatalf("override snapshot reused another state: %v", err)
		}
		requests := performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
		if len(requests[1].Params) != 3 || string(requests[1].Params[1]) != `"0x384"` {
			t.Fatal("overrides lost their explicit object or number pin")
		}
	}
	for _, overrides := range []map[common.Address]gethclient.OverrideAccount{nil, {}} {
		got, err := tracker.GetNewPoolStateWithOverrides(context.Background(), verifiedEntity(), pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
		if err != nil || got.Reserves[0] != "1230000000" {
			t.Fatalf("empty overrides bypass failed: %v", err)
		}
		performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
	}
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || got.Reserves[0] != "1230000000" {
		t.Fatalf("override state contaminated normal cache: %v", err)
	}
	performanceMethods(t, f, "eth_getBlockByNumber")
}

func TestPerformanceSnapshotMalformedRefreshNeverUsesCachedFields(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{}); err != nil {
		t.Fatal(err)
	}
	f.setHead(901, 1, 888000000)
	f.mu.Lock()
	f.malformed = true
	f.mu.Unlock()
	p := verifiedEntity()
	before := publicJSON(t, p)
	got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	if err == nil || !bytes.Equal(before, publicJSON(t, got)) {
		t.Fatal("failed getter was repaired from cached fields")
	}
	f.mu.Lock()
	f.malformed = false
	f.mu.Unlock()
	got, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	if err != nil || got.BlockNumber != 901 || got.Reserves[0] != "888000000" {
		t.Fatalf("malformed refresh poisoned cache: %v", err)
	}
}

func TestPerformanceSnapshotCanceledCallerDoesNotCancelPeer(t *testing.T) {
	type callerKey struct{}
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{}); err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	var once sync.Once
	f.setHook(func(ctx context.Context, _ verifiedRequest) error {
		if ctx.Value(callerKey{}) == true {
			once.Do(func() { close(entered) })
			<-ctx.Done()
			return ctx.Err()
		}
		return nil
	})
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), callerKey{}, true), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := tracker.GetNewPoolState(ctx, verifiedEntity(), pool.GetNewPoolStateParams{})
		done <- err
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("first request did not start")
	}
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || got.Reserves[0] != "1230000000" {
		t.Fatalf("independent waiter was blocked or canceled: %v", err)
	}
	cancel()
	select {
	case err = <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("first cancellation was lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("first request failed to cancel promptly")
	}
}

func TestPerformanceSnapshotCacheFailureAndCancellationRecover(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{}); err != nil {
		t.Fatal(err)
	}
	f.takeRequests()
	f.mu.Lock()
	f.failNext = "eth_getBlockByNumber"
	f.mu.Unlock()
	p := verifiedEntity()
	before := publicJSON(t, p)
	if got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{}); err == nil || !bytes.Equal(before, publicJSON(t, got)) {
		t.Fatal("cached result bypassed failed canonical check")
	}
	f.takeRequests()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := tracker.GetNewPoolState(canceled, p, pool.GetNewPoolStateParams{}); err == nil || !bytes.Equal(before, publicJSON(t, got)) {
		t.Fatal("already canceled request returned cached success")
	}
	performanceMethods(t, f)
	entered := make(chan struct{})
	var once sync.Once
	f.setHook(func(ctx context.Context, _ verifiedRequest) error {
		once.Do(func() { close(entered) })
		<-ctx.Done()
		return ctx.Err()
	})
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	done := make(chan error, 1)
	go func() { _, err := tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{}); done <- err }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("RPC did not start")
	}
	stop()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation was lost: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled RPC did not exit promptly")
	}
	f.setHook(nil)
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || got.Reserves[0] != "1230000000" {
		t.Fatalf("cache did not recover after failure/cancellation: %v", err)
	}
}

func TestPerformanceSnapshotReturnedEntityCannotPoisonCache(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	first, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil {
		t.Fatal(err)
	}
	first.Reserves[0] = "0"
	first.Extra = `{"paused":true}`
	first.Tokens[0].Address = "0x0000000000000000000000000000000000000001"
	first.StaticExtra = `{"n":true}`
	second, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || second.Reserves[0] != "1230000000" || publicExtra(t, second).Paused || second.Tokens[0].Address != verifiedEntity().Tokens[0].Address {
		t.Fatalf("caller mutation poisoned cached state: %v", err)
	}
}

func TestPerformanceSnapshotConcurrentPoolsAndClients(t *testing.T) {
	first := newPerformanceFixture(t)
	second := newPerformanceFixture(t)
	second.setHead(900, 7, 999000000)
	other := verifiedEntity()
	other.Address = "0x0000000000000000000000000000000000004567"
	first.poolReserves[common.HexToAddress(other.Address)] = 777000000
	tracker1, tracker2 := publicTracker(first.client(t)), publicTracker(second.client(t))
	type scenario struct {
		tracker *PoolTracker
		p       entity.Pool
		reserve string
	}
	cases := []scenario{{tracker1, verifiedEntity(), "1230000000"}, {tracker1, other, "777000000"}, {tracker2, verifiedEntity(), "999000000"}}
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	for i := 0; i < 24; i++ {
		tc := cases[i%len(cases)]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 5; n++ {
				got, err := tc.tracker.GetNewPoolState(context.Background(), tc.p, pool.GetNewPoolStateParams{})
				if err != nil {
					errs <- err
					return
				}
				if got.Address != tc.p.Address || got.Reserves[0] != tc.reserve {
					errs <- fmt.Errorf("cross-pool/client state: address=%s reserve=%s want=%s", got.Address, got.Reserves[0], tc.reserve)
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestPerformanceSnapshotCacheBoundedByEviction(t *testing.T) {
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	firstHint := performanceHint(f.headHeader())
	for i := uint64(0); i < 70; i++ {
		h := f.setHead(900+i, 1, int64(1230000000+i))
		if _, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), performanceHint(h)); err != nil {
			t.Fatal(err)
		}
	}
	f.takeRequests()
	// Restore the fixture tip so this specifically tests bounded eviction,
	// independently of the rule forbidding historical hinted snapshots.
	f.setHead(900, 1, 1230000000)
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), firstHint)
	if err != nil || got.BlockNumber != 900 || got.Reserves[0] != "1230000000" {
		t.Fatalf("evicted snapshot did not safely reload: %v", err)
	}
	performanceMethods(t, f, "eth_getBlockByNumber", "eth_call", "eth_getBlockByNumber")
}

func TestPerformanceSnapshotCacheClonesMutableIntegers(t *testing.T) {
	f := newPerformanceFixture(t)
	client := f.client(t)
	cache := new(snapshotCache)
	first, err := fetchRPCStateWithCache(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, client, cache)
	if err != nil {
		t.Fatal(err)
	}
	wantPrice := first.extra.SqrtPriceX96.String()
	first.reserveX.SetInt64(0)
	first.reserveY.SetInt64(1)
	first.extra.SqrtPriceX96.Clear()
	for i := 0; i < 2; i++ {
		got, err := fetchRPCStateWithCache(context.Background(), verifiedPoolAddress, valueobject.ChainIDBSC, client, cache)
		if err != nil {
			t.Fatal(err)
		}
		if got.reserveX.String() != "1230000000" || got.reserveY.String() != "4560000000" || got.extra.SqrtPriceX96.String() != wantPrice {
			t.Fatal("returned mutable integers alias cached state")
		}
		got.reserveX.SetInt64(777)
		got.reserveY.SetInt64(888)
		got.extra.SqrtPriceX96.Clear()
	}
}

func TestPerformanceSnapshotOutOfOrderCompletionPreservesNewTip(t *testing.T) {
	type olderCallerKey struct{}
	f := newPerformanceFixture(t)
	tracker := publicTracker(f.client(t))
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.setHook(func(ctx context.Context, req verifiedRequest) error {
		if ctx.Value(olderCallerKey{}) == true && req.Method == "eth_getBlockByNumber" && string(req.Params[0]) == `"latest"` {
			once.Do(func() { close(entered) })
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), olderCallerKey{}, true), 5*time.Second)
	defer cancel()
	type result struct {
		pool entity.Pool
		err  error
	}
	done := make(chan result, 1)
	go func() {
		got, err := tracker.GetNewPoolState(ctx, verifiedEntity(), pool.GetNewPoolStateParams{})
		done <- result{got, err}
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("older refresh did not reach its header")
	}
	f.setHead(901, 1, 888000000)
	newer, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || newer.BlockNumber != 901 {
		t.Fatalf("newer refresh failed: %v", err)
	}
	close(release)
	select {
	case older := <-done:
		if older.err != nil || older.pool.BlockNumber != 900 {
			t.Fatalf("older independently canonical refresh failed: %v", older.err)
		}
	case <-ctx.Done():
		t.Fatal("older refresh did not finish")
	}
	f.setHook(nil)
	f.takeRequests()
	got, err := tracker.GetNewPoolState(context.Background(), verifiedEntity(), pool.GetNewPoolStateParams{})
	if err != nil || got.BlockNumber != 901 || got.Reserves[0] != "888000000" {
		t.Fatalf("late old completion replaced fresh cached state: %v", err)
	}
	performanceMethods(t, f, "eth_getBlockByNumber")
}
