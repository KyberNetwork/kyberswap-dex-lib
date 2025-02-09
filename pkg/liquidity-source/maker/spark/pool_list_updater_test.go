package spark

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsdai"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/savingsusds"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolListUpdaterTestSuite struct {
	suite.Suite
}

func (ts *PoolListUpdaterTestSuite) TestGetNewPools() {
	rpcClientByChainID := map[valueobject.ChainID]*ethrpc.Client{
		1: ethrpc.New("https://ethereum.kyberengineering.io").
			SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696")),
	}

	testCases := []struct {
		chainID valueobject.ChainID
		config  Config
	}{
		{
			chainID: valueobject.ChainIDEthereum,
			config: Config{
				DexID:             savingsdai.DexType,
				DepositToken:      "0x6b175474e89094c44da98b954eedeac495271d0f",
				SavingsToken:      "0x83f20f44975d03b1b09e64809b757c47f942beea",
				Pot:               "0x197e90f9fad81970ba7976f33cbd77088e5d7cf7",
				SavingsRateSymbol: "dsr",
			},
		},
		{
			chainID: valueobject.ChainIDEthereum,
			config: Config{
				DexID:             savingsusds.DexType,
				DepositToken:      "0xdc035d45d973e3ec169d2276ddab16f1e407384f",
				SavingsToken:      "0xa3931d71877c0e7a3148cb7eb4463524fec27fbd",
				Pot:               "0xa3931d71877c0e7a3148cb7eb4463524fec27fbd",
				SavingsRateSymbol: "ssr",
			},
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.config.DexID, func(t *testing.T) {
			updater := PoolListUpdater{
				config:       &tc.config,
				ethrpcClient: rpcClientByChainID[tc.chainID],
				initialized:  false,
			}

			tracker := NewPoolTracker(
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
