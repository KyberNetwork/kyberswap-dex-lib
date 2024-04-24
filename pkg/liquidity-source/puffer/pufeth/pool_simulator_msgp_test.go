package pufeth

import (
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{PUFETH, STETH, WSTETH},
				},
			},
			totalSupply: number.NewUint256("379989503452489947895013"),
			totalAssets: number.NewUint256("382649667359278267721330"),
		},
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{PUFETH, STETH, WSTETH},
				},
			},
			totalSupply:      number.NewUint256("379677392580527064900714"),
			totalAssets:      number.NewUint256("382335371516233372457736"),
			totalPooledEther: number.NewUint256("9408886941382666867434878"),
			totalShares:      number.NewUint256("8085737150987915500442326"),
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
