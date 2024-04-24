package tricrypto

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
			Reserves:    entity.PoolReserves{"54743954382801", "212871488312", "32759437840549558629494"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"A\":\"1707629\",\"D\":\"162458225493710120387117207\",\"gamma\":\"11809167828997\",\"priceScale\":[\"25182439404844022315525\",\"1651754874918630176109\",\"\"],\"lastPrices\":[\"25550848343816062635020\",\"1663587698754935470890\",\"\"],\"priceOracle\":[\"25509537194730788716548\",\"1663683592023356857621\",\"\"],\"feeGamma\":\"500000000000000\",\"midFee\":\"3000000\",\"outFee\":\"30000000\",\"futureAGammaTime\":0,\"futureAGamma\":\"581076037942835227425498917514114728328226821\",\"initialAGammaTime\":1633548703,\"initialAGamma\":\"183752478137306770270222288013175834186240000\",\"lastPricesTimestamp\":1686880115,\"lpSupply\":\"151463393077555004737648\",\"xcpProfit\":\"1063768763992698993\",\"virtualPrice\":\"1031885802695565056\",\"allowedExtraProfit\":\"2000000000000\",\"adjustmentStep\":\"490000000000000\",\"maHalfTime\":\"600\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1000000000000\",\"10000000000\",\"1\"]}",
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
