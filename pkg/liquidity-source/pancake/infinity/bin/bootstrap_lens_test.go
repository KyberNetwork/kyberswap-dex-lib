package bin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

var aggregateABI = lo.Must(abi.JSON(strings.NewReader(`[{"name":"aggregate","type":"function",
"stateMutability":"view","inputs":[{"name":"calls","type":"tuple[]","components":[{"name":"target",
"type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber",
"type":"uint256"},{"name":"returnData","type":"bytes[]"}]}]`))).Methods["aggregate"]

// binChain is a node at block 100 holding one bin pool whose active bin is activeID and holds
// reserves: the lens reads that bin, and a multicall reads the pool's slot0.
func binChain(t *testing.T, activeID uint32) *httptest.Server {
	var doc struct {
		ABI json.RawMessage `json:"abi"`
	}
	require.NoError(t, json.Unmarshal(lensJSON, &doc))
	parsed, err := abi.JSON(bytes.NewReader(doc.ABI))
	require.NoError(t, err)

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
		id := new(big.Int).SetUint64(uint64(activeID))
		switch req.Method {
		case "eth_chainId":
			reply("result", hexutil.Uint64(56))
		case "eth_blockNumber":
			reply("result", hexutil.Uint64(100))
		case "eth_call":
			var msg struct {
				To *common.Address `json:"to"`
			}
			require.NoError(t, json.Unmarshal(req.Params[0], &msg))
			if msg.To == nil {
				bins := []struct{ Id, ReserveX, ReserveY *big.Int }{{id, big.NewInt(5), big.NewInt(7)}}
				out, err := parsed.Methods["getSnapshot"].Outputs.Pack(id, big.NewInt(100), big.NewInt(0), true, bins)
				require.NoError(t, err)
				reply("error", map[string]any{"code": 3, "message": "execution reverted", "data": hexutil.Bytes(out)})
				return
			}
			slot0 := append(common.LeftPadBytes(id.Bytes(), 32), make([]byte, 64)...)
			out, err := aggregateABI.Outputs.Pack(big.NewInt(100), [][]byte{slot0})
			require.NoError(t, err)
			reply("result", hexutil.Bytes(out))
		default:
			t.Errorf("unexpected %s", req.Method)
		}
	}))
}

// TestLensBootstrapIsTimestamped: a pool given its state from the lens at the batch's head is
// stamped with the time it was read, as every other read of it is.
func TestLensBootstrapIsTimestamped(t *testing.T) {
	const activeID = 8388608
	chain := binChain(t, activeID)
	defer chain.Close()
	client := ethrpc.New(chain.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true,
		BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, client, nil)

	p := entity.Pool{
		Address:     "0x" + strings.Repeat("ab", 32),
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"fee":3000}`,
	}
	got, err := tr.GetNewPoolState(t.Context(), p, poolpkg.GetNewPoolStateParams{
		BlockHeaders: map[uint64]entity.BlockHeader{100: {}},
	})
	require.NoError(t, err)
	require.True(t, hasState(got), "the lens did not give the pool its state")
	assert.EqualValues(t, 100, got.BlockNumber)
	assert.NotZero(t, got.Timestamp)
}
