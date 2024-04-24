package velodromev1

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Address:  "0xe08d427724d8a2673fe0be3a81b7db17be835b36",
					Tokens:   []string{"0x7f5c764cbc14f9669b88837ca1490cca17c31607", "0x94b008aa00579c1307b0ef2c499ad98a8ce58e58"},
					Reserves: []*big.Int{utils.NewBig10("6110873648"), utils.NewBig10("6651345170")},
				},
			},
			isPaused:     false,
			stable:       true,
			decimals0:    number.NewUint256("1000000"),
			decimals1:    number.NewUint256("1000000"),
			fee:          uint256.NewInt(5),
			feePrecision: uint256.NewInt(10000),
		},
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolSimulator{})...))
	}
}
