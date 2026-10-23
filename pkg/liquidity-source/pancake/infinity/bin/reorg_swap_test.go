package bin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// binNodeAt serves one bin pool through multicalls at any block, its slot0 and getBin from state, and
// no lens.
func binNodeAt(t *testing.T, active uint32, bins map[uint32][2]int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			reply("result", hexutil.Uint64(56))
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
		if msg.To == nil || !bytes.Equal(data[:4], aggregateABI.ID) {
			reply("error", map[string]any{"code": -32000, "message": "not served"})
			return
		}
		args, err := aggregateABI.Inputs.Unpack(data[4:])
		require.NoError(t, err)
		calls := reflect.ValueOf(args[0])
		results := make([][]byte, calls.Len())
		for i := range results {
			cd := calls.Index(i).FieldByName("CallData").Bytes()
			m, err := shared.BinPoolManagerABI.MethodById(cd[:4])
			require.NoError(t, err)
			switch m.Name {
			case "getSlot0":
				results[i], err = m.Outputs.Pack(active, uint32(0), uint32(0))
			case "getBin":
				in, err2 := m.Inputs.Unpack(cd[4:])
				require.NoError(t, err2)
				b := bins[uint32(in[1].(*big.Int).Uint64())]
				results[i], err = m.Outputs.Pack(big.NewInt(b[0]), big.NewInt(b[1]), new(big.Int), new(big.Int))
			default:
				t.Errorf("unexpected %s", m.Name)
			}
			require.NoError(t, err)
		}
		out, err := aggregateABI.Outputs.Pack(big.NewInt(110), results)
		require.NoError(t, err)
		reply("result", hexutil.Bytes(out))
	}))
}

// TestRemovedSwapRereadsTheBinsItCrossed: a swap at block 110 moved the active bin from 100 to 105,
// and the pool was stored as it left bins 100 to 105. A reorg drops that swap, and the chain is back
// at active bin 100 with bins 101 to 104 holding X. Its removed event names only bin 105, so the
// bins it crossed are read again as those between the stored active bin and the one read now.
func TestRemovedSwapRereadsTheBinsItCrossed(t *testing.T) {
	chain := map[uint32][2]int64{100: {10, 10}, 101: {10, 0}, 102: {10, 0}, 103: {10, 0}, 104: {10, 0}, 105: {10, 0}}
	node := binNodeAt(t, 100, chain)
	defer node.Close()
	manager := common.HexToAddress("0xC697d2898e0D09264376196696c51D7aBbbAA4a9")
	client := ethrpc.New(node.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true, BinPoolManagerAddress: manager.Hex()}, client, nil)

	poolID := common.HexToHash("0x" + strings.Repeat("ab", 32))
	swap := shared.BinPoolManagerABI.Events["Swap"]
	logData, err := swap.Inputs.NonIndexed().Pack(big.NewInt(-1), big.NewInt(1), big.NewInt(105), big.NewInt(3000),
		uint16(0))
	require.NoError(t, err)
	removed := ethtypes.Log{Address: manager, BlockNumber: 110, Removed: true,
		Topics: []common.Hash{swap.ID, poolID, {}}, Data: logData}

	p := entity.Pool{
		Address:     strings.ToLower(poolID.Hex()),
		BlockNumber: 110,
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"fee":3000}`,
		Extra: `{"protocolFee":0,"activeBinId":105,"bins":[{"id":100,"reserveX":"0","reserveY":"20"},` +
			`{"id":101,"reserveX":"0","reserveY":"10"},{"id":102,"reserveX":"0","reserveY":"10"},` +
			`{"id":103,"reserveX":"0","reserveY":"10"},{"id":104,"reserveX":"0","reserveY":"10"},` +
			`{"id":105,"reserveX":"5","reserveY":"5"}]}`,
	}
	got, err := tr.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{Logs: []ethtypes.Log{removed}})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
	assert.EqualValues(t, 100, extra.ActiveBinID)
	for _, b := range extra.Bins {
		want := chain[b.ID]
		assert.Equal(t, fmt.Sprint(want), fmt.Sprintf("[%s %s]", b.ReserveX, b.ReserveY), "bin %d", b.ID)
	}
}
