package bin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

// TestLensBootstrapFallsBackWithoutTheLens: a bootstrap whose lens read fails, as one paging across
// blocks does on a node that ignores a pinned block, goes on without the lens rather than leaving
// the pool unsaved.
func TestLensBootstrapFallsBackWithoutTheLens(t *testing.T) {
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &req))
		switch req.Method {
		case "eth_chainId":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, hexutil.Uint64(56).String())
		default:
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"header not found"}}`, req.ID)
		}
	}))
	defer node.Close()
	var subgraphAsked atomic.Bool
	subgraph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		subgraphAsked.Store(true)
		_, _ = w.Write([]byte(`{"data":{"lbpair":null}}`))
	}))
	defer subgraph.Close()

	client := ethrpc.New(node.URL).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true,
		BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, client, graphqlpkg.NewClient(subgraph.URL))
	_, _ = tr.BootstrapPoolState(context.Background(), entity.Pool{
		Address:     "0x" + strings.Repeat("ab", 32),
		Tokens:      []*entity.PoolToken{{Address: "0xa"}, {Address: "0xb"}},
		StaticExtra: `{"fee":3000}`,
	}, poolpkg.GetNewPoolStateParams{})
	assert.True(t, subgraphAsked.Load(), "the bootstrap without the lens ran")
}
