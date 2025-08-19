package meth

import (
	"context"
	"strings"
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
	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	ts.client = rpcClient

	config := Config{
		DexID: DexType,
	}

	ts.updater = PoolsListUpdater{
		config:       &config,
		ethrpcClient: ts.client,

		hasInitialized: false,
	}
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	pools, _, _ := ts.updater.GetNewPools(context.Background(), nil)

	ts.Require().NotNil(pools)
	require.NotNil(ts.T(), pools)
	require.Equal(ts.T(), 1, len(pools))
	require.Equal(ts.T(), MantleLSPStaking, pools[0].Address)
	require.Equal(ts.T(), DexType, pools[0].Exchange)
	require.Equal(ts.T(), DexType, pools[0].Type)
	require.Equal(ts.T(), 2, len(pools[0].Tokens))
	require.Equal(ts.T(), uint8(18), pools[0].Tokens[0].Decimals)
	require.Equal(ts.T(), uint8(18), pools[0].Tokens[1].Decimals)
	require.Equal(ts.T(), strings.ToLower(WETH), pools[0].Tokens[0].Address)
	require.Equal(ts.T(), strings.ToLower(METH), pools[0].Tokens[1].Address)
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	t.Parallel()
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
