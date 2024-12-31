package maverickv1

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	rawPool entity.Pool
	_       = json.Unmarshal([]byte(`{"address":"0x5bdb08ae195c8f085704582a27d566028a719265","reserveUsd":6125.460669340948,"amplifiedTvl":1.5317046591839606e+36,"swapFee":0.0002,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1733251521,"reserves":["171389885714232604","5520719430817218266406"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"","symbol":"","decimals":18,"weight":50,"swappable":true},{"address":"0x50c5725949a6f0c72e6c4a641f24049a917db0cb","name":"","symbol":"","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":200000000000000,\"protocFeeRatio\":0,\"tick\":-414,\"bins\":{\"10\":{\"rA\":0,\"rB\":366061918078398110287,\"lT\":-382,\"k\":0},\"11\":{\"rA\":0,\"rB\":6883201841157069692,\"lT\":-381,\"k\":0},\"110\":{\"rA\":132717611565218673,\"rB\":0,\"lT\":-419,\"k\":2},\"12\":{\"rA\":0,\"rB\":11618854453023835182,\"lT\":-380,\"k\":0},\"13\":{\"rA\":0,\"rB\":12437524481813913437,\"lT\":-379,\"k\":0},\"14\":{\"rA\":0,\"rB\":14401602762318260592,\"lT\":-378,\"k\":0},\"15\":{\"rA\":0,\"rB\":20285507739545552151,\"lT\":-377,\"k\":0},\"16\":{\"rA\":0,\"rB\":31435697110122283023,\"lT\":-376,\"k\":0},\"17\":{\"rA\":0,\"rB\":788483706236166152,\"lT\":-413,\"k\":3},\"179\":{\"rA\":21032489559537,\"rB\":0,\"lT\":-418,\"k\":2},\"18\":{\"rA\":0,\"rB\":51073932713153201416,\"lT\":-375,\"k\":0},\"19\":{\"rA\":0,\"rB\":222307057404855279,\"lT\":-371,\"k\":1},\"20\":{\"rA\":0,\"rB\":34704569490663933549,\"lT\":-374,\"k\":0},\"21\":{\"rA\":0,\"rB\":20840557123719804560,\"lT\":-373,\"k\":0},\"22\":{\"rA\":0,\"rB\":12589042725147163216,\"lT\":-372,\"k\":0},\"229\":{\"rA\":61392297107235,\"rB\":0,\"lT\":-417,\"k\":2},\"23\":{\"rA\":0,\"rB\":7693776205599244096,\"lT\":-371,\"k\":0},\"24\":{\"rA\":124749357780963,\"rB\":0,\"lT\":-420,\"k\":2},\"26\":{\"rA\":0,\"rB\":4596108634618399595,\"lT\":-370,\"k\":0},\"28\":{\"rA\":0,\"rB\":602275039904554226,\"lT\":-369,\"k\":0},\"32\":{\"rA\":0,\"rB\":136281208928706118,\"lT\":-368,\"k\":0},\"34\":{\"rA\":0,\"rB\":63986755290069521,\"lT\":-367,\"k\":0},\"38\":{\"rA\":0,\"rB\":33174150853905978,\"lT\":-366,\"k\":0},\"40\":{\"rA\":0,\"rB\":21828411692540749,\"lT\":-379,\"k\":1},\"43\":{\"rA\":0,\"rB\":409922429801838876933,\"lT\":-387,\"k\":0},\"44\":{\"rA\":0,\"rB\":492481576830109445182,\"lT\":-391,\"k\":0},\"45\":{\"rA\":0,\"rB\":456855429833934711881,\"lT\":-390,\"k\":0},\"46\":{\"rA\":0,\"rB\":426147559216634848646,\"lT\":-389,\"k\":0},\"47\":{\"rA\":0,\"rB\":412526863298037002614,\"lT\":-388,\"k\":0},\"50\":{\"rA\":0,\"rB\":515602886856476035020,\"lT\":-392,\"k\":0},\"52\":{\"rA\":0,\"rB\":23418069379099730041,\"lT\":-395,\"k\":0},\"53\":{\"rA\":0,\"rB\":30757477984328343036,\"lT\":-394,\"k\":0},\"54\":{\"rA\":0,\"rB\":45386269011903114868,\"lT\":-393,\"k\":0},\"55\":{\"rA\":0,\"rB\":15265455037673920297,\"lT\":-396,\"k\":0},\"56\":{\"rA\":0,\"rB\":5499269864763066638,\"lT\":-398,\"k\":0},\"57\":{\"rA\":0,\"rB\":10172273266654089188,\"lT\":-397,\"k\":0},\"59\":{\"rA\":0,\"rB\":253084860784686155015,\"lT\":-390,\"k\":1},\"6\":{\"rA\":0,\"rB\":403448772752780592117,\"lT\":-386,\"k\":0},\"66\":{\"rA\":0,\"rB\":3808911750074946479,\"lT\":-400,\"k\":0},\"67\":{\"rA\":0,\"rB\":4688520546816020959,\"lT\":-399,\"k\":0},\"7\":{\"rA\":0,\"rB\":388504816646649170875,\"lT\":-385,\"k\":0},\"70\":{\"rA\":0,\"rB\":3798807329530486046,\"lT\":-401,\"k\":0},\"71\":{\"rA\":0,\"rB\":10071572048021564215,\"lT\":-404,\"k\":0},\"72\":{\"rA\":0,\"rB\":7209707634661330902,\"lT\":-403,\"k\":0},\"73\":{\"rA\":0,\"rB\":4814826238401964821,\"lT\":-402,\"k\":0},\"75\":{\"rA\":0,\"rB\":9819174046092950091,\"lT\":-405,\"k\":0},\"77\":{\"rA\":0,\"rB\":10099921674660620524,\"lT\":-406,\"k\":0},\"79\":{\"rA\":0,\"rB\":9995893308994472369,\"lT\":-407,\"k\":0},\"8\":{\"rA\":0,\"rB\":379893740110116757293,\"lT\":-384,\"k\":0},\"80\":{\"rA\":0,\"rB\":16142986122983682089,\"lT\":-409,\"k\":0},\"81\":{\"rA\":0,\"rB\":13506369485623416752,\"lT\":-408,\"k\":0},\"85\":{\"rA\":0,\"rB\":22690331704117185877,\"lT\":-410,\"k\":0},\"86\":{\"rA\":0,\"rB\":30883697387599607669,\"lT\":-411,\"k\":0},\"87\":{\"rA\":10309755817688041,\"rB\":17311324165380548343,\"lT\":-414,\"k\":0},\"88\":{\"rA\":0,\"rB\":71542867536434719695,\"lT\":-413,\"k\":0},\"89\":{\"rA\":0,\"rB\":46464456107563260886,\"lT\":-412,\"k\":0},\"9\":{\"rA\":0,\"rB\":372411683364983853951,\"lT\":-383,\"k\":0},\"90\":{\"rA\":5222845979037199,\"rB\":0,\"lT\":-417,\"k\":0},\"91\":{\"rA\":7004064916583540,\"rB\":0,\"lT\":-416,\"k\":0},\"92\":{\"rA\":10362207657057631,\"rB\":0,\"lT\":-415,\"k\":0},\"93\":{\"rA\":155404053438863,\"rB\":0,\"lT\":-422,\"k\":0},\"94\":{\"rA\":276984568134181,\"rB\":0,\"lT\":-421,\"k\":0},\"95\":{\"rA\":559418756547705,\"rB\":0,\"lT\":-420,\"k\":0},\"96\":{\"rA\":1242440220441584,\"rB\":0,\"lT\":-419,\"k\":0},\"97\":{\"rA\":3300929649906974,\"rB\":0,\"lT\":-418,\"k\":0},\"98\":{\"rA\":885667833527,\"rB\":0,\"lT\":-424,\"k\":0},\"99\":{\"rA\":30162717691326,\"rB\":0,\"lT\":-423,\"k\":0}},\"binPosMap\":{\"-366\":{\"0\":38},\"-367\":{\"0\":34},\"-368\":{\"0\":32},\"-369\":{\"0\":28},\"-370\":{\"0\":26},\"-371\":{\"0\":23,\"1\":19},\"-372\":{\"0\":22},\"-373\":{\"0\":21},\"-374\":{\"0\":20},\"-375\":{\"0\":18},\"-376\":{\"0\":16},\"-377\":{\"0\":15},\"-378\":{\"0\":14},\"-379\":{\"0\":13,\"1\":40},\"-380\":{\"0\":12},\"-381\":{\"0\":11},\"-382\":{\"0\":10},\"-383\":{\"0\":9},\"-384\":{\"0\":8},\"-385\":{\"0\":7},\"-386\":{\"0\":6},\"-387\":{\"0\":43},\"-388\":{\"0\":47},\"-389\":{\"0\":46},\"-390\":{\"0\":45,\"1\":59},\"-391\":{\"0\":44},\"-392\":{\"0\":50},\"-393\":{\"0\":54},\"-394\":{\"0\":53},\"-395\":{\"0\":52},\"-396\":{\"0\":55},\"-397\":{\"0\":57},\"-398\":{\"0\":56},\"-399\":{\"0\":67},\"-400\":{\"0\":66},\"-401\":{\"0\":70},\"-402\":{\"0\":73},\"-403\":{\"0\":72},\"-404\":{\"0\":71},\"-405\":{\"0\":75},\"-406\":{\"0\":77},\"-407\":{\"0\":79},\"-408\":{\"0\":81},\"-409\":{\"0\":80},\"-410\":{\"0\":85},\"-411\":{\"0\":86},\"-412\":{\"0\":89},\"-413\":{\"0\":88,\"3\":17},\"-414\":{\"0\":87},\"-415\":{\"0\":92},\"-416\":{\"0\":91},\"-417\":{\"0\":90,\"2\":229},\"-418\":{\"0\":97,\"2\":179},\"-419\":{\"0\":96,\"2\":110},\"-420\":{\"0\":95,\"2\":24},\"-421\":{\"0\":94},\"-422\":{\"0\":93},\"-423\":{\"0\":99},\"-424\":{\"0\":98}},\"binMap\":{\"-6\":5037199922260211732753,\"-7\":7719486419313773276032307203424262894732243725133358814731333581099956699136},\"liquidity\":91798642659145483704,\"sqrtPriceX96\":16711602883141162}","staticExtra":"{\"tickSpacing\":198}"}`),
		&rawPool)
	maverickPool, err = NewPoolSimulator(rawPool)
)

func TestPoolCalcAmountOut(t *testing.T) {
	assert.Nil(t, err)

	// make sure that we can calculate min/max index if pool-service hasn't done that yet
	assert.EqualValues(t, -7, maverickPool.state.minBinMapIndex)
	assert.EqualValues(t, -6, maverickPool.state.maxBinMapIndex)

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
	assert.EqualValues(t, -7, maverickPool.state.minBinMapIndex)
	assert.EqualValues(t, -6, maverickPool.state.maxBinMapIndex)

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
	poolRedis := `{"address":"0x012245db1919bbb6d727b9ce787c3169f963a898","reserveUsd":1.3045263641356901,"amplifiedTvl":8.068244485638408e+40,"swapFee":0.00008,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1704265258,"reserves":["1171608824435142257","76716840233381"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","decimals":6,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{,\"fee\":80000000000000,\"protocFeeRatio\":0,\"tick\":1502,\"binCounter\":36,\"bins\":{\"1\":{\"rA\":314516285521548227,\"rB\":0,\"lT\":1500,\"k\":0},\"2\":{\"rA\":191245215895503843,\"rB\":0,\"lT\":1501,\"k\":0},\"3\":{\"rA\":114504301688631519,\"rB\":963774576010,\"lT\":1502,\"k\":0},\"31\":{\"rA\":108753991059500386,\"rB\":0,\"lT\":1500,\"k\":2},\"32\":{\"rA\":25486000000000000,\"rB\":0,\"lT\":1495,\"k\":0},\"33\":{\"rA\":42126000000000000,\"rB\":0,\"lT\":1496,\"k\":0},\"34\":{\"rA\":69628000000000000,\"rB\":0,\"lT\":1497,\"k\":0},\"35\":{\"rA\":115099589497454909,\"rB\":0,\"lT\":1498,\"k\":0},\"36\":{\"rA\":190249440772503320,\"rB\":0,\"lT\":1499,\"k\":0},\"4\":{\"rA\":0,\"rB\":38435140947772,\"lT\":1503,\"k\":0},\"5\":{\"rA\":0,\"rB\":23251195184809,\"lT\":1504,\"k\":0},\"6\":{\"rA\":0,\"rB\":14066729524731,\"lT\":1505,\"k\":0}},\"binPosMap\":{\"1495\":{\"0\":32},\"1496\":{\"0\":33},\"1497\":{\"0\":34},\"1498\":{\"0\":35},\"1499\":{\"0\":36},\"1500\":{\"0\":1,\"2\":31},\"1501\":{\"0\":2},\"1502\":{\"0\":3},\"1503\":{\"0\":4},\"1504\":{\"0\":5},\"1505\":{\"0\":6}},\"binMap\":{\"23\":5807506497971120465074964654080854589440},\"liquidity\":1087229757983496926,\"sqrtPriceX96\":42831515231783862772}","staticExtra":"{\"tickSpacing\":50}"}`
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
    "extra": "{\"fee\":1000000000000000,\"protocFeeRatio\":0,\"tick\":-1470,\"bins\":{},\"binPosMap\":{},\"binMap\":{},\"liquidity\":0,\"sqrtPriceX96\":479523079949611800}",
    "staticExtra": "{\"tickSpacing\":10}"
  }`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	_, err = NewPoolSimulator(poolEnt)
	assert.True(t, errors.Is(err, ErrEmptyBins))
}

func TestUpdateBalance(t *testing.T) {
	poolRedis := `{"address":"0x5fdf78aef906cbad032fbaea032aaae3accf9dc3","reserveUsd":47625.963767453606,"amplifiedTvl":2.0145226157464416e+41,"swapFee":0.0005,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1704957203,"reserves":["108363845032166910770488","2097024497432052549"],"tokens":[{"address":"0x04506dddbf689714487f91ae1397047169afcf34","decimals":18,"weight":50,"swappable":true},{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":500000000000000,\"protocFeeRatio\":0,\"tick\":10,\"bins\":{\"1\":{\"rA\":1880866557485545835609,\"rB\":0,\"lT\":-5,\"k\":0},\"10\":{\"rA\":2013495774191474777406,\"rB\":0,\"lT\":4,\"k\":0},\"11\":{\"rA\":411993441413380258157,\"rB\":0,\"lT\":5,\"k\":0},\"12\":{\"rA\":491298562692665969507,\"rB\":0,\"lT\":6,\"k\":0},\"13\":{\"rA\":620606767055018215315,\"rB\":0,\"lT\":7,\"k\":0},\"14\":{\"rA\":725257522405584599699,\"rB\":0,\"lT\":8,\"k\":0},\"15\":{\"rA\":897478209865575805530,\"rB\":0,\"lT\":9,\"k\":0},\"16\":{\"rA\":2142944919078882824342,\"rB\":0,\"lT\":-6,\"k\":0},\"17\":{\"rA\":1022668409565365293976,\"rB\":2097024497432052514,\"lT\":10,\"k\":0},\"2\":{\"rA\":1634106566195962389560,\"rB\":0,\"lT\":-4,\"k\":0},\"3\":{\"rA\":1405424035812355050009,\"rB\":0,\"lT\":-3,\"k\":0},\"4\":{\"rA\":1233705168748319240144,\"rB\":0,\"lT\":-2,\"k\":0},\"5\":{\"rA\":47686688533077328269486,\"rB\":0,\"lT\":-1,\"k\":0},\"6\":{\"rA\":30071745509492793533770,\"rB\":0,\"lT\":0,\"k\":0},\"7\":{\"rA\":6925596663250336094803,\"rB\":0,\"lT\":1,\"k\":0},\"8\":{\"rA\":5442282585416271863178,\"rB\":0,\"lT\":2,\"k\":0},\"9\":{\"rA\":3757685806420050749903,\"rB\":0,\"lT\":3,\"k\":0}},\"binPosMap\":{\"-1\":{\"0\":5},\"-2\":{\"0\":4},\"-3\":{\"0\":3},\"-4\":{\"0\":2},\"-5\":{\"0\":1},\"-6\":{\"0\":16},\"0\":{\"0\":6},\"1\":{\"0\":7},\"10\":{\"0\":17},\"2\":{\"0\":8},\"3\":{\"0\":9},\"4\":{\"0\":10},\"5\":{\"0\":11},\"6\":{\"0\":12},\"7\":{\"0\":13},\"8\":{\"0\":14},\"9\":{\"0\":15}},\"binMap\":{\"-1\":7719472155704656575533813171595469705082968814302106124604735256359260389376,\"0\":1172812402961},\"liquidity\":399352711841249002240879,\"sqrtPriceX96\":1027874653953720738}","staticExtra":"{\"tickSpacing\":50}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	assert.Equal(t, int16(-1), sim.state.minBinMapIndex)
	assert.Equal(t, int16(0), sim.state.maxBinMapIndex)

	testCases := []struct {
		tokenIn      string
		tokenOut     string
		amountIn     string
		expAmountOut string
	}{
		{"0x04506dddbf689714487f91ae1397047169afcf34", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd",
			"1000000000000000000", "946022415423519310"},
		{"0x04506dddbf689714487f91ae1397047169afcf34", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd",
			"500000000000000000", "473009480095110835"},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0x04506dddbf689714487f91ae1397047169afcf34",
			"200000000000000000", "211201042322131096"},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0x04506dddbf689714487f91ae1397047169afcf34",
			"900000000000000000", "950402000485391540"},
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
				require.NotEqual(t, result.TokenAmountOut.Amount.String(),
					resultAfterUpdate.TokenAmountOut.Amount.String())
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
	poolRedis := `{"address":"0xd50c68c7fbaee4f469e04cebdcfbf1113b4cdadf","reserveUsd":52056.74739685542,"amplifiedTvl":3.641901122084877e+44,"swapFee":0.01,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1704959580,"reserves":["13095016099313357610018","26336470622025877177"],"tokens":[{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":10000000000000000,\"protocFeeRatio\":0,\"tick\":433,\"bins\":{\"1\":{\"rA\":0,\"rB\":91157437341918885,\"lT\":434,\"k\":3},\"10\":{\"rA\":0,\"rB\":1239975611309030976,\"lT\":447,\"k\":0},\"11\":{\"rA\":0,\"rB\":1141328409485710165,\"lT\":448,\"k\":0},\"12\":{\"rA\":0,\"rB\":1090262378803153846,\"lT\":449,\"k\":0},\"13\":{\"rA\":0,\"rB\":1058956124010387881,\"lT\":450,\"k\":0},\"14\":{\"rA\":0,\"rB\":373789640683233838,\"lT\":451,\"k\":0},\"15\":{\"rA\":0,\"rB\":357135930104930106,\"lT\":452,\"k\":0},\"16\":{\"rA\":0,\"rB\":338216301716701110,\"lT\":453,\"k\":0},\"17\":{\"rA\":0,\"rB\":320760828172399247,\"lT\":454,\"k\":0},\"18\":{\"rA\":0,\"rB\":313627396754223600,\"lT\":455,\"k\":0},\"19\":{\"rA\":0,\"rB\":307794867765749325,\"lT\":456,\"k\":0},\"20\":{\"rA\":0,\"rB\":298219676795168630,\"lT\":457,\"k\":0},\"21\":{\"rA\":0,\"rB\":294481079018034973,\"lT\":458,\"k\":0},\"22\":{\"rA\":0,\"rB\":2773410496175111720,\"lT\":455,\"k\":1},\"29\":{\"rA\":1609295705362818753486,\"rB\":0,\"lT\":426,\"k\":0},\"3\":{\"rA\":0,\"rB\":1598645710773758142,\"lT\":440,\"k\":0},\"30\":{\"rA\":1713963852223063753018,\"rB\":0,\"lT\":427,\"k\":0},\"31\":{\"rA\":1796069142786354277710,\"rB\":0,\"lT\":428,\"k\":0},\"32\":{\"rA\":1739178743674569940344,\"rB\":0,\"lT\":429,\"k\":0},\"33\":{\"rA\":1677169405367640113180,\"rB\":0,\"lT\":430,\"k\":0},\"34\":{\"rA\":1771627492707184684589,\"rB\":0,\"lT\":431,\"k\":0},\"35\":{\"rA\":1872397651819441245354,\"rB\":0,\"lT\":432,\"k\":0},\"36\":{\"rA\":915314105372284841505,\"rB\":217573456205038785,\"lT\":433,\"k\":0},\"37\":{\"rA\":0,\"rB\":423191801919726618,\"lT\":434,\"k\":0},\"38\":{\"rA\":0,\"rB\":425853602425037489,\"lT\":435,\"k\":0},\"39\":{\"rA\":0,\"rB\":434870246675316320,\"lT\":436,\"k\":0},\"4\":{\"rA\":0,\"rB\":1747275673731363525,\"lT\":441,\"k\":0},\"40\":{\"rA\":0,\"rB\":428431372113458941,\"lT\":437,\"k\":0},\"41\":{\"rA\":0,\"rB\":1032183470388298339,\"lT\":438,\"k\":0},\"42\":{\"rA\":0,\"rB\":985419259209570776,\"lT\":439,\"k\":0},\"43\":{\"rA\":0,\"rB\":242182693866501568,\"lT\":459,\"k\":0},\"44\":{\"rA\":0,\"rB\":239797032983525254,\"lT\":460,\"k\":0},\"45\":{\"rA\":0,\"rB\":237434872449643063,\"lT\":461,\"k\":0},\"46\":{\"rA\":0,\"rB\":235095980770748030,\"lT\":462,\"k\":0},\"47\":{\"rA\":0,\"rB\":232780128733125118,\"lT\":463,\"k\":0},\"48\":{\"rA\":0,\"rB\":230487089380947529,\"lT\":464,\"k\":0},\"49\":{\"rA\":0,\"rB\":225934471614129577,\"lT\":465,\"k\":0},\"5\":{\"rA\":0,\"rB\":1461855769991903582,\"lT\":442,\"k\":0},\"50\":{\"rA\":0,\"rB\":221901118128804058,\"lT\":466,\"k\":0},\"6\":{\"rA\":0,\"rB\":1480772137646768965,\"lT\":443,\"k\":0},\"7\":{\"rA\":0,\"rB\":1324401440806630210,\"lT\":444,\"k\":0},\"8\":{\"rA\":0,\"rB\":1411870610798131358,\"lT\":445,\"k\":0},\"9\":{\"rA\":0,\"rB\":1499396503277694755,\"lT\":446,\"k\":0}},\"binPosMap\":{\"426\":{\"0\":29},\"427\":{\"0\":30},\"428\":{\"0\":31},\"429\":{\"0\":32},\"430\":{\"0\":33},\"431\":{\"0\":34},\"432\":{\"0\":35},\"433\":{\"0\":36},\"434\":{\"0\":37,\"3\":1},\"435\":{\"0\":38},\"436\":{\"0\":39},\"437\":{\"0\":40},\"438\":{\"0\":41},\"439\":{\"0\":42},\"440\":{\"0\":3},\"441\":{\"0\":4},\"442\":{\"0\":5},\"443\":{\"0\":6},\"444\":{\"0\":7},\"445\":{\"0\":8},\"446\":{\"0\":9},\"447\":{\"0\":10},\"448\":{\"0\":11},\"449\":{\"0\":12},\"450\":{\"0\":13},\"451\":{\"0\":14},\"452\":{\"0\":15},\"453\":{\"0\":16},\"454\":{\"0\":17},\"455\":{\"0\":18,\"1\":22},\"456\":{\"0\":19},\"457\":{\"0\":20},\"458\":{\"0\":21},\"459\":{\"0\":43},\"460\":{\"0\":44},\"461\":{\"0\":45},\"462\":{\"0\":46},\"463\":{\"0\":47},\"464\":{\"0\":48},\"465\":{\"0\":49},\"466\":{\"0\":50}},\"binMap\":{\"6\":7719472615821092550409086380891770248800661236333969968563225660742308986880,\"7\":5037190915061491765521},\"liquidity\":2878336312141942146424,\"sqrtPriceX96\":73028492082257348963}","staticExtra":"{\"tickSpacing\":198}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	assert.EqualValues(t, 6, sim.state.minBinMapIndex)
	assert.EqualValues(t, 7, sim.state.maxBinMapIndex)

	testCases := []struct {
		tokenIn      string
		tokenOut     string
		amountIn     string
		expAmountOut string
		expNextTick  int32
	}{
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"10000000000000000000000", "1784415750858931428", 437},
		{"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"50000000000000000000000", "7995459204101958875", 443},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd",
			"5000000000000000000", "31297333580169335152628", 440},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd",
			"900000000000000000", "5437425224138046326996", 439},
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
			require.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick)

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
	poolRedis := `{"address":"0xbd278792260a68ee81a42adba23befdba87e30eb","reserveUsd":15864.05368210012,"amplifiedTvl":4.188628090182823e+41,"swapFee":0.0001,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1705301925,"reserves":["2938066235974926310","4262165529103738357"],"tokens":[{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":100000000000000,\"protocFeeRatio\":0,\"tick\":-8,\"bins\":{\"12\":{\"rA\":0,\"rB\":4242237013037562,\"lT\":-7,\"k\":3},\"43\":{\"rA\":0,\"rB\":1000000000000000000,\"lT\":343,\"k\":0},\"44\":{\"rA\":0,\"rB\":978339781816359,\"lT\":693,\"k\":0},\"45\":{\"rA\":0,\"rB\":957148728684,\"lT\":1043,\"k\":0},\"46\":{\"rA\":0,\"rB\":936416678,\"lT\":1393,\"k\":0},\"47\":{\"rA\":0,\"rB\":916133,\"lT\":1743,\"k\":0},\"48\":{\"rA\":1000000000000000000,\"rB\":0,\"lT\":-357,\"k\":0},\"49\":{\"rA\":978339781816359,\"rB\":0,\"lT\":-707,\"k\":0},\"5\":{\"rA\":1937086938107048420,\"rB\":516466083168804034,\"lT\":-8,\"k\":0},\"50\":{\"rA\":957148728684,\"rB\":0,\"lT\":-1057,\"k\":0},\"51\":{\"rA\":936416678,\"rB\":0,\"lT\":-1407,\"k\":0},\"52\":{\"rA\":916133,\"rB\":0,\"lT\":-1757,\"k\":0},\"6\":{\"rA\":0,\"rB\":2740477911054018869,\"lT\":-7,\"k\":0}},\"binPosMap\":{\"-1057\":{\"0\":50},\"-1407\":{\"0\":51},\"-1757\":{\"0\":52},\"-357\":{\"0\":48},\"-7\":{\"0\":6,\"3\":12},\"-707\":{\"0\":49},\"-8\":{\"0\":5},\"1043\":{\"0\":45},\"1393\":{\"0\":46},\"1743\":{\"0\":47},\"343\":{\"0\":43},\"693\":{\"0\":44}},\"binMap\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-12\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"-17\":21267647932558653966460912964485513216,\"-22\":16,\"-28\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"10\":6582018229284824168619876730229402019930943462534319453394436096,\"16\":75557863725914323419136,\"21\":100433627766186892221372630771322662657637687111424552206336,\"27\":1152921504606846976,\"5\":4951760157141521099596496896},\"liquidity\":259584104169810274047,\"sqrtPriceX96\":931321064190656607}","staticExtra":"{\"tickSpacing\":198}"}`
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
		expNextTick  int32
	}{
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"10000000000000000", "11527622736373110", -8},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"1000000000000000000", "1148063760188665172", -7},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"1900000000000000000", "2173275581711676990", -7},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"2890000000000000000", "3261217293522172144", 343},
		{"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0", "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"2900000000000000000", "3261228530154579452", 343},

		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			"50000000000000000", "43355832734019328", -8},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			"500000000000000000", "432859677357507302", -8},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			"5000000000000000000", "1939474350528474792", -357},
		{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
			"5000000000000000000000", "2938066235974285626", -1757},
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
			assert.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick)
		})
	}
}

func TestLsb(t *testing.T) {
	testCases := []struct {
		x        *uint256.Int
		expected uint
	}{
		{uint256.MustFromDecimal("4951760157141521099596496896"), 92},
		{uint256.MustFromDecimal("100433627766186892221372630771322662657637687111424552206336"), 196},
		{uint256.MustFromDecimal("126"), 1},
		{uint256.MustFromDecimal("28269553036454149273332760011886696253239742350009903329945699220681916416"), 244},
		{uint256.MustFromHex("0x100000000000000000000000000000000"), 128},
		{uint256.MustFromHex("0x100000000000000000000000000000001"), 0},
		{uint256.MustFromHex("0x10"), 4},
		{uint256.MustFromHex("0x10000000000000"), 52},
		{uint256.MustFromHex("0x10000000000000000"), 64},
		{uint256.MustFromHex("0x1000000000000000000000000000000000000000"), 156},
		{uint256.MustFromHex("0x1000000000000000000000000000000010000000000000000"), 64},
		{uint256.MustFromHex("0x7e000000000000000000000000000000000000000000000000"), 193},
		{uint256.MustFromHex("0xcde86a1e89d763bf0c446c982aab796721e864761df619b7000000005ef4a747"), 0},
		{uint256.MustFromHex("0x89d76318000000002aab7967cde86a1e1df619b70c446c980000a74721e86476"), 1},
	}
	for _, tt := range testCases {
		t.Run(tt.x.String(), func(t *testing.T) {
			assert.EqualValues(t, tt.expected, int(lsb(tt.x)))
		})
	}
}

func TestMsb(t *testing.T) {
	testCases := []struct {
		x        *uint256.Int
		expected uint
	}{
		{uint256.MustFromDecimal("4951760157141521099596496896"), 92},
		{uint256.MustFromDecimal("100433627766186892221372630771322662657637687111424552206336"), 196},
		{uint256.MustFromDecimal("126"), 6},
		{uint256.MustFromDecimal("28269553036454149273332760011886696253239742350009903329945699220681916416"), 244},
		{uint256.MustFromHex("0x100000000000000000000000000000000"), 128},
		{uint256.MustFromHex("0x100000000000000000000000000000001"), 128},
		{uint256.MustFromHex("0x10"), 4},
		{uint256.MustFromHex("0x10000000000000"), 52},
		{uint256.MustFromHex("0x10000000000000000"), 64},
		{uint256.MustFromHex("0x1000000000000000000000000000000000000000"), 156},
		{uint256.MustFromHex("0x1000000000000000000000000000000010000000000000000"), 192},
		{uint256.MustFromHex("0x7e000000000000000000000000000000000000000000000000"), 198},
		{uint256.MustFromHex("0xcde86a1e89d763bf0c446c982aab796721e864761df619b7000000005ef4a747"), 255},
		{uint256.MustFromHex("0x89d76318000000002aab7967cde86a1e1df619b70c446c980000a74721e86476"), 255},
	}
	for _, tt := range testCases {
		t.Run(tt.x.String(), func(t *testing.T) {
			assert.EqualValues(t, tt.expected, int(msb(tt.x)))
		})
	}
}

func TestGas(t *testing.T) {
	poolRedis := `{"address":"0xbd278792260a68ee81a42adba23befdba87e30eb","reserveUsd":15059.478927527987,"amplifiedTvl":4.184931466034053e+41,"swapFee":0.0001,"exchange":"maverick-v1","type":"maverick-v1","timestamp":1706603958,"reserves":["2722240380725257133","4511247270069585288"],"tokens":[{"address":"A","decimals":18,"weight":50,"swappable":true},{"address":"B","decimals":18,"weight":50,"swappable":true}],"extra":"{\"fee\":100000000000000,\"protocFeeRatio\":0,\"tick\":-8,\"bins\":{\"12\":{\"rA\":0,\"rB\":4242237013037562,\"lT\":-7,\"k\":3},\"43\":{\"rA\":0,\"rB\":1000000000000000000,\"lT\":343,\"k\":0},\"44\":{\"rA\":0,\"rB\":978339781816359,\"lT\":693,\"k\":0},\"45\":{\"rA\":0,\"rB\":957148728684,\"lT\":1043,\"k\":0},\"46\":{\"rA\":0,\"rB\":936416678,\"lT\":1393,\"k\":0},\"47\":{\"rA\":0,\"rB\":916133,\"lT\":1743,\"k\":0},\"48\":{\"rA\":1000000000000000000,\"rB\":0,\"lT\":-357,\"k\":0},\"49\":{\"rA\":978339781816359,\"rB\":0,\"lT\":-707,\"k\":0},\"5\":{\"rA\":1721261082857379243,\"rB\":765547824134650965,\"lT\":-8,\"k\":0},\"50\":{\"rA\":957148728684,\"rB\":0,\"lT\":-1057,\"k\":0},\"51\":{\"rA\":936416678,\"rB\":0,\"lT\":-1407,\"k\":0},\"52\":{\"rA\":916133,\"rB\":0,\"lT\":-1757,\"k\":0},\"6\":{\"rA\":0,\"rB\":2740477911054018869,\"lT\":-7,\"k\":0}},\"binPosMap\":{\"-1057\":{\"0\":50},\"-1407\":{\"0\":51},\"-1757\":{\"0\":52},\"-357\":{\"0\":48},\"-7\":{\"0\":6,\"3\":12},\"-707\":{\"0\":49},\"-8\":{\"0\":5},\"1043\":{\"0\":45},\"1393\":{\"0\":46},\"1743\":{\"0\":47},\"343\":{\"0\":43},\"693\":{\"0\":44}},\"binMap\":{\"-1\":3909192266736842770226717187617846447677385941268383009760023486136320,\"-12\":28269553036454149273332760011886696253239742350009903329945699220681916416,\"-17\":21267647932558653966460912964485513216,\"-22\":16,\"-28\":1393796574908163946345982392040522594123776,\"-6\":324518553658426726783156020576256,\"10\":6582018229284824168619876730229402019930943462534319453394436096,\"16\":75557863725914323419136,\"21\":100433627766186892221372630771322662657637687111424552206336,\"27\":1152921504606846976,\"5\":4951760157141521099596496896},\"liquidity\":259586774308826574234,\"sqrtPriceX96\":930489566587878568}","staticExtra":"{\"tickSpacing\":198}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	testCases := []struct {
		tokenIn     string
		tokenOut    string
		amountIn    string
		expNextTick int32
		expGas      int64
	}{
		{"A", "B", "10000000000000000", -8, 145000},    // use 1 tick, sim 110969
		{"A", "B", "1000000000000000000", -7, 165000},  // use 2 tick, sim 144530
		{"A", "B", "1900000000000000000", -7, 165000},  // use 2 tick, sim 144530
		{"A", "B", "3890000000000000000", 343, 185000}, // use 3 tick, sim 173534
		{"A", "B", "4000000000000000000", 343, 185000}, // use 3 tick, sim 173534

		{"B", "A", "50000000000000000", -8, 145000},         // use 1 tick, sim 106139
		{"B", "A", "500000000000000000", -8, 145000},        // use 1 tick, sim 106139
		{"B", "A", "5000000000000000000", -357, 165000},     // use 2 tick, sim 138592
		{"B", "A", "5000000000000000000000", -1757, 245000}, // use 6 tick, sim 273005
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

			assert.Equal(t, tc.expNextTick, result.SwapInfo.(maverickSwapInfo).activeTick)
			assert.Equal(t, tc.expGas, result.Gas)
		})
	}
}
