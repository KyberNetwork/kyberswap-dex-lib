package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const multicallABIJSON = `[{"name":"tryBlockAndAggregate","type":"function","stateMutability":"payable",
"inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],
"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`

func TestPoolTracker_GetNewPoolState_RPCError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"rpc error"}}`, req.ID)
	}))
	defer srv.Close()

	ethrpcClient := ethrpc.New(srv.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	tracker, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDEthereum}, ethrpcClient)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		poolType    string
		staticExtra string
		extra       string
	}{
		{
			name:        "DodoV2_DVM",
			poolType:    "dodo-dvm",
			staticExtra: `{"type":"DVM"}`,
			extra:       `{"i":"1","K":"0","B":"100","Q":"200","B0":"100","Q0":"200","R":"0","mtFeeRate":"0","lpFeeRate":"0"}`,
		},
		{
			name:        "DodoV1_Classical",
			poolType:    "dodo-classical",
			staticExtra: `{"type":"CLASSICAL"}`,
			extra:       `{"B":"100","Q":"200"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := entity.Pool{
				Address:     "0x0000000000000000000000000000000000000001",
				Exchange:    tc.poolType,
				Type:        tc.poolType,
				Reserves:    entity.PoolReserves{"100", "200"},
				Extra:       tc.extra,
				StaticExtra: tc.staticExtra,
			}

			got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
			require.Error(t, err)
			assert.Equal(t, entity.Pool{}, got)
		})
	}
}

func TestPoolTracker_GetNewPoolState_DodoV2DeadPool(t *testing.T) {
	t.Parallel()

	mcABI, err := abi.JSON(strings.NewReader(multicallABIJSON))
	require.NoError(t, err)

	pmmStateRet, err := V2PoolABI.Methods[dodoV2MethodGetPMMStateForCall].Outputs.Pack(
		big.NewInt(1), // i
		big.NewInt(0), // K == 0
		big.NewInt(0), // B == 0
		big.NewInt(0), // Q == 0
		big.NewInt(0), // B0
		big.NewInt(0), // Q0
		big.NewInt(0), // R
	)
	require.NoError(t, err)

	lpFeeRet, err := V2PoolABI.Methods[dodoV1MethodLpFeeRate].Outputs.Pack(big.NewInt(0))
	require.NoError(t, err)

	type returnDataItem struct {
		Success    bool
		ReturnData []byte
	}
	packedResp, err := mcABI.Methods["tryBlockAndAggregate"].Outputs.Pack(
		big.NewInt(12345),
		[32]byte{1},
		[]returnDataItem{
			{Success: true, ReturnData: pmmStateRet},
			{Success: true, ReturnData: lpFeeRet},
			{Success: false, ReturnData: nil}, // getUserFeeRate reverts on dead pool
			{Success: false, ReturnData: nil},
			{Success: false, ReturnData: nil},
		},
	)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, hexutil.Encode(packedResp))
	}))
	defer srv.Close()

	ethrpcClient := ethrpc.New(srv.URL).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	tracker, err := NewPoolTracker(&Config{ChainID: valueobject.ChainIDEthereum}, ethrpcClient)
	require.NoError(t, err)

	p := entity.Pool{
		Address:     "0x0000000000000000000000000000000000000001",
		Exchange:    "dodo-dvm",
		Type:        "dodo-dvm",
		Reserves:    entity.PoolReserves{"100", "200"},
		Extra:       `{"i":"1","K":"0","B":"100","Q":"200"}`,
		StaticExtra: `{"type":"DVM"}`,
	}

	got, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Empty(t, got.Extra)
	assert.Equal(t, entity.PoolReserves{"0", "0"}, got.Reserves)
	assert.Equal(t, uint64(12345), got.BlockNumber)
}
