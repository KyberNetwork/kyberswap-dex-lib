package generic_rate

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	skypsm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/sky-psm"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	rpcClientByChainID := map[valueobject.ChainID]*ethrpc.Client{
		1: ethrpc.New("https://ethereum.drpc.org").
			SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696")),
		8453: ethrpc.New("https://base.drpc.org").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}

	testCases := []struct {
		chainID valueobject.ChainID
		config  Config
	}{
		{
			chainID: valueobject.ChainIDBase,
			config: Config{
				DexID:    skypsm.DexType,
				PoolPath: "pools/base.json",
			},
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.config.DexID, func(t *testing.T) {
			updater := PoolsListUpdater{
				config:         &tc.config,
				ethrpcClient:   rpcClientByChainID[tc.chainID],
				hasInitialized: false,
			}

			tracker, _ := NewPoolTracker(
				&tc.config,
				rpcClientByChainID[tc.chainID],
			)

			pools, _, err := updater.GetNewPools(context.Background(), nil)
			require.NoError(t, err)
			require.NotNil(t, pools)

			for _, pool := range pools {
				entityPool, err := tracker.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
				require.NoError(t, err)
				require.NotNil(t, entityPool)

				prettyJSON, err := json.MarshalIndent(entityPool, "", "    ")
				require.NoError(t, err)
				require.NotNil(t, pools)
				t.Log(string(prettyJSON))
			}
		})
	}
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
