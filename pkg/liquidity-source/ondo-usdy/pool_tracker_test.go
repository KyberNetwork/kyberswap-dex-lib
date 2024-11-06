package ondo_usdy

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
	rpcClient := ethrpc.New("https://rpc.mantle.xyz")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	ts.client = rpcClient

	config := Config{
		DexID:    DexType,
		PoolPath: "pools/mantle.json",
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x5be26527e817998a7206475496fde1e68957c5a6",
			},
			{
				Address: "0xab575258d37eaa5c8956efabe71f4ee8f6397cf3",
			},
		},
		Extra: "{\"paused\":false,\"oraclePrice\":\"1064060720000000000\",\"priceTimeStamp\":1729749978,\"rwaDynamicOracleAddress\":\"0xa96abbe61afedeb0d14a20440ae7100d9ab4882f\"}",
	}, pool.GetNewPoolStateParams{})

	require.Nil(ts.T(), err)

	var poolExtra PoolExtra
	if err := json.Unmarshal([]byte(newPoolState.Extra), &poolExtra); err != nil {
		require.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
