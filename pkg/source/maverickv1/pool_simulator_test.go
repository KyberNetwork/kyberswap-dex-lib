package maverickv1

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var maverickPool, err = NewPoolSimulator(entity.Pool{
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
	poolRedis := "{\"address\":\"0x012245db1919bbb6d727b9ce787c3169f963a898\",\"reserveUsd\":1.3045263641356901,\"amplifiedTvl\":8.068244485638408e+40,\"swapFee\":0.00008,\"exchange\":\"maverick-v1\",\"type\":\"maverick-v1\",\"timestamp\":1704265258,\"reserves\":[\"1171608824435142257\",\"76716840233381\"],\"tokens\":[{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"fee\\\":80000000000000,\\\"protocolFeeRatio\\\":0,\\\"activeTick\\\":1502,\\\"binCounter\\\":36,\\\"bins\\\":{\\\"1\\\":{\\\"reserveA\\\":314516285521548227,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"2\\\":{\\\"reserveA\\\":191245215895503843,\\\"reserveB\\\":0,\\\"lowerTick\\\":1501,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"3\\\":{\\\"reserveA\\\":114504301688631519,\\\"reserveB\\\":963774576010,\\\"lowerTick\\\":1502,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"31\\\":{\\\"reserveA\\\":108753991059500386,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":2,\\\"mergeId\\\":0},\\\"32\\\":{\\\"reserveA\\\":25486000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1495,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"33\\\":{\\\"reserveA\\\":42126000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1496,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"34\\\":{\\\"reserveA\\\":69628000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1497,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"35\\\":{\\\"reserveA\\\":115099589497454909,\\\"reserveB\\\":0,\\\"lowerTick\\\":1498,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"36\\\":{\\\"reserveA\\\":190249440772503320,\\\"reserveB\\\":0,\\\"lowerTick\\\":1499,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"4\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":38435140947772,\\\"lowerTick\\\":1503,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"5\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":23251195184809,\\\"lowerTick\\\":1504,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"6\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":14066729524731,\\\"lowerTick\\\":1505,\\\"kind\\\":0,\\\"mergeId\\\":0}},\\\"binPositions\\\":{\\\"1495\\\":{\\\"0\\\":32},\\\"1496\\\":{\\\"0\\\":33},\\\"1497\\\":{\\\"0\\\":34},\\\"1498\\\":{\\\"0\\\":35},\\\"1499\\\":{\\\"0\\\":36},\\\"1500\\\":{\\\"0\\\":1,\\\"2\\\":31},\\\"1501\\\":{\\\"0\\\":2},\\\"1502\\\":{\\\"0\\\":3},\\\"1503\\\":{\\\"0\\\":4},\\\"1504\\\":{\\\"0\\\":5},\\\"1505\\\":{\\\"0\\\":6}},\\\"binMap\\\":{\\\"23\\\":5807506497971120465074964654080854589440},\\\"binMapHex\\\":{\\\"17\\\":5807506497971120465074964654080854589440},\\\"liquidity\\\":1087229757983496926,\\\"sqrtPriceX96\\\":42831515231783862772}\",\"staticExtra\":\"{\\\"tickSpacing\\\":50}\"}"
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(b, err)

	maverickPool, err := NewPoolSimulator(poolEnt)
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

func TestEmptyPool(t *testing.T) {
	poolRedis := `{
    "address": "0xccd9eb9480f7beaa2bcac7d0cf5d4143f328ac06",
    "swapFee": 0.001,
    "exchange": "maverick-v1",
    "type": "maverick-v1",
    "timestamp": 1704940286,
    "reserves": ["0", "0"],
    "tokens": [
      { "address": "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "name": "", "symbol": "", "decimals": 18, "weight": 50, "swappable": true },
      { "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "name": "", "symbol": "", "decimals": 6, "weight": 50, "swappable": true }
    ],
    "extra": "{\"fee\":1000000000000000,\"protocolFeeRatio\":0,\"activeTick\":-1470,\"binCounter\":1,\"bins\":{},\"binPositions\":{},\"binMap\":{},\"binMapHex\":{},\"liquidity\":0,\"sqrtPriceX96\":479523079949611800}",
    "staticExtra": "{\"tickSpacing\":10}"
  }`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	_, err = NewPoolSimulator(poolEnt)
	assert.True(t, errors.Is(err, ErrEmptyBins))
}

func TestUpdateBalance(t *testing.T) {
	poolRedis := `{"address":"0x5fdf78aef906cbad032fbaea032aaae3accf9dc3","reserveUsd":47625.963767453606,"amplifiedTvl":2.0145226157464416e+41,"swapFee":0.0005,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1704957203,"reserves":["108363845032166910770488","2097024497432052549"],"tokens":[{"address":"0x04506dddbf689714487f91ae1397047169afcf34","decimals":18,"weight":50,"swappable":true},{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":500000000000000,\"protocolFeeRatio\":0,\"activeTick\":10,\"binCounter\":18,\"bins\":{\"1\":{\"reserveA\":1880866557485545835609,\"reserveB\":0,\"lowerTick\":-5,\"kind\":0,\"mergeId\":0},\"10\":{\"reserveA\":2013495774191474777406,\"reserveB\":0,\"lowerTick\":4,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":411993441413380258157,\"reserveB\":0,\"lowerTick\":5,\"kind\":0,\"mergeId\":0},\"12\":{\"reserveA\":491298562692665969507,\"reserveB\":0,\"lowerTick\":6,\"kind\":0,\"mergeId\":0},\"13\":{\"reserveA\":620606767055018215315,\"reserveB\":0,\"lowerTick\":7,\"kind\":0,\"mergeId\":0},\"14\":{\"reserveA\":725257522405584599699,\"reserveB\":0,\"lowerTick\":8,\"kind\":0,\"mergeId\":0},\"15\":{\"reserveA\":897478209865575805530,\"reserveB\":0,\"lowerTick\":9,\"kind\":0,\"mergeId\":0},\"16\":{\"reserveA\":2142944919078882824342,\"reserveB\":0,\"lowerTick\":-6,\"kind\":0,\"mergeId\":0},\"17\":{\"reserveA\":1022668409565365293976,\"reserveB\":2097024497432052514,\"lowerTick\":10,\"kind\":0,\"mergeId\":0},\"2\":{\"reserveA\":1634106566195962389560,\"reserveB\":0,\"lowerTick\":-4,\"kind\":0,\"mergeId\":0},\"3\":{\"reserveA\":1405424035812355050009,\"reserveB\":0,\"lowerTick\":-3,\"kind\":0,\"mergeId\":0},\"4\":{\"reserveA\":1233705168748319240144,\"reserveB\":0,\"lowerTick\":-2,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":47686688533077328269486,\"reserveB\":0,\"lowerTick\":-1,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":30071745509492793533770,\"reserveB\":0,\"lowerTick\":0,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":6925596663250336094803,\"reserveB\":0,\"lowerTick\":1,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":5442282585416271863178,\"reserveB\":0,\"lowerTick\":2,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":3757685806420050749903,\"reserveB\":0,\"lowerTick\":3,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"-1\":{\"0\":5},\"-2\":{\"0\":4},\"-3\":{\"0\":3},\"-4\":{\"0\":2},\"-5\":{\"0\":1},\"-6\":{\"0\":16},\"0\":{\"0\":6},\"1\":{\"0\":7},\"10\":{\"0\":17},\"2\":{\"0\":8},\"3\":{\"0\":9},\"4\":{\"0\":10},\"5\":{\"0\":11},\"6\":{\"0\":12},\"7\":{\"0\":13},\"8\":{\"0\":14},\"9\":{\"0\":15}},\"binMap\":{\"-1\":7719472155704656575533813171595469705082968814302106124604735256359260389376,\"0\":1172812402961},\"binMapHex\":{\"-1\":7719472155704656575533813171595469705082968814302106124604735256359260389376,\"0\":1172812402961},\"liquidity\":399352711841249002240879,\"sqrtPriceX96\":1027874653953720738}","staticExtra":"{\"tickSpacing\":50}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	testCases := []struct {
		tokenIn      string
		tokenOut     string
		amountIn     string
		expAmountOut string
	}{
		{"0x04506dddbf689714487f91ae1397047169afcf34", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "1000000000000000000", "946022415423519310"},
		{"0x04506dddbf689714487f91ae1397047169afcf34", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "500000000000000000", "473009480095110835"},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0x04506dddbf689714487f91ae1397047169afcf34", "200000000000000000", "211201042322131096"},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0x04506dddbf689714487f91ae1397047169afcf34", "900000000000000000", "950402000485391540"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			in := pool.TokenAmount{
				Token:  tc.tokenIn,
				Amount: bignumber.NewBig10(tc.amountIn),
			}
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: in,
				TokenOut:      tc.tokenOut,
				Limit:         nil,
			})
			require.Nil(t, err)
			require.Equal(t, tc.expAmountOut, result.TokenAmountOut.Amount.String())

			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			sim.UpdateBalance(updateBalanceParams)
		})
	}
}

func TestUpdateBalanceNextTick(t *testing.T) {
	poolRedis := `{"address":"0xd50c68c7fbaee4f469e04cebdcfbf1113b4cdadf","reserveUsd":52056.74739685542,"amplifiedTvl":3.641901122084877e+44,"swapFee":0.01,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1704959580,"reserves":["13095016099313357610018","26336470622025877177"],"tokens":[{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":10000000000000000,\"protocolFeeRatio\":0,\"activeTick\":433,\"binCounter\":55,\"bins\":{\"1\":{\"reserveA\":0,\"reserveB\":91157437341918885,\"lowerTick\":434,\"kind\":3,\"mergeId\":0},\"10\":{\"reserveA\":0,\"reserveB\":1239975611309030976,\"lowerTick\":447,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":0,\"reserveB\":1141328409485710165,\"lowerTick\":448,\"kind\":0,\"mergeId\":0},\"12\":{\"reserveA\":0,\"reserveB\":1090262378803153846,\"lowerTick\":449,\"kind\":0,\"mergeId\":0},\"13\":{\"reserveA\":0,\"reserveB\":1058956124010387881,\"lowerTick\":450,\"kind\":0,\"mergeId\":0},\"14\":{\"reserveA\":0,\"reserveB\":373789640683233838,\"lowerTick\":451,\"kind\":0,\"mergeId\":0},\"15\":{\"reserveA\":0,\"reserveB\":357135930104930106,\"lowerTick\":452,\"kind\":0,\"mergeId\":0},\"16\":{\"reserveA\":0,\"reserveB\":338216301716701110,\"lowerTick\":453,\"kind\":0,\"mergeId\":0},\"17\":{\"reserveA\":0,\"reserveB\":320760828172399247,\"lowerTick\":454,\"kind\":0,\"mergeId\":0},\"18\":{\"reserveA\":0,\"reserveB\":313627396754223600,\"lowerTick\":455,\"kind\":0,\"mergeId\":0},\"19\":{\"reserveA\":0,\"reserveB\":307794867765749325,\"lowerTick\":456,\"kind\":0,\"mergeId\":0},\"20\":{\"reserveA\":0,\"reserveB\":298219676795168630,\"lowerTick\":457,\"kind\":0,\"mergeId\":0},\"21\":{\"reserveA\":0,\"reserveB\":294481079018034973,\"lowerTick\":458,\"kind\":0,\"mergeId\":0},\"22\":{\"reserveA\":0,\"reserveB\":2773410496175111720,\"lowerTick\":455,\"kind\":1,\"mergeId\":0},\"29\":{\"reserveA\":1609295705362818753486,\"reserveB\":0,\"lowerTick\":426,\"kind\":0,\"mergeId\":0},\"3\":{\"reserveA\":0,\"reserveB\":1598645710773758142,\"lowerTick\":440,\"kind\":0,\"mergeId\":0},\"30\":{\"reserveA\":1713963852223063753018,\"reserveB\":0,\"lowerTick\":427,\"kind\":0,\"mergeId\":0},\"31\":{\"reserveA\":1796069142786354277710,\"reserveB\":0,\"lowerTick\":428,\"kind\":0,\"mergeId\":0},\"32\":{\"reserveA\":1739178743674569940344,\"reserveB\":0,\"lowerTick\":429,\"kind\":0,\"mergeId\":0},\"33\":{\"reserveA\":1677169405367640113180,\"reserveB\":0,\"lowerTick\":430,\"kind\":0,\"mergeId\":0},\"34\":{\"reserveA\":1771627492707184684589,\"reserveB\":0,\"lowerTick\":431,\"kind\":0,\"mergeId\":0},\"35\":{\"reserveA\":1872397651819441245354,\"reserveB\":0,\"lowerTick\":432,\"kind\":0,\"mergeId\":0},\"36\":{\"reserveA\":915314105372284841505,\"reserveB\":217573456205038785,\"lowerTick\":433,\"kind\":0,\"mergeId\":0},\"37\":{\"reserveA\":0,\"reserveB\":423191801919726618,\"lowerTick\":434,\"kind\":0,\"mergeId\":0},\"38\":{\"reserveA\":0,\"reserveB\":425853602425037489,\"lowerTick\":435,\"kind\":0,\"mergeId\":0},\"39\":{\"reserveA\":0,\"reserveB\":434870246675316320,\"lowerTick\":436,\"kind\":0,\"mergeId\":0},\"4\":{\"reserveA\":0,\"reserveB\":1747275673731363525,\"lowerTick\":441,\"kind\":0,\"mergeId\":0},\"40\":{\"reserveA\":0,\"reserveB\":428431372113458941,\"lowerTick\":437,\"kind\":0,\"mergeId\":0},\"41\":{\"reserveA\":0,\"reserveB\":1032183470388298339,\"lowerTick\":438,\"kind\":0,\"mergeId\":0},\"42\":{\"reserveA\":0,\"reserveB\":985419259209570776,\"lowerTick\":439,\"kind\":0,\"mergeId\":0},\"43\":{\"reserveA\":0,\"reserveB\":242182693866501568,\"lowerTick\":459,\"kind\":0,\"mergeId\":0},\"44\":{\"reserveA\":0,\"reserveB\":239797032983525254,\"lowerTick\":460,\"kind\":0,\"mergeId\":0},\"45\":{\"reserveA\":0,\"reserveB\":237434872449643063,\"lowerTick\":461,\"kind\":0,\"mergeId\":0},\"46\":{\"reserveA\":0,\"reserveB\":235095980770748030,\"lowerTick\":462,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":0,\"reserveB\":232780128733125118,\"lowerTick\":463,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":0,\"reserveB\":230487089380947529,\"lowerTick\":464,\"kind\":0,\"mergeId\":0},\"49\":{\"reserveA\":0,\"reserveB\":225934471614129577,\"lowerTick\":465,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":0,\"reserveB\":1461855769991903582,\"lowerTick\":442,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":0,\"reserveB\":221901118128804058,\"lowerTick\":466,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":1480772137646768965,\"lowerTick\":443,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":0,\"reserveB\":1324401440806630210,\"lowerTick\":444,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":0,\"reserveB\":1411870610798131358,\"lowerTick\":445,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":0,\"reserveB\":1499396503277694755,\"lowerTick\":446,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"426\":{\"0\":29},\"427\":{\"0\":30},\"428\":{\"0\":31},\"429\":{\"0\":32},\"430\":{\"0\":33},\"431\":{\"0\":34},\"432\":{\"0\":35},\"433\":{\"0\":36},\"434\":{\"0\":37,\"3\":1},\"435\":{\"0\":38},\"436\":{\"0\":39},\"437\":{\"0\":40},\"438\":{\"0\":41},\"439\":{\"0\":42},\"440\":{\"0\":3},\"441\":{\"0\":4},\"442\":{\"0\":5},\"443\":{\"0\":6},\"444\":{\"0\":7},\"445\":{\"0\":8},\"446\":{\"0\":9},\"447\":{\"0\":10},\"448\":{\"0\":11},\"449\":{\"0\":12},\"450\":{\"0\":13},\"451\":{\"0\":14},\"452\":{\"0\":15},\"453\":{\"0\":16},\"454\":{\"0\":17},\"455\":{\"0\":18,\"1\":22},\"456\":{\"0\":19},\"457\":{\"0\":20},\"458\":{\"0\":21},\"459\":{\"0\":43},\"460\":{\"0\":44},\"461\":{\"0\":45},\"462\":{\"0\":46},\"463\":{\"0\":47},\"464\":{\"0\":48},\"465\":{\"0\":49},\"466\":{\"0\":50}},\"binMap\":{\"6\":7719472615821092550409086380891770248800661236333969968563225660742308986880,\"7\":5037190915061491765521},\"binMapHex\":{\"6\":7719472615821092550409086380891770248800661236333969968563225660742308986880,\"7\":5037190915061491765521},\"liquidity\":2878336312141942146424,\"sqrtPriceX96\":73028492082257348963}","staticExtra":"{\"tickSpacing\":198}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	testCases := []struct {
		tokenIn      string
		tokenOut     string
		amountIn     string
		expAmountOut string
		expNextTick  string
	}{
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "10000000000000000000000", "1784415750858931428", "437"},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "50000000000000000000000", "7995459204101958875", "443"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "5000000000000000000", "31297333580169335152628", "440"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "900000000000000000", "5437425224138046326996", "439"},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			in := pool.TokenAmount{
				Token:  tc.tokenIn,
				Amount: bignumber.NewBig10(tc.amountIn),
			}
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: in,
				TokenOut:      tc.tokenOut,
				Limit:         nil,
			})
			require.Nil(t, err)
			require.Equal(t, tc.expAmountOut, result.TokenAmountOut.Amount.String())
			require.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick.String())

			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			sim.UpdateBalance(updateBalanceParams)
		})
	}
}
