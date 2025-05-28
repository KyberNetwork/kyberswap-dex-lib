package usd0pp

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	client  *ethrpc.Client
	tracker PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	// Setup RPC server
	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	ts.client = rpcClient

	config := Config{
		DexID: DexType,
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, err := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0x35d8949372d46b7a3d5a56006ae77b215fc69bc0",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x73A15FeD60Bf67631dC6cd7Bc5B6e8da8190aCF5",
			},
			{
				Address: "0x35d8949372d46b7a3d5a56006ae77b215fc69bc0",
			},
		},
		Reserves: []string{defaultReserves, defaultReserves},
		Extra:    `{"Paused":false,"endTime":1844335800,"startTime":1718105400}`,
	}, pool.GetNewPoolStateParams{})
	if err != nil {
		panic(err)
	}

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(pool.Extra), &poolExtra); err != nil {
		require.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}

	require.NotNil(ts.Suite.T(), pool)
	require.NotNil(ts.Suite.T(), poolExtra)

	require.Equal(ts.Suite.T(), false, poolExtra.Paused)
	require.Equal(ts.Suite.T(), int64(1844335800), poolExtra.EndTime)
	require.Equal(ts.Suite.T(), int64(1718105400), poolExtra.StartTime)

	require.Equal(ts.Suite.T(), 2, len(pool.Reserves))
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[0])
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[1])

	require.Equal(ts.Suite.T(), 2, len(pool.Tokens))
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[0])
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[1])
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
