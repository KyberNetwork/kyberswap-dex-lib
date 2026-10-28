package eth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func recordBatch(t *testing.T) (*rpc.Client, *[][]json.RawMessage) {
	t.Helper()
	var params [][]json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqs []struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqs))
		resp := make([]map[string]any, len(reqs))
		for i, req := range reqs {
			params = append(params, req.Params)
			resp[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x01"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	client, err := rpc.Dial(srv.URL)
	require.NoError(t, err)
	t.Cleanup(client.Close)
	return client, &params
}

func TestBatchGetStorageAt_BlockTagRetryAndLegacyBehavior(t *testing.T) {
	for _, tc := range []struct {
		name, tag string
		mode      string
		wantErr   string
		wantCalls int
	}{
		{name: "legacy-latest", wantCalls: 1},
		{name: "pinned-unknown-retry", tag: "0x3039", mode: "unknown", wantCalls: 2},
		{name: "pinned-rate-retry-raw-element-error", tag: "0x3039", mode: "rate-limit", wantErr: "element broke", wantCalls: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var tags []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if tc.mode == "rate-limit" && calls == 1 {
					http.Error(w, "429 rate limit", http.StatusTooManyRequests)
					return
				}
				var reqs []struct {
					ID     json.RawMessage   `json:"id"`
					Params []json.RawMessage `json:"params"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&reqs))
				resp := make([]map[string]any, len(reqs))
				for i, req := range reqs {
					var tag string
					_ = json.Unmarshal(req.Params[2], &tag)
					tags = append(tags, tag)
					if tc.mode == "unknown" && calls == 1 {
						resp[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "unknown block"}}
					} else if tc.mode == "rate-limit" {
						resp[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32000, "message": "element broke"}}
					} else {
						resp[i] = map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x" + strings.Repeat("0", 64)}
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			defer srv.Close()
			client, err := rpc.Dial(srv.URL)
			require.NoError(t, err)
			defer client.Close()
			reads := []StorageRead{{Slot: common.Hash{1}}}
			if tc.tag == "" {
				err = BatchGetStorageAt(context.Background(), client, common.Address{1}, reads, BatchRetry{MaxAttempts: 2})
			} else {
				err = BatchGetStorageAt(context.Background(), client, common.Address{1}, reads, BatchRetry{MaxAttempts: 2}, tc.tag)
			}
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantCalls, calls)
			wantTag := tc.tag
			if wantTag == "" {
				wantTag = "latest"
			}
			for _, got := range tags {
				require.Equal(t, wantTag, got)
			}
		})
	}
}

func TestBatchEthCall_Overrides(t *testing.T) {
	client, params := recordBatch(t)

	runAt := common.HexToAddress("0x8f10b468b06c6fd214b65f87778827f7d113f996")
	slot0 := common.Hash{}
	value := common.BytesToHash(common.HexToAddress("0x28e2Ea090877bF75740558f6BFB36A5ffeE9e9dF").Bytes())

	calls := []BatchCall{
		{
			To:   runAt.Hex(),
			Data: []byte{0xaa},
			Overrides: map[common.Address]gethclient.OverrideAccount{
				runAt: {Code: []byte{0x60, 0x00}, StateDiff: map[common.Hash]common.Hash{slot0: value}},
			},
		},
		{To: runAt.Hex(), Data: []byte{0xaa}},
	}

	results, callErrs, err := BatchEthCall(context.Background(), client, calls, nil)
	require.NoError(t, err)
	require.Len(t, *params, 2)
	for i := range calls {
		require.NoError(t, callErrs[i])
		assert.Equal(t, []byte{0x01}, results[i])
	}

	withOverride := (*params)[0]
	require.Len(t, withOverride, 3, "override must be sent as eth_call's third parameter")
	var set map[common.Address]struct {
		Code      string            `json:"code"`
		StateDiff map[string]string `json:"stateDiff"`
		State     json.RawMessage   `json:"state"`
	}
	require.NoError(t, json.Unmarshal(withOverride[2], &set))
	acc, ok := set[runAt]
	require.True(t, ok, "override keyed by the address it applies to, got %s", withOverride[2])
	assert.Equal(t, "0x6000", acc.Code)
	assert.Equal(t, map[string]string{slot0.Hex(): value.Hex()}, acc.StateDiff)
	assert.Empty(t, acc.State)

	assert.Len(t, (*params)[1], 2, "a call without overrides must stay a plain two-parameter eth_call")
}
