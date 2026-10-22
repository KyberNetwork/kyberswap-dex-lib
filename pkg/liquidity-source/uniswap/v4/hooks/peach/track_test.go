package peach

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func TestTrackPinnedSnapshotOverridesAndReset(t *testing.T) {
	aggregate := mustABI(`[{"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}],"stateMutability":"nonpayable"}]`)
	address := common.HexToAddress("0x98C819a522b91f76bAEAf499F3801e76890F2080")
	manager := uniswapv4.PoolManager[56]
	timestamp, auto, fee := int64(1000), int64(700), int64(700)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "eth_call", request.Method)
		require.Len(t, request.Params, 3)
		require.JSONEq(t, `"0x2a"`, string(request.Params[1]))
		var overrides map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(request.Params[2], &overrides))
		require.Contains(t, overrides, hexutil.Encode(address[:]))
		var call struct {
			Data  string `json:"data"`
			Input string `json:"input"`
		}
		require.NoError(t, json.Unmarshal(request.Params[0], &call))
		data := call.Data
		if data == "" {
			data = call.Input
		}
		encoded, err := hexutil.Decode(data)
		require.NoError(t, err)
		unpacked, err := aggregate.Methods["aggregate"].Inputs.Unpack(encoded[4:])
		require.NoError(t, err)
		// Every state getter and the timestamp live in one aggregate at block 42.
		require.Len(t, unpacked, 1)
		require.Equal(t, 13, reflect.ValueOf(unpacked[0]).Len())
		values := []struct {
			name string
			args []any
		}{
			{"currentFee", []any{big.NewInt(fee)}}, {"currentFee", []any{big.NewInt(fee)}},
			{"autonomousFee", []any{big.NewInt(auto)}}, {"autonomousFee", []any{big.NewInt(auto)}},
			{"pokeFloor", []any{big.NewInt(100)}}, {"maxFee", []any{big.NewInt(6000)}},
			{"ABSOLUTE_MAX_FEE", []any{big.NewInt(1_000_000)}},
			{"pokeOf", []any{big.NewInt(0), big.NewInt(0), big.NewInt(0)}},
			{"pausedUntil", []any{big.NewInt(0)}}, {"pausedUntilOf", []any{big.NewInt(0)}},
			{"midBlockConfig", []any{true, big.NewInt(100), big.NewInt(500_000), big.NewInt(5000)}},
			{"poolManager", []any{manager}},
		}
		returned := make([][]byte, 0, 13)
		for _, v := range values {
			b, e := hookABI.Methods[v.name].Outputs.Pack(v.args...)
			require.NoError(t, e)
			returned = append(returned, b)
		}
		b, e := clockABI.Methods["getCurrentBlockTimestamp"].Outputs.Pack(big.NewInt(timestamp))
		require.NoError(t, e)
		returned = append(returned, b)
		b, e = aggregate.Methods["aggregate"].Outputs.Pack(big.NewInt(42), returned)
		require.NoError(t, e)
		calls++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": hexutil.Encode(b)}))
	}))
	defer server.Close()
	param := &uniswapv4.HookParam{Cfg: &uniswapv4.Config{ChainID: 56}, HookAddress: address, Pool: &entity.Pool{Address: "0x1234"}, BlockNumber: big.NewInt(42), RpcClient: ethrpc.New(server.URL), Overrides: map[common.Address]gethclient.OverrideAccount{address: {Nonce: 1}}}
	h := testHook()
	h.Advanced = true
	h.OpenPrice.SetUint64(123)
	raw, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	require.NotEmpty(t, raw)
	require.Equal(t, 1, calls)
	require.True(t, h.RWA)
	require.False(t, h.Advanced)
	require.True(t, h.OpenPrice.IsZero())
	require.EqualValues(t, 700, quote(t, h, "1000", true).SwapFee)
	// RWA fee changes are taken from a fresh snapshot, not extrapolated from an old curve.
	timestamp, auto, fee = 1001, 1500, 1500
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	h.QuoteAt = 1001
	require.EqualValues(t, 1500, quote(t, h, "1000", true).SwapFee)
	require.EqualValues(t, 1002, h.NextTrackAt(param))
	// Autonomous fee may exceed one million before the pool cap is applied.
	auto, fee = 3_000_000, 6000
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	require.EqualValues(t, 6000, quote(t, h, "1000", true).SwapFee)
	before := h.Extra
	fee = 1
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrInvalidState)
	require.Equal(t, before, h.Extra)
	manager = common.Address{}
	_, err = h.Track(context.Background(), param)
	require.ErrorIs(t, err, ErrDeployment)
}
