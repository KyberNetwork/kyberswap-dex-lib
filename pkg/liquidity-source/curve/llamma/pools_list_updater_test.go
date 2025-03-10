package llamma

import (
	"context"
	"fmt"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/kr/pretty"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
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
	graphqlClientByChainID := map[valueobject.ChainID]*graphqlpkg.Client{
		valueobject.ChainIDEthereum: graphqlpkg.NewClient("https://gateway.thegraph.com/api/0d3cd5a52941499ee6dde42cc4852a20/subgraphs/id/B6jNaCpuWbz1BfEVq5D8EyZHtvX55oBY3viaYnLeFgQ3"),
	}

	testCases := []struct {
		chainID valueobject.ChainID
		config  Config
	}{
		{
			chainID: valueobject.ChainIDEthereum,
			config: Config{
				DexID:               DexType,
				FactoryAddress:      "0xc9332fdcb1c491dcc683bae86fe3cb70360738bc",
				StableCoin:          "0xf939E0A03FB07F59A73314E73794Be0E57ac1b4E",
				NewPoolLimit:        10,
				LlammaHelperAddress: "0xb52C1ce530ad6F651bc01C234f11F91517970aC9",
			},
		},
	}

	for _, tc := range testCases {
		ts.T().Run(tc.config.DexID, func(t *testing.T) {
			updater := NewPoolsListUpdater(&tc.config, rpcClientByChainID[tc.chainID])

			pools, _, err := updater.GetNewPools(context.Background(), nil)
			require.NoError(t, err)
			require.NotNil(t, pools)

			fmt.Println("Pools: ", len(pools))

			tracker := NewPoolTracker(&tc.config, rpcClientByChainID[tc.chainID], graphqlClientByChainID[tc.chainID])
			for _, pool := range pools {
				fmt.Println(pool.Address)
				newPool, err := tracker.GetNewPoolState(context.Background(), pool, poolpkg.GetNewPoolStateParams{})
				require.NoError(t, err)
				require.NotNil(t, newPool)

				poolStr, err := json.Marshal(newPool)
				require.NoError(t, err)
				// fmt.Println(string(poolStr))

				// print pretty
				fmt.Println(pretty.Println(string(poolStr)))
				bytes, err := updater.getStateFromHelper(context.Background(), pool.Address)
				if err != nil {
					return
				}
				fmt.Println(common.Bytes2Hex(bytes))
			}
		})
	}
}

func TestPoolListUpdaterTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(PoolListUpdaterTestSuite))
}
