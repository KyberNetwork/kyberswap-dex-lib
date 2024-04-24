package smardex

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolSimulator
	{
		extra := SmardexPair{
			PairFee: PairFee{
				FeesLP:   feesLP,
				FeesPool: feesPool,
				FeesBase: FEES_BASE,
			},
			FictiveReserve: FictiveReserve{
				FictiveReserve0: resFicT0,
				FictiveReserve1: resFicT1,
			},
			PriceAverage: PriceAverage{
				PriceAverage0:             priceAvT0,
				PriceAverage1:             priceAvT1,
				PriceAverageLastTimestamp: big.NewInt(TIMESTAMP_JAN_2020),
			},
			FeeToAmount: FeeToAmount{
				Fees0: big.NewInt(0),
				Fees1: big.NewInt(0),
			},
		}
		extraJson, _ := json.Marshal(extra)

		token0 := entity.PoolToken{
			Address:   "token0",
			Swappable: true,
		}
		token1 := entity.PoolToken{
			Address:   "token1",
			Swappable: true,
		}

		pool := entity.Pool{
			Reserves: entity.PoolReserves{resT0.String(), resT1.String()},
			Tokens:   []*entity.PoolToken{&token0, &token1},
			Extra:    string(extraJson),
		}
		poolSimulator, err := NewPoolSimulator(pool)
		require.NoError(t, err)
		pools = append(pools, poolSimulator)
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
