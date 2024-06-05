package utils

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestMarshalUnmarshal[P any](t *testing.T, pools []*P) {
	require.NotEmpty(t, pools)

	for _, pool := range pools {
		var encoded bytes.Buffer
		en := msgpack.NewEncoder(&encoded)
		err := en.Encode(pool)
		msgpack.PutEncoder(en)
		require.NoError(t, err)

		decoded := new(P)
		de := msgpack.NewDecoder(&encoded)
		err = de.Decode(decoded)
		msgpack.PutDecoder(de)
		require.NoError(t, err)

		require.Empty(t, cmp.Diff(pool, decoded, testutil.CmpOpts()...))

		// Skip check pointers relationship equality for `synthetix.PoolSimulator``.
		// Because there is a bug in spew.Sdump() which panics when dump a `synthetix.PoolSimulator`:
		// "panic: reflect.Value.Interface: cannot return value obtained from unexported field or method".
		// `synthetix.PoolSimulators` doesn't need to check pointers relationship equality anyway because it doesn't contain pointer alias.
		if reflect.TypeFor[P]() != reflect.TypeFor[synthetix.PoolSimulator]() {
			require.Equal(t, testutil.DumpWithNormalizedPointers(pool), testutil.DumpWithNormalizedPointers(decoded))
		}
	}
}
