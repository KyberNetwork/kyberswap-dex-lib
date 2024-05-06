package makerpsm

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*PoolSimulator{
		newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0)),
		newPool(t, big.NewInt(100), TOLL_ONE_PCT, big.NewInt(0)),
		newPool(t, big.NewInt(100), new(big.Int).Mul(big.NewInt(5), TOLL_ONE_PCT), new(big.Int).Mul(big.NewInt(10), TOLL_ONE_PCT)),
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
