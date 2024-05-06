package algebrav1

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		pool, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2985,"feeOtz":2985,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)

		pool, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":2822091172725,"globalState":{"price":93065132232889433968150957834858946,"tick":279543,"feeZto":2979,"feeOtz":2979,"timepoint_index":65,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":2822091172725,"LiquidityNet":2822091172725},{"Index":273540,"LiquidityGross":116315447200034,"LiquidityNet":116315447200034},{"Index":279120,"LiquidityGross":116315447200034,"LiquidityNet":-116315447200034},{"Index":285480,"LiquidityGross":2822091172725,"LiquidityNet":-2822091172725}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)

		pool, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"10963601168695220226", "357336560175387760"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":0,"globalState":{"price":4295128740,"tick":-887272,"feeZto":1622,"feeOtz":1622,"timepoint_index":2497,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},"ticks":[{"Index":-3420,"LiquidityGross":3425867281055637406,"LiquidityNet":3425867281055637406},{"Index":-1680,"LiquidityGross":54492387444405553633,"LiquidityNet":54492387444405553633},{"Index":-1500,"LiquidityGross":11191922902152224210,"LiquidityNet":11191922902152224210},{"Index":0,"LiquidityGross":2148740956490219135,"LiquidityNet":2148740956490219135},{"Index":60,"LiquidityGross":5964987541425314734,"LiquidityNet":5964987541425314734},{"Index":120,"LiquidityGross":5964987541425314734,"LiquidityNet":-5964987541425314734},{"Index":180,"LiquidityGross":2148740956490219135,"LiquidityNet":-2148740956490219135},{"Index":1200,"LiquidityGross":54492387444405553633,"LiquidityNet":-54492387444405553633},{"Index":1380,"LiquidityGross":11191922902152224210,"LiquidityNet":-11191922902152224210},{"Index":2160,"LiquidityGross":3425867281055637406,"LiquidityNet":-3425867281055637406}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)

		pool, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"4972738711862929441043", "1959593146565760679885786"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":98714460437307995596273,"globalState":{"price":1572768200222810245774927517376,"tick":59768,"feeZto":11076,"feeOtz":11076,"timepoint_index":45,"community_fee_token0":1000,"community_fee_token1":1000,"unlocked":true},"ticks":[{"Index":-887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":98714460437307995596273},{"Index":887220,"LiquidityGross":98714460437307995596273,"LiquidityNet":-98714460437307995596273}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)

		pool, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)

		pool, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"723924", "36031866872048609640"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:    `{"liquidity":954140562773509808028,"globalState":{"price":84125210470736011805469300802,"tick":1199,"feeZto":100,"feeOtz":3000,"timepoint_index":104,"community_fee_token0":150,"community_fee_token1":150,"unlocked":true},"ticks":[{"Index":480,"LiquidityGross":954140562773509808028,"LiquidityNet":954140562773509808028},{"Index":1200,"LiquidityGross":954140562773509808028,"LiquidityNet":-954140562773509808028}],"tickSpacing":60}`,
		}, 1001)
		require.NoError(t, err)
		pools = append(pools, pool)
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
