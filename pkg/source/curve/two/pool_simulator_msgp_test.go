package two

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*Pool
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"2575977394749099472751", "1447320191806527553931"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:       "{\"A\":\"200000000\",\"D\":\"4344269418800893049364\",\"gamma\":\"100000000000000\",\"priceScale\":\"1250033866036595049\",\"lastPrices\":\"1241874208010789089\",\"priceOracle\":\"1199834141509881054\",\"feeGamma\":\"5000000000000000\",\"midFee\":\"10000000\",\"outFee\":\"90000000\",\"futureAGammaTime\":0,\"futureAGamma\":\"68056473384187692692674921486353742291200000000\",\"initialAGammaTime\":0,\"initialAGamma\":\"68056473384187692692674921486353742291200000000\",\"lastPricesTimestamp\":1686876995,\"lpSupply\":\"1894549993474267797965\",\"xcpProfit\":\"1034188512253919548\",\"virtualPrice\":\"1025462529694819838\",\"allowedExtraProfit\":\"10000000000\",\"adjustmentStep\":\"5500000000000\",\"maHalfTime\":\"600\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\"]}",
		})
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
