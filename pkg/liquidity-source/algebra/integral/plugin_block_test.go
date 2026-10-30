package integral

import (
	"bytes"
	"slices"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

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
