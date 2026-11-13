package pancakestable

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// The encoder keys the swap path on the pool type: curve-base would pick the Curve
// int128 path, but these pools need the uint256 exchange path.
func TestListerEmitsOwnPoolType(t *testing.T) {
	p := newEntityPool("zkswap-stable", common.HexToAddress("0x1"), nil, []byte("{}"))
	require.Equal(t, "pancake-stable", p.Type)
	require.NotEqual(t, curve.PoolTypeBase, p.Type)
}

func TestFactoryBuildsSimulatorForOwnType(t *testing.T) {
	ep := entity.Pool{
		Address:  "0x1",
		Exchange: "zkswap-stable",
		Type:     DexType,
		Reserves: entity.PoolReserves{"100000000", "100000000", "200000000"},
		Tokens:   []*entity.PoolToken{{Address: "a"}, {Address: "b"}},
		Extra:    `{"swapFee":"3000000","adminFee":"5000000000","initialA":"150000","futureA":"150000"}`,
		StaticExtra: `{"lpToken":"LP","aPrecision":"100","precisionMultipliers":["1","1"],` +
			`"rates":["1000000000000000000","1000000000000000000"]}`,
	}
	f := pool.Factory(DexType)
	require.NotNil(t, f)
	sim, err := f(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	require.Equal(t, DexType, sim.GetType())
}
