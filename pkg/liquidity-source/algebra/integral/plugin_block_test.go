package integral

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// TestPluginReadAtThePinnedBlock: FetchRPCData pinned to block 256 on a node whose multicall
// reports 257, as Etherlink's does, reads the plugin at 256 too.
func TestPluginReadAtThePinnedBlock(t *testing.T) {
	mc := lo.Must(abi.JSON(strings.NewReader(multicallAggregateABI)))
	var mu sync.Mutex
	blocks := map[string][]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var msg struct {
			Input hexutil.Bytes `json:"input"`
			Data  hexutil.Bytes `json:"data"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		data := append(msg.Data, msg.Input...)
		m := lo.Must(mc.MethodById(data[:4]))
		var tag string
		_ = json.Unmarshal(req.Params[1], &tag)
		mu.Lock()
		blocks[m.Name] = append(blocks[m.Name], tag)
		mu.Unlock()
		args := lo.Must(m.Inputs.Unpack(data[4:]))
		var out []byte
		if m.Name == "aggregate" { // the pool's reads: zeros, but a plugin address last
			calls := args[0].([]struct {
				Target   common.Address `json:"target"`
				CallData []byte         `json:"callData"`
			})
			replies := make([][]byte, len(calls))
			for i := range replies {
				replies[i] = make([]byte, 6*32)
			}
			replies[len(replies)-1] = common.LeftPadBytes(common.HexToAddress("0xabc").Bytes(), 32)
			out = lo.Must(m.Outputs.Pack(big.NewInt(257), replies))
		} else { // the plugin's settings: every call fails
			calls := args[1].([]struct {
				Target   common.Address `json:"target"`
				CallData []byte         `json:"callData"`
			})
			results := make([]callResult, len(calls))
			out = lo.Must(m.Outputs.Pack(results))
		}
		res, _ := json.Marshal(hexutil.Bytes(out))
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, res)
	}))
	defer srv.Close()

	client := ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr := NewPoolTracker(&Config{}, client, nil)
	_, err := tr.FetchRPCData(t.Context(), &entity.Pool{Address: "0x1"}, 256, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"0x100"}, blocks["tryAggregate"], "plugin read at another block than the pool")
}
