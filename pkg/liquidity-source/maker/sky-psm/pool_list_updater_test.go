package skypsm

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	rpcClientByChainID := map[valueobject.ChainID]*ethrpc.Client{
		valueobject.ChainIDArbitrumOne: ethrpc.New("https://arbitrum.drpc.org").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}

	testCases := []struct {
		chainID valueobject.ChainID
		config  Config
	}{
		{
			chainID: valueobject.ChainIDArbitrumOne,
			config: Config{
				DexID:      DexType,
				PsmAddress: "0x2B05F8e1cACC6974fD79A673a341Fe1f58d27266",
				Tokens: []string{
					"0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
					"0x6491c05A82219b8D1479057361ff1654749b876b",
					"0xdDb46999F8891663a8F2828d25298f70416d7610",
				},
			},
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.config.DexID, func(t *testing.T) {
			updater := PoolListUpdater{
				config:         &tc.config,
				ethrpcClient:   rpcClientByChainID[tc.chainID],
				hasInitialized: false,
			}

			tracker := NewPoolTracker(
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
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
