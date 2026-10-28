package liquiditybookv21

import (
	"encoding/json"
	"slices"
	"testing"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// trackerFixture is a pair across one block that emptied its active bin, recorded from the chain:
// its bins at the block before, as the tracker stores them; its logs in the block; what the tracker
// reads from it at the block; and its active bin, non-empty bins and quotes after the block.
type trackerFixture struct {
	Chain          string            `json:"chain"`
	Pair           string            `json:"pair"`
	Block          uint64            `json:"block"`
	BlockTimestamp uint64            `json:"blockTimestamp"`
	Txs            []string          `json:"txs"`
	Pool           entity.Pool       `json:"pool"`
	Logs           []ethtypes.Log    `json:"logs"`
	Calls          map[string]string `json:"calls"`
	ActiveBinID    uint32            `json:"activeBinId"`
	Bins           []Bin             `json:"bins"`
	Quotes         []onchainQuote    `json:"quotes"`
}

func binStrings(bins []Bin) []string {
	s := make([]string, len(bins))
	for i, b := range bins {
		s[i] = binString(b.ID, b.ReserveX, b.ReserveY)
	}
	return s
}

// TestTrackerKeepsEmptyActiveBinOnchain: each fixture is a block in which a withdrawal emptied a
// pair's active bin and left liquidity on one side of it. The tracker takes the block's logs and
// reads the pair at that block from what the chain returned there. It must store the pair's
// non-empty bins and its empty active bin, and the simulator built from what it stores must quote
// as the pair's getSwapOut and getSwapIn did at that block. A set without the active bin cannot be
// quoted either way.
func TestTrackerKeepsEmptyActiveBinOnchain(t *testing.T) {
	for name, fixture := range loadFixtures[trackerFixture](t, "testdata/tracker/*.json") {
		t.Run(name, func(t *testing.T) {
			rpc := testutil.NewMulticallStub(t, testutil.RecordedCalls(t, fixture.Calls)).
				At(fixture.Block, fixture.BlockTimestamp)
			tracker := NewPoolTracker(&Config{}, rpc.Client(), nil)
			got, err := tracker.GetNewPoolState(t.Context(), fixture.Pool,
				pool.GetNewPoolStateParams{Logs: fixture.Logs})
			require.NoError(t, err)

			var extra Extra
			require.NoError(t, json.Unmarshal([]byte(got.Extra), &extra))
			require.Equal(t, fixture.ActiveBinID, extra.ActiveBinID)
			assert.Equal(t, binStrings(fixture.Bins), binStrings(filterEmptyAndSortBins(extra.Bins)),
				"the pair's non-empty bins")
			assert.True(t, slices.ContainsFunc(extra.Bins, func(b Bin) bool { return b.ID == extra.ActiveBinID }),
				"the empty active bin %d is stored", extra.ActiveBinID)

			// The tracker stamps the state with the time it reads it; the pair quoted at the block's.
			extra.RpcBlockTimestamp = fixture.BlockTimestamp
			extraBytes, err := json.Marshal(extra)
			require.NoError(t, err)
			got.Extra = string(extraBytes)
			sim, err := NewPoolSimulator(got)
			require.NoError(t, err)

			assertQuotes(t, sim, fixture.Quotes)
		})
	}
}
