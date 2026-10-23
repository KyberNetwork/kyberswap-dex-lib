package liquiditybookv21

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

// lensState is a pair at one block.
type lensState struct {
	active uint32
	rx, ry int64
	bins   map[uint32][2]int64
}

// lensChain serves a pair whose state is states[block]: multicalls of its views, and the lens.
// It records the block each lens call was pinned to.
func lensChain(t *testing.T, head uint64, states map[uint64]lensState) (*httptest.Server, func() []string) {
	var doc struct {
		ABI json.RawMessage `json:"abi"`
	}
	require.NoError(t, json.Unmarshal(lensJSON, &doc))
	lensABI, err := abi.JSON(bytes.NewReader(doc.ABI))
	require.NoError(t, err)

	var mu sync.Mutex
	var lensBlocks []string
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
			reply("result", hexutil.Uint64(43114))
			return
		case "eth_blockNumber":
			reply("result", hexutil.Uint64(head))
			return
		case "eth_call":
		default:
			t.Errorf("unexpected %s", req.Method)
			return
		}
		var msg struct {
			To    *common.Address `json:"to"`
			Data  hexutil.Bytes   `json:"data"`
			Input hexutil.Bytes   `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := msg.Data
		if len(data) == 0 {
			data = msg.Input
		}
		var blockArg string
		require.NoError(t, json.Unmarshal(req.Params[1], &blockArg))
		block := head
		if blockArg != "latest" {
			block, err = hexutil.DecodeUint64(blockArg)
			require.NoError(t, err)
		}
		st, ok := states[block]
		if msg.To == nil {
			if len(data) < 64 { // the gas cap probe
				reply("error", map[string]any{"code": -32000, "message": "probe refused"})
				return
			}
			mu.Lock()
			lensBlocks = append(lensBlocks, blockArg)
			mu.Unlock()
			if !ok {
				reply("error", map[string]any{"code": -32000, "message": "header not found"})
				return
			}
			var bins []struct{ Id, ReserveX, ReserveY *big.Int }
			for _, id := range slices.Sorted(maps.Keys(st.bins)) {
				if b := st.bins[id]; b[0] != 0 || b[1] != 0 {
					bins = append(bins, struct{ Id, ReserveX, ReserveY *big.Int }{
						big.NewInt(int64(id)), big.NewInt(b[0]), big.NewInt(b[1])})
				}
			}
			out, err := lensABI.Methods["getSnapshot"].Outputs.Pack(big.NewInt(int64(st.active)), big.NewInt(st.rx),
				big.NewInt(st.ry), new(big.Int).SetUint64(block), big.NewInt(0), true, bins)
			require.NoError(t, err)
			reply("error", map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Bytes(out)})
			return
		}
		if !ok {
			reply("error", map[string]any{"code": -32000, "message": "header not found"})
			return
		}
		if !bytes.Equal(data[:4], aggregateABI.ID) { // a direct call: getPriceFromId
			reply("result", hexutil.Bytes(make([]byte, 32)))
			return
		}
		args, err := aggregateABI.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := reflect.ValueOf(args[0])
		results := make([][]byte, calls.Len())
		for i := range results {
			cd := calls.Index(i).FieldByName("CallData").Bytes()
			m, err := pairABI.MethodById(cd[:4])
			require.NoError(t, err)
			switch m.Name {
			case pairMethodGetReserves:
				results[i], err = m.Outputs.Pack(big.NewInt(st.rx), big.NewInt(st.ry))
			case pairMethodGetActiveID:
				results[i], err = m.Outputs.Pack(big.NewInt(int64(st.active)))
			case pairMethodGetBinStep:
				results[i], err = m.Outputs.Pack(uint16(25))
			case pairMethodGetBin:
				in, err2 := m.Inputs.Unpack(cd[4:])
				require.NoError(t, err2)
				b := st.bins[uint32(in[0].(*big.Int).Uint64())]
				results[i], err = m.Outputs.Pack(big.NewInt(b[0]), big.NewInt(b[1]))
			default:
				results[i] = make([]byte, 32*len(m.Outputs))
			}
			require.NoError(t, err)
		}
		out, err := aggregateABI.Outputs.Pack(new(big.Int).SetUint64(block), results)
		require.NoError(t, err)
		reply("result", hexutil.Bytes(out))
	}))
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), lensBlocks...)
	}
}
