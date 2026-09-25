package eth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNode answers every eth_call with reply and records the params it got.
func fakeNode(t *testing.T, reply map[string]any) (*rpc.Client, *[]json.RawMessage) {
	t.Helper()
	var params []json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		params = req.Params
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		for k, v := range reply {
			resp[k] = v
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

func TestDeploylessCall_ReturnsRevertData(t *testing.T) {
	client, params := fakeNode(t, map[string]any{
		"error": map[string]any{"code": 3, "message": "execution reverted", "data": "0x04cb9605abcd"},
	})

	data, err := DeploylessCall(context.Background(), client, []byte{0x60, 0x00}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x04, 0xcb, 0x96, 0x05, 0xab, 0xcd}, data)

	require.Len(t, *params, 2, "no overrides: plain two-parameter eth_call")
	var call map[string]json.RawMessage
	require.NoError(t, json.Unmarshal((*params)[0], &call))
	_, hasTo := call["to"]
	assert.False(t, hasTo, "creation call must omit \"to\", not send null")
	assert.JSONEq(t, `"0x6000"`, string(call["data"]))
}

// A block override alone still needs the state-override slot filled, since
// eth_call's parameters are positional.
func TestDeploylessCall_BlockOverrideKeepsPositions(t *testing.T) {
	client, params := fakeNode(t, map[string]any{
		"error": map[string]any{"code": 3, "message": "execution reverted", "data": "0x01"},
	})

	_, err := DeploylessCall(context.Background(), client, []byte{0x00}, nil,
		&gethclient.BlockOverrides{Time: 1790316383})
	require.NoError(t, err)

	require.Len(t, *params, 4)
	assert.JSONEq(t, `{}`, string((*params)[2]))
	assert.JSONEq(t, `{"time":"0x6ab60f5f"}`, string((*params)[3]))
}

func TestDeploylessCall_OverridesSent(t *testing.T) {
	client, params := fakeNode(t, map[string]any{
		"error": map[string]any{"code": 3, "message": "execution reverted", "data": "0x01"},
	})
	registry := common.HexToAddress("0xda7afeed021eafc1c1af9c362de477dad0396b81")
	slot, value := common.HexToHash("0x01"), common.HexToHash("0x02")

	_, err := DeploylessCall(context.Background(), client, []byte{0x00},
		map[common.Address]gethclient.OverrideAccount{registry: {StateDiff: map[common.Hash]common.Hash{slot: value}}},
		nil)
	require.NoError(t, err)

	require.Len(t, *params, 3)
	var set map[common.Address]struct {
		StateDiff map[string]string `json:"stateDiff"`
	}
	require.NoError(t, json.Unmarshal((*params)[2], &set))
	assert.Equal(t, map[string]string{slot.Hex(): value.Hex()}, set[registry].StateDiff)
}

// A constructor that returns normally means the initcode is not a deployless
// helper; surfacing that beats decoding its runtime code as a payload.
func TestDeploylessCall_NoRevert(t *testing.T) {
	client, _ := fakeNode(t, map[string]any{"result": "0x6000"})

	_, err := DeploylessCall(context.Background(), client, []byte{0x00}, nil, nil)
	assert.ErrorIs(t, err, ErrDeploylessNoRevert)
}

// A revert without data (e.g. out of gas, or a provider that strips it) is a
// failure, not an empty payload.
func TestDeploylessCall_RevertWithoutData(t *testing.T) {
	client, _ := fakeNode(t, map[string]any{
		"error": map[string]any{"code": -32000, "message": "out of gas"},
	})

	data, err := DeploylessCall(context.Background(), client, []byte{0x00}, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, data)
}
