package v2

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	updater *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://base-rpc.kyberswap.com").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	config := Config{
		ChainID:      valueobject.ChainIDBase,
		DexId:        DexType,
		Factory:      "0xD7D3C85B4f2e9bee1998cD2E98820e647792d284",
		NewPoolLimit: 100,
	}

	ts.updater = NewPoolsListUpdater(&config, rpcClient)
	ts.tracker = NewPoolTracker(&config, rpcClient)
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	var metadataBytes []byte
	for {
		pools, newMetadataBytes, err := ts.updater.GetNewPools(ts.T().Context(), metadataBytes)
		require.NoError(ts.T(), err)
		require.Greater(ts.T(), len(pools), 0)

		metadataBytes = newMetadataBytes
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	suite.Run(t, new(PoolListTrackerTestSuite))
}
