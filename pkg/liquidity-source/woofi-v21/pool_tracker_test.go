package woofiv21

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
	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF"))

	ts.client = rpcClient

	config := Config{
		DexID:                    DexTypeWooFiV21,
		WooPPV2Address:           "0xEd9e3f98bBed560e66B89AaC922E29D4596A9642",
		IntegrationHelperAddress: "0x28D2B949024FE50627f1EbC5f0Ca3Ca721148E40",
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	pool, _ := ts.tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address: "0xed9e3f98bbed560e66b89aac922e29d4596a9642",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			},
			{
				Address: "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
			},
			{
				Address: "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
			},
			{
				Address: "0x912ce59144191c1204e64559fe8253a0e49e6548",
			},
			{
				Address: "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
			},
			{
				Address: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
			},
		},
	}, pool.GetNewPoolStateParams{})

	var extra Extra
	if err := json.Unmarshal([]byte(pool.Extra), &extra); err != nil {
		assert.Fail(ts.Suite.T(), "Failed to unmarshal pool extra %e", err)
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
