package integral

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
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

// TestPluginReadAtThePinnedBlock: FetchRPCData pinned to block 256 on a node whose multicall
// reports 257, as Etherlink's does, reads the plugin at 256 too.
func TestPluginReadAtThePinnedBlock(t *testing.T) {
	var mu sync.Mutex
	var pluginTags []string // the block of each read of the plugin's settings
	pluginAddr, plugin := common.HexToAddress("0xabc"), poolV12ABI.Methods[poolPluginMethod].ID
	node := testutil.NewNodeStub(t, 1, 300, func(c testutil.NodeCall) ([]byte, error) {
		require.NotNil(t, c.To, "a lens call")
		// The pool's reads: zeros, but its plugin's address. The plugin's settings: every call fails.
		ret, targets, err := testutil.Multicall(*c.To, c.Data, 257, 0, func(target common.Address, callData []byte) (bool, []byte) {
			if bytes.HasPrefix(callData, plugin) {
				return true, common.LeftPadBytes(pluginAddr.Bytes(), 32)
			}
			return target != pluginAddr, make([]byte, 6*32)
		})
		if slices.Contains(targets, pluginAddr) {
			mu.Lock()
			pluginTags = append(pluginTags, c.Tag)
			mu.Unlock()
		}
		return ret, err
	})

	tr := NewPoolTracker(&Config{}, node.Client(), nil)
	_, err := tr.FetchRPCData(t.Context(), &entity.Pool{Address: "0x1"}, 256, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"0x100"}, pluginTags, "plugin read at another block than the pool")
}
