package ondo_usdy

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite
	client  *ethrpc.Client
	updater PoolsListUpdater
}

func (ts *PoolListUpdaterTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://rpc.mantle.xyz")
	rpcClient.SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	ts.client = rpcClient

	config := Config{
		DexID:    DexType,
		PoolPath: "pools/mantle.json",
	}

	ts.updater = PoolsListUpdater{
		config:       &config,
		ethrpcClient: ts.client,

		hasInitialized: false,
	}
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	pools, _, _ := ts.updater.GetNewPools(context.Background(), nil)
	require.NotNil(ts.T(), pools)
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
