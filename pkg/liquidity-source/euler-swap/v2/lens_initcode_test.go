package v2

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
)

// maxInitCode is EIP-3860's limit, which geth's state transition applies to a to-less eth_call
// (core/vm.CheckMaxInitCodeSize) before running it: a plain error, no revert, no data.
const maxInitCode = 49152

// initCodeNode answers a lens call like a Shanghai node: an oversize one fails outright, a smaller
// one reads every pool with empty rounds.
func initCodeNode(t *testing.T, lensCalls, oversize *atomic.Int64) *httptest.Server {
	var doc struct {
		ABI      json.RawMessage `json:"abi"`
		Bytecode hexutil.Bytes   `json:"bytecode"`
	}
	require.NoError(t, json.Unmarshal(lensJSON, &doc))
	parsed, err := abi.JSON(bytes.NewReader(doc.ABI))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		reply := func(field string, v any) {
			b, _ := json.Marshal(v)
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,%q:%s}`, req.ID, field, b)
		}
		switch req.Method {
		case "eth_chainId":
			reply("result", hexutil.Uint64(1))
		case "eth_call":
			var msg struct {
				To    *common.Address `json:"to"`
				Input hexutil.Bytes   `json:"input"`
				Data  hexutil.Bytes   `json:"data"`
				Gas   hexutil.Uint64  `json:"gas"`
			}
			require.NoError(t, json.Unmarshal(req.Params[0], &msg))
			data := append(msg.Data, msg.Input...)
			if msg.To == nil && bytes.Equal(data, []byte{0x5a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}) {
				// gas cap probe: run with what it asked, capped at 50M
				g := min(uint64(msg.Gas), 50_000_000) - 53_124
				reply("result", hexutil.Bytes(common.BigToHash(new(big.Int).SetUint64(g)).Bytes()))
				return
			}
			if msg.To != nil {
				reply("error", map[string]any{"code": -32000, "message": "no multicall here"})
				return
			}
			lensCalls.Add(1)
			if len(data) > maxInitCode {
				oversize.Add(1)
				reply("error", map[string]any{"code": -32000,
					"message": fmt.Sprintf("max initcode size exceeded: code size %d limit %d", len(data), maxInitCode)})
				return
			}
			args, err := parsed.Constructor.Inputs.Unpack(data[len(doc.Bytecode):])
			require.NoError(t, err)
			part := *abi.ConvertType(args[0], new([]poolInput)).(*[]poolInput)
			out, err := parsed.Methods["getPoolStates"].Outputs.Pack(big.NewInt(100), make([]poolReads, len(part)))
			require.NoError(t, err)
			reply("error", map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Bytes(out)})
		default:
			t.Errorf("unexpected %s", req.Method)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func manyPools(n int) []entity.Pool {
	pools := make([]entity.Pool, n)
	for i := range pools {
		pools[i] = entity.Pool{
			Address: common.BigToAddress(big.NewInt(int64(0x1000 + i))).Hex(),
			Tokens:  []*entity.PoolToken{{Address: "0x00000000000000000000000000000000000000aa", Decimals: 6}, {Address: "0x00000000000000000000000000000000000000bb", Decimals: 18}},
			StaticExtra: fmt.Sprintf(`{"sv0":"%s","sv1":"%s","ea":"%s","evc":"0x0000000000000000000000000000000000000c0c"}`,
				common.BigToAddress(big.NewInt(int64(0x2000+i))).Hex(), common.BigToAddress(big.NewInt(int64(0x3000+i))).Hex(),
				common.BigToAddress(big.NewInt(int64(0x4000+i))).Hex()),
		}
	}
	return pools
}

// TestLensBatchOverInitCodeLimitIsSplit: GetNewPoolStates hands readLens every pool of a batch in
// one call. Past ~83 pools of two vaults the call data passes EIP-3860's 49152 bytes, which a node
// refuses before running it; the batch is then read in smaller parts, not given up for multicalls.
func TestLensBatchOverInitCodeLimitIsSplit(t *testing.T) {
	var lensCalls, oversize atomic.Int64
	srv := initCodeNode(t, &lensCalls, &oversize)
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, ethrpc.New(srv.URL))
	require.NoError(t, err)

	pools := manyPools(100)
	var inputs []poolInput
	for _, p := range pools {
		se, vaults, err := poolVaults(p)
		require.NoError(t, err)
		inputs = append(inputs, lensInput(p.Address, se, vaults))
	}
	full, err := lens.PackCall(inputs)
	require.NoError(t, err)
	half, err := lens.PackCall(inputs[:50])
	require.NoError(t, err)
	require.Greater(t, len(full), maxInitCode)
	require.Less(t, len(half), maxInitCode)

	_, reads, err := tr.readLens(t.Context(), inputs, nil, 0)
	assert.NoError(t, err, "a batch over the init code limit should be read in smaller parts, not given up")
	assert.Len(t, reads, len(inputs))
}
