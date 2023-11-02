package levelfinance_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	levelfinance "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/level-finance"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalcAmountOut(t *testing.T) {
	levelFinancePool, err := levelfinance.NewPoolSimulator(entity.Pool{
		Address:  "0x73c3a78e5ff0d216a50b11d51b262ca839fcfe17",
		Exchange: "level-finance",
		Type:     "level-finance",
		Reserves: entity.PoolReserves{"779313577917429145001279", "1386752187018865380131"},
		Tokens: []*entity.PoolToken{
			{
				Address:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
				Decimals: 18,
			},
			{
				Address:  "0x55d398326f99059ff775485246999027b3197955",
				Decimals: 18,
			},
		},
		Extra: "{\"oracle\":\"0x347a868537c96650608b0C38a40d65fA8668bb61\",\"tokenInfos\":{\"0x0e09fabb73bd3ade0a17ecc321fd13a19e81ce82\":{\"isStableCoin\":false,\"targetWeight\":0,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":1515612264199876,\"reserveAmount\":0}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":100000,\"minPrice\":1525411950000,\"maxPrice\":1525411950000},\"0x2170ed0880ac9a755fd29b2688956bd959f933f8\":{\"isStableCoin\":false,\"targetWeight\":25000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":88409477196046758709,\"reserveAmount\":3758081405615144417},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":1273285914369262213583,\"reserveAmount\":19626301909258985334},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":25056795453556407839,\"reserveAmount\":4829823313157371109}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"minPrice\":1800186152960000,\"maxPrice\":1800186152960000},\"0x55d398326f99059ff775485246999027b3197955\":{\"isStableCoin\":true,\"targetWeight\":41000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":184341536385327250951544,\"reserveAmount\":29495041636499676110955},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":521599008999365338023501,\"reserveAmount\":14800885608374279815880},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":73373032532736556026234,\"reserveAmount\":35744357844692178501361}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":0,\"minPrice\":1000270130000,\"maxPrice\":1000270130000},\"0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c\":{\"isStableCoin\":false,\"targetWeight\":30800,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":5816697803176260078,\"reserveAmount\":3383010407777685588},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":111906233603518278521,\"reserveAmount\":4501534065981811378},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":4938928540475400967,\"reserveAmount\":3832600695744573765}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"minPrice\":34699370809700000,\"maxPrice\":34699370809700000},\"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c\":{\"isStableCoin\":false,\"targetWeight\":3000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":359302236490299536203,\"reserveAmount\":103226605503390418504},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":454243991764384413972,\"reserveAmount\":7741757526752034806},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":264772809488184251242,\"reserveAmount\":140152480740415789817}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"minPrice\":230162156910000,\"maxPrice\":230162156910000},\"0xe9e7cea3dedca5984780bafc599bd69add087d56\":{\"isStableCoin\":true,\"targetWeight\":0,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":4351267527855370,\"reserveAmount\":0}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":0,\"minPrice\":1000370160000,\"maxPrice\":1000370160000}},\"totalWeight\":99800,\"virtualPoolValue\":7631639747905684688745700698456428608,\"stableCoinBaseSwapFee\":1000000,\"stableCoinTaxBasisPoint\":5000000,\"baseSwapFee\":25000000,\"taxBasisPoint\":40000000,\"daoFee\":5500000000}",
	})

	assert.Nil(t, err)

	result, err := levelFinancePool.CalcAmountOut(
		pool.TokenAmount{
			Token:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		"0x55d398326f99059ff775485246999027b3197955",
	)

	assert.Nil(t, err)
	assert.Equal(t, "1789789741024165775187", result.TokenAmountOut.Amount.String())
}
