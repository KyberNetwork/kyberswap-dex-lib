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
		common.HexToAddress("0xd876ec2ee0816c019cc54299a8184e8111694865"),
		common.HexToAddress("0xf7b3e9697fd769104cd6cf653c179fb452505a3e"),
		quoting.Config{
			Fee:         9223372036854775,
			TickSpacing: 1000,
			Extension:   common.Address{},
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
