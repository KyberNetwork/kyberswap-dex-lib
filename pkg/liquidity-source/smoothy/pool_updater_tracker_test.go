package smoothy

import (
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/account"
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
	rpcClient := ethrpc.New("https://eth.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	config := Config{
		DexId: DexType,
		Pool:  common.HexToAddress("0xe5859f4EFc09027A9B718781DCb2C6910CAc6E91"),
	}

	ts.updater = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pools, _, err := ts.updater.GetNewPools(ts.T().Context(), nil)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), len(pools), 1)

	newPoolState, err := ts.tracker.GetNewPoolState(ts.T().Context(), pools[0], pool.GetNewPoolStateParams{})
	require.NoError(ts.T(), err)
	require.NotNil(ts.T(), newPoolState)

	for _, token := range newPoolState.Tokens {
		require.True(ts.T(), account.IsValidAddress(token.Address))
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
