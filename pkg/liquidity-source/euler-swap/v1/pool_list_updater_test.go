package v1

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolsListUpdater(&shared.Config{
		DexID:          DexType,
		FactoryAddress: "0x45b146BC07c9985589B52df651310e75C6BE066A",
	}, ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(valueobject.AddrMulticall3))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	for _, p := range newPools {
		tracker, err := NewPoolTracker(plUpdater.config, plUpdater.ethrpcClient)
		require.NoError(t, err)

		p, err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotNil(t, p)
	}
}
