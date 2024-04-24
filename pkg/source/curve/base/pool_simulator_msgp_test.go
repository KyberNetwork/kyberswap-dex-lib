package base

import (
	"fmt"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*PoolBaseSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\"}",
				"3000000",    // 0.0003
				"5000000000", // 0.5
				150000, 150000),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
				"100",
				"1000000000000", "1000000000000",
				"1000000000000000000000000000000", "1000000000000000000000000000000"),
		})
		require.NoError(t, err)
		pools = append(pools, p)

		now := time.Now().Unix()
		p, err = NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"101940884", "107546110", "208092128367874420986"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"initialA\": \"%v\", \"futureA\": \"%v\", \"futureATime\": %v}",
				"3000000",    // 0.0003
				"5000000000", // 0.5
				100000, 200000,
				now*2),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"0x0\", \"aPrecision\": \"%v\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"rates\": [\"%v\", \"%v\"]}",
				"100",
				"1000000000000", "1000000000000",
				"1000000000000000000000000000000", "1000000000000000000000000000000"),
		})
		require.NoError(t, err)
		pools = append(pools, p)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(PoolBaseSimulator)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(PoolBaseSimulator{})...))
	}
}
