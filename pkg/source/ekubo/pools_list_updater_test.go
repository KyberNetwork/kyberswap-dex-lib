package ekubo

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	ethclient, err := clientFromEnv()
	require.NoError(t, err)

	plUpdater := NewPoolListUpdater(&SepoliaConfig, ethrpc.NewWithClient(ethclient))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)

	testPk := quoting.NewPoolKey(
		common.Address{},
		common.HexToAddress("0x618c25b11a5e9b5ad60b04bb64fcbdfbad7621d1"),
		quoting.Config{
			Fee:         0,
			TickSpacing: 0,
			Extension:   common.HexToAddress(SepoliaConfig.Oracle),
		},
	)

	require.True(t, slices.ContainsFunc(newPools, func(el entity.Pool) bool {
		var staticExtra StaticExtra
		err := json.Unmarshal([]byte(el.StaticExtra), &staticExtra)
		require.NoError(t, err)

		pk := staticExtra.PoolKey

		return pk.Token0.Cmp(testPk.Token0) == 0 && pk.Token1.Cmp(testPk.Token1) == 0 && slices.Equal(pk.Config.Compressed(), testPk.Config.Compressed())
	}))

	newPools, _, err = plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, len(newPools), 0)
}
