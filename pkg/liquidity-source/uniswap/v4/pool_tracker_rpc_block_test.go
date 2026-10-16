package uniswapv4

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// On Arbitrum-family chains, EVM BLOCKNUMBER (including Multicall.aggregate's
// first return value) can be a parent-chain height. JSON-RPC eth_call's block
// selector must use the rollup's RPC height instead. Deliberately make them
// different so Ethereum-only fixtures cannot hide the bug.
func TestFetchOnchainState_UsesRPCBlockHeight(t *testing.T) {
	const rpcHead, evmBlock = uint64(90327817), uint64(26207189)
	for _, tc := range []struct {
		name          string
		requested     uint64
		want          uint64
		wantHeadCalls int32
		withOverrides bool
	}{
		{"latest resolves RPC height", 0, rpcHead, 1, false},
		{"explicit RPC height retained", rpcHead - 10, rpcHead - 10, 0, false},
		{"overrides retain RPC height", 0, rpcHead, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			uintT, err := abi.NewType("uint256", "", nil)
			require.NoError(t, err)
			bytesT, err := abi.NewType("bytes[]", "", nil)
			require.NoError(t, err)
			liquidity, err := stateViewABI.Methods["getLiquidity"].Outputs.Pack(big.NewInt(1e18))
			require.NoError(t, err)
			sqrt := new(big.Int).Lsh(big.NewInt(1), 96)
			slot, err := stateViewABI.Methods["getSlot0"].Outputs.Pack(sqrt, big.NewInt(0), uint32(0), uint32(3000))
			require.NoError(t, err)
			output, err := (abi.Arguments{{Type: uintT}, {Type: bytesT}}).Pack(new(big.Int).SetUint64(evmBlock), [][]byte{liquidity, slot})
			require.NoError(t, err)
			var headCalls atomic.Int32
			var callBlock atomic.Value
			var callOverrides atomic.Value
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				var result string
				switch req.Method {
				case "eth_blockNumber":
					headCalls.Add(1)
					result = hexutil.EncodeUint64(rpcHead)
				case "eth_call":
					var block string
					if len(req.Params) > 1 {
						_ = json.Unmarshal(req.Params[1], &block)
					}
					callBlock.Store(block)
					if len(req.Params) > 2 {
						callOverrides.Store(string(req.Params[2]))
					}
					result = hexutil.Encode(output)
				default:
					http.Error(w, "unexpected method", http.StatusBadRequest)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer server.Close()
			tracker := NewPoolTracker(&Config{ChainID: 4663}, ethrpc.New(server.URL), nil)
			p := &entity.Pool{Address: "0x1", StaticExtra: `{"fee":3000,"tS":60}`, Extra: `{}`}
			var overrides map[common.Address]gethclient.OverrideAccount
			if tc.withOverrides {
				overrides = map[common.Address]gethclient.OverrideAccount{
					common.HexToAddress("0x1"): {Nonce: 7},
				}
			}
			state, _, param, err := tracker.fetchOnchainState(context.Background(), p, tc.requested, overrides)
			require.NoError(t, err)
			require.NotNil(t, state)
			require.Equal(t, tc.want, param.BlockNumber.Uint64(), "dependent hook reads must use the RPC height, never EVM BLOCKNUMBER")
			require.Equal(t, hexutil.EncodeUint64(tc.want), callBlock.Load(), "base and hook reads must share one RPC block")
			require.Equal(t, tc.wantHeadCalls, headCalls.Load())
			require.Equal(t, overrides, param.Overrides)
			if tc.withOverrides {
				require.JSONEq(t, `{"0x0000000000000000000000000000000000000001":{"nonce":"0x7"}}`, callOverrides.Load().(string))
			} else {
				require.Nil(t, callOverrides.Load())
			}
		})
	}
}

func TestFetchOnchainState_RPCHeadFailure(t *testing.T) {
	var stateCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Method != "eth_blockNumber" {
			stateCalls.Add(1)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]any{"code": -32000, "message": "RPC head unavailable"}})
	}))
	defer server.Close()
	tracker := NewPoolTracker(&Config{ChainID: 4663}, ethrpc.New(server.URL), nil)
	p := &entity.Pool{Address: "0x1", StaticExtra: `{"fee":3000,"tS":60}`, Extra: `{}`}
	_, _, param, err := tracker.fetchOnchainState(context.Background(), p, 0, nil)
	require.ErrorContains(t, err, "RPC head unavailable")
	require.Zero(t, stateCalls.Load(), "do not fetch unpinned state after RPC height resolution fails")
	require.Nil(t, param.BlockNumber)
}
