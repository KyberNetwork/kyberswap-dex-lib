package business

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateAmplifiedTVL(t *testing.T) {
	prices := map[string]*routerEntity.OnchainPrice{
		// ethereum, quote=weth
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("150000000000000000", "129226320769630792"),
				Sell: floatRatio("150000000000000000", "129122116394826877"),
			},
		},

		// optimism
		"0x4200000000000000000000000000000000000006": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  big.NewFloat(1),
				Sell: big.NewFloat(1),
			},
		},
		"0x7f5c764cbc14f9669b88837ca1490cca17c31607": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("150000000000000000", "465021449"),
				Sell: floatRatio("150000000000000000", "465162980"),
			},
		},
		"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58": {
			Decimals: 18,
			NativePriceRaw: routerEntity.Price{
				Buy:  floatRatio("150000000000000000", "465167378"),
				Sell: floatRatio("150000000000000000", "465683118"),
			},
		},
	}

	testcases := []struct {
		poolRedis string
		expATVL   float64
	}{
		// ethereum
		{`{"address":"0xbd278792260a68ee81a42adba23befdba87e30eb","reserveUsd":23670.769320999876,"amplifiedTvl":7.422774029174093e+41,"swapFee":0.0001,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1713233418,"reserves":["2067211957590464268","5270326964488675348"],"tokens":[{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","name":"","symbol":"","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"","symbol":"","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":100000000000000,\"protocolFeeRatio\":0,\"activeTick\":-8,\"binCounter\":52,\"bins\":{\"12\":{\"reserveA\":0,\"reserveB\":4242237013037562,\"lowerTick\":-7,\"kind\":3,\"mergeId\":0},\"43\":{\"reserveA\":0,\"reserveB\":1000000000000000000,\"lowerTick\":343,\"kind\":0,\"mergeId\":0},\"44\":{\"reserveA\":0,\"reserveB\":978339781816359,\"lowerTick\":693,\"kind\":0,\"mergeId\":0},\"45\":{\"reserveA\":0,\"reserveB\":957148728684,\"lowerTick\":1043,\"kind\":0,\"mergeId\":0},\"46\":{\"reserveA\":0,\"reserveB\":936416678,\"lowerTick\":1393,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":0,\"reserveB\":916133,\"lowerTick\":1743,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":1000000000000000000,\"reserveB\":0,\"lowerTick\":-357,\"kind\":0,\"mergeId\":0},\"49\":{\"reserveA\":978339781816359,\"reserveB\":0,\"lowerTick\":-707,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":1066232659722586378,\"reserveB\":1524627518553741025,\"lowerTick\":-8,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":957148728684,\"reserveB\":0,\"lowerTick\":-1057,\"kind\":0,\"mergeId\":0},\"51\":{\"reserveA\":936416678,\"reserveB\":0,\"lowerTick\":-1407,\"kind\":0,\"mergeId\":0},\"52\":{\"reserveA\":916133,\"reserveB\":0,\"lowerTick\":-1757,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":2740477911054018869,\"lowerTick\":-7,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"-1057\":{\"0\":50},\"-1407\":{\"0\":51},\"-1757\":{\"0\":52},\"-357\":{\"0\":48},\"-7\":{\"0\":6,\"3\":12},\"-707\":{\"0\":49},\"-8\":{\"0\":5},\"1043\":{\"0\":45},\"1393\":{\"0\":46},\"1743\":{\"0\":47},\"343\":{\"0\":43},\"693\":{\"0\":44}},\"binMap\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-12\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"-17\":21267647932558653966460912964485513216,\"-22\":16,\"-28\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"10\":6582018229284824168619876730229402019930943462534319453394436096,\"16\":75557863725914323419136,\"21\":100433627766186892221372630771322662657637687111424552206336,\"27\":1152921504606846976,\"5\":4951760157141521099596496896},\"binMapHex\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-11\":21267647932558653966460912964485513216,\"-16\":16,\"-1c\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"-c\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"10\":75557863725914323419136,\"15\":100433627766186892221372630771322662657637687111424552206336,\"1b\":1152921504606846976,\"5\":4951760157141521099596496896,\"a\":6582018229284824168619876730229402019930943462534319453394436096},\"liquidity\":259631256293446401185,\"sqrtPriceX96\":927965512348602641,\"minBinMapIndex\":-28,\"maxBinMapIndex\":27}","staticExtra":"{\"tickSpacing\":198}"}`,
			565.8220545687768},
		{`{"address":"0x109830a1aaad605bbf02a9dfa7b0b92ec2fb7daa","reserveUsd":23083842.250331726,"amplifiedTvl":1.2906142634758348e+57,"swapFee":100,"exchange":"uniswapv3","type":"uniswapv3","timestamp":1713232886,"reserves":["5295013391771434011842","1312300090791952654850"],"tokens":[{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","name":"Wrapped liquid staked Ether 2.0","symbol":"wstETH","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liquidity\":4905260927646506776145973,\"sqrtPriceX96\":85387168208687470790908522417,\"tick\":1497,\"ticks\":[{\"index\":-887272,\"liquidityGross\":1631845377254394488,\"liquidityNet\":1631845377254394488},{\"index\":356,\"liquidityGross\":104053785111247877,\"liquidityNet\":104053785111247877},{\"index\":887,\"liquidityGross\":232557099163221,\"liquidityNet\":232557099163221},{\"index\":907,\"liquidityGross\":232557099163221,\"liquidityNet\":-232557099163221},{\"index\":1098,\"liquidityGross\":4758053312817345187,\"liquidityNet\":4758053312817345187},{\"index\":1165,\"liquidityGross\":360594364322346894745,\"liquidityNet\":360594364322346894745},{\"index\":1283,\"liquidityGross\":170854122338412466859,\"liquidityNet\":170854122338412466859},{\"index\":1286,\"liquidityGross\":170854122338412466859,\"liquidityNet\":-170854122338412466859},{\"index\":1289,\"liquidityGross\":388430653403919838,\"liquidityNet\":388430653403919838},{\"index\":1298,\"liquidityGross\":4503599627370496,\"liquidityNet\":4503599627370496},{\"index\":1299,\"liquidityGross\":4503599627370496,\"liquidityNet\":-4503599627370496},{\"index\":1303,\"liquidityGross\":4767393453492560686851,\"liquidityNet\":4767393453492560686851},{\"index\":1304,\"liquidityGross\":12988107486219802790,\"liquidityNet\":12988107486219802790},{\"index\":1309,\"liquidityGross\":388430653403919838,\"liquidityNet\":-388430653403919838},{\"index\":1323,\"liquidityGross\":12988107486219802790,\"liquidityNet\":-12988107486219802790},{\"index\":1368,\"liquidityGross\":144891793848856447904,\"liquidityNet\":144891793848856447904},{\"index\":1372,\"liquidityGross\":4503599627370496,\"liquidityNet\":4503599627370496},{\"index\":1373,\"liquidityGross\":4503599627370496,\"liquidityNet\":-4503599627370496},{\"index\":1382,\"liquidityGross\":4503599627370496,\"liquidityNet\":4503599627370496},{\"index\":1383,\"liquidityGross\":4503599627370496,\"liquidityNet\":-4503599627370496},{\"index\":1448,\"liquidityGross\":20476139588820970572435,\"liquidityNet\":20476139588820970572435},{\"index\":1457,\"liquidityGross\":63510139830079766279421,\"liquidityNet\":63510139830079766279421},{\"index\":1462,\"liquidityGross\":20476139588820970572435,\"liquidityNet\":-20476139588820970572435},{\"index\":1472,\"liquidityGross\":62099469887018330673171,\"liquidityNet\":62099469887018330673171},{\"index\":1473,\"liquidityGross\":30826549417159087929,\"liquidityNet\":30826549417159087929},{\"index\":1474,\"liquidityGross\":5389828135553814540943,\"liquidityNet\":5389828135553814540943},{\"index\":1475,\"liquidityGross\":5298464631849982774415,\"liquidityNet\":5298464631849982774415},{\"index\":1477,\"liquidityGross\":1681025057818434330314,\"liquidityNet\":-1681025057818434330314},{\"index\":1485,\"liquidityGross\":50007371200730372746258,\"liquidityNet\":-11701569950051705735994},{\"index\":1487,\"liquidityGross\":24295946931080279413223,\"liquidityNet\":24295946931080279413223},{\"index\":1488,\"liquidityGross\":568905979220352646483491,\"liquidityNet\":568905979220352646483491},{\"index\":1492,\"liquidityGross\":699234586999753158131996,\"liquidityNet\":699163417794293205265764},{\"index\":1493,\"liquidityGross\":41518454898797788045943,\"liquidityNet\":41518454898797788045943},{\"index\":1494,\"liquidityGross\":1118287104065060475757299,\"liquidityNet\":1055797105441805892893209},{\"index\":1495,\"liquidityGross\":2387354515769989103737720,\"liquidityNet\":2387354515769989103737720},{\"index\":1498,\"liquidityGross\":3105884066864560252410893,\"liquidityNet\":1707497908844803509467471},{\"index\":1499,\"liquidityGross\":376893647949360478799203,\"liquidityNet\":376893647949360478799203},{\"index\":1500,\"liquidityGross\":46908283034351602586886,\"liquidityNet\":-46908283034351602586886},{\"index\":1501,\"liquidityGross\":1584092862936145228044141,\"liquidityNet\":-1584092862936145228044141},{\"index\":1502,\"liquidityGross\":2458853726657581534523503,\"liquidityNet\":-2458853726657581534523503},{\"index\":1505,\"liquidityGross\":2783940645954090531717206,\"liquidityNet\":-2783940645954090531717206},{\"index\":1506,\"liquidityGross\":24295946931080279413223,\"liquidityNet\":-24295946931080279413223},{\"index\":1510,\"liquidityGross\":24451365257189316279547,\"liquidityNet\":-24451365257189316279547},{\"index\":1517,\"liquidityGross\":61829114772261331949107,\"liquidityNet\":-61829114772261331949107},{\"index\":1527,\"liquidityGross\":6443199038016676016036023,\"liquidityNet\":6443199038016676016036023},{\"index\":1528,\"liquidityGross\":6443199038016676016036023,\"liquidityNet\":-6443199038016676016036023},{\"index\":1549,\"liquidityGross\":144891793848856447904,\"liquidityNet\":-144891793848856447904},{\"index\":1652,\"liquidityGross\":5923387144810227169,\"liquidityNet\":-5923387144810227169},{\"index\":1791,\"liquidityGross\":360594364322346894745,\"liquidityNet\":-360594364322346894745},{\"index\":2363,\"liquidityGross\":104053785111247877,\"liquidityNet\":-104053785111247877},{\"index\":27081,\"liquidityGross\":4767393453492560686851,\"liquidityNet\":-4767393453492560686851},{\"index\":887272,\"liquidityGross\":1631845377254394488,\"liquidityNet\":-1631845377254394488}]}","staticExtra":"{\"poolId\":\"0x109830a1aaad605bbf02a9dfa7b0b92ec2fb7daa\"}"}`,
			1.057182234689844e+07},
		{`{"address":"0x4370e48e610d2e02d3d091a9d79c8eb9a54c5b1c","reserveUsd":986512.1880237974,"amplifiedTvl":2.8406926067935803e+56,"swapFee":50,"exchange":"solidly-v3","type":"solidly-v3","timestamp":1713234601,"reserves":["62512009608749472875","249389515233930374665"],"tokens":[{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","name":"Wrapped liquid staked Ether 2.0","symbol":"wstETH","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true}],"extra":"{\"liquidity\":1087546934421704115010548,\"sqrtPriceX96\":85384644566379596234634507826,\"tickSpacing\":1,\"tick\":1496,\"ticks\":[{\"index\":1380,\"liquidityGross\":6331680774659780993,\"liquidityNet\":6331680774659780993},{\"index\":1436,\"liquidityGross\":4857666970279581058392,\"liquidityNet\":4857666970279581058392},{\"index\":1450,\"liquidityGross\":4863998651054240839385,\"liquidityNet\":-4863998651054240839385},{\"index\":1488,\"liquidityGross\":190323912053901855036809,\"liquidityNet\":190323912053901855036809},{\"index\":1491,\"liquidityGross\":199117888154065885424429,\"liquidityNet\":199117888154065885424429},{\"index\":1493,\"liquidityGross\":190323912053901855036809,\"liquidityNet\":-190323912053901855036809},{\"index\":1494,\"liquidityGross\":1286664822575770000434977,\"liquidityNet\":888429046267638229586119},{\"index\":1498,\"liquidityGross\":1087546934421704115010548,\"liquidityNet\":-1087546934421704115010548}]}"}`,
			2.3438820311444025e+06},

		// optimism
		{`{"address":"0x5feabb69432930c0271ba1991c55e6640ac8b388","reserveUsd":79.89203796522139,"amplifiedTvl":3.2366085615282165e+32,"swapFee":2000,"exchange":"iziswap","type":"iziswap","timestamp":1713235136,"reserves":["3658248613549940","68798763"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"WETH","symbol":"WETH","decimals":18,"weight":50,"swappable":true},{"address":"0x7f5c764cbc14f9669b88837ca1490cca17c31607","name":"USDC","symbol":"USDC","decimals":6,"weight":50,"swappable":true}],"extra":"{\"CurrentPoint\":-196003,\"PointDelta\":40,\"LeftMostPt\":-800000,\"RightMostPt\":800000,\"Fee\":2000,\"Liquidity\":73793537,\"LiquidityX\":479052,\"Liquidities\":[{\"LiqudityDelta\":497435646,\"Point\":-198040},{\"LiqudityDelta\":-925993119,\"Point\":-197640},{\"LiqudityDelta\":-283502,\"Point\":-197120},{\"LiqudityDelta\":-52770697,\"Point\":-194680},{\"LiqudityDelta\":-13,\"Point\":-194280}],\"LimitOrders\":[]}"}`,
			2.6504987469338684e-06},
		{`{"address":"0xa87d6ad50b113ca01933df263ddb55479bab8759","reserveUsd":22143.892605880603,"amplifiedTvl":1.8126787005103896e+41,"exchange":"zyberswap-v3","type":"algebra-v1","timestamp":1706772678,"reserves":["13294894449","8838858735"],"tokens":[{"address":"0x7f5c764cbc14f9669b88837ca1490cca17c31607","name":"USD Coin","symbol":"USDC","decimals":6,"weight":50,"swappable":true},{"address":"0x94b008aa00579c1307b0ef2c499ad98a8ce58e58","name":"Tether USD","symbol":"USDT","decimals":6,"weight":50,"swappable":true}],"extra":"{\"liquidity\":2287792441075,\"globalState\":{\"price\":79260951533741818140777192181,\"tick\":8,\"feeZto\":99,\"feeOtz\":99,\"timepoint_index\":2280,\"community_fee_token0\":100,\"community_fee_token1\":100,\"unlocked\":true},\"ticks\":[],\"tickSpacing\":60}","blockNumber":115586943}`,
			1475.1746599174974},
	}

	ctx := context.TODO()
	var poolEntity entity.Pool
	for _, tc := range testcases {
		err := json.Unmarshal([]byte(tc.poolRedis), &poolEntity)
		require.Nil(t, err)
		t.Run(poolEntity.Address, func(t *testing.T) {
			aTVL, useTvl, err := CalculatePoolAmplifiedTVL(ctx, &poolEntity, prices)
			require.Nil(t, err)
			require.False(t, useTvl)
			assert.InDelta(t, tc.expATVL, aTVL, 0.01)
		})
	}
}
