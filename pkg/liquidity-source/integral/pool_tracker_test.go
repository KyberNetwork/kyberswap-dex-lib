package integral

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
		DexID:          DexTypeIntegral,
		RelayerAddress: "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E",
		PoolPagingSize: 20,
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
		isFirstRun:   true,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, err := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0x2fe16Dd18bba26e457B7dD2080d5674312b026a2",
		Tokens: []*entity.PoolToken{
			{
				Address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
			},
			{
				Address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			},
		},
	}, pool.GetNewPoolStateParams{})
	if err != nil {
		panic(err)
	}

	var pair IntegralPair
	if err := json.Unmarshal([]byte(pool.Extra), &pair); err != nil {
		require.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}

	require.NotNil(ts.Suite.T(), pair)

	require.NotEqual(ts.Suite.T(), number.Zero, pair.SpotPrice)
	require.NotEqual(ts.Suite.T(), number.Zero, pair.AveragePrice)
	require.NotEqual(ts.Suite.T(), 0, pair.X_Decimals)
	require.NotEqual(ts.Suite.T(), 0, pair.Y_Decimals)

	require.Equal(ts.Suite.T(), 2, len(pool.Reserves))
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[0])
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[1])

	require.Equal(ts.Suite.T(), 2, len(pool.Tokens))
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[0])
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[1])
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
