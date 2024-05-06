package lido_steth

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		tokens := []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", "0xae7ab96520de3a18e5e111b5eaab095312d7fe84"}
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: []string{"1", "1"},
			Tokens:   lo.Map(tokens, func(adr string, _ int) *entity.PoolToken { return &entity.PoolToken{Address: adr} }),
		}, valueobject.ChainIDEthereum)
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
