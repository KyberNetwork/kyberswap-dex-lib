package shared

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const testMulticall = "0xcA11bde05977b3631167028862bE2a173976CA11"

var testMulticallABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"aggregate","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"returnData","type":"bytes[]"}]},
 {"type":"function","name":"tryAggregate","inputs":[{"type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`)))

// onchainRPC serves a factory holding pools, plus zero-valued vault reads (no hook, no buffer).
func onchainRPC(t *testing.T, pools []common.Address) *ethrpc.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ID     json.RawMessage
			Params []json.RawMessage
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		var call struct{ Data, Input hexutil.Bytes }
		require.NoError(t, json.Unmarshal(body.Params[0], &call))
		data := append(call.Input, call.Data...)
		method, err := testMulticallABI.MethodById(data[:4])
		require.NoError(t, err)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		type param struct {
			Target   common.Address
			CallData []byte
		}
		calls := *abi.ConvertType(args[len(args)-1], new([]param)).(*[]param)

		returns := make([][]byte, len(calls))
		for i, c := range calls {
			var m *abi.Method
			if pm, err := PoolFactoryABI.MethodById(c.CallData[:4]); err == nil {
				m = pm
			} else {
				m, err = VaultExplorerABI.MethodById(c.CallData[:4])
				require.NoError(t, err)
			}
			var out []any
			switch m.Name {
			case "getPoolCount":
				out = []any{big.NewInt(int64(len(pools)))}
			case "getPoolsInRange":
				in, err := m.Inputs.Unpack(c.CallData[4:])
				require.NoError(t, err)
				start, n := int(in[0].(*big.Int).Int64()), int(in[1].(*big.Int).Int64())
				out = []any{pools[start : start+n]}
			case "getPoolTokens":
				out = []any{[]common.Address{common.HexToAddress("0xaa"), common.HexToAddress("0xbb")}}
			default: // getHooksConfig, getBufferAsset: zero value
				for _, o := range m.Outputs {
					out = append(out, reflect.New(o.Type.GetType()).Elem().Interface())
				}
			}
			returns[i], err = m.Outputs.Pack(out...)
			require.NoError(t, err)
		}

		var encoded []byte
		if method.Name == "aggregate" {
			encoded, err = method.Outputs.Pack(big.NewInt(1), returns)
		} else {
			type result struct {
				Success    bool
				ReturnData []byte
			}
			results := lo.Map(returns, func(b []byte, _ int) result { return result{true, b} })
			encoded, err = method.Outputs.Pack(results)
		}
		require.NoError(t, err)
		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID,
			"result": hexutil.Encode(encoded)}))
	}))
	t.Cleanup(server.Close)
	return ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress(testMulticall))
}

// An on-chain factory's cursor is reset every RelistInterval runs so that every pool is emitted again.
// StaticExtra.BufferTokens is only set at listing, so a buffer initialized later is picked up on relist.
func TestGetNewPoolsOnchainRelist(t *testing.T) {
	factory := "0x00000000000000000000000000000000000000FF"
	listed := []common.Address{common.HexToAddress("0x01"), common.HexToAddress("0x02")}
	u := NewPoolsListUpdater(&Config{DexID: "bush-v3-weighted", ChainID: valueobject.ChainIDRobinhood,
		Factories: []string{factory}, NewPoolLimit: 10}, onchainRPC(t, listed), nil)
	ctx := t.Context()

	pools, md, err := u.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.Len(t, pools, 2)
	var metadata Metadata
	require.NoError(t, json.Unmarshal(md, &metadata))
	require.Equal(t, map[string]int{strings.ToLower(factory): 2}, metadata.Offsets)

	pools, _, err = u.GetNewPools(ctx, md)
	require.NoError(t, err)
	require.Empty(t, pools, "cursor advances between relists")

	u.count = RelistInterval
	pools, md, err = u.GetNewPools(ctx, md)
	require.NoError(t, err)
	require.Len(t, pools, 2, "relist run re-emits every pool")
	require.NoError(t, json.Unmarshal(md, &metadata))
	require.Equal(t, map[string]int{strings.ToLower(factory): 2}, metadata.Offsets)
}
