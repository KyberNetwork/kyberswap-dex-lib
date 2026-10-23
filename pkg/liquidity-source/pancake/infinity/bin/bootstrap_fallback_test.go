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
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// fallbackChain is a node whose lens read, when noProgress, comes back with no bin read and its
// cursor where it started, as one under a gas cap too low for the pool does, and otherwise fails as
// a node behind does; every other call fails too. subgraphAsked tells whether the bootstrap without
// the lens ran.
func fallbackChain(t *testing.T, noProgress bool) (node, subgraph *httptest.Server, subgraphAsked *atomic.Bool) {
	var doc struct {
		ABI json.RawMessage `json:"abi"`
	}
	require.NoError(t, json.Unmarshal(lensJSON, &doc))
	parsed, err := abi.JSON(bytes.NewReader(doc.ABI))
	require.NoError(t, err)
	node = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage   `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		if req.Method == "eth_chainId" {
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, hexutil.Uint64(56).String())
			return
		}
		var msg struct {
			To    *common.Address `json:"to"`
			Data  hexutil.Bytes   `json:"data"`
			Input hexutil.Bytes   `json:"input"`
		}
		require.NoError(t, json.Unmarshal(req.Params[0], &msg))
		if msg.To == nil && noProgress && len(msg.Data)+len(msg.Input) > 32 {
			out, err := parsed.Methods["getSnapshot"].Outputs.Pack(big.NewInt(8388608), big.NewInt(100), big.NewInt(0),
				false, []struct{ Id, ReserveX, ReserveY *big.Int }{})
			require.NoError(t, err)
			b, _ := json.Marshal(hexutil.Bytes(out))
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":3,"message":"execution reverted","data":%s}}`, req.ID, b)
			return
		}
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"header not found"}}`, req.ID)
	}))
	subgraphAsked = new(atomic.Bool)
	subgraph = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		subgraphAsked.Store(true)
		_, _ = w.Write([]byte(`{"data":{"lbpair":null}}`))
	}))
	return node, subgraph, subgraphAsked
}

// TestLensBootstrapFallsBackWhereTheLensCannotRead: a node that cannot serve the lens read at all,
// here as one that caps a call under the pool's densest step, has the pool bootstrapped without the
// lens rather than left unsaved; a failure of the node's own is the bootstrap's, to be tried again.
func TestLensBootstrapFallsBackWhereTheLensCannotRead(t *testing.T) {
	for _, noProgress := range []bool{true, false} {
		node, subgraph, subgraphAsked := fallbackChain(t, noProgress)
		client := ethrpc.New(node.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
		tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true,
			BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, client, graphqlpkg.NewClient(subgraph.URL))
		_, _ = tr.BootstrapPoolState(t.Context(), entity.Pool{
			Address:     "0x" + strings.Repeat("ab", 32),
			Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
			StaticExtra: `{"fee":3000}`,
		}, poolpkg.GetNewPoolStateParams{})
		assert.Equal(t, noProgress, subgraphAsked.Load(), "the bootstrap without the lens ran")
		node.Close()
		subgraph.Close()
	}
}
