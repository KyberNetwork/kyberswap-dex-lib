package thogprop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// multicallReply encodes a successful tryAggregate reply for the fixture snapshot.
func multicallReply(t *testing.T, sample contractSnapshot, blockTime uint64) string {
	t.Helper()
	results := []tryResult{{Success: true, ReturnData: raw(t, sample.Snapshot)},
		{Success: true, ReturnData: encodedWords(u(blockTime))}}
	for _, token := range xStocks {
		holding, ok := sample.ShareBalances[token]
		if !ok {
			results = append(results, tryResult{}, tryResult{})
			continue
		}
		results = append(results, tryResult{Success: true, ReturnData: encodedWords(holding.Shares)},
			tryResult{Success: true, ReturnData: encodedWords(holding.Multiplier, u(0), u(0))})
	}
	out, err := abipkg.Multicall3ABI.Methods[abipkg.Multicall3TryAggregate].Outputs.Pack(results)
	require.NoError(t, err)
	return hexutil.Encode(out)
}

// A provider that refuses the lens keeps refusing it: after one refusal the tracker
// reads with Multicall3 alone until the cooldown ends, then probes the lens again.
func TestTrackerRemembersLensRefusal(t *testing.T) {
	f := fixture(t, "execution-model3.json.gz")
	sample := f.Snapshots[0]
	expected := stateFrom(t, sample)
	proxy := common.HexToAddress(f.Proxy)
	reply := multicallReply(t, sample, 1793310000)

	var lens, multicall atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request rpcRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		var call map[string]string
		require.NoError(t, json.Unmarshal(request.Params[0], &call))
		out := map[string]any{"jsonrpc": "2.0", "id": request.ID}
		if call["to"] == "" {
			lens.Add(1)
			out["error"] = map[string]any{"code": -32000, "message": "user-specified gas exceeds provider limit"}
		} else {
			multicall.Add(1)
			out["result"] = reply
		}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(out))
	}))
	defer srv.Close()

	tracker, err := NewPoolTracker(&Config{}, ethrpc.New(srv.URL))
	require.NoError(t, err)
	now := time.UnixMilli(1_800_000_000_000)
	tracker.now = func() time.Time { return now }

	for range 3 {
		state, _, err := tracker.readSnapshot(context.Background(), proxy)
		require.NoError(t, err)
		require.Equal(t, expected, state)
	}
	require.EqualValues(t, 1, lens.Load(), "a refused lens is not retried during the cooldown")
	require.EqualValues(t, 3, multicall.Load())

	now = now.Add(lensRetryAfter)
	_, _, err = tracker.readSnapshot(context.Background(), proxy)
	require.NoError(t, err)
	require.EqualValues(t, 2, lens.Load(), "the lens is probed again after the cooldown")
	require.EqualValues(t, 4, multicall.Load())
}

// Request payloads are fixed per proxy, so repeated refreshes reuse one encoding.
func TestSnapshotRequestsAreBuiltOncePerProxy(t *testing.T) {
	tracker, err := NewPoolTracker(&Config{}, ethrpc.New("http://127.0.0.1:0"))
	require.NoError(t, err)
	a, b := common.HexToAddress("0x01"), common.HexToAddress("0x02")
	ra, err := tracker.requests(a)
	require.NoError(t, err)
	again, err := tracker.requests(a)
	require.NoError(t, err)
	require.Same(t, ra, again)
	rb, err := tracker.requests(b)
	require.NoError(t, err)
	require.NotEqual(t, ra.multicall, rb.multicall)
	require.NotEqual(t, ra.lensInit, rb.lensInit)
}
