package usd0pp

import (
	"context"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/bytedance/sonic"
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
	// Setup RPC server
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

	require.NotNil(ts.Suite.T(), pools)
	require.Equal(ts.Suite.T(), 1, len(pools))
	require.Equal(ts.Suite.T(), USD0PP, pools[0].Address)
	require.Equal(ts.Suite.T(), DexType, pools[0].Exchange)
	require.Equal(ts.Suite.T(), DexType, pools[0].Type)
	require.Equal(ts.Suite.T(), 2, len(pools[0].Tokens))
	require.Equal(ts.Suite.T(), uint8(18), pools[0].Tokens[0].Decimals)
	require.Equal(ts.Suite.T(), uint8(18), pools[0].Tokens[1].Decimals)
	require.Equal(ts.Suite.T(), strings.ToLower(USD0), pools[0].Tokens[0].Address)
	require.Equal(ts.Suite.T(), strings.ToLower(USD0PP), pools[0].Tokens[1].Address)

	var poolExtra PoolExtra
	if err := sonic.Unmarshal([]byte(pools[0].Extra), &poolExtra); err != nil {
		require.NoError(ts.Suite.T(), err)
	}

	require.Equal(ts.Suite.T(), int64(1718105400), poolExtra.StartTime)
	require.Equal(ts.Suite.T(), int64(1844335800), poolExtra.EndTime)
	require.Equal(ts.Suite.T(), false, poolExtra.Paused)
	require.Equal(ts.Suite.T(), int64(126230400), poolExtra.EndTime-poolExtra.StartTime)
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
