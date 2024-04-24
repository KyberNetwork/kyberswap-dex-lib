package ezeth

import (
	"testing"

	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xbf5495efe5db9ce00f80364c8b423567e58d2110",
						"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"0xa2e3356610840701bdf5611a53974510ae27e2e1",
						"0xae7ab96520de3a18e5e111b5eaab095312d7fe84",
					},
				},
			},
			paused:        false,
			totalTVL:      bignumber.NewBig("846148216510217972629804"),
			totalSupply:   bignumber.NewBig("839310921147858962585526"),
			maxDepositTVL: bignumber.ZeroBI,
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
