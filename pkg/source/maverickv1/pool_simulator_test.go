package maverickv1

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	rawPool entity.Pool
	_ = json.Unmarshal([]byte(`{"address":"0x5bdb08ae195c8f085704582a27d566028a719265","reserveUsd":6125.460669340948,"amplifiedTvl":1.5317046591839606e+36,"swapFee":0.0002,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1733251521,"reserves":["171389885714232604","5520719430817218266406"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"","symbol":"","decimals":18,"weight":50,"swappable":true},{"address":"0x50c5725949a6f0c72e6c4a641f24049a917db0cb","name":"","symbol":"","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":200000000000000,\"protocolFeeRatio\":0,\"activeTick\":-414,\"binCounter\":288,\"bins\":{\"10\":{\"reserveA\":0,\"reserveB\":366061918078398110287,\"lowerTick\":-382,\"kind\":0,\"mergeId\":0},\"11\":{\"reserveA\":0,\"reserveB\":6883201841157069692,\"lowerTick\":-381,\"kind\":0,\"mergeId\":0},\"110\":{\"reserveA\":132717611565218673,\"reserveB\":0,\"lowerTick\":-419,\"kind\":2,\"mergeId\":0},\"12\":{\"reserveA\":0,\"reserveB\":11618854453023835182,\"lowerTick\":-380,\"kind\":0,\"mergeId\":0},\"13\":{\"reserveA\":0,\"reserveB\":12437524481813913437,\"lowerTick\":-379,\"kind\":0,\"mergeId\":0},\"14\":{\"reserveA\":0,\"reserveB\":14401602762318260592,\"lowerTick\":-378,\"kind\":0,\"mergeId\":0},\"15\":{\"reserveA\":0,\"reserveB\":20285507739545552151,\"lowerTick\":-377,\"kind\":0,\"mergeId\":0},\"16\":{\"reserveA\":0,\"reserveB\":31435697110122283023,\"lowerTick\":-376,\"kind\":0,\"mergeId\":0},\"17\":{\"reserveA\":0,\"reserveB\":788483706236166152,\"lowerTick\":-413,\"kind\":3,\"mergeId\":0},\"179\":{\"reserveA\":21032489559537,\"reserveB\":0,\"lowerTick\":-418,\"kind\":2,\"mergeId\":0},\"18\":{\"reserveA\":0,\"reserveB\":51073932713153201416,\"lowerTick\":-375,\"kind\":0,\"mergeId\":0},\"19\":{\"reserveA\":0,\"reserveB\":222307057404855279,\"lowerTick\":-371,\"kind\":1,\"mergeId\":0},\"20\":{\"reserveA\":0,\"reserveB\":34704569490663933549,\"lowerTick\":-374,\"kind\":0,\"mergeId\":0},\"21\":{\"reserveA\":0,\"reserveB\":20840557123719804560,\"lowerTick\":-373,\"kind\":0,\"mergeId\":0},\"22\":{\"reserveA\":0,\"reserveB\":12589042725147163216,\"lowerTick\":-372,\"kind\":0,\"mergeId\":0},\"229\":{\"reserveA\":61392297107235,\"reserveB\":0,\"lowerTick\":-417,\"kind\":2,\"mergeId\":0},\"23\":{\"reserveA\":0,\"reserveB\":7693776205599244096,\"lowerTick\":-371,\"kind\":0,\"mergeId\":0},\"24\":{\"reserveA\":124749357780963,\"reserveB\":0,\"lowerTick\":-420,\"kind\":2,\"mergeId\":0},\"26\":{\"reserveA\":0,\"reserveB\":4596108634618399595,\"lowerTick\":-370,\"kind\":0,\"mergeId\":0},\"28\":{\"reserveA\":0,\"reserveB\":602275039904554226,\"lowerTick\":-369,\"kind\":0,\"mergeId\":0},\"32\":{\"reserveA\":0,\"reserveB\":136281208928706118,\"lowerTick\":-368,\"kind\":0,\"mergeId\":0},\"34\":{\"reserveA\":0,\"reserveB\":63986755290069521,\"lowerTick\":-367,\"kind\":0,\"mergeId\":0},\"38\":{\"reserveA\":0,\"reserveB\":33174150853905978,\"lowerTick\":-366,\"kind\":0,\"mergeId\":0},\"40\":{\"reserveA\":0,\"reserveB\":21828411692540749,\"lowerTick\":-379,\"kind\":1,\"mergeId\":0},\"43\":{\"reserveA\":0,\"reserveB\":409922429801838876933,\"lowerTick\":-387,\"kind\":0,\"mergeId\":0},\"44\":{\"reserveA\":0,\"reserveB\":492481576830109445182,\"lowerTick\":-391,\"kind\":0,\"mergeId\":0},\"45\":{\"reserveA\":0,\"reserveB\":456855429833934711881,\"lowerTick\":-390,\"kind\":0,\"mergeId\":0},\"46\":{\"reserveA\":0,\"reserveB\":426147559216634848646,\"lowerTick\":-389,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":0,\"reserveB\":412526863298037002614,\"lowerTick\":-388,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":0,\"reserveB\":515602886856476035020,\"lowerTick\":-392,\"kind\":0,\"mergeId\":0},\"52\":{\"reserveA\":0,\"reserveB\":23418069379099730041,\"lowerTick\":-395,\"kind\":0,\"mergeId\":0},\"53\":{\"reserveA\":0,\"reserveB\":30757477984328343036,\"lowerTick\":-394,\"kind\":0,\"mergeId\":0},\"54\":{\"reserveA\":0,\"reserveB\":45386269011903114868,\"lowerTick\":-393,\"kind\":0,\"mergeId\":0},\"55\":{\"reserveA\":0,\"reserveB\":15265455037673920297,\"lowerTick\":-396,\"kind\":0,\"mergeId\":0},\"56\":{\"reserveA\":0,\"reserveB\":5499269864763066638,\"lowerTick\":-398,\"kind\":0,\"mergeId\":0},\"57\":{\"reserveA\":0,\"reserveB\":10172273266654089188,\"lowerTick\":-397,\"kind\":0,\"mergeId\":0},\"59\":{\"reserveA\":0,\"reserveB\":253084860784686155015,\"lowerTick\":-390,\"kind\":1,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":403448772752780592117,\"lowerTick\":-386,\"kind\":0,\"mergeId\":0},\"66\":{\"reserveA\":0,\"reserveB\":3808911750074946479,\"lowerTick\":-400,\"kind\":0,\"mergeId\":0},\"67\":{\"reserveA\":0,\"reserveB\":4688520546816020959,\"lowerTick\":-399,\"kind\":0,\"mergeId\":0},\"7\":{\"reserveA\":0,\"reserveB\":388504816646649170875,\"lowerTick\":-385,\"kind\":0,\"mergeId\":0},\"70\":{\"reserveA\":0,\"reserveB\":3798807329530486046,\"lowerTick\":-401,\"kind\":0,\"mergeId\":0},\"71\":{\"reserveA\":0,\"reserveB\":10071572048021564215,\"lowerTick\":-404,\"kind\":0,\"mergeId\":0},\"72\":{\"reserveA\":0,\"reserveB\":7209707634661330902,\"lowerTick\":-403,\"kind\":0,\"mergeId\":0},\"73\":{\"reserveA\":0,\"reserveB\":4814826238401964821,\"lowerTick\":-402,\"kind\":0,\"mergeId\":0},\"75\":{\"reserveA\":0,\"reserveB\":9819174046092950091,\"lowerTick\":-405,\"kind\":0,\"mergeId\":0},\"77\":{\"reserveA\":0,\"reserveB\":10099921674660620524,\"lowerTick\":-406,\"kind\":0,\"mergeId\":0},\"79\":{\"reserveA\":0,\"reserveB\":9995893308994472369,\"lowerTick\":-407,\"kind\":0,\"mergeId\":0},\"8\":{\"reserveA\":0,\"reserveB\":379893740110116757293,\"lowerTick\":-384,\"kind\":0,\"mergeId\":0},\"80\":{\"reserveA\":0,\"reserveB\":16142986122983682089,\"lowerTick\":-409,\"kind\":0,\"mergeId\":0},\"81\":{\"reserveA\":0,\"reserveB\":13506369485623416752,\"lowerTick\":-408,\"kind\":0,\"mergeId\":0},\"85\":{\"reserveA\":0,\"reserveB\":22690331704117185877,\"lowerTick\":-410,\"kind\":0,\"mergeId\":0},\"86\":{\"reserveA\":0,\"reserveB\":30883697387599607669,\"lowerTick\":-411,\"kind\":0,\"mergeId\":0},\"87\":{\"reserveA\":10309755817688041,\"reserveB\":17311324165380548343,\"lowerTick\":-414,\"kind\":0,\"mergeId\":0},\"88\":{\"reserveA\":0,\"reserveB\":71542867536434719695,\"lowerTick\":-413,\"kind\":0,\"mergeId\":0},\"89\":{\"reserveA\":0,\"reserveB\":46464456107563260886,\"lowerTick\":-412,\"kind\":0,\"mergeId\":0},\"9\":{\"reserveA\":0,\"reserveB\":372411683364983853951,\"lowerTick\":-383,\"kind\":0,\"mergeId\":0},\"90\":{\"reserveA\":5222845979037199,\"reserveB\":0,\"lowerTick\":-417,\"kind\":0,\"mergeId\":0},\"91\":{\"reserveA\":7004064916583540,\"reserveB\":0,\"lowerTick\":-416,\"kind\":0,\"mergeId\":0},\"92\":{\"reserveA\":10362207657057631,\"reserveB\":0,\"lowerTick\":-415,\"kind\":0,\"mergeId\":0},\"93\":{\"reserveA\":155404053438863,\"reserveB\":0,\"lowerTick\":-422,\"kind\":0,\"mergeId\":0},\"94\":{\"reserveA\":276984568134181,\"reserveB\":0,\"lowerTick\":-421,\"kind\":0,\"mergeId\":0},\"95\":{\"reserveA\":559418756547705,\"reserveB\":0,\"lowerTick\":-420,\"kind\":0,\"mergeId\":0},\"96\":{\"reserveA\":1242440220441584,\"reserveB\":0,\"lowerTick\":-419,\"kind\":0,\"mergeId\":0},\"97\":{\"reserveA\":3300929649906974,\"reserveB\":0,\"lowerTick\":-418,\"kind\":0,\"mergeId\":0},\"98\":{\"reserveA\":885667833527,\"reserveB\":0,\"lowerTick\":-424,\"kind\":0,\"mergeId\":0},\"99\":{\"reserveA\":30162717691326,\"reserveB\":0,\"lowerTick\":-423,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"-366\":{\"0\":38},\"-367\":{\"0\":34},\"-368\":{\"0\":32},\"-369\":{\"0\":28},\"-370\":{\"0\":26},\"-371\":{\"0\":23,\"1\":19},\"-372\":{\"0\":22},\"-373\":{\"0\":21},\"-374\":{\"0\":20},\"-375\":{\"0\":18},\"-376\":{\"0\":16},\"-377\":{\"0\":15},\"-378\":{\"0\":14},\"-379\":{\"0\":13,\"1\":40},\"-380\":{\"0\":12},\"-381\":{\"0\":11},\"-382\":{\"0\":10},\"-383\":{\"0\":9},\"-384\":{\"0\":8},\"-385\":{\"0\":7},\"-386\":{\"0\":6},\"-387\":{\"0\":43},\"-388\":{\"0\":47},\"-389\":{\"0\":46},\"-390\":{\"0\":45,\"1\":59},\"-391\":{\"0\":44},\"-392\":{\"0\":50},\"-393\":{\"0\":54},\"-394\":{\"0\":53},\"-395\":{\"0\":52},\"-396\":{\"0\":55},\"-397\":{\"0\":57},\"-398\":{\"0\":56},\"-399\":{\"0\":67},\"-400\":{\"0\":66},\"-401\":{\"0\":70},\"-402\":{\"0\":73},\"-403\":{\"0\":72},\"-404\":{\"0\":71},\"-405\":{\"0\":75},\"-406\":{\"0\":77},\"-407\":{\"0\":79},\"-408\":{\"0\":81},\"-409\":{\"0\":80},\"-410\":{\"0\":85},\"-411\":{\"0\":86},\"-412\":{\"0\":89},\"-413\":{\"0\":88,\"3\":17},\"-414\":{\"0\":87},\"-415\":{\"0\":92},\"-416\":{\"0\":91},\"-417\":{\"0\":90,\"2\":229},\"-418\":{\"0\":97,\"2\":179},\"-419\":{\"0\":96,\"2\":110},\"-420\":{\"0\":95,\"2\":24},\"-421\":{\"0\":94},\"-422\":{\"0\":93},\"-423\":{\"0\":99},\"-424\":{\"0\":98}},\"binMap\":{\"-6\":5037199922260211732753,\"-7\":7719486419313773276032307203424262894732243725133358814731333581099956699136},\"binMapHex\":{\"-6\":5037199922260211732753,\"-7\":7719486419313773276032307203424262894732243725133358814731333581099956699136},\"liquidity\":91798642659145483704,\"sqrtPriceX96\":16711602883141162,\"minBinMapIndex\":-7,\"maxBinMapIndex\":-6}","staticExtra":"{\"tickSpacing\":198}"}`), &rawPool)
	maverickPool, err = NewPoolSimulator(rawPool)
)

func TestPoolCalcAmountOut(t *testing.T) {
	assert.Nil(t, err)

	// make sure that we can calculate min/max index if pool-service hasn't done that yet
	assert.Equal(t, big.NewInt(-7), maverickPool.state.minBinMapIndex)
	assert.Equal(t, big.NewInt(-6), maverickPool.state.maxBinMapIndex)

	result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
		return maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x4200000000000000000000000000000000000006",
				Amount: bignumber.NewBig10("100100100100100100"),
			},
			TokenOut: "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
		})
	})

	if assert.Nil(t, err) {
		assert.Equal(t, "319754866834816685427", result.TokenAmountOut.Amount.String())
	}
}

func TestPoolCalcAmountOut_RevertL(t *testing.T) {
	for _, amtIn := range []string{"17299124533583919",
		"3717299124533583919",
		"37172991245335839190",
		"371729912453358391945",
		"37172991245335839191"} {
		_, err := maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
				Amount: bignumber.NewBig10(amtIn),
			},
			TokenOut: "0x4200000000000000000000000000000000000006",
		})
		assert.NoError(t, err, amtIn)
	}
	for _, amtIn := range []string{"37172991245335839192",
		"37172991245335839193",
		"37172991245335839196789"} {
		_, err := maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
				Amount: bignumber.NewBig10(amtIn),
			},
			TokenOut: "0x4200000000000000000000000000000000000006",
		})
		assert.Error(t, err, amtIn)
	}
}

func TestPoolCalcAmountIn(t *testing.T) {
	// make sure that we can calculate min/max index if pool-service hasn't done that yet
	assert.Equal(t, big.NewInt(-7), maverickPool.state.minBinMapIndex)
	assert.Equal(t, big.NewInt(-6), maverickPool.state.maxBinMapIndex)

	result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountInResult, error) {
		return maverickPool.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{
				Token:  "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
				Amount: bignumber.NewBig10("319754866834816685427"),
			},
			TokenIn: "0x4200000000000000000000000000000000000006",
		})
	})

	if assert.Nil(t, err) {
		assert.Equal(t, "100100100100099386", result.TokenAmountIn.Amount.String())
	}
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: bignumber.NewBig10("1000000000000000000"),
			},
			TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		})
	}
}

func BenchmarkNextActive(b *testing.B) {
	poolRedis := "{\"address\":\"0x012245db1919bbb6d727b9ce787c3169f963a898\",\"reserveUsd\":1.3045263641356901,\"amplifiedTvl\":8.068244485638408e+40,\"swapFee\":0.00008,\"exchange\":\"maverick-v1\",\"type\":\"maverick-v1\",\"timestamp\":1704265258,\"reserves\":[\"1171608824435142257\",\"76716840233381\"],\"tokens\":[{\"address\":\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\",\"decimals\":6,\"weight\":50,\"swappable\":true},{\"address\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"decimals\":18,\"weight\":50,\"swappable\":true}],\"extra\":\"{\\\"minBinMapIndex\\\":23,\\\"maxBinMapIndex\\\":23,\\\"fee\\\":80000000000000,\\\"protocolFeeRatio\\\":0,\\\"activeTick\\\":1502,\\\"binCounter\\\":36,\\\"bins\\\":{\\\"1\\\":{\\\"reserveA\\\":314516285521548227,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"2\\\":{\\\"reserveA\\\":191245215895503843,\\\"reserveB\\\":0,\\\"lowerTick\\\":1501,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"3\\\":{\\\"reserveA\\\":114504301688631519,\\\"reserveB\\\":963774576010,\\\"lowerTick\\\":1502,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"31\\\":{\\\"reserveA\\\":108753991059500386,\\\"reserveB\\\":0,\\\"lowerTick\\\":1500,\\\"kind\\\":2,\\\"mergeId\\\":0},\\\"32\\\":{\\\"reserveA\\\":25486000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1495,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"33\\\":{\\\"reserveA\\\":42126000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1496,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"34\\\":{\\\"reserveA\\\":69628000000000000,\\\"reserveB\\\":0,\\\"lowerTick\\\":1497,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"35\\\":{\\\"reserveA\\\":115099589497454909,\\\"reserveB\\\":0,\\\"lowerTick\\\":1498,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"36\\\":{\\\"reserveA\\\":190249440772503320,\\\"reserveB\\\":0,\\\"lowerTick\\\":1499,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"4\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":38435140947772,\\\"lowerTick\\\":1503,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"5\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":23251195184809,\\\"lowerTick\\\":1504,\\\"kind\\\":0,\\\"mergeId\\\":0},\\\"6\\\":{\\\"reserveA\\\":0,\\\"reserveB\\\":14066729524731,\\\"lowerTick\\\":1505,\\\"kind\\\":0,\\\"mergeId\\\":0}},\\\"binPositions\\\":{\\\"1495\\\":{\\\"0\\\":32},\\\"1496\\\":{\\\"0\\\":33},\\\"1497\\\":{\\\"0\\\":34},\\\"1498\\\":{\\\"0\\\":35},\\\"1499\\\":{\\\"0\\\":36},\\\"1500\\\":{\\\"0\\\":1,\\\"2\\\":31},\\\"1501\\\":{\\\"0\\\":2},\\\"1502\\\":{\\\"0\\\":3},\\\"1503\\\":{\\\"0\\\":4},\\\"1504\\\":{\\\"0\\\":5},\\\"1505\\\":{\\\"0\\\":6}},\\\"binMap\\\":{\\\"23\\\":5807506497971120465074964654080854589440},\\\"binMapHex\\\":{\\\"17\\\":5807506497971120465074964654080854589440},\\\"liquidity\\\":1087229757983496926,\\\"sqrtPriceX96\\\":42831515231783862772}\",\"staticExtra\":\"{\\\"tickSpacing\\\":50}\"}"
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

	assert.Equal(t, int64(-1), sim.state.minBinMapIndex.Int64())
	assert.Equal(t, int64(0), sim.state.maxBinMapIndex.Int64())

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
			cloned := sim.CloneState()
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.tokenOut,
				})
			})
			require.Nil(t, err)
			require.Equal(t, tc.expAmountOut, result.TokenAmountOut.Amount.String())
			resultBeforeUpdate, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.tokenOut,
				})
			})
			require.Nil(t, err)
			require.Equal(t, result.TokenAmountOut.Amount.String(), resultBeforeUpdate.TokenAmountOut.Amount.String())

			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			sim.UpdateBalance(updateBalanceParams)

			resultAfterUpdate, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.tokenOut,
				})
			})
			if err == nil {
				require.NotEqual(t, result.TokenAmountOut.Amount.String(), resultAfterUpdate.TokenAmountOut.Amount.String())
			}

			resultOfCloned, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return cloned.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.tokenOut,
				})
			})
			require.Nil(t, err)
			require.Equal(t, tc.expAmountOut, resultOfCloned.TokenAmountOut.Amount.String())
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

	assert.Equal(t, big.NewInt(6), sim.state.minBinMapIndex)
	assert.Equal(t, big.NewInt(7), sim.state.maxBinMapIndex)

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
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: in,
					TokenOut:      tc.tokenOut,
				})
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

func TestNextActive(t *testing.T) {
	poolRedis := `{"address":"0xbd278792260a68ee81a42adba23befdba87e30eb","reserveUsd":15864.05368210012,"amplifiedTvl":4.188628090182823e+41,"swapFee":0.0001,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1705301925,"reserves":["2938066235974926310","4262165529103738357"],"tokens":[{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":100000000000000,\"protocolFeeRatio\":0,\"activeTick\":-8,\"binCounter\":52,\"bins\":{\"12\":{\"reserveA\":0,\"reserveB\":4242237013037562,\"lowerTick\":-7,\"kind\":3,\"mergeId\":0},\"43\":{\"reserveA\":0,\"reserveB\":1000000000000000000,\"lowerTick\":343,\"kind\":0,\"mergeId\":0},\"44\":{\"reserveA\":0,\"reserveB\":978339781816359,\"lowerTick\":693,\"kind\":0,\"mergeId\":0},\"45\":{\"reserveA\":0,\"reserveB\":957148728684,\"lowerTick\":1043,\"kind\":0,\"mergeId\":0},\"46\":{\"reserveA\":0,\"reserveB\":936416678,\"lowerTick\":1393,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":0,\"reserveB\":916133,\"lowerTick\":1743,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":1000000000000000000,\"reserveB\":0,\"lowerTick\":-357,\"kind\":0,\"mergeId\":0},\"49\":{\"reserveA\":978339781816359,\"reserveB\":0,\"lowerTick\":-707,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":1937086938107048420,\"reserveB\":516466083168804034,\"lowerTick\":-8,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":957148728684,\"reserveB\":0,\"lowerTick\":-1057,\"kind\":0,\"mergeId\":0},\"51\":{\"reserveA\":936416678,\"reserveB\":0,\"lowerTick\":-1407,\"kind\":0,\"mergeId\":0},\"52\":{\"reserveA\":916133,\"reserveB\":0,\"lowerTick\":-1757,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":2740477911054018869,\"lowerTick\":-7,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"-1057\":{\"0\":50},\"-1407\":{\"0\":51},\"-1757\":{\"0\":52},\"-357\":{\"0\":48},\"-7\":{\"0\":6,\"3\":12},\"-707\":{\"0\":49},\"-8\":{\"0\":5},\"1043\":{\"0\":45},\"1393\":{\"0\":46},\"1743\":{\"0\":47},\"343\":{\"0\":43},\"693\":{\"0\":44}},\"binMap\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-12\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"-17\":21267647932558653966460912964485513216,\"-22\":16,\"-28\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"10\":6582018229284824168619876730229402019930943462534319453394436096,\"16\":75557863725914323419136,\"21\":100433627766186892221372630771322662657637687111424552206336,\"27\":1152921504606846976,\"5\":4951760157141521099596496896},\"binMapHex\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-11\":21267647932558653966460912964485513216,\"-16\":16,\"-1c\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"-c\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"10\":75557863725914323419136,\"15\":100433627766186892221372630771322662657637687111424552206336,\"1b\":1152921504606846976,\"5\":4951760157141521099596496896,\"a\":6582018229284824168619876730229402019930943462534319453394436096},\"liquidity\":259584104169810274047,\"sqrtPriceX96\":931321064190656607}","staticExtra":"{\"tickSpacing\":198}"}`
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
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "10000000000000000", "11527622736373110", "-8"},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1000000000000000000", "1148063760188665172", "-7"},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "1900000000000000000", "2173275581711676990", "-7"},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "2890000000000000000", "3261217293522172144", "343"},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "2900000000000000000", "3261228530154579452", "343"},

		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "50000000000000000", "43355832734019328", "-8"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "500000000000000000", "432859677357507302", "-8"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "5000000000000000000", "1939474350528474792", "-357"},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "5000000000000000000000", "2938066235974285626", "-1757"},
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
			})
			require.Nil(t, err)
			assert.Equal(t, tc.expAmountOut, result.TokenAmountOut.Amount.String())
			assert.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick.String())
		})
	}
}

func TestLSBMSB(t *testing.T) {
	assert.Equal(t, big.NewInt(92), lsb(bignumber.NewBig10("4951760157141521099596496896")))
	assert.Equal(t, big.NewInt(196), lsb(bignumber.NewBig10("100433627766186892221372630771322662657637687111424552206336")))
	assert.Equal(t, big.NewInt(1), lsb(bignumber.NewBig10("126")))
	assert.Equal(t, int64(0), lsb(bignumber.NewBig10("-2923")).Int64())
	assert.Equal(t, big.NewInt(244), lsb(bignumber.NewBig10("28269553036454149273332760011886696253239742350009903329945699220681916416")))
	assert.Equal(t, big.NewInt(128), lsb(bignumber.NewBig("0x100000000000000000000000000000000")))
	assert.Equal(t, int64(0), lsb(bignumber.NewBig("0x100000000000000000000000000000001")).Int64())

	assert.Equal(t, big.NewInt(92), msb(bignumber.NewBig10("4951760157141521099596496896")))
	assert.Equal(t, big.NewInt(196), msb(bignumber.NewBig10("100433627766186892221372630771322662657637687111424552206336")))
	assert.Equal(t, big.NewInt(6), msb(bignumber.NewBig10("126")))
	assert.Equal(t, int64(0), msb(bignumber.NewBig10("-2923")).Int64())
	assert.Equal(t, big.NewInt(244), msb(bignumber.NewBig10("28269553036454149273332760011886696253239742350009903329945699220681916416")))
	assert.Equal(t, big.NewInt(128), msb(bignumber.NewBig("0x100000000000000000000000000000000")))
	assert.Equal(t, big.NewInt(128), msb(bignumber.NewBig("0x100000000000000000000000000000001")))
}

func TestGas(t *testing.T) {
	poolRedis := `{"address":"0xbd278792260a68ee81a42adba23befdba87e30eb","reserveUsd":15059.478927527987,"amplifiedTvl":4.184931466034053e+41,"swapFee":0.0001,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1706603958,"reserves":["2722240380725257133","4511247270069585288"],"tokens":[{"address":"A","decimals":18,"weight":50,"swappable":true},{"address":"B","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":100000000000000,\"protocolFeeRatio\":0,\"activeTick\":-8,\"binCounter\":52,\"bins\":{\"12\":{\"reserveA\":0,\"reserveB\":4242237013037562,\"lowerTick\":-7,\"kind\":3,\"mergeId\":0},\"43\":{\"reserveA\":0,\"reserveB\":1000000000000000000,\"lowerTick\":343,\"kind\":0,\"mergeId\":0},\"44\":{\"reserveA\":0,\"reserveB\":978339781816359,\"lowerTick\":693,\"kind\":0,\"mergeId\":0},\"45\":{\"reserveA\":0,\"reserveB\":957148728684,\"lowerTick\":1043,\"kind\":0,\"mergeId\":0},\"46\":{\"reserveA\":0,\"reserveB\":936416678,\"lowerTick\":1393,\"kind\":0,\"mergeId\":0},\"47\":{\"reserveA\":0,\"reserveB\":916133,\"lowerTick\":1743,\"kind\":0,\"mergeId\":0},\"48\":{\"reserveA\":1000000000000000000,\"reserveB\":0,\"lowerTick\":-357,\"kind\":0,\"mergeId\":0},\"49\":{\"reserveA\":978339781816359,\"reserveB\":0,\"lowerTick\":-707,\"kind\":0,\"mergeId\":0},\"5\":{\"reserveA\":1721261082857379243,\"reserveB\":765547824134650965,\"lowerTick\":-8,\"kind\":0,\"mergeId\":0},\"50\":{\"reserveA\":957148728684,\"reserveB\":0,\"lowerTick\":-1057,\"kind\":0,\"mergeId\":0},\"51\":{\"reserveA\":936416678,\"reserveB\":0,\"lowerTick\":-1407,\"kind\":0,\"mergeId\":0},\"52\":{\"reserveA\":916133,\"reserveB\":0,\"lowerTick\":-1757,\"kind\":0,\"mergeId\":0},\"6\":{\"reserveA\":0,\"reserveB\":2740477911054018869,\"lowerTick\":-7,\"kind\":0,\"mergeId\":0}},\"binPositions\":{\"-1057\":{\"0\":50},\"-1407\":{\"0\":51},\"-1757\":{\"0\":52},\"-357\":{\"0\":48},\"-7\":{\"0\":6,\"3\":12},\"-707\":{\"0\":49},\"-8\":{\"0\":5},\"1043\":{\"0\":45},\"1393\":{\"0\":46},\"1743\":{\"0\":47},\"343\":{\"0\":43},\"693\":{\"0\":44}},\"binMap\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-12\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"-17\":21267647932558653966460912964485513216,\"-22\":16,\"-28\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"10\":6582018229284824168619876730229402019930943462534319453394436096,\"16\":75557863725914323419136,\"21\":100433627766186892221372630771322662657637687111424552206336,\"27\":1152921504606846976,\"5\":4951760157141521099596496896},\"binMapHex\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-11\":21267647932558653966460912964485513216,\"-16\":16,\"-1c\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"-c\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"10\":75557863725914323419136,\"15\":100433627766186892221372630771322662657637687111424552206336,\"1b\":1152921504606846976,\"5\":4951760157141521099596496896,\"a\":6582018229284824168619876730229402019930943462534319453394436096},\"liquidity\":259586774308826574234,\"sqrtPriceX96\":930489566587878568,\"minBinMapIndex\":-28,\"maxBinMapIndex\":27}","staticExtra":"{\"tickSpacing\":198}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	testCases := []struct {
		tokenIn     string
		tokenOut    string
		amountIn    string
		expNextTick string
		expGas      int64
	}{
		{"A", "B", "10000000000000000", "-8", 145000},    // use 1 tick, sim 110969
		{"A", "B", "1000000000000000000", "-7", 165000},  // use 2 tick, sim 144530
		{"A", "B", "1900000000000000000", "-7", 165000},  // use 2 tick, sim 144530
		{"A", "B", "3890000000000000000", "343", 185000}, // use 3 tick, sim 173534
		{"A", "B", "4000000000000000000", "343", 185000}, // use 3 tick, sim 173534

		{"B", "A", "50000000000000000", "-8", 145000},         // use 1 tick, sim 106139
		{"B", "A", "500000000000000000", "-8", 145000},        // use 1 tick, sim 106139
		{"B", "A", "5000000000000000000", "-357", 165000},     // use 2 tick, sim 138592
		{"B", "A", "5000000000000000000000", "-1757", 245000}, // use 6 tick, sim 273005
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
			})
			require.Nil(t, err)

			assert.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick.String())
			assert.Equal(t, tc.expGas, result.Gas)
		})
	}
}
