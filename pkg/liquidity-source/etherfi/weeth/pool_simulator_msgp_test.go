package weeth

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
					Tokens: []string{"0x35fa164735182de50811e8e2e824cfb9b6118ac2", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
				},
			},
			totalPooledEther: bignumber.NewBig("479746451523543911039175"),
			totalShares:      bignumber.NewBig("464768412137509601320862"),
		},
		{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{"0x35fa164735182de50811e8e2e824cfb9b6118ac2", "0xCd5fE23C85820F7B72D0926FC9b05b43E359b7ee"},
				},
			},
			totalPooledEther: bignumber.NewBig("482437159360194010684174"),
			totalShares:      bignumber.NewBig("467375114083494601305331"),
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
