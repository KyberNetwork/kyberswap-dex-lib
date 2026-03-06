package kuruob

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolsListUpdater_GetNewPools(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcURL := "https://rpc.monad.xyz"
	multicallAddress := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	chainId := valueobject.ChainIDMonad
	kuruApiURL := "https://api.kuru.io"

	plUpdater := NewPoolListUpdater(&Config{
		DexID: DexType,
		HTTPConfig: HTTPConfig{
			BaseURL: kuruApiURL,
		},
		ChainId: chainId,
	}, ethrpc.New(rpcURL).SetMulticallContract(multicallAddress))

	pools, poolsMetadataBytes, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(pools), 0)

	tracker, err := NewPoolTracker(plUpdater.config, plUpdater.ethrpcClient)
	require.NoError(t, err)

	for i, p := range pools {
		pools[i], err = tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
	}
	t.Log(string(lo.Must(json.Marshal(pools[10]))))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), poolsMetadataBytes)
	require.NoError(t, err)
	require.Equal(t, 0, len(newPools))
}
