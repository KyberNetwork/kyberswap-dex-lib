package kokonutcrypto

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
		kokonutPool, err := NewPoolSimulator(entity.Pool{
			Address:  "0x73c3a78e5ff0d216a50b11d51b262ca839fcfe17",
			Exchange: "kokonut-crypto",
			Type:     "kokonut-crypto",
			Reserves: entity.PoolReserves{"952708662862", "589902580550233792806"},
			Tokens: []*entity.PoolToken{
				{
					Address:  "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
					Decimals: 6,
				},
				{
					Address:  "0x4200000000000000000000000000000000000006",
					Decimals: 18,
				},
			},
			Extra:       "{\"A\":\"400000\",\"D\":\"1981441302805325624637942\",\"gamma\":\"145000000000000\",\"priceScale\":\"1745382367410361004355\",\"lastPrices\":\"1641929899604339515825\",\"priceOracle\":\"1641934566575895837347\",\"feeGamma\":\"230000000000000\",\"midFee\":\"10000000\",\"outFee\":\"100000000\",\"futureAGammaTime\":0,\"futureA\":\"400000\",\"futureGamma\":\"145000000000000\",\"initialAGammaTime\":0,\"initialA\":\"400000\",\"initialGamma\":\"145000000000000\",\"lastPricesTimestamp\":1694139013,\"lpSupply\":\"23698540246446124166400\",\"xcpProfit\":\"1000781771675844506\",\"virtualPrice\":\"1000654903935132927\",\"allowedExtraProfit\":\"2000000000000\",\"adjustmentStep\":\"146000000000000\",\"maHalfTime\":\"600\"}",
			StaticExtra: "{\"lpToken\":\"0x5b15fc22233315d4f4064a00268e5efc95795a23\",\"precisionMultipliers\":[\"1000000000000\",\"1\"]}",
		})
		require.NoError(t, err)
		pools = append(pools, kokonutPool)
	}
	{
		kokonutPool, err := NewPoolSimulator(entity.Pool{
			Address:  "0x73c3a78e5ff0d216a50b11d51b262ca839fcfe17",
			Exchange: "kokonut-crypto",
			Type:     "kokonut-crypto",
			Reserves: entity.PoolReserves{"952708662862", "589902580550233792806"},
			Tokens: []*entity.PoolToken{
				{
					Address:  "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
					Decimals: 6,
				},
				{
					Address:  "0x4200000000000000000000000000000000000006",
					Decimals: 18,
				},
			},
			Extra:       "{\"A\":\"400000\",\"D\":\"1981441302805325624637942\",\"gamma\":\"145000000000000\",\"priceScale\":\"1745382367410361004355\",\"lastPrices\":\"1641929899604339515825\",\"priceOracle\":\"1641934566575895837347\",\"feeGamma\":\"230000000000000\",\"midFee\":\"10000000\",\"outFee\":\"100000000\",\"futureAGammaTime\":0,\"futureA\":\"400000\",\"futureGamma\":\"145000000000000\",\"initialAGammaTime\":0,\"initialA\":\"400000\",\"initialGamma\":\"145000000000000\",\"lastPricesTimestamp\":1694139013,\"lpSupply\":\"23698540246446124166400\",\"xcpProfit\":\"1000781771675844506\",\"virtualPrice\":\"1000654903935132927\",\"allowedExtraProfit\":\"2000000000000\",\"adjustmentStep\":\"146000000000000\",\"maHalfTime\":\"600\",\"minRemainingPostRebalanceRatio\":\"8000000000\"}",
			StaticExtra: "{\"lpToken\":\"0x5b15fc22233315d4f4064a00268e5efc95795a23\",\"precisionMultipliers\":[\"1000000000000\",\"1\"]}",
		})
		require.NoError(t, err)
		pools = append(pools, kokonutPool)
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
