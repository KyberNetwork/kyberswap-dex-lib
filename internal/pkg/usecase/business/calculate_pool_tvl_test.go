package business

import (
	"context"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func floatRatio(s1, s2 string) *big.Float {
	f1, ok := new(big.Float).SetString(s1)
	if !ok {
		panic("not float:" + s1)
	}
	f2, ok := new(big.Float).SetString(s2)
	if !ok {
		panic("not float:" + s2)
	}
	return new(big.Float).Quo(f1, f2)
}

func TestCalculateTVL(t *testing.T) {

	prices := map[string]*routerEntity.OnchainPrice{
		// polygon, quote=wmatic
		"0x03b54a6e9a984069379fae1a4fc4dbae93b3bccd": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "107982486527405436"),
				Sell: floatRatio("500000000000000000000", "108113490068707647"),
			},
		},
		"0x7ceb23fd6bc0add59e62ac25578270cff1b9f619": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "125438368340517111"),
				Sell: floatRatio("500000000000000000000", "125567287703903826"),
			},
		},
		"0xdc31233e09f3bf5bfe5c10da2014677c23b6894c": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "123488826095110521"),
				Sell: floatRatio("500000000000000000000", "123929598688426320"),
			},
		},
		"0x27f8d03b3a2196956ed754badc28d73be8830a6e": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "6302039740113774694"),
				Sell: nil,
			},
		},
		"0x1a13f4ca1d028320a707d99520abfefca3998b7f": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "36075951"),
				Sell: nil,
			},
		},
		"0x60d55f02a771d515e077c9c2403a1ef324885cec": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("500000000000000000000", "1460667"),
				Sell: nil,
			},
		},

		// ethereum, quote=weth
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"0xd9a442856c234a39a81a089c06451ebaa4306a72": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("150000000000000000", "151851020595749149"),
				Sell: floatRatio("150000000000000000", "151783585912295583"),
			},
		},
	}

	testcases := []struct {
		poolStr string
		tvl     float64
	}{
		// polygon pools, native is wMatic -> reserveNative should be in the same order of magtitude as reserveUsd
		{
			`{"address":"0xdc31233e09f3bf5bfe5c10da2014677c23b6894c","reserveUsd":7193.863367447958,"amplifiedTvl":7193.863367447958,"exchange":"balancer-v2-composable-stable","type":"balancer-v2-composable-stable","timestamp":1712891812,"reserves":["1542587670201991059","252813278536006178","2596148429267413812242977489125310"],"tokens":[{"address":"0x03b54a6e9a984069379fae1a4fc4dbae93b3bccd","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0x7ceb23fd6bc0add59e62ac25578270cff1b9f619","name":"","symbol":"","decimals":0,"weight":0,"swappable":true},{"address":"0xdc31233e09f3bf5bfe5c10da2014677c23b6894c","name":"","symbol":"","decimals":0,"weight":0,"swappable":true}],"extra":"{\"canNotUpdateTokenRates\":false,\"scalingFactors\":[\"1163261138292932841\",\"1000000000000000000\",\"1000000000000000000\"],\"bptTotalSupply\":\"2596148429267415826563233829245265\",\"amp\":\"5000000\",\"lastJoinExit\":{\"lastJoinExitAmplification\":\"5000000\",\"lastPostJoinExitInvariant\":\"2046020665983438353\"},\"rateProviders\":[\"0x693a9aca2f7b699bbd3d55d980ac8a5d7a66868b\",\"0x0000000000000000000000000000000000000000\",\"0x0000000000000000000000000000000000000000\"],\"tokenRateCaches\":[{\"rate\":\"1163261138292932841\",\"oldRate\":\"1162941713456930360\",\"duration\":\"21600\",\"expires\":\"1712913412\"},{\"rate\":null,\"oldRate\":null,\"duration\":null,\"expires\":null},{\"rate\":null,\"oldRate\":null,\"duration\":null,\"expires\":null}],\"swapFeePercentage\":\"500000000000000\",\"protocolFeePercentageCache\":{\"0\":\"0\",\"2\":\"0\"},\"isTokenExemptFromYieldProtocolFee\":[false,false,false],\"isExemptFromYieldProtocolFee\":false,\"inRecoveryMode\":false,\"paused\":false}","staticExtra":"{\"poolId\":\"0xdc31233e09f3bf5bfe5c10da2014677c23b6894c000000000000000000000c23\",\"poolType\":\"ComposableStable\",\"poolTypeVer\":5,\"bptIndex\":2,\"scalingFactors\":[\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\",\"0xde0b6b3a7640000\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"}`,
			8145.6421,
		},

		// skipped, amDAI price is so wrong right now
		// {
		// 	`{"address":"0x445fe580ef8d70ff569ab36e80c647af338db351","reserveUsd":7926044.148083694,"amplifiedTvl":7926044.148083694,"exchange":"curve","type":"curve-aave","timestamp":1712896418,"reserves":["1981417630713856686472173","2781811090444","3134387404226","7017390969891275648684750"],"tokens":[{"address":"0x27f8d03b3a2196956ed754badc28d73be8830a6e","name":"","symbol":"","decimals":0,"weight":1,"swappable":true},{"address":"0x1a13f4ca1d028320a707d99520abfefca3998b7f","name":"","symbol":"","decimals":0,"weight":1,"swappable":true},{"address":"0x60d55f02a771d515e077c9c2403a1ef324885cec","name":"","symbol":"","decimals":0,"weight":1,"swappable":true}],"extra":"{\"initialA\":\"100000\",\"futureA\":\"200000\",\"initialATime\":1620408998,\"futureATime\":1621013782,\"swapFee\":\"3000000\",\"adminFee\":\"5000000000\",\"offpegFeeMultiplier\":\"20000000000\"}","staticExtra":"{\"lpToken\":\"0xe7a24ef0c5e95ffb0f6684b813a78f2a3ad7d171\",\"underlyingTokens\":[\"0x8f3cf7ad23cd3cadbd9735aff958023239c6a063\",\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\",\"0xc2132d05d31c914a87c6611c10748aeb04b58e8f\"],\"precisionMultipliers\":[\"1\",\"1000000000000\",\"1000000000000\"]}"}`,
		// 	,
		// },

		// ethereum
		{
			`{"address":"0x0fd9444ebfcbf5b2be47a71d1dae17a43d341e7b","reserveUsd":165.78178329088746,"amplifiedTvl":6.073888839359977e+40,"swapFee":0.003,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1712896385,"reserves":["19999979999999989","27177572251738528"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"","symbol":"","decimals":18,"weight":50,"swappable":true},{"address":"0xd9a442856c234a39a81a089c06451ebaa4306a72","name":"","symbol":"","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":3000000000000000,\"protocolFeeRatio\":0,\"activeTick\":7,\"binCounter\":11,\"bins\":{\"1\":{\"reserveA\":2518329576855117,\"reserveB\":0,\"lowerTick\":2,\"kind\":0,\"mergeId\":0},\"10\":{\"reserveA\":0,\"reserveB\":2855070905259045,\"lowerTick\":11,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":0,\"reserveB\":2499513693565321,\"lowerTick\":12,\"kind\":0,\"mergeId\":0},\"2\":{\"reserveA\":2876563358402835,\"reserveB\":0,\"lowerTick\":3,\"kind\":0,\"mergeId\":0},\"3\":{\"reserveA\":3466847490748800,\"reserveB\":0,\"lowerTick\":4,\"kind\":0,\"mergeId\":0},\"4\":{\"reserveA\":4439908984252495,\"reserveB\":0,\"lowerTick\":5,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":6044378945078618,\"reserveB\":0,\"lowerTick\":6,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":653951644662124,\"reserveB\":7976089159664525,\"lowerTick\":7,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":0,\"reserveB\":5999217926503902,\"lowerTick\":8,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":0,\"reserveB\":4406735880131370,\"lowerTick\":9,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":0,\"reserveB\":3440944686614365,\"lowerTick\":10,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"10\":{\"0\":9},\"11\":{\"0\":10},\"12\":{\"0\":11},\"2\":{\"0\":1},\"3\":{\"0\":2},\"4\":{\"0\":3},\"5\":{\"0\":4},\"6\":{\"0\":5},\"7\":{\"0\":6},\"8\":{\"0\":7},\"9\":{\"0\":8}},\"binMap\":{\"0\":300239975158016},\"binMapHex\":{\"0\":300239975158016},\"liquidity\":17316584538384865422,\"sqrtPriceX96\":1003543721020663876,\"minBinMapIndex\":0,\"maxBinMapIndex\":0}","staticExtra":"{\"tickSpacing\":10}"}`,
			0.0468,
		},
	}

	logger.SetLogLevel("debug")
	for _, tc := range testcases {
		var ent entity.Pool
		err := json.Unmarshal([]byte(tc.poolStr), &ent)
		require.Nil(t, err)
		t.Run(ent.Address, func(t *testing.T) {
			tvl, err := CalculatePoolTVL(context.TODO(), &ent, prices)
			require.Nil(t, err)
			assert.InDelta(t, tc.tvl, tvl, 0.0001)
		})
	}
}
