package eulerswap

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolsListUpdater(&Config{
		DexID:          DexType,
		FactoryAddress: "0x45b146BC07c9985589B52df651310e75C6BE066A",
	}, ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))

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
