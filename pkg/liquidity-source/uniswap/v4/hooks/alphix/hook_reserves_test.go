package alphix

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// fakeHookState is what the fake node answers for the hook's GetReserves batch.
type fakeHookState struct {
	balances  map[common.Address]*big.Int // balanceOf(poolManager) per token
	yield     map[common.Address]*big.Int // getAmountInYieldSource per token
	fee       uint64
	tickLower int64
	tickUpper int64
}

// answer packs the return data of one inner multicall call.
func (s *fakeHookState) answer(t *testing.T, target common.Address, data []byte) []byte {
	sel := data[:4]
	switch {
	case string(sel) == string(abi.Erc20ABI.Methods[abi.Erc20BalanceOfMethod].ID):
		out, err := abi.Erc20ABI.Methods[abi.Erc20BalanceOfMethod].Outputs.Pack(s.balances[target])
		require.NoError(t, err)
		return out
	case string(sel) == string(alphixHookABI.Methods["getAmountInYieldSource"].ID):
		args, err := alphixHookABI.Methods["getAmountInYieldSource"].Inputs.Unpack(data[4:])
		require.NoError(t, err)
		out, err := alphixHookABI.Methods["getAmountInYieldSource"].Outputs.Pack(s.yield[args[0].(common.Address)])
		require.NoError(t, err)
		return out
	case string(sel) == string(alphixHookABI.Methods["getFee"].ID):
		out, err := alphixHookABI.Methods["getFee"].Outputs.Pack(s.fee)
		require.NoError(t, err)
		return out
	case string(sel) == string(alphixHookABI.Methods["getReHypothecationConfig"].ID):
		out, err := alphixHookABI.Methods["getReHypothecationConfig"].Outputs.Pack(struct {
			TickLower *big.Int
			TickUpper *big.Int
		}{big.NewInt(s.tickLower), big.NewInt(s.tickUpper)})
		require.NoError(t, err)
		return out
	}
	t.Errorf("unexpected call to %s selector %x", target, sel)
	return nil
}

// fakeMulticallNode serves eth_call to Multicall3.aggregate from state, offline.
func fakeMulticallNode(t *testing.T, state *fakeHookState) string {
	aggregate := abi.Multicall3ABI.Methods["aggregate"]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Method != "eth_call" {
			t.Errorf("unexpected request %s: %v", req.Method, err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var msg struct {
			Data  hexutil.Bytes `json:"data"`
			Input hexutil.Bytes `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := msg.Input
		if len(data) == 0 {
			data = msg.Data
		}
		args, err := aggregate.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := args[0].([]struct {
			Target   common.Address `json:"target"`
			CallData []byte         `json:"callData"`
		})
		returns := make([][]byte, len(calls))
		for i, c := range calls {
			returns[i] = state.answer(t, c.Target, c.CallData)
		}
		out, err := aggregate.Outputs.Pack(big.NewInt(1), returns)
		require.NoError(t, err)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Bytes(out)})
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

var (
	testToken0 = common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831")
	testToken1 = common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9")
)

func newFakeParam(t *testing.T, state *fakeHookState, extra string) *uniswapv4.HookParam {
	rpcClient := ethrpc.New(fakeMulticallNode(t, state)).SetMulticallContract(multicall3)
	return &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: 42161, Multicall3Address: multicall3.Hex()},
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[1],
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: strings.ToLower(testToken0.Hex())},
				{Address: strings.ToLower(testToken1.Hex())},
			},
			Extra: extra,
		},
	}
}

func defaultFakeState() *fakeHookState {
	return &fakeHookState{
		balances:  map[common.Address]*big.Int{testToken0: big.NewInt(700), testToken1: big.NewInt(5_000)},
		yield:     map[common.Address]*big.Int{testToken0: big.NewInt(1_000), testToken1: big.NewInt(3_000)},
		fee:       500,
		tickLower: -8,
		tickUpper: 8,
	}
}

// With no ticks, the reserves are min(yield, poolManager balance) per token, and GetReserves must not write into
// shared values (the tick estimate used to hand back big256.U0 for both tokens, which Add then overwrote).
func TestGetReserves_NoTicks_Offline(t *testing.T) {
	t.Cleanup(func() { big256.U0.Clear() })
	param := newFakeParam(t, defaultFakeState(), `{"sqrtPriceX96":79228162514264337593543950336}`)
	hook, _ := uniswapv4.GetHook(HookAddresses[1], param)

	reserves, err := hook.GetReserves(t.Context(), param)
	require.NoError(t, err)
	assert.Equal(t, entity.PoolReserves{"700", "3000"}, reserves)
	assert.True(t, big256.U0.IsZero(), "big256.U0 was written: %s", big256.U0)

	h := hook.(*Hook)
	assert.Equal(t, uniswapv4.FeeAmount(500), h.SwapFee)
	assert.Equal(t, -8, h.TickLower)
	assert.Equal(t, 8, h.TickUpper)
}

// With ticks, the reserves are the tick estimate plus the JIT amounts: the output the fix must keep.
func TestGetReserves_WithTicks_Offline(t *testing.T) {
	extra := `{"sqrtPriceX96":"79228162514264337593543950336","ticks":[{"index":-60,"liquidityGross":"1000000000000","liquidityNet":"1000000000000"},{"index":60,"liquidityGross":"1000000000000","liquidityNet":"-1000000000000"}]}`
	param := newFakeParam(t, defaultFakeState(), extra)
	hook, _ := uniswapv4.GetHook(HookAddresses[1], param)
	h := hook.(*Hook)
	est0, est1 := uniswapv4.EstimateReservesFromTicksU256(h.SqrtPriceX96, h.Ticks)
	require.False(t, est0.IsZero())
	want := entity.PoolReserves{
		new(uint256.Int).AddUint64(est0, 700).String(),
		new(uint256.Int).AddUint64(est1, 3000).String(),
	}

	reserves, err := hook.GetReserves(t.Context(), param)
	require.NoError(t, err)
	assert.Equal(t, want, reserves)
}

// Concurrent GetReserves/Track on separate hooks (the pool tracker runs pools in parallel) must not race.
func TestGetReserves_Concurrent_Offline(t *testing.T) {
	t.Cleanup(func() { big256.U0.Clear() })
	const workers = 8
	params := make([]*uniswapv4.HookParam, workers)
	hooks := make([]uniswapv4.Hook, workers)
	for i := range params {
		params[i] = newFakeParam(t, defaultFakeState(), `{"sqrtPriceX96":79228162514264337593543950336}`)
		hooks[i], _ = uniswapv4.GetHook(HookAddresses[1], params[i])
	}
	var wg sync.WaitGroup
	results := make([]entity.PoolReserves, workers)
	errs := make([]error, workers)
	for i := range hooks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for range 20 {
				if results[i], errs[i] = hooks[i].GetReserves(t.Context(), params[i]); errs[i] != nil {
					return
				}
				if _, errs[i] = hooks[i].Track(t.Context(), params[i]); errs[i] != nil {
					return
				}
			}
		}(i)
	}
	wg.Wait()
	for i := range hooks {
		require.NoError(t, errs[i])
		assert.Equal(t, entity.PoolReserves{"700", "3000"}, results[i])
	}
	assert.True(t, big256.U0.IsZero(), "big256.U0 was written: %s", big256.U0)
}
