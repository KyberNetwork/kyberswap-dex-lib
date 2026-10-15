package uscoreprop

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

const (
	testPool  = "0xed2ef1b02f2d82d238d6af17e6404a4977b0fefa"
	testQuote = "0xb88339cb7199b77e23db6e890353e22632ba630f"
	testBase  = "0x5555555555555555555555555555555555555555"
	multicall = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

var multicallABI = lo.Must(abi.JSON(strings.NewReader(`[
 {"type":"function","name":"tryAggregate","inputs":[{"type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}
]`)))

// listerRPC answers the lister's tryAggregate of quote()/base(); failBase makes base() revert.
func listerRPC(t *testing.T, failBase bool, count *int) *ethrpc.Client {
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
		method, err := multicallABI.MethodById(data[:4])
		require.NoError(t, err)
		args, err := method.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := *abi.ConvertType(args[1], new([]struct {
			Target   common.Address
			CallData []byte
		})).(*[]struct {
			Target   common.Address
			CallData []byte
		})
		type result struct {
			Success    bool
			ReturnData []byte
		}
		var results []result
		for _, c := range calls {
			m, err := poolABI.MethodById(c.CallData[:4])
			require.NoError(t, err)
			token, success := testQuote, true
			if m.Name == "base" {
				token, success = testBase, !failBase
			}
			out, err := m.Outputs.Pack(common.HexToAddress(token))
			require.NoError(t, err)
			results = append(results, result{success, out})
		}
		encoded, err := method.Outputs.Pack(results)
		require.NoError(t, err)
		if count != nil {
			*count++
		}
		w.Header().Set("content-type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID, "result": hexutil.Encode(encoded)}))
	}))
	t.Cleanup(server.Close)
	return ethrpc.New(server.URL).SetMulticallContract(common.HexToAddress(multicall))
}

func TestRegisteredLister(t *testing.T) {
	factory := poollist.Factory(DexType)
	require.NotNil(t, factory)
	lister, err := factory(DexType, poollist.FactoryParams{
		Properties:   poollist.Properties{"pools": []string{testPool}},
		Dependencies: poollist.Dependencies{EthrpcClient: listerRPC(t, false, nil)},
	})
	require.NoError(t, err)
	pools, _, err := lister.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, DexType, pools[0].Exchange)
	require.Equal(t, DexType, pools[0].Type)
}

func TestDiscoveryDeduplicatesAndRetriesFailures(t *testing.T) {
	cfg := &Config{DexID: DexType, Pools: []string{testPool, common.HexToAddress(testPool).Hex()}}
	failed := NewPoolsListUpdater(cfg, listerRPC(t, true, nil))
	pools, meta, err := failed.GetNewPools(t.Context(), nil)
	require.NoError(t, err)
	require.Empty(t, pools)
	count := 0
	lister := NewPoolsListUpdater(cfg, listerRPC(t, false, &count))
	pools, meta, err = lister.GetNewPools(t.Context(), meta)
	require.NoError(t, err)
	require.Len(t, pools, 1)
	require.Equal(t, testQuote, pools[0].Tokens[0].Address)
	require.Equal(t, testBase, pools[0].Tokens[1].Address)
	require.Equal(t, entity.PoolReserves{"0", "0"}, pools[0].Reserves)
	pools, _, err = lister.GetNewPools(t.Context(), meta)
	require.NoError(t, err)
	require.Empty(t, pools)
	require.Equal(t, 1, count)
}
