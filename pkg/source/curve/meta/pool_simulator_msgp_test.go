package meta

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/curve/base"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*Pool
	{
		base, err := base.NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243", "352290453972395231054279357"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
			StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\",\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}",
		})
		require.NoError(t, err)
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
			Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
			Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
			StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
		}, base)
		require.NoError(t, err)
		pools = append(pools, p)
	}
	{
		base, err := base.NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"93649867132724477811796755", "92440712316473", "175421309630243", "352290453972395231054279357"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
			StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\",\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}",
		})
		require.NoError(t, err)
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
			Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
			Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
			StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
		}, base)
		require.NoError(t, err)
		pools = append(pools, p)
	}
	{
		base, err := base.NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"93650900813860355891321787", "92392098150103", "175345980953129", "352170672490633463630226070"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"5000\",\"futureA\":\"2000\",\"initialATime\":1653559305,\"futureATime\":1654158027,\"swapFee\":\"1000000\",\"adminFee\":\"5000000000\"}",
			StaticExtra: "{\"lpToken\":\"LPBase\",\"aPrecision\":\"1\",\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}",
		})
		require.NoError(t, err)
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"4763102571534863472313821", "15272752439110430673281", "0"},
			Tokens:      []*entity.PoolToken{{Address: "Am"}, {Address: "Bm"}},
			Extra:       "{\"initialA\":\"10000\",\"futureA\":\"25000\",\"initialATime\":1649327847,\"futureATime\":1649925962,\"swapFee\":\"4000000\",\"adminFee\":\"0\"}",
			StaticExtra: "{\"lpToken\":\"LPMeta\",\"basePool\":\"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7\",\"rateMultiplier\":\"1000000000000000000\",\"aPrecision\":\"100\",\"underlyingTokens\":[\"0x674c6ad92fd080e4004b2312b45f796a192d27a0\",\"0x6b175474e89094c44da98b954eedeac495271d0f\",\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"0xdac17f958d2ee523a2206206994597c13d831ec7\"],\"precisionMultipliers\":[\"1\",\"1\"],\"rates\":[\"\",\"\"]}",
		}, base)
		require.NoError(t, err)
		pools = append(pools, p)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(Pool)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(Pool{})...))
	}
}
