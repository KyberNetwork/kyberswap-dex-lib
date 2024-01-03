package maverickv1_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var maverickPool, err = maverickv1.NewPoolSimulator(entity.Pool{
	Tokens: []*entity.PoolToken{
		{
			Address:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Decimals: 6,
		},
		{
			Address:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Decimals: 18,
		},
	},
	Extra:       "{\"fee\":400000000000000,\"protocolFeeRatio\":0,\"activeTick\":379,\"binCounter\":122,\"bins\":{\"1\":{\"reserveA\":17453201008635512394640,\"reserveB\":0,\"lowerTick\":375,\"kind\":0,\"mergeId\":0},\"10\":{\"reserveA\":0,\"reserveB\":632634315831505118,\"lowerTick\":384,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":0,\"reserveB\":568174206788937614,\"lowerTick\":385,\"kind\":0,\"mergeId\":0},\"12\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":1,\"mergeId\":0},\"13\":{\"reserveA\":0,\"reserveB\":24179624473369718938,\"lowerTick\":384,\"kind\":1,\"mergeId\":0},\"15\":{\"reserveA\":7448514891591076678798,\"reserveB\":4960078724015931105,\"lowerTick\":379,\"kind\":3,\"mergeId\":0},\"16\":{\"reserveA\":1631153083876778654919,\"reserveB\":0,\"lowerTick\":373,\"kind\":0,\"mergeId\":0},\"17\":{\"reserveA\":14604684077518837517486,\"reserveB\":0,\"lowerTick\":374,\"kind\":0,\"mergeId\":0},\"2\":{\"reserveA\":21271280872855300434039,\"reserveB\":0,\"lowerTick\":376,\"kind\":0,\"mergeId\":0},\"23\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":2,\"mergeId\":0},\"25\":{\"reserveA\":426150857836353291022,\"reserveB\":0,\"lowerTick\":373,\"kind\":2,\"mergeId\":0},\"3\":{\"reserveA\":25965452451154862154091,\"reserveB\":0,\"lowerTick\":377,\"kind\":0,\"mergeId\":0},\"32\":{\"reserveA\":0,\"reserveB\":30757567785565755,\"lowerTick\":386,\"kind\":0,\"mergeId\":0},\"34\":{\"reserveA\":0,\"reserveB\":0,\"lowerTick\":379,\"kind\":3,\"mergeId\":0},\"37\":{\"reserveA\":973003208635914825127,\"reserveB\":0,\"lowerTick\":372,\"kind\":0,\"mergeId\":0},\"4\":{\"reserveA\":22309339486762762891065,\"reserveB\":0,\"lowerTick\":378,\"kind\":0,\"mergeId\":0},\"41\":{\"reserveA\":28773102441950282148,\"reserveB\":0,\"lowerTick\":371,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":596733989717113121,\"reserveB\":0,\"lowerTick\":369,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":993638242463261223,\"reserveB\":0,\"lowerTick\":370,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":9361000987001231865441,\"reserveB\":6233632141023853827,\"lowerTick\":379,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":968206263636201246648,\"reserveB\":0,\"lowerTick\":376,\"kind\":2,\"mergeId\":0},\"53\":{\"reserveA\":2153035881950200782250,\"reserveB\":1433739158145507548,\"lowerTick\":379,\"kind\":2,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":10375023547668913537,\"lowerTick\":380,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":0,\"reserveB\":9381324932473456976,\"lowerTick\":381,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":0,\"reserveB\":8271837842446867401,\"lowerTick\":382,\"kind\":0,\"mergeId\":0},\"84\":{\"reserveA\":0,\"reserveB\":821816663509517,\"lowerTick\":387,\"kind\":0,\"mergeId\":0},\"87\":{\"reserveA\":0,\"reserveB\":140916273379942,\"lowerTick\":388,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":0,\"reserveB\":732155171838690157,\"lowerTick\":383,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"369\":{\"0\":47},\"370\":{\"0\":48},\"371\":{\"0\":41},\"372\":{\"0\":37},\"373\":{\"0\":16,\"2\":25},\"374\":{\"0\":17},\"375\":{\"0\":1},\"376\":{\"0\":2,\"2\":50},\"377\":{\"0\":3},\"378\":{\"0\":4},\"379\":{\"0\":5,\"1\":12,\"2\":53,\"3\":15},\"380\":{\"0\":6},\"381\":{\"0\":7},\"382\":{\"0\":8},\"383\":{\"0\":9},\"384\":{\"0\":10,\"1\":13},\"385\":{\"0\":11},\"386\":{\"0\":32},\"387\":{\"0\":84},\"388\":{\"0\":87}},\"binMap\":{\"5\":7721018714868875516017241010155757617493946277325927722722110067420054945792,\"6\":69907},\"liquidity\":60474424673766490639024,\"sqrtPriceX96\":42792872587486068317}",
	StaticExtra: "{\"tickSpacing\":198}",
})

func TestPoolCalcAmountOut(t *testing.T) {
	assert.Nil(t, err)

	result, err := maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		Limit:    nil,
	})

	assert.Nil(t, err)
	assert.Equal(t, "1829711602", result.TokenAmountOut.Amount.String())

	//var bins = map[string]maverickv1.Bin{
	//	"1": {
	//		ReserveA:  bignumber.NewBig10("17453201008635512394640"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("375"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"2": {
	//		ReserveA:  bignumber.NewBig10("21271280872855300434039"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("376"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"3": {
	//		ReserveA:  bignumber.NewBig10("25965452451154862154091"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("377"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"4": {
	//		ReserveA:  bignumber.NewBig10("22309339486762762891065"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("378"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"5": {
	//		ReserveA:  bignumber.NewBig10("9361000987001231865441"),
	//		ReserveB:  bignumber.NewBig10("6233632141023853827"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"6": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("10375023547668914000"),
	//		LowerTick: bignumber.NewBig10("380"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"7": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("9381324932473457000"),
	//		LowerTick: bignumber.NewBig10("381"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"8": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("8271837842446867000"),
	//		LowerTick: bignumber.NewBig10("382"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"9": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("732155171838690200"),
	//		LowerTick: bignumber.NewBig10("383"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"10": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("632634315831505200"),
	//		LowerTick: bignumber.NewBig10("384"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"11": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("568174206788937600"),
	//		LowerTick: bignumber.NewBig10("385"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"12": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("1"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"13": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("24179624473369720000"),
	//		LowerTick: bignumber.NewBig10("384"),
	//		Kind:      bignumber.NewBig10("1"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"15": {
	//		ReserveA:  bignumber.NewBig10("7448514891591076678798"),
	//		ReserveB:  bignumber.NewBig10("4960078724015931105"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("3"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"16": {
	//		ReserveA:  bignumber.NewBig10("1631153083876778654919"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("373"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"17": {
	//		ReserveA:  bignumber.NewBig10("14604684077518837517486"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("374"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"23": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("2"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"25": {
	//		ReserveA:  bignumber.NewBig10("426150857836353300000"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("373"),
	//		Kind:      bignumber.NewBig10("2"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"32": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("30757567785565756"),
	//		LowerTick: bignumber.NewBig10("386"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"34": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("3"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"37": {
	//		ReserveA:  bignumber.NewBig10("973003208635914800000"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("372"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"41": {
	//		ReserveA:  bignumber.NewBig10("28773102441950280000"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("371"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"47": {
	//		ReserveA:  bignumber.NewBig10("596733989717113100"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("369"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"48": {
	//		ReserveA:  bignumber.NewBig10("993638242463261200"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("370"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"50": {
	//		ReserveA:  bignumber.NewBig10("968206263636201300000"),
	//		ReserveB:  bignumber.NewBig10("0"),
	//		LowerTick: bignumber.NewBig10("376"),
	//		Kind:      bignumber.NewBig10("2"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"53": {
	//		ReserveA:  bignumber.NewBig10("2153035881950200782250"),
	//		ReserveB:  bignumber.NewBig10("1433739158145507548"),
	//		LowerTick: bignumber.NewBig10("379"),
	//		Kind:      bignumber.NewBig10("2"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"84": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("821816663509517"),
	//		LowerTick: bignumber.NewBig10("387"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//	"87": {
	//		ReserveA:  bignumber.NewBig10("0"),
	//		ReserveB:  bignumber.NewBig10("140916273379942"),
	//		LowerTick: bignumber.NewBig10("388"),
	//		Kind:      bignumber.NewBig10("0"),
	//		MergeID:   bignumber.NewBig10("0"),
	//	},
	//}
	//
	//var binPositions = map[string]map[string]*big.Int{
	//	"369": {
	//		"0": bignumber.NewBig10("47"),
	//	},
	//	"370": {
	//		"0": bignumber.NewBig10("48"),
	//	},
	//	"371": {
	//		"0": bignumber.NewBig10("41"),
	//	},
	//	"372": {
	//		"0": bignumber.NewBig10("37"),
	//	},
	//	"373": {
	//		"0": bignumber.NewBig10("16"),
	//		"2": bignumber.NewBig10("25"),
	//	},
	//	"374": {
	//		"0": bignumber.NewBig10("17"),
	//	},
	//	"375": {
	//		"0": bignumber.NewBig10("1"),
	//	},
	//	"376": {
	//		"0": bignumber.NewBig10("2"),
	//		"2": bignumber.NewBig10("50"),
	//	},
	//	"377": {
	//		"0": bignumber.NewBig10("3"),
	//	},
	//	"378": {
	//		"0": bignumber.NewBig10("4"),
	//	},
	//	"379": {
	//		"0": bignumber.NewBig10("5"),
	//		"1": bignumber.NewBig10("12"),
	//		"2": bignumber.NewBig10("53"),
	//		"3": bignumber.NewBig10("34"),
	//	},
	//	"380": {
	//		"0": bignumber.NewBig10("6"),
	//	},
	//	"381": {
	//		"0": bignumber.NewBig10("7"),
	//	},
	//	"382": {
	//		"0": bignumber.NewBig10("8"),
	//	},
	//	"383": {
	//		"0": bignumber.NewBig10("9"),
	//	},
	//	"384": {
	//		"0": bignumber.NewBig10("10"),
	//		"1": bignumber.NewBig10("13"),
	//	},
	//	"385": {
	//		"0": bignumber.NewBig10("11"),
	//	},
	//	"386": {
	//		"0": bignumber.NewBig10("32"),
	//	},
	//	"387": {
	//		"0": bignumber.NewBig10("84"),
	//	},
	//	"388": {
	//		"0": bignumber.NewBig10("87"),
	//	},
	//}
	//
	//var binMap = map[string]*big.Int{
	//	"5": bignumber.NewBig10("7721018714868875516017241010155757617493946277325927722722110067420054945792"),
	//}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err = maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: bignumber.NewBig10("1000000000000000000"),
			},
			TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Limit:    nil,
		})
	}
}

func BenchmarkNextActive(b *testing.B) {
	poolRedis := "{\"address\":\"0x012245db1919bbb6d727b9ce787c3169f963a898\",\"reserveUsd\":1.3045263641356901,\"amplifiedTvl\":8.068244485638408e+40,\"swapFee\":0.00008,\"exchange\":\"maverick-v1\",\"type\":\"maverick-v1\",\"timestamp\":1704265258,\"reserves\":[\"1171608824435142257\",\"76716840233381\"],\"tokens\":[{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"fee\\\":80000000000000,\\\"protocolFeeRatio\\\":0,\\\"activeTick\\\":1502,\\\"binCounter\\\":36,\\\"bins\\\":{\\\"1\\\":{\\\"reserveA\\\":314516285521548227,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"2\\\":{\\\"reserveA\\\":191245215895503843,\\\"reserveB\\\":0,\\\"lowerTick\\\":1501,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"3\\\":{\\\"reserveA\\\":114504301688631519,\\\"reserveB\\\":963774576010,\\\"lowerTick\\\":1502,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"31\\\":{\\\"reserveA\\\":108753991059500386,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":2,\\\"mergeId\\\":0},\\\"32\\\":{\\\"reserveA\\\":25486000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1495,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"33\\\":{\\\"reserveA\\\":42126000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1496,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"34\\\":{\\\"reserveA\\\":69628000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1497,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"35\\\":{\\\"reserveA\\\":115099589497454909,\\\"reserveB\\\":0,\\\"lowerTick\\\":1498,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"36\\\":{\\\"reserveA\\\":190249440772503320,\\\"reserveB\\\":0,\\\"lowerTick\\\":1499,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"4\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":38435140947772,\\\"lowerTick\\\":1503,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"5\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":23251195184809,\\\"lowerTick\\\":1504,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"6\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":14066729524731,\\\"lowerTick\\\":1505,\\\"kind\\\":0,\\\"mergeId\\\":0}},\\\"binPositions\\\":{\\\"1495\\\":{\\\"0\\\":32},\\\"1496\\\":{\\\"0\\\":33},\\\"1497\\\":{\\\"0\\\":34},\\\"1498\\\":{\\\"0\\\":35},\\\"1499\\\":{\\\"0\\\":36},\\\"1500\\\":{\\\"0\\\":1,\\\"2\\\":31},\\\"1501\\\":{\\\"0\\\":2},\\\"1502\\\":{\\\"0\\\":3},\\\"1503\\\":{\\\"0\\\":4},\\\"1504\\\":{\\\"0\\\":5},\\\"1505\\\":{\\\"0\\\":6}},\\\"binMap\\\":{\\\"23\\\":5807506497971120465074964654080854589440},\\\"liquidity\\\":1087229757983496926,\\\"sqrtPriceX96\\\":42831515231783862772}\",\"staticExtra\":\"{\\\"tickSpacing\\\":50}\"}"
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(b, err)

	maverickPool, err := maverickv1.NewPoolSimulator(poolEnt)
	require.Nil(b, err)

	for i := 0; i < b.N; i++ {
		_, _ = maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Amount: bignumber.NewBig10("1000000000"),
			},
			TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Limit:    nil,
		})
	}
}
