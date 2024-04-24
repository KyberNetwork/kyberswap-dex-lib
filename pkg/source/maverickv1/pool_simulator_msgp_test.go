package maverickv1

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	pools := []*Pool{
		maverickPool,
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(Pool)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Equal(t, "", cmp.Diff(pool, actual, testutil.CmpOpts(Pool{})...))
	}
}
