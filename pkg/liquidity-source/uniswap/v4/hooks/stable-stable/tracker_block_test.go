package stablestable

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// The registered Robinhood deployment uses BlockNumberish/ArbSys, not the EVM
// NUMBER opcode, for feeState.blockNumber. See the deployment verification in
// #1669 and Uniswap/v4-hooks-public@e4eabe52/src/stable/StablePairHook.sol.
func TestTrack_RobinhoodFeeClockUsesRPCHeight(t *testing.T) {
	const rpcBlock, parentBlock = uint64(90327817), uint64(26207189)
	ref := new(big.Int).Lsh(big.NewInt(1), 96)
	cached := new(big.Int).Div(new(big.Int).Mul(ref, big.NewInt(1001)), big.NewInt(1000))
	current := new(big.Int).Div(new(big.Int).Mul(ref, big.NewInt(1002)), big.NewInt(1000))
	config, err := stableStableHookV2ABI.Methods["feeConfig"].Outputs.Pack(big.NewInt(100000), big.NewInt(7), uint8(50), ref)
	require.NoError(t, err)
	state, err := stableStableHookABI.Methods["feeState"].Outputs.Pack(big.NewInt(500000000), cached, new(big.Int).SetUint64(rpcBlock))
	require.NoError(t, err)
	uintType, err := abi.NewType("uint256", "", nil)
	require.NoError(t, err)
	bytesType, err := abi.NewType("bytes[]", "", nil)
	require.NoError(t, err)
	// Standard Multicall3 reports the parent height, but the hook's clock is L2.
	output, err := (abi.Arguments{{Type: uintType}, {Type: bytesType}}).Pack(new(big.Int).SetUint64(parentBlock), [][]byte{config, state})
	require.NoError(t, err)
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
		var block string
		if len(req.Params) < 2 || json.Unmarshal(req.Params[1], &block) != nil || req.Method != "eth_call" || block != hexutil.EncodeUint64(rpcBlock) {
			http.Error(w, "hook state must be pinned to the RPC height", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": hexutil.Encode(output)})
	}))
	defer server.Close()
	param := &uniswapv4.HookParam{
		RpcClient: ethrpc.New(server.URL), HookAddress: common.HexToAddress("0x3b64660a35a09afde554ce545bca9166d6a23cc0"),
		BlockNumber: new(big.Int).SetUint64(rpcBlock), Pool: &entity.Pool{Address: "0x1"},
	}
	hook, ok := uniswapv4.GetHook(param.HookAddress, param)
	require.True(t, ok)
	h := hook.(*Hook)
	_, err = h.Track(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, rpcBlock, h.TrackedBlock)
	require.Equal(t, h.BlockNumber, h.TrackedBlock, "feeState and tracking must use the same L2 clock")
	h.SqrtPriceX96, _ = uint256.FromBig(current)
	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1000)})
	require.NoError(t, err)
	require.Equal(t, uint32(500), uint32(result.SwapFee), "same-block swaps reuse the stored decaying fee")
	require.Nil(t, result.SwapInfo, "same-block swaps must not replace the cached start-of-block price")
}
