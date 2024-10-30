package deltaswapv1

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PoolListTrackerTestSuite struct {
	suite.Suite

	client  *ethrpc.Client
	tracker PoolTracker
}

func (ts *PoolListTrackerTestSuite) SetupTest() {
	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x7eCfBaa8742fDf5756DAC92fbc8b90a19b8815bF"))

	ts.client = rpcClient

	config := Config{
		DexID:          DexType,
		FactoryAddress: "0xcb85e1222f715a81b8edaeb73b28182fa37cffa8",
	}

	ts.tracker = PoolTracker{
		config:       &config,
		ethrpcClient: ts.client,
	}
}

func (ts *PoolListTrackerTestSuite) TestGetNewPoolState() {
	testCases := []struct {
		name       string
		pool       entity.Pool
		expectFail bool
	}{
		{
			name: "Test 1",
			pool: entity.Pool{
				Address: "0xa688899fed3e2cee02747a8193ab38f6be595070",
			},
			expectFail: false,
		},
		{
			name: "Test 2",
			pool: entity.Pool{
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			},
			expectFail: true,
		},
	}

	for _, tc := range testCases {
		ts.Run(tc.name, func() {
			newPoolState, err := ts.tracker.GetNewPoolState(context.Background(), tc.pool, pool.GetNewPoolStateParams{})

			if tc.expectFail {
				require.Error(ts.T(), err, "Expected an error but got none")
			} else {
				require.NoError(ts.T(), err, "Expected no error but got one")
				require.NotNil(ts.T(), newPoolState, "Expected non-nil pool state")

				var poolExtra Extra
				err = json.Unmarshal([]byte(newPoolState.Extra), &poolExtra)
				require.NoError(ts.T(), err, "Failed to unmarshal pool extra")
			}
		})
	}
}

func TestPoolListTrackerTestSuite(t *testing.T) {
	t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListTrackerTestSuite))
}
