package ramses

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	velodromev1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/velodrome-v1"
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
	rpcClient.SetMulticallContract(common.HexToAddress("0x7ecfbaa8742fdf5756dac92fbc8b90a19b8815bf"))

	ts.client = rpcClient

	config := Config{
		DexID:          "ramses",
		FactoryAddress: "0xAAA20D08e59F6561f242b08513D36266C5A29415",
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}

}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	staticExtraJson, _ := json.Marshal(StaticExtra{
		Stable: false,
	})
	pool, _ := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address:     "0xf9c642d206e7974d7d01758568d3e30019c7f022",
		StaticExtra: string(staticExtraJson),
	}, pool.GetNewPoolStateParams{})

	var pair velodromev1.PoolExtra
	if err := json.Unmarshal([]byte(pool.Extra), &pair); err != nil {
		assert.Fail(ts.T(), "Failed to unmarshal pool extra %e", err)
	}

	assert.NotNil(ts.T(), pair.IsPaused)
	assert.NotNil(ts.T(), pair.Fee)
	assert.NotNil(ts.T(), pool.Reserves)
	assert.NotNil(ts.T(), pool.Extra)
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
