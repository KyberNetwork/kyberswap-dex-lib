package aavev3

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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

	lister := NewPoolsListUpdater(&Config{
		DexID:           DexType,
		AavePoolAddress: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
	}, client)

	newPools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	tracker, err := NewPoolTracker(&Config{
		DexID:           DexType,
		AavePoolAddress: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
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

		require.NotEqual(t, "{}", newPool.Extra, newPool.Address)
	}

}
