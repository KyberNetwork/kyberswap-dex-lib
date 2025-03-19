package llamma

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
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
		valueobject.ChainIDEthereum: ethrpc.New("https://ethereum.kyberengineering.io").
			SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")),
	}

	testCases := []struct {
		chainID valueobject.ChainID
		config  Config
	}{
		{
			chainID: valueobject.ChainIDEthereum,
			config: Config{
				DexID:          DexType,
				FactoryAddress: "0xc9332fdcb1c491dcc683bae86fe3cb70360738bc",
				BorrowedToken:  "0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E",
				MaxBandLimit:   50,
				NewPoolLimit:   10,
			},
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.config.DexID, func(t *testing.T) {
			updater := NewPoolsListUpdater(&tc.config, rpcClientByChainID[tc.chainID])
			tracker := NewPoolTracker(&tc.config, rpcClientByChainID[tc.chainID])

			pools, _, err := updater.GetNewPools(context.Background(), nil)
			require.NoError(t, err)
			require.NotNil(t, pools)

			for _, pool := range pools {
				newPool, err := tracker.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
				require.NoError(t, err)
				require.NotNil(t, newPool)
			}
		})
	}
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
