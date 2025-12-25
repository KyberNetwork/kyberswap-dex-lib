package nabla

import (
	"context"
	"testing"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
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
		DexId:           DexType,
		Portal:          "0x1F917Fe724F186a1fFA7744A73afed18C335b9eC",
		Oracle:          "0x6d6190Da8fD73E0C911929DED2D6B47cE066e441",
		PythAdapterV2:   "0x9B5a425a9F4b4411D42B21caacf86d026dce43Ec",
		SkipPriceUpdate: false,
		PriceAPI:        "https://antenna.nabla.fi/v1/updates/price/latest",
		PriceTimeout:    durationjson.Duration{Duration: 10 * time.Second},
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

	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
