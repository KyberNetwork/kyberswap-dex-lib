package nabla

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	lister  *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://berachain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := Config{
		DexId:  DexType,
		Portal: "0x1F917Fe724F186a1fFA7744A73afed18C335b9eC",
	}

	ts.lister = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pools, _, err := ts.lister.GetNewPools(context.Background(), nil)
	require.NoError(ts.T(), err)
	require.Greater(ts.T(), len(pools), 0)

	for _, p := range pools {
		newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(ts.T(), err)

		poolBytes, err := json.Marshal(newPoolState)
		require.NoError(ts.T(), err)

		ts.T().Log(string(poolBytes))
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PoolListTrackerTestSuite))
}
