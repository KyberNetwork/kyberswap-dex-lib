package xpress

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcURL := "https://rpc.soniclabs.com"
	multicallAddress := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	chainId := valueobject.ChainID(146)
	xpressApiURL := "https://api.xpressprotocol.com"
	helperAddress := "0x38e577290CAF18D07B5719CC9DA1E91BD753F8C0"

	plUpdater := NewPoolListUpdater(&Config{
		DexId: DexType,
		HTTPConfig: HTTPConfig{
			BaseURL: xpressApiURL,
		},
		HelperAddress: helperAddress,
		ChainId:       chainId,
	}, ethrpc.New(rpcURL).
		SetMulticallContract(multicallAddress))

	pools, poolsMetadataBytes, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(pools), 0)

	for _, p := range pools {
		tracker, err := NewPoolTracker(plUpdater.config, plUpdater.ethrpcClient)
		require.NoError(t, err)

		pool, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotNil(t, pool)
	}

	newPools, _, err := plUpdater.GetNewPools(context.Background(), poolsMetadataBytes)
	require.NoError(t, err)
	require.Equal(t, 0, len(newPools))
}
