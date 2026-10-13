package st0x

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestGetReservesFetchesWithoutTrack(t *testing.T) {
	r0, err := propAMMHookABI.Methods["getBalance"].Outputs.Pack(big.NewInt(1000))
	require.NoError(t, err)
	r1, err := propAMMHookABI.Methods["getBalance"].Outputs.Pack(big.NewInt(2000))
	require.NoError(t, err)
	u, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	b, err := abi.NewType("bytes[]", "", nil)
	require.NoError(t, err)
	ret, err := (abi.Arguments{{Type: u}, {Type: b}}).Pack(big.NewInt(100), [][]byte{r0, r1})
	require.NoError(t, err)
	var calls atomic.Int32
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Params []json.RawMessage `json:"params"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&req)) || !assert.Len(t, req.Params, 2) {
			return
		}
		assert.JSONEq(t, `"0x64"`, string(req.Params[1]))
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(ret)}))
	}))
	defer stub.Close()
	param := &uniswapv4.HookParam{
		RpcClient: ethrpc.New(stub.URL), HookAddress: hookAddress, BlockNumber: big.NewInt(100),
		Pool: &entity.Pool{Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000001"},
			{Address: "0x0000000000000000000000000000000000000002"},
		}},
	}
	// Both an uninitialized hook and an existing snapshot must fetch fresh balances.
	for _, h := range []*Hook{{}, {Reserve0: uint256.NewInt(1), Reserve1: uint256.NewInt(2)}} {
		reserves, err := h.GetReserves(context.Background(), param)
		require.NoError(t, err)
		require.Equal(t, entity.PoolReserves{"1000", "2000"}, reserves)
		require.Equal(t, "1000", h.Reserve0.Dec())
		require.Equal(t, "2000", h.Reserve1.Dec())
	}
	require.Equal(t, int32(2), calls.Load())
}

func TestHookFactoryUsesPoolReserves(t *testing.T) {
	for name, extra := range map[string]string{
		"missing": `{"p":1000000000000000000}`,
		"stale":   `{"p":1000000000000000000,"r0":"1","r1":"2"}`,
	} {
		t.Run(name, func(t *testing.T) {
			hook, ok := uniswapv4.GetHook(hookAddress, &uniswapv4.HookParam{
				Pool: &entity.Pool{Reserves: entity.PoolReserves{"1000", "2000"}}, HookExtra: uniswapv4.HookExtra(extra),
			})
			require.True(t, ok)
			h := hook.(*Hook)
			require.Equal(t, "1000", h.Reserve0.Dec())
			require.Equal(t, "2000", h.Reserve1.Dec())
			// Round-tripping HookExtra must not duplicate the reserves persisted on the pool.
			encoded, err := json.Marshal(h)
			require.NoError(t, err)
			var fields map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(encoded, &fields))
			require.NotContains(t, fields, "r0")
			require.NotContains(t, fields, "r1")
			for _, direction := range []bool{true, false} {
				for _, calcOut := range []bool{true, false} {
					result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: direction, AmountSpecified: big.NewInt(10)})
					require.NoError(t, err)
					require.Equal(t, "1000", h.Reserve0.Dec())
					require.Equal(t, "2000", h.Reserve1.Dec())
					clone := h.CloneState().(*Hook)
					clone.UpdateBalance(result.SwapInfo)
					if direction {
						require.Equal(t, "1010", clone.Reserve0.Dec())
						require.Equal(t, "1990", clone.Reserve1.Dec())
					} else {
						require.Equal(t, "990", clone.Reserve0.Dec())
						require.Equal(t, "2010", clone.Reserve1.Dec())
					}
					require.Equal(t, "1000", h.Reserve0.Dec())
					require.Equal(t, "2000", h.Reserve1.Dec())
				}
			}
			_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(2001)})
			require.ErrorIs(t, err, ErrInsufficientRsv)
		})
	}
}

func TestPoolSimulatorWithMissingHookExtraReserves(t *testing.T) {
	const token0 = "0x0000000000000000000000000000000000000001"
	const token1 = "0x0000000000000000000000000000000000000002"
	p := entity.Pool{
		Address:     "0xe26507ae709887aa01dab0e959ee4b3a8a1c025e5c29e1b2644b62ea950a4253",
		Reserves:    entity.PoolReserves{"1000", "2000"},
		Tokens:      []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 18}},
		Extra:       `{"liquidity":0,"sqrtPriceX96":79228162514264337593543950336,"tick":0,"tickSpacing":1,"ticks":[],"hX":{"p":1000000000000000000}}`,
		StaticExtra: `{"hooks":"0x9af1a97021b92c47219d2fb24deba51c248a2a88","fee":0,"tS":1}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainID(8453))
	require.NoError(t, err)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10)}, TokenOut: token1}
	for i := 0; i < 2; i++ {
		result, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(10), result.TokenAmountOut.Amount)
	}
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
