package thogprop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	abipkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// pageTokens lists the snapshot's tokens in pool order, as the tracker passes them.
func pageTokens(t *testing.T, sample contractSnapshot) []common.Address {
	t.Helper()
	page := fixturePage(t, sample)
	tokens := make([]common.Address, len(page.Tokens))
	for i, token := range page.Tokens {
		tokens[i] = token.Token
	}
	return tokens
}

// multicallReply encodes a successful tryAggregate reply for the fixture snapshot
// to a batch built from tokens: page, block timestamp, then storedMultiplier and
// sharesOf per token. Ordinary tokens answer a zero multiplier and fail sharesOf.
func multicallReply(t *testing.T, sample contractSnapshot, tokens []common.Address, blockTime uint64) string {
	t.Helper()
	results := []tryResult{{Success: true, ReturnData: raw(t, sample.Snapshot)},
		{Success: true, ReturnData: encodedWords(u(blockTime))}}
	for _, token := range tokens {
		holding, ok := sample.ShareBalances[hexutil.Encode(token[:])]
		if !ok {
			results = append(results, tryResult{Success: true, ReturnData: encodedWords(u(0))}, tryResult{})
			continue
		}
		results = append(results, tryResult{Success: true, ReturnData: encodedWords(holding.Multiplier)},
			tryResult{Success: true, ReturnData: encodedWords(holding.Shares)})
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
	tokens := pageTokens(t, sample)
	reply := multicallReply(t, sample, tokens, 1793310000)

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
		state, _, err := tracker.readSnapshot(context.Background(), proxy, tokens)
		require.NoError(t, err)
		require.Equal(t, expected, state)
	}
	require.EqualValues(t, 1, lens.Load(), "a refused lens is not retried during the cooldown")
	require.EqualValues(t, 3, multicall.Load())

	now = now.Add(lensRetryAfter)
	_, _, err = tracker.readSnapshot(context.Background(), proxy, tokens)
	require.NoError(t, err)
	require.EqualValues(t, 2, lens.Load(), "the lens is probed again after the cooldown")
	require.EqualValues(t, 4, multicall.Load())
}

// Request payloads depend only on the proxy and its token list, so repeated
// refreshes reuse one encoding until the list changes.
func TestSnapshotRequestsAreBuiltOncePerTokenList(t *testing.T) {
	tracker, err := NewPoolTracker(&Config{}, ethrpc.New("http://127.0.0.1:0"))
	require.NoError(t, err)
	a, b := common.HexToAddress("0x01"), common.HexToAddress("0x02")
	tokens := []common.Address{common.HexToAddress("0x11"), common.HexToAddress("0x12")}
	ra, err := tracker.requests(a, tokens)
	require.NoError(t, err)
	again, err := tracker.requests(a, slices.Clone(tokens))
	require.NoError(t, err)
	require.Same(t, ra, again)
	rb, err := tracker.requests(b, tokens)
	require.NoError(t, err)
	require.NotEqual(t, ra.multicall, rb.multicall)
	require.NotEqual(t, ra.lensInit, rb.lensInit)
	relisted, err := tracker.requests(a, tokens[:1])
	require.NoError(t, err)
	require.NotSame(t, ra, relisted)
	require.Equal(t, ra.lensInit, relisted.lensInit)
	require.NotEqual(t, ra.multicall, relisted.multicall)
}
