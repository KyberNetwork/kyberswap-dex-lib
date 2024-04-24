package lido

import (
	"fmt"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"2264571555224494676557305", "2005870067403083354670050"},
			Tokens:   []*entity.PoolToken{{Address: "stETH"}, {Address: "wstETH"}},
			Extra: fmt.Sprintf("{\"stEthPerToken\": %v, \"tokensPerStEth\": %v}",
				"1128972205632615487",
				"885761398740240572"),
			StaticExtra: "{\"lpToken\": \"wstETH\"}",
		})
		require.NoError(t, err)
		pools = append(pools, p)
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
