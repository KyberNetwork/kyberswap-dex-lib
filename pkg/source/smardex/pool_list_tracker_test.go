package smardex

import (
	"context"
	"fmt"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
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

	// var pair SmardexPair
	// if err := json.Unmarshal([]byte(pool.Extra), &pair); err != nil {
	// 	fmt.Printf("cmnguyen %s\n", pool.Extra)
	// 	assert.Fail(ts.Suite.T(), "Failed to unmarshal pool extra")
	// }

	// assert.NotNil(ts.Suite.T(), pair.PairFee)
	fmt.Printf("cmnguyen %s\n", pool.TotalSupply)
	// fmt.Printf("cmnguyen reserves %s, %s\n", pool.)
	// assert.Equal(ts.Suite.T(), big.NewInt(1500), pair.PairFee.feesLP)
	// assert.Equal(ts.Suite.T(), big.NewInt(900), pair.PairFee.feesPool)
	// assert.NotNil(ts.Suite.T(), pair.FictiveReserve)
	// assert.NotNil(ts.Suite.T(), pair.PriceAverage)
	// assert.NotNil(ts.Suite.T(), pair.FeeToAmount)
	// assert.NotNil(ts.Suite.T(), pair.Reserve)

}

func TestPoolListTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(PoolListTrackerTestSuite))
}
