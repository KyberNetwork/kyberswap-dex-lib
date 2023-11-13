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
		Reserves: entity.PoolReserves{"1166858757615124990262", "1492325299408887313876906"},
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
		Extra: "{\"oracle\":\"0x347a868537c96650608b0C38a40d65fA8668bb61\",\"tokenInfos\":{\"0x0e09fabb73bd3ade0a17ecc321fd13a19e81ce82\":{\"isStableCoin\":false,\"targetWeight\":0,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":1515612264199876,\"reserveAmount\":0}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":30000,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":70000},\"totalRiskFactor\":100000,\"maxLiquidity\":30000000000000000000000,\"minPrice\":2154937010000,\"maxPrice\":2154937010000},\"0x2170ed0880ac9a755fd29b2688956bd959f933f8\":{\"isStableCoin\":false,\"targetWeight\":25000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":3520297887173991266,\"reserveAmount\":3107655192521716074},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":1158814358081360824799,\"reserveAmount\":42345363161251319034},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":4524101646590174197,\"reserveAmount\":3993561039180106090}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":1,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"maxLiquidity\":0,\"minPrice\":2058445054830000,\"maxPrice\":2058445054830000},\"0x55d398326f99059ff775485246999027b3197955\":{\"isStableCoin\":true,\"targetWeight\":41000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":7505022308831360591917,\"reserveAmount\":4050878017979352186019},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":1473156522410332348533984,\"reserveAmount\":2403305927215349054403},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":11655141347659753989297,\"reserveAmount\":7545688712255318153812}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":0,\"maxLiquidity\":0,\"minPrice\":1000435010000,\"maxPrice\":1000435010000},\"0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c\":{\"isStableCoin\":false,\"targetWeight\":30800,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":10432339777844960805,\"reserveAmount\":3314299896355151925},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":90676698266657793134,\"reserveAmount\":9341443798917242741},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":8262498579033104067,\"reserveAmount\":3758402928161385148}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":1,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"maxLiquidity\":0,\"minPrice\":37039615597090000,\"maxPrice\":37039615597090000},\"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c\":{\"isStableCoin\":false,\"targetWeight\":3000,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":86848770936987931277,\"reserveAmount\":57288002510698643098},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":147953546466901169449,\"reserveAmount\":17249934251612702673},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":112335861432756827835,\"reserveAmount\":60714813704662884006}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":1,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":1,\"maxLiquidity\":5000000000000000000000,\"minPrice\":246507186460000,\"maxPrice\":246507186460000},\"0xe9e7cea3dedca5984780bafc599bd69add087d56\":{\"isStableCoin\":true,\"targetWeight\":0,\"trancheAssets\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":{\"poolAmount\":0,\"reserveAmount\":0},\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":{\"poolAmount\":4351267527855370,\"reserveAmount\":0}},\"riskFactor\":{\"0x4265af66537F7BE1Ca60Ca6070D97531EC571BDd\":0,\"0xB5C42F84Ab3f786bCA9761240546AA9cEC1f8821\":0,\"0xcC5368f152453D497061CB1fB578D2d3C54bD0A0\":0},\"totalRiskFactor\":0,\"maxLiquidity\":0,\"minPrice\":999934790000,\"maxPrice\":999934790000}},\"totalWeight\":99800,\"virtualPoolValue\":7858172229621476737899266820565538095,\"stableCoinBaseSwapFee\":1000000,\"stableCoinTaxBasisPoint\":5000000,\"baseSwapFee\":25000000,\"taxBasisPoint\":40000000,\"daoFee\":5500000000}",
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
	assert.Equal(t, "2047979446915897120965", result.TokenAmountOut.Amount.String())
}
