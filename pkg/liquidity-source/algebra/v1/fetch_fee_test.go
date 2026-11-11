package algebrav1

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// TestFetchRPCDataFeeAtBlock: each fixture in testdata/fetch holds the RPC calls the tracker makes for a pool at one
// block, recorded from the chain. The fee the tracker computes must be the one a swap paid in that block, which
// testdata/fee holds. A call the tracker makes at another block, the latest one, is not in the recording.
func TestFetchRPCDataFeeAtBlock(t *testing.T) {
	files, err := filepath.Glob("testdata/fetch/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			parts := strings.Split(strings.TrimSuffix(filepath.Base(file), ".json"), "-")
			chain, addr := parts[0], parts[1]
			block, err := strconv.ParseUint(parts[2], 10, 64)
			require.NoError(t, err)

			fx := readJSON[feeFixture](t, filepath.Join("testdata/fee", chain+"-"+addr+".json"))
			require.Equal(t, fx.Block, block)

			tracker := NewPoolTracker(&Config{}, ethrpc.New(replayRPC(t, file)).SetMulticallContract(
				common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")), nil)
			res, err := tracker.FetchRPCData(t.Context(), &entity.Pool{Address: addr}, block)
			require.NoError(t, err)
			assert.Equal(t, fx.SwapFee, res.State.FeeZto)
			assert.Equal(t, fx.SwapFee, res.State.FeeOtz)
		})
	}
}
