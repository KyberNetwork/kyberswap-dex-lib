package obric

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://bsc.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := Config{
		DexId:        DexType,
		Factory:      "0x749837Fd609232941920a826eb7997C9c4bF4120",
		NewPoolLimit: 100,
	}

	ts.updater = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	ctx := ts.T().Context()
	var metadataBytes []byte
	for {
		pools, newMetadataBytes, err := ts.updater.GetNewPools(ctx, metadataBytes)
		require.NoError(ts.T(), err)

		if len(pools) == 0 {
			break
		}

		for _, p := range pools {
			newPoolState, err := ts.tracker.GetNewPoolState(ctx, p, pool.GetNewPoolStateParams{})
			require.NoError(ts.T(), err)
			require.NotNil(ts.T(), newPoolState)
		}

		metadataBytes = newMetadataBytes
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
