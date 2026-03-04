package v2

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	client := ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(valueobject.AddrMulticall3)

	cfg := &Config{
		ChainID:     1,
		DexID:       DexType,
		Comptroller: "0x3d9819210A31b4961b30EF54bE2aeD79B9c9Cd3B",
	}

	lister := NewPoolsListUpdater(cfg, client)

	newPools, _, err := lister.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	tracker, err := NewPoolTracker(cfg, client)
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
