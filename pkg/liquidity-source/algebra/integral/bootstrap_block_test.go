package integral

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// zeroChain is a node at block head whose multicall reports block multicallBlock, as Etherlink's
// reports the block after the one a call is pinned to, and answers every read with zeros.
func zeroChain(t *testing.T, head, multicallBlock uint64) *testutil.NodeStub {
	return testutil.NewNodeStub(t, 1, head, func(c testutil.NodeCall) ([]byte, error) {
		require.NotNil(t, c.To, "a lens call")
		ret, _, err := testutil.Multicall(*c.To, c.Data, multicallBlock, 0, func(common.Address, []byte) (bool, []byte) {
			return true, make([]byte, 6*32)
		})
		return ret, err
	})
}

// TestBootstrapIsDatedByTheBlockItRead: a pool bootstrapped without the lens is dated by the block
// its state was read at, not left at an older one, such as its creation block, at which the caller
// would read its ticks again and drop those created since; and that block is the one the read was
// pinned to, not the one a multicall reports.
func TestBootstrapIsDatedByTheBlockItRead(t *testing.T) {
	subgraph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":{"pool":null}}`)
	}))
	defer subgraph.Close()

	tr := NewPoolTracker(&Config{}, zeroChain(t, 256, 257).Client(), graphqlpkg.NewClient(subgraph.URL))

	p, err := tr.BootstrapPoolState(t.Context(), entity.Pool{Address: "0x1", BlockNumber: 50},
		sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.EqualValues(t, 256, p.BlockNumber)
}
