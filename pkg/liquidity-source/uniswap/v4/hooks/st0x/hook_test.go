package st0x

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

func TestTrackPersistsReserves(t *testing.T) {
	price, err := priceOracleABI.Methods["getPrice"].Outputs.Pack(struct{ Price, UpdatedAt, SpreadBps *big.Int }{big.NewInt(1e18), big.NewInt(100), big.NewInt(0)})
	require.NoError(t, err)
	stale, err := propAMMHookABI.Methods["maxStaleness"].Outputs.Pack(big.NewInt(100))
	require.NoError(t, err)
	r0, err := propAMMHookABI.Methods["getBalance"].Outputs.Pack(big.NewInt(1000))
	require.NoError(t, err)
	r1, err := propAMMHookABI.Methods["getBalance"].Outputs.Pack(big.NewInt(2000))
	require.NoError(t, err)
	u, _ := abi.NewType("uint256", "", nil)
	b, _ := abi.NewType("bytes[]", "", nil)
	ret, err := (abi.Arguments{{Type: u}, {Type: b}}).Pack(big.NewInt(100), [][]byte{price, stale, r0, r1})
	require.NoError(t, err)
	calls := 0
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.Len(t, req.Params, 2)
		require.JSONEq(t, `"0x64"`, string(req.Params[1]))
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)}))
	}))
	defer stub.Close()
	h := &Hook{Reserve0: uint256.NewInt(1), Reserve1: uint256.NewInt(2)}
	param := &uniswapv4.HookParam{RpcClient: ethrpc.New(stub.URL), HookAddress: hookAddress, BlockNumber: big.NewInt(100), Pool: &entity.Pool{Tokens: []*entity.PoolToken{{Address: "0x0000000000000000000000000000000000000001"}, {Address: "0x0000000000000000000000000000000000000002"}}}}
	extra, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	reserves, err := h.GetReserves(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{"1000", "2000"}, reserves)
	require.Equal(t, 1, calls)
	var restored Hook
	require.NoError(t, json.Unmarshal(extra, &restored))
	require.Equal(t, h.Reserve0, restored.Reserve0)
	require.Equal(t, h.Reserve1, restored.Reserve1)
	for _, direction := range []bool{true, false} {
		result, err := restored.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: direction, AmountSpecified: big.NewInt(10)})
		require.NoError(t, err)
		require.Equal(t, int64(-10), result.DeltaUnspecified.Int64())
	}
	require.Equal(t, "1000", restored.Reserve0.Dec())
	require.Equal(t, "2000", restored.Reserve1.Dec())
}

func TestBeforeSwapMissingReserves(t *testing.T) {
	for _, reserves := range [][2]*uint256.Int{{nil, nil}, {nil, uint256.NewInt(100)}, {uint256.NewInt(100), nil}} {
		h := &Hook{Price: 1e18, Reserve0: reserves[0], Reserve1: reserves[1]}
		for _, direction := range []bool{true, false} {
			for _, calcOut := range []bool{true, false} {
				_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: direction, AmountSpecified: big.NewInt(10)})
				require.ErrorIs(t, err, ErrMissingReserves)
			}
		}
	}
}
