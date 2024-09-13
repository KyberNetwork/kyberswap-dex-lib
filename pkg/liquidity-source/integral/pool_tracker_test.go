package integral

import (
	"context"
	"encoding/json"
	"testing"

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
		FactoryAddress: "0xC480b33eE5229DE3FbDFAD1D2DCD3F3BAD0C56c6",
		PoolPagingSize: 20,
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, err := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0xd17b3c9784510E33cD5B87b490E79253BcD81e2E",
	}, pool.GetNewPoolStateParams{})
	if err != nil {
		panic(err)
	}

	var pair IntegralPair
	if err := json.Unmarshal([]byte(pool.Extra), &pair); err != nil {
		require.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}

	require.NotNil(ts.Suite.T(), pair)

	require.NotEqual(ts.Suite.T(), uZERO, pair.AveragePrice)
	// require.NotEqual(ts.Suite.T(), ZERO, pair.DecimalsConverter)
	// require.NotEqual(ts.Suite.T(), uZERO, pair.SwapFee)

	require.Equal(ts.Suite.T(), 2, len(pool.Reserves))
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[0])
	require.NotEqual(ts.Suite.T(), "", pool.Reserves[1])

	require.Equal(ts.Suite.T(), 2, len(pool.Tokens))
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[0])
	require.NotEqual(ts.Suite.T(), "", pool.Tokens[1])
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
