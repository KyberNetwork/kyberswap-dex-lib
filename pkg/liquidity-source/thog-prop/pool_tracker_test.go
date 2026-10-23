package thogprop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestTrackerOneAtomicSnapshotRoundTripsIntoSimulator(t *testing.T) {
	f := fixture(t, "deployed-v7.json.gz")
	expected := stateFrom(t, f.Snapshots[0].Snapshot)
	data, err := thogAMMABI.Pack(methodGetPoolData, uint8(0), uint8(32))
	require.NoError(t, err)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "eth_call", request.Method)
		require.Len(t, request.Params, 2)
		var call map[string]string
		require.NoError(t, json.Unmarshal(request.Params[0], &call))
		require.Equal(t, f.Proxy, call["to"])
		require.Equal(t, "0x1", call["gasPrice"])
		input := call["input"]
		if input == "" {
			input = call["data"]
		}
		require.Equal(t, hexutil.Encode(data), input)
		require.JSONEq(t, `"latest"`, string(request.Params[1]))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": f.Snapshots[0].Snapshot}))
	}))
	defer server.Close()
	tracker, err := NewPoolTracker(&Config{Proxy: f.Proxy}, ethrpc.New(server.URL))
	require.NoError(t, err)
	before := entity.Pool{Address: f.Proxy, Type: DexType, StaticExtra: `{"poolId":"0xce389e78282dedac7b18ba7f775b7602d2ab3ab171bbd6711eb0239be6ef4dcc"}`}
	after, err := tracker.GetNewPoolState(context.Background(), before, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.EqualValues(t, 1, calls.Load())
	sim, err := NewPoolSimulator(after)
	require.NoError(t, err)
	require.Equal(t, expected, sim.state)
	require.Equal(t, expected.BlockNumber, sim.Info.BlockNumber)
	require.Empty(t, before.Extra)
	require.Empty(t, before.Tokens)
}
func TestSnapshotRejectsIncompleteOrIncompatibleImages(t *testing.T) {
	f := fixture(t, "deployed-v7.json.gz")
	for _, mutate := range []func(*poolData){
		func(p *poolData) { p.SchemaVersion = 6 }, func(p *poolData) { p.Words = p.Words[:125] }, func(p *poolData) { p.PortfolioBalances = p.PortfolioBalances[:7] },
		func(p *poolData) { p.Words[1].Slot = 2 }, func(p *poolData) { p.Tokens[0].VolumeRate = 0 }, func(p *poolData) { p.Categories[0].AnchorTokenIndex = 4 }, func(p *poolData) { p.BaseFeePerGas = nil },
	} {
		p, err := decodeSnapshot(raw(t, f.Snapshots[0].Snapshot))
		require.NoError(t, err)
		mutate(&p)
		_, err = snapshotState(p)
		require.Error(t, err)
	}
}
