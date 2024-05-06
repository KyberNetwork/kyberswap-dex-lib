package velodromev2

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
					Address:  "0x1ad06ca54de04dbe9e2817f4c13ecb406dcbeaf0",
					Tokens:   []string{"0x3e29d3a9316dab217754d13b28646b76607c5f04", "0x6806411765af15bddd26f8f544a34cc40cb9838b"},
					Reserves: []*big.Int{utils.NewBig10("165363502891169888414"), utils.NewBig10("70707320014274856246")},
				},
			},
			isPaused:     false,
			stable:       true,
			decimals0:    number.NewUint256("1000000000000000000"),
			decimals1:    number.NewUint256("1000000000000000000"),
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
