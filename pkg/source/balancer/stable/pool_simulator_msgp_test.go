package balancerstable

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	poolEntities := []*entity.Pool{
		{
			Address:    "0x06df3b2bbb68adc8b0e302443692037ed9f91b42",
			ReserveUsd: 0,
			SwapFee:    0.0004,
			Exchange:   "balancer",
			Type:       "balancer-stable",
			Timestamp:  13529165,
			Reserves: []string{"4362365955985",
				"4342743177527924936049411",
				"6921895060068041759669604",
				"4198113236810"},
			Tokens: entity.PoolTokens{
				&entity.PoolToken{
					Address: "A",
					Weight:  250000000000000000,
				},
				&entity.PoolToken{
					Address: "B",
					Weight:  250000000000000000,
				},
				&entity.PoolToken{
					Address: "C",
					Weight:  250000000000000000,
				},
				&entity.PoolToken{
					Address: "D",
					Weight:  250000000000000000,
				},
			},
			Extra:       "{\"amplificationParameter\":{\"value\":60000,\"isUpdating\":false,\"precision\":1000}}",
			StaticExtra: "{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x06df3b2bbb68adc8b0e302443692037ed9f91b42000000000000000000000012\",\"tokenDecimals\":[6,18,18,6]}",
		},
	}
	var err error
	pools := make([]*StablePool, len(poolEntities))
	for i, poolEntity := range poolEntities {
		pools[i], err = NewPoolSimulator(*poolEntity)
		require.NoError(t, err)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(StablePool)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(StablePool{})...))
	}
}
