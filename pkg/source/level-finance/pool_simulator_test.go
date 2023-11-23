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
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x2170ed0880ac9a755fd29b2688956bd959f933f8",
				Amount: bignumber.NewBig10("1000000000000000000"),
			},
			TokenOut: "0x55d398326f99059ff775485246999027b3197955",
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, "2047979446915897120965", result.TokenAmountOut.Amount.String())
}

func TestCalcAmountOutStalbeToken(t *testing.T) {
	levelFinancePool, err := levelfinance.NewPoolSimulator(entity.Pool{
		Address:  "0x32b7bf19cb8b95c27e644183837813d4b595dcc6",
		Exchange: "level-finance",
		Type:     "level-finance",
		Reserves: entity.PoolReserves{"1522182766006", "466620964426"},
		Tokens: []*entity.PoolToken{
			{
				Address:  "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
				Decimals: 18,
			},
			{
				Address:  "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
				Decimals: 18,
			},
		},
		Extra: "{\"oracle\":\"0x82B585a8F15701BBD671850f0a9F1feE57a8DCB5\",\"tokenInfos\":{\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"isStableCoin\":false,\"targetWeight\":22000,\"trancheAssets\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":{\"poolAmount\":256502946,\"reserveAmount\":3548375},\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":{\"poolAmount\":3687504900,\"reserveAmount\":247367443},\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":{\"poolAmount\":219402575,\"reserveAmount\":3104828}},\"riskFactor\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":0,\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":1,\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":0},\"totalRiskFactor\":1,\"maxLiquidity\":0,\"minPrice\":373463781553900000000000000,\"maxPrice\":373463781553900000000000000},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"isStableCoin\":false,\"targetWeight\":35000,\"trancheAssets\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":{\"poolAmount\":17458989988881172,\"reserveAmount\":17458989988881172},\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":{\"poolAmount\":897706368784770877858,\"reserveAmount\":53366506867359689166},\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":{\"poolAmount\":13579214435796465,\"reserveAmount\":13579214435796465}},\"riskFactor\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":0,\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":1,\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":0},\"totalRiskFactor\":1,\"maxLiquidity\":0,\"minPrice\":2068413620680000,\"maxPrice\":2068413620680000},\"0x912ce59144191c1204e64559fe8253a0e49e6548\":{\"isStableCoin\":false,\"targetWeight\":1000,\"trancheAssets\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":{\"poolAmount\":0,\"reserveAmount\":0},\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":{\"poolAmount\":53181849675852502167367,\"reserveAmount\":50255467926189998076758},\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":{\"poolAmount\":0,\"reserveAmount\":0}},\"riskFactor\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":0,\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":1,\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":0},\"totalRiskFactor\":1,\"maxLiquidity\":100000000000000000000000,\"minPrice\":1025605090000,\"maxPrice\":1025605090000},\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\":{\"isStableCoin\":true,\"targetWeight\":10000,\"trancheAssets\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":{\"poolAmount\":37320430,\"reserveAmount\":0},\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":{\"poolAmount\":466546323566,\"reserveAmount\":55250639218},\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":{\"poolAmount\":37320430,\"reserveAmount\":0}},\"riskFactor\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":0,\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":0,\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":0},\"totalRiskFactor\":0,\"maxLiquidity\":0,\"minPrice\":1000099990000000000000000,\"maxPrice\":1000099990000000000000000},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"isStableCoin\":true,\"targetWeight\":32000,\"trancheAssets\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":{\"poolAmount\":49873143222,\"reserveAmount\":49867628734},\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":{\"poolAmount\":1433924520125,\"reserveAmount\":33543210917},\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":{\"poolAmount\":38385102659,\"reserveAmount\":38379588171}},\"riskFactor\":{\"0x502697AF336F7413Bb4706262e7C506Edab4f3B9\":0,\"0x5573405636F4b895E511C9C54aAfbefa0E7Ee458\":0,\"0xb076f79f8D1477165E2ff8fa99930381FB7d94c1\":0},\"totalRiskFactor\":0,\"maxLiquidity\":0,\"minPrice\":1000200010000000000000000,\"maxPrice\":1000200010000000000000000}},\"totalWeight\":100000,\"virtualPoolValue\":5436646567387990705054968055050223481,\"stableCoinBaseSwapFee\":1000000,\"stableCoinTaxBasisPoint\":5000000,\"baseSwapFee\":25000000,\"taxBasisPoint\":60000000,\"daoFee\":5500000000}",
	})

	assert.Nil(t, err)

	result, err := levelFinancePool.CalcAmountOut(
		pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
				Amount: bignumber.NewBig10("100000000"),
			},
			TokenOut: "0xaf88d065e77c8cc2239327c5edb3a432268e5831",
		},
	)

	assert.Nil(t, err)
	assert.Equal(t, "99992913", result.TokenAmountOut.Amount.String())
}
