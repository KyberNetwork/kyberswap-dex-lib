package reth

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
					Tokens: []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xae78736cd615f374d3085123a210448e74fc6393"},
				},
			},
			depositEnabled:         true,
			minimumDeposit:         bignumber.NewBig("10000000000000000"),
			balance:                bignumber.NewBig("17963940799090443727000"),
			maximumDepositPoolSize: bignumber.NewBig("18000000000000000000000"),
			depositFee:             bignumber.NewBig("500000000000000"),
			totalRETHSupply:        bignumber.NewBig("563912813663573766722840"),
			totalETHBalance:        bignumber.NewBig("619583685490020782650352"),
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
