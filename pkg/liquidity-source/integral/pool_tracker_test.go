package integral

import (
	"context"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/ethrpc"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
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
	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	ts.client = rpcClient

	config := Config{
		DexID:          DexTypeIntegral,
		RelayerAddress: "0x3c6951fdb433b5b8442e7aa126d50fbfb54b5f42",
		PoolPagingSize: 20,
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, err := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0x12b8bC27Ca8A997680F49d1A6FC1D93D552aacbe",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			},
			{
				Address: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			},
		},
	}, pool.GetNewPoolStateParams{})
	if err != nil {
		panic(err)
	}

	var pair IntegralPair
	if err := sonic.Unmarshal([]byte(pool.Extra), &pair); err != nil {
		require.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}

	require.NotNil(ts.Suite.T(), pair)

	require.NotEqual(ts.Suite.T(), number.Zero, pair.Price)
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
