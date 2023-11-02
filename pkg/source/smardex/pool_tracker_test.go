package smardex

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
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
		DexID:          "smardex",
		FactoryAddress: "0x7753F36E711B66a0350a753aba9F5651BAE76A1D",
		PoolPagingSize: 20,
		ChainID:        1,
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}

}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, _ := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0x9fcF8F5Bd54db123470c96620441cA5C342A8Bd4",
	}, pool.GetNewPoolStateParams{})

	var pair SmardexPair
	if err := json.Unmarshal([]byte(pool.Extra), &pair); err != nil {
		assert.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}

	assert.NotNil(ts.Suite.T(), pair.PairFee.FeesLP)
	assert.NotNil(ts.Suite.T(), pair.PairFee.FeesPool)
	assert.NotNil(ts.Suite.T(), pair.FictiveReserve.FictiveReserve0)
	assert.NotNil(ts.Suite.T(), pair.FictiveReserve.FictiveReserve1)
	assert.NotNil(ts.Suite.T(), pair.PriceAverage.PriceAverage0)
	assert.NotNil(ts.Suite.T(), pair.PriceAverage.PriceAverage1)
	assert.NotNil(ts.Suite.T(), pair.FeeToAmount.Fees0)
	assert.NotNil(ts.Suite.T(), pair.FeeToAmount.Fees1)
	if ts.tracker.config.ChainID == 1 {
		assert.Equal(ts.Suite.T(), pair.PairFee.FeesBase.Cmp(FEES_BASE_ETHEREUM), 0)
	} else {
		assert.Equal(ts.Suite.T(), pair.PairFee.FeesBase.Cmp(FEES_BASE), 0)
	}
	assert.NotNil(ts.Suite.T(), pool.Reserves)
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
