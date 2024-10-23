package ondo_usdy

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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

	require.NotNil(ts.T(), pools)

	a := uint256.MustFromDecimal("53919893334301279589334030174039261347274288845081144962207220498432")
	// get bit 255 of a
	a = a.Rsh(a, 255)
	require.Equal(ts.T(), big.NewInt(1), a)
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
