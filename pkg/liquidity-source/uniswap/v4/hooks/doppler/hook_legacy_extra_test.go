package doppler

import (
	"context"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Robinhood pools created before the initializer was registered still carry the
// auto-calibration hX, whose "a" (anchors, a list) collides with DHook's "a" (Asset).
const autoHX = `{"f":[{"o":10500},{}],"b":36,"lg":[10001,0],"lc":[5,0],"u":1785000000,"k":3,` +
	`"v":1785000100,"a":[[{"i":"1000","r":"10500"}],null],"r":[4481,0]}`

var robinhoodInitializer = common.HexToAddress("0x4e3468951D49f2EEa976eD0D6e75fFCb44a9a544")

// The Doppler factory must survive an auto-shaped hX: no scheduling, Asset not taken from the
// colliding "a", and Track's first RPC asks the initializer for getState of both pool tokens.
func TestAutoShapedHookExtra(t *testing.T) {
	token0 := common.HexToAddress("0x00000000000000000000000000000000000000a0")
	token1 := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	p := entity.Pool{
		Address: "0xpool",
		Tokens: []*entity.PoolToken{
			{Address: hexutil.Encode(token0[:])}, {Address: hexutil.Encode(token1[:])},
		},
	}

	hook, ok := uniswapv4.GetHook(robinhoodInitializer, &uniswapv4.HookParam{
		HookExtra: uniswapv4.HookExtra(json.RawMessage(autoHX)), Pool: &p,
	})
	require.True(t, ok)
	require.IsType(t, &DHook{}, hook)
	assert.Equal(t, string(valueobject.ExchangeUniswapV4Doppler), hook.GetExchange())
	assert.NotImplements(t, (*uniswapv4.HookWithSchedule)(nil), hook)

	// Same through the tracker: no scheduled update, so pool-service drops the pool.
	extra, err := json.Marshal(uniswapv4.Extra{HookExtra: json.RawMessage(autoHX)})
	require.NoError(t, err)
	p.StaticExtra = `{"hooks":"` + robinhoodInitializer.Hex() + `"}`
	p.Extra = string(extra)
	assert.Zero(t, uniswapv4.NewPoolTracker(&uniswapv4.Config{}, nil, nil).NextUpdateAt(p))

	assert.Equal(t, common.Address{}, hook.(*DHook).Asset, "colliding \"a\" must not leave a garbage Asset")

	// Stub the multicall: token0 is not an asset (revert), token1 is the asset.
	var gotTargets []common.Address
	var gotCalls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCalls.Add(1)
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage `json:"id"`
			Params []json.RawMessage
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var msg struct{ Data, Input string }
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := hexutil.MustDecode(max(msg.Data, msg.Input))
		gotTargets = decodeGetStateAssets(t, data)

		resp, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(tryAggregateResult(t, token0, token1)),
		})
		_, _ = w.Write(resp)
	}))
	defer stub.Close()

	got, err := hook.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: ethrpc.New(stub.URL), Pool: &p, HookAddress: robinhoodInitializer,
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), gotCalls.Load())
	assert.Equal(t, []common.Address{token0, token1}, gotTargets)

	var out DHook
	require.NoError(t, json.Unmarshal(got, &out))
	assert.Equal(t, token1, out.Asset)
}

// decodeGetStateAssets returns the getState(asset) argument of each call in a tryAggregate.
func decodeGetStateAssets(t *testing.T, data []byte) []common.Address {
	callT, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "target", Type: "address"}, {Name: "callData", Type: "bytes"},
	})
	require.NoError(t, err)
	boolT, _ := abi.NewType("bool", "", nil)
	args, err := abi.Arguments{{Type: boolT}, {Type: callT}}.Unpack(data[4:])
	require.NoError(t, err)
	var assets []common.Address
	for _, c := range args[1].([]struct {
		Target   common.Address `json:"target"`
		CallData []byte         `json:"callData"`
	}) {
		assert.Equal(t, robinhoodInitializer, c.Target)
		assert.Equal(t, poolStateABI.Methods["getState"].ID, c.CallData[:4])
		assets = append(assets, common.BytesToAddress(c.CallData[4:]))
	}
	return assets
}

// tryAggregateResult answers token0 with a revert and token1 with a state whose numeraire is token0.
func tryAggregateResult(t *testing.T, token0, token1 common.Address) []byte {
	state, err := poolStateABI.Methods["getState"].Outputs.Pack(
		token0, big.NewInt(1), common.Address{}, []byte{}, uint8(1),
		uniswapv4.PoolKey{Currency0: token0, Currency1: token1, Fee: big.NewInt(0), TickSpacing: big.NewInt(60)},
		int32(0))
	require.NoError(t, err)
	resT, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "success", Type: "bool"}, {Name: "returnData", Type: "bytes"},
	})
	require.NoError(t, err)
	type res struct {
		Success    bool
		ReturnData []byte
	}
	out, err := abi.Arguments{{Type: resT}}.Pack([]res{{false, nil}, {true, state}})
	require.NoError(t, err)
	return out
}
