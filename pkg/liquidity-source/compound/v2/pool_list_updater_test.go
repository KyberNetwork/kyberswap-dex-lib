package v2

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
	"github.com/ethereum/go-ethereum/common"

	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	client := ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	graphqlClient := graphqlpkg.NewClient("https://gateway.thegraph.com/api/18edf1db0b785b022d29fa48dc740617/subgraphs/id/3D5iVSy3jiTKSQAT8UjW9in6ZuiDA7WnDiDRBzYVT2yw")

	lister := NewPoolsListUpdater(&Config{
		DexID: DexType,
	}, client, graphqlClient)

	newPools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	tracker, err := NewPoolTracker(&Config{
		DexID: DexType,
	}, client)
	require.NoError(t, err)

	for _, p := range newPools {
		require.Equal(t, p.Tokens[0].Address, p.Address)

		newPool, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.Equal(t,
			entity.PoolReserves{
				strconv.Itoa(defaultReserve),
				strconv.Itoa(defaultReserve),
			},
			newPool.Reserves,
		)
	}

}
