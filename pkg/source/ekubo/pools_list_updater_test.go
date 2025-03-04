package ekubo

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	ethclient, err := clientFromEnv()
	require.NoError(t, err)

	plUpdater := NewPoolListUpdater(&SepoliaConfig, ethrpc.NewWithClient(ethclient))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	newPools, _, err = plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, len(newPools), 0)
}
