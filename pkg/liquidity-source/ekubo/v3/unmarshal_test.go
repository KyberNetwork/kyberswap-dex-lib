package ekubov3

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/v3/pools"
)

func TestUnmarshalVe33FullRangePool(t *testing.T) {
	t.Parallel()

	const swapFee = uint64(123)
	ve33 := common.HexToAddress("0xd100000000000000000000000000000000000000")
	key := pools.AnyPoolKey{PoolKey: pools.NewPoolKey(
		common.HexToAddress("0x1"),
		common.HexToAddress("0x2"),
		pools.NewPoolConfig(ve33, 0, pools.PoolTypeConfig(pools.NewFullRangePoolTypeConfig())),
	)}
	state := pools.NewVe33PoolState(
		pools.NewFullRangePoolState(
			pools.NewFullRangePoolSwapState(new(uint256.Int).Lsh(uint256.NewInt(1), 128)),
			uint256.NewInt(1_000_000),
		),
		swapFee,
	)
	extra, err := json.Marshal(state)
	require.NoError(t, err)

	pool, err := unmarshalPool(extra, &StaticExtra{
		ExtensionType: ExtensionTypeVe33,
		PoolKey:       key,
	})
	require.NoError(t, err)

	roundTrip, err := json.Marshal(pool.GetState())
	require.NoError(t, err)
	require.JSONEq(t, string(extra), string(roundTrip))
}
