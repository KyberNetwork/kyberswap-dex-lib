package integral

import (
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
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var aggregateABI = lo.Must(abi.JSON(strings.NewReader(multicallAggregateABI))).Methods["aggregate"]

// zeroChain is a node at block head whose multicall reports block multicallBlock, as Etherlink's
// reports the block after the one a call is pinned to, and answers every read with zeros.
func zeroChain(t *testing.T, head, multicallBlock uint64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.Unmarshal(body, &req))
		var result any
		switch req.Method {
		case "eth_blockNumber":
			result = hexutil.Uint64(head)
		case "eth_call":
			var msg struct {
				Input hexutil.Bytes `json:"input"`
				Data  hexutil.Bytes `json:"data"`
			}
			require.NoError(t, json.Unmarshal(req.Params[0], &msg))
			data := append(msg.Data, msg.Input...)
			args, err := aggregateABI.Inputs.Unpack(data[4:])
			require.NoError(t, err)
			calls := args[0].([]struct {
				Target   common.Address `json:"target"`
				CallData []byte         `json:"callData"`
			})
			replies := make([][]byte, len(calls))
			for i := range replies {
				replies[i] = make([]byte, 6*32)
			}
			out, err := aggregateABI.Outputs.Pack(new(big.Int).SetUint64(multicallBlock), replies)
			require.NoError(t, err)
			result = hexutil.Bytes(out)
		default:
			t.Fatalf("unexpected %s", req.Method)
		}
		res, _ := json.Marshal(result)
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, res)
	}))
}

// TestBootstrapIsDatedByTheBlockItRead: a pool bootstrapped without the lens is dated by the block
// its state was read at, not left at an older one, such as its creation block, at which the caller
// would read its ticks again and drop those created since; and that block is the one the read was
// pinned to, not the one a multicall reports.
func TestBootstrapIsDatedByTheBlockItRead(t *testing.T) {
	chain := zeroChain(t, 256, 257)
	defer chain.Close()
	subgraph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"pool":null}}`)
	}))
	defer subgraph.Close()

	client := ethrpc.New(chain.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr := NewPoolTracker(&Config{}, client, graphqlpkg.NewClient(subgraph.URL))

	p, err := tr.BootstrapPoolState(t.Context(), entity.Pool{Address: "0x1", BlockNumber: 50},
		sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.EqualValues(t, 256, p.BlockNumber)
}
