package bin

import (
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// fallbackChain is a node whose lens read, when noProgress, comes back with no bin read and its
// cursor where it started, as one under a gas cap too low for the pool does, and otherwise fails as
// a node behind does; every other call fails too. subgraphAsked tells whether the bootstrap without
// the lens ran.
func fallbackChain(t *testing.T, noProgress bool) (node *testutil.NodeStub, subgraph *httptest.Server,
	subgraphAsked *atomic.Bool) {
	node = testutil.NewNodeStub(t, 56, 100, func(c testutil.NodeCall) ([]byte, error) {
		if c.To != nil || !noProgress {
			return nil, errors.New("header not found")
		}
		out, err := lens.PackResult(big.NewInt(8388608), big.NewInt(100), big.NewInt(0), false,
			[]struct{ Id, ReserveX, ReserveY *big.Int }{})
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
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
		tr := NewPoolTracker(&Config{ChainID: 56, UseLens: true,
			BinPoolManagerAddress: "0xC697d2898e0D09264376196696c51D7aBbbAA4a9"}, node.Client(), graphqlpkg.NewClient(subgraph.URL))
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
