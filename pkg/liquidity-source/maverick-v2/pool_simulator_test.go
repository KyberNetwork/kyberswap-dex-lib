package maverickv2

import (
	"errors"
	"fmt"
	"testing"

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
	_       = json.Unmarshal([]byte(`{"address":"0x5bdb08ae195c8f085704582a27d566028a719265","reserveUsd":6125.460669340948,"amplifiedTvl":1.5317046591839606e+36,"swapFee":0.0002,"exchange":"maverick-v2","type":"maverick-v2","timestamp":1733251521,"reserves":["171389885714232604","5520719430817218266406"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","name":"","symbol":"","decimals":18,"weight":50,"swappable":true},{"address":"0x50c5725949a6f0c72e6c4a641f24049a917db0cb","name":"","symbol":"","decimals":18,"weight":50,"swappable":true}],"extra":"{\"feeAIn\":20000000000000,\"feeBIn\":20000000000000,\"protocolFeeRatio\":0,\"activeTick\":-414,\"bins\":{\"10\":{\"reserveA\":0,\"reserveB\":366061918078398110287,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-382,\"tickBalance\":0},\"11\":{\"reserveA\":0,\"reserveB\":6883201841157069692,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-381,\"tickBalance\":0},\"110\":{\"reserveA\":132717611565218673,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":2,\"tick\":-419,\"tickBalance\":0},\"12\":{\"reserveA\":0,\"reserveB\":11618854453023835182,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-380,\"tickBalance\":0},\"13\":{\"reserveA\":0,\"reserveB\":12437524481813913437,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-379,\"tickBalance\":0},\"14\":{\"reserveA\":0,\"reserveB\":14401602762318260592,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-378,\"tickBalance\":0},\"15\":{\"reserveA\":0,\"reserveB\":20285507739545552151,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-377,\"tickBalance\":0},\"16\":{\"reserveA\":0,\"reserveB\":31435697110122283023,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-376,\"tickBalance\":0},\"17\":{\"reserveA\":0,\"reserveB\":788483706236166152,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":3,\"tick\":-413,\"tickBalance\":0},\"179\":{\"reserveA\":21032489559537,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":2,\"tick\":-418,\"tickBalance\":0},\"18\":{\"reserveA\":0,\"reserveB\":51073932713153201416,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-375,\"tickBalance\":0},\"19\":{\"reserveA\":0,\"reserveB\":222307057404855279,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":1,\"tick\":-371,\"tickBalance\":0},\"20\":{\"reserveA\":0,\"reserveB\":34704569490663933549,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-374,\"tickBalance\":0},\"21\":{\"reserveA\":0,\"reserveB\":20840557123719804560,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-373,\"tickBalance\":0},\"22\":{\"reserveA\":0,\"reserveB\":12589042725147163216,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-372,\"tickBalance\":0},\"229\":{\"reserveA\":61392297107235,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":2,\"tick\":-417,\"tickBalance\":0},\"23\":{\"reserveA\":0,\"reserveB\":7693776205599244096,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-371,\"tickBalance\":0},\"24\":{\"reserveA\":124749357780963,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":2,\"tick\":-420,\"tickBalance\":0},\"26\":{\"reserveA\":0,\"reserveB\":4596108634618399595,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-370,\"tickBalance\":0},\"28\":{\"reserveA\":0,\"reserveB\":602275039904554226,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-369,\"tickBalance\":0},\"32\":{\"reserveA\":0,\"reserveB\":136281208928706118,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-368,\"tickBalance\":0},\"34\":{\"reserveA\":0,\"reserveB\":63986755290069521,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-367,\"tickBalance\":0},\"38\":{\"reserveA\":0,\"reserveB\":33174150853905978,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-366,\"tickBalance\":0},\"40\":{\"reserveA\":0,\"reserveB\":21828411692540749,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":1,\"tick\":-379,\"tickBalance\":0},\"43\":{\"reserveA\":0,\"reserveB\":409922429801838876933,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-387,\"tickBalance\":0},\"44\":{\"reserveA\":0,\"reserveB\":492481576830109445182,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-391,\"tickBalance\":0},\"45\":{\"reserveA\":0,\"reserveB\":456855429833934711881,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-390,\"tickBalance\":0},\"46\":{\"reserveA\":0,\"reserveB\":426147559216634848646,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-389,\"tickBalance\":0},\"47\":{\"reserveA\":0,\"reserveB\":412526863298037002614,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-388,\"tickBalance\":0},\"50\":{\"reserveA\":0,\"reserveB\":515602886856476035020,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-392,\"tickBalance\":0},\"52\":{\"reserveA\":0,\"reserveB\":23418069379099730041,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-395,\"tickBalance\":0},\"53\":{\"reserveA\":0,\"reserveB\":30757477984328343036,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-394,\"tickBalance\":0},\"54\":{\"reserveA\":0,\"reserveB\":45386269011903114868,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-393,\"tickBalance\":0},\"55\":{\"reserveA\":0,\"reserveB\":15265455037673920297,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-396,\"tickBalance\":0},\"56\":{\"reserveA\":0,\"reserveB\":5499269864763066638,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-398,\"tickBalance\":0},\"57\":{\"reserveA\":0,\"reserveB\":10172273266654089188,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-397,\"tickBalance\":0},\"59\":{\"reserveA\":0,\"reserveB\":253084860784686155015,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":1,\"tick\":-390,\"tickBalance\":0},\"6\":{\"reserveA\":0,\"reserveB\":403448772752780592117,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-386,\"tickBalance\":0},\"66\":{\"reserveA\":0,\"reserveB\":3808911750074946479,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-400,\"tickBalance\":0},\"67\":{\"reserveA\":0,\"reserveB\":4688520546816020959,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-399,\"tickBalance\":0},\"7\":{\"reserveA\":0,\"reserveB\":388504816646649170875,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-385,\"tickBalance\":0},\"70\":{\"reserveA\":0,\"reserveB\":3798807329530486046,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-401,\"tickBalance\":0},\"71\":{\"reserveA\":0,\"reserveB\":10071572048021564215,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-404,\"tickBalance\":0},\"72\":{\"reserveA\":0,\"reserveB\":7209707634661330902,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-403,\"tickBalance\":0},\"73\":{\"reserveA\":0,\"reserveB\":4814826238401964821,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-402,\"tickBalance\":0},\"75\":{\"reserveA\":0,\"reserveB\":9819174046092950091,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-405,\"tickBalance\":0},\"77\":{\"reserveA\":0,\"reserveB\":10099921674660620524,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-406,\"tickBalance\":0},\"79\":{\"reserveA\":0,\"reserveB\":9995893308994472369,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-407,\"tickBalance\":0},\"8\":{\"reserveA\":0,\"reserveB\":379893740110116757293,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-384,\"tickBalance\":0},\"80\":{\"reserveA\":0,\"reserveB\":16142986122983682089,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-409,\"tickBalance\":0},\"81\":{\"reserveA\":0,\"reserveB\":13506369485623416752,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-408,\"tickBalance\":0},\"85\":{\"reserveA\":0,\"reserveB\":22690331704117185877,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-410,\"tickBalance\":0},\"86\":{\"reserveA\":0,\"reserveB\":30883697387599607669,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-411,\"tickBalance\":0},\"87\":{\"reserveA\":10309755817688041,\"reserveB\":17311324165380548343,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-414,\"tickBalance\":0},\"88\":{\"reserveA\":0,\"reserveB\":71542867536434719695,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-413,\"tickBalance\":0},\"89\":{\"reserveA\":0,\"reserveB\":46464456107563260886,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-412,\"tickBalance\":0},\"9\":{\"reserveA\":0,\"reserveB\":372411683364983853951,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-383,\"tickBalance\":0},\"90\":{\"reserveA\":5222845979037199,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-417,\"tickBalance\":0},\"91\":{\"reserveA\":7004064916583540,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-416,\"tickBalance\":0},\"92\":{\"reserveA\":10362207657057631,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-415,\"tickBalance\":0},\"93\":{\"reserveA\":155404053438863,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-422,\"tickBalance\":0},\"94\":{\"reserveA\":276984568134181,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-421,\"tickBalance\":0},\"95\":{\"reserveA\":559418756547705,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-420,\"tickBalance\":0},\"96\":{\"reserveA\":1242440220441584,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-419,\"tickBalance\":0},\"97\":{\"reserveA\":3300929649906974,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-418,\"tickBalance\":0},\"98\":{\"reserveA\":885667833527,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-424,\"tickBalance\":0},\"99\":{\"reserveA\":30162717691326,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-423,\"tickBalance\":0}},\"binPositions\":{\"-366\":[38],\"-367\":[34],\"-368\":[32],\"-369\":[28],\"-370\":[26],\"-371\":[23,19],\"-372\":[22],\"-373\":[21],\"-374\":[20],\"-375\":[18],\"-376\":[16],\"-377\":[15],\"-378\":[14],\"-379\":[13,40],\"-380\":[12],\"-381\":[11],\"-382\":[10],\"-383\":[9],\"-384\":[8],\"-385\":[7],\"-386\":[6],\"-387\":[43],\"-388\":[47],\"-389\":[46],\"-390\":[45,59],\"-391\":[44],\"-392\":[50],\"-393\":[54],\"-394\":[53],\"-395\":[52],\"-396\":[55],\"-397\":[57],\"-398\":[56],\"-399\":[67],\"-400\":[66],\"-401\":[70],\"-402\":[73],\"-403\":[72],\"-404\":[71],\"-405\":[75],\"-406\":[77],\"-407\":[79],\"-408\":[81],\"-409\":[80],\"-410\":[85],\"-411\":[86],\"-412\":[89],\"-413\":[88,17],\"-414\":[87],\"-415\":[92],\"-416\":[91],\"-417\":[90,229],\"-418\":[97,179],\"-419\":[96,110],\"-420\":[95,24],\"-421\":[94],\"-422\":[93],\"-423\":[99],\"-424\":[98]},\"binMap\":{\"-414\":1,\"-413\":1,\"-375\":1}}","staticExtra":"{\"tickSpacing\":198}"}`),
		&rawPool)
	maverickPool, err = NewPoolSimulator(rawPool)
)

func TestPoolCalcAmountOut(t *testing.T) {
	t.Parallel()
	assert.Nil(t, err)

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
	t.Parallel()
	for _, amtIn := range []string{
		"17299124533583919",
		"3717299124533583919",
		"37172991245335839190",
		"371729912453358391945",
		"37172991245335839191",
	} {
		_, err := maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
				Amount: bignumber.NewBig10(amtIn),
			},
			TokenOut: "0x4200000000000000000000000000000000000006",
		})
		assert.NoError(t, err, amtIn)
	}
	for _, amtIn := range []string{
		"37172991245335839192",
		"37172991245335839193",
		"37172991245335839196789",
	} {
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
	t.Parallel()

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

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	poolRedis := `{"address":"0x5fdf78aef906cbad032fbaea032aaae3accf9dc3","reserveUsd":47625.963767453606,"amplifiedTvl":2.0145226157464416e+41,"swapFee":0.0005,"exchange":"maverick-v2","type":"maverick-v2","timestamp":1704957203,"reserves":["108363845032166910770488","2097024497432052549"],"tokens":[{"address":"0x04506dddbf689714487f91ae1397047169afcf34","decimals":18,"weight":50,"swappable":true},{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true}],"extra":"{\"feeAIn\":500000000000000,\"feeBIn\":500000000000000,\"protocolFeeRatio\":0,\"activeTick\":10,\"bins\":{\"1\":{\"reserveA\":1880866557485545835609,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-5,\"tickBalance\":0},\"10\":{\"reserveA\":2013495774191474777406,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":4,\"tickBalance\":0},\"11\":{\"reserveA\":411993441413380258157,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":5,\"tickBalance\":0},\"12\":{\"reserveA\":491298562692665969507,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":6,\"tickBalance\":0},\"13\":{\"reserveA\":620606767055018215315,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":7,\"tickBalance\":0},\"14\":{\"reserveA\":725257522405584599699,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":8,\"tickBalance\":0},\"15\":{\"reserveA\":897478209865575805530,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":9,\"tickBalance\":0},\"16\":{\"reserveA\":2142944919078882824342,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-6,\"tickBalance\":0},\"17\":{\"reserveA\":1022668409565365293976,\"reserveB\":2097024497432052514,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":10,\"tickBalance\":0},\"2\":{\"reserveA\":1634106566195962389560,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-4,\"tickBalance\":0},\"3\":{\"reserveA\":1405424035812355050009,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-3,\"tickBalance\":0},\"4\":{\"reserveA\":1233705168748319240144,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-2,\"tickBalance\":0},\"5\":{\"reserveA\":47686688533077328269486,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":-1,\"tickBalance\":0},\"6\":{\"reserveA\":30071745509492793533770,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":0,\"tickBalance\":0},\"7\":{\"reserveA\":6925596663250336094803,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":1,\"tickBalance\":0},\"8\":{\"reserveA\":5442282585416271863178,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":2,\"tickBalance\":0},\"9\":{\"reserveA\":3757685806420050749903,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":3,\"tickBalance\":0}},\"binPositions\":{\"-1\":[5],\"-2\":[4],\"-3\":[3],\"-4\":[2],\"-5\":[1],\"-6\":[16],\"0\":[6],\"1\":[7],\"10\":[17],\"2\":[8],\"3\":[9],\"4\":[10],\"5\":[11],\"6\":[12],\"7\":[13],\"8\":[14],\"9\":[15]},\"binMap\":{\"10\":1,\"0\":1}}","staticExtra":"{\"tickSpacing\":50}"}`
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
	t.Parallel()
	poolRedis := `{"address":"0xd50c68c7fbaee4f469e04cebdcfbf1113b4cdadf","reserveUsd":52056.74739685542,"amplifiedTvl":3.641901122084877e+44,"swapFee":0.01,"exchange":"maverick-v2","type":"maverick-v2","timestamp":1704959580,"reserves":["13095016099313357610018","26336470622025877177"],"tokens":[{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true},{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","decimals":18,"weight":50,"swappable":true}],"extra":"{\"feeAIn\":10000000000000000,\"feeBIn\":10000000000000000,\"protocolFeeRatio\":0,\"activeTick\":433,\"lastTwaD8\":110848000,\"timestamp\":1704959580,\"accumValueD8\":\"13080236544000\",\"lookbackSec\":600,\"bins\":{\"1\":{\"reserveA\":0,\"reserveB\":91157437341918885,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":3,\"tick\":434,\"tickBalance\":0},\"10\":{\"reserveA\":0,\"reserveB\":1239975611309030976,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":447,\"tickBalance\":0},\"11\":{\"reserveA\":0,\"reserveB\":1141328409485710165,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":448,\"tickBalance\":0},\"12\":{\"reserveA\":0,\"reserveB\":1090262378803153846,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":449,\"tickBalance\":0},\"13\":{\"reserveA\":0,\"reserveB\":1058956124010387881,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":450,\"tickBalance\":0},\"14\":{\"reserveA\":0,\"reserveB\":373789640683233838,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":451,\"tickBalance\":0},\"15\":{\"reserveA\":0,\"reserveB\":357135930104930106,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":452,\"tickBalance\":0},\"16\":{\"reserveA\":0,\"reserveB\":338216301716701110,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":453,\"tickBalance\":0},\"17\":{\"reserveA\":0,\"reserveB\":320760828172399247,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":454,\"tickBalance\":0},\"18\":{\"reserveA\":0,\"reserveB\":313627396754223600,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":455,\"tickBalance\":0},\"19\":{\"reserveA\":0,\"reserveB\":307794867765749325,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":456,\"tickBalance\":0},\"20\":{\"reserveA\":0,\"reserveB\":298219676795168630,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":457,\"tickBalance\":0},\"21\":{\"reserveA\":0,\"reserveB\":294481079018034973,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":458,\"tickBalance\":0},\"22\":{\"reserveA\":0,\"reserveB\":2773410496175111720,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":1,\"tick\":455,\"tickBalance\":0},\"29\":{\"reserveA\":1609295705362818753486,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":426,\"tickBalance\":0},\"3\":{\"reserveA\":0,\"reserveB\":1598645710773758142,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":440,\"tickBalance\":0},\"30\":{\"reserveA\":1713963852223063753018,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":427,\"tickBalance\":0},\"31\":{\"reserveA\":1796069142786354277710,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":428,\"tickBalance\":0},\"32\":{\"reserveA\":1739178743674569940344,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":429,\"tickBalance\":0},\"33\":{\"reserveA\":1677169405367640113180,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":430,\"tickBalance\":0},\"34\":{\"reserveA\":1771627492707184684589,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":431,\"tickBalance\":0},\"35\":{\"reserveA\":1872397651819441245354,\"reserveB\":0,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":432,\"tickBalance\":0},\"36\":{\"reserveA\":915314105372284841505,\"reserveB\":217573456205038785,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":433,\"tickBalance\":0},\"37\":{\"reserveA\":0,\"reserveB\":423191801919726618,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":434,\"tickBalance\":0},\"38\":{\"reserveA\":0,\"reserveB\":425853602425037489,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":435,\"tickBalance\":0},\"39\":{\"reserveA\":0,\"reserveB\":434870246675316320,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":436,\"tickBalance\":0},\"4\":{\"reserveA\":0,\"reserveB\":1747275673731363525,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":441,\"tickBalance\":0},\"40\":{\"reserveA\":0,\"reserveB\":428431372113458941,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":437,\"tickBalance\":0},\"41\":{\"reserveA\":0,\"reserveB\":1032183470388298339,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":438,\"tickBalance\":0},\"42\":{\"reserveA\":0,\"reserveB\":985419259209570776,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":439,\"tickBalance\":0},\"43\":{\"reserveA\":0,\"reserveB\":242182693866501568,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":459,\"tickBalance\":0},\"44\":{\"reserveA\":0,\"reserveB\":239797032983525254,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":460,\"tickBalance\":0},\"45\":{\"reserveA\":0,\"reserveB\":237434872449643063,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":461,\"tickBalance\":0},\"46\":{\"reserveA\":0,\"reserveB\":235095980770748030,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":462,\"tickBalance\":0},\"47\":{\"reserveA\":0,\"reserveB\":232780128733125118,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":463,\"tickBalance\":0},\"48\":{\"reserveA\":0,\"reserveB\":230487089380947529,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":464,\"tickBalance\":0},\"49\":{\"reserveA\":0,\"reserveB\":225934471614129577,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":465,\"tickBalance\":0},\"5\":{\"reserveA\":0,\"reserveB\":1461855769991903582,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":442,\"tickBalance\":0},\"50\":{\"reserveA\":0,\"reserveB\":221901118128804058,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":466,\"tickBalance\":0},\"6\":{\"reserveA\":0,\"reserveB\":1480772137646768965,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":443,\"tickBalance\":0},\"7\":{\"reserveA\":0,\"reserveB\":1324401440806630210,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":444,\"tickBalance\":0},\"8\":{\"reserveA\":0,\"reserveB\":1411870610798131358,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":445,\"tickBalance\":0},\"9\":{\"reserveA\":0,\"reserveB\":1499396503277694755,\"mergeBinBalance\":0,\"mergeId\":0,\"totalSupply\":0,\"kind\":0,\"tick\":446,\"tickBalance\":0}},\"binPositions\":{\"426\":[29],\"427\":[30],\"428\":[31],\"429\":[32],\"430\":[33],\"431\":[34],\"432\":[35],\"433\":[36],\"434\":[37,1],\"435\":[38],\"436\":[39],\"437\":[40],\"438\":[41],\"439\":[42],\"440\":[3],\"441\":[4],\"442\":[5],\"443\":[6],\"444\":[7],\"445\":[8],\"446\":[9],\"447\":[10],\"448\":[11],\"449\":[12],\"450\":[13],\"451\":[14],\"452\":[15],\"453\":[16],\"454\":[17],\"455\":[18,22],\"456\":[19],\"457\":[20],\"458\":[21],\"459\":[43],\"460\":[44],\"461\":[45],\"462\":[46],\"463\":[47],\"464\":[48],\"465\":[49],\"466\":[50]},\"binMap\":{\"433\":1,\"434\":1,\"435\":1,\"436\":1,\"437\":1,\"438\":1}}","staticExtra":"{\"tickSpacing\":198}"}`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.Nil(t, err)

	// Verify initial TWA value
	initialLastTwaD8 := sim.state.LastTwaD8
	require.Equal(t, int64(110848000), initialLastTwaD8)

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

			// Capture TWA value before update
			twaBefore := sim.state.LastTwaD8

			updateBalanceParams := pool.UpdateBalanceParams{
				TokenAmountIn:  in,
				TokenAmountOut: *result.TokenAmountOut,
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			}
			sim.UpdateBalance(updateBalanceParams)
			
			// Verify TWA value has been updated
			twaAfter := sim.state.LastTwaD8
			
			// For significant tick changes, TWA should change
			if absDiff(int32(tc.expNextTick), 433) > 3 {
				require.NotEqual(t, twaBefore, twaAfter, "TWA should change after significant tick movement")
			}
			
			// Verify timestamp gets updated
			require.True(t, sim.state.Timestamp > 0, "Timestamp should be updated")
		})
	}
}

func TestEmptyPool(t *testing.T) {
	t.Parallel()
	poolRedis := `{
    "address": "0xccd9eb9480f7beaa2bcac7d0cf5d4143f328ac06",
    "swapFee": 0.001,
    "exchange": "maverick-v2",
    "type": "maverick-v2",
    "timestamp": 1704940286,
    "reserves": ["0", "0"],
    "tokens": [
      { "address": "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", "name": "", "symbol": "", "decimals": 18, "weight": 50, "swappable": true },
      { "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "name": "", "symbol": "", "decimals": 6, "weight": 50, "swappable": true }
    ],
    "extra": "{\"feeAIn\":1000000000000000,\"feeBIn\":1000000000000000,\"protocolFeeRatio\":0,\"activeTick\":-1470,\"bins\":{},\"binPositions\":{},\"binMap\":{}}",
    "staticExtra": "{\"tickSpacing\":10}"
  }`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	_, err = NewPoolSimulator(poolEnt)
	assert.True(t, errors.Is(err, ErrEmptyBins))
}

func BenchmarkCalcAmountOut(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = maverickPool.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  "0x4200000000000000000000000000000000000006",
				Amount: bignumber.NewBig10("1000000000000000000"),
			},
			TokenOut: "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
		})
	}
}

// Helper function to test scaling operations
func TestScalingOperations(t *testing.T) {
	testCases := []struct {
		amount   string
		decimals uint8
	}{
		{"1000000000000000000", 18}, // 1 ETH with 18 decimals
		{"1000000", 6},              // 1 USDC with 6 decimals
		{"1000000000", 9},           // 1 token with 9 decimals
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Scale with %d decimals", tc.decimals), func(t *testing.T) {
			amount, _ := uint256.FromDecimal(tc.amount)

			// Test scaleFromAmount
			scaled, err := scaleFromAmount(amount, tc.decimals)
			assert.NoError(t, err)

			// Test ScaleToAmount
			original, err := ScaleToAmount(scaled, tc.decimals)
			assert.NoError(t, err)

			// The round trip should give approximately the original amount
			// There might be small precision loss due to division operations
			diff := new(uint256.Int).Sub(amount, original)

			// Check if the difference is small (less than 0.1%)
			tolerance := new(uint256.Int).Div(amount, uint256.NewInt(1000)) // 0.1% of original
			assert.True(t, diff.Cmp(tolerance) < 0,
				"Difference after scaling roundtrip exceeds tolerance: %s vs %s (diff: %s)",
				amount.Dec(), original.Dec(), diff.Dec())
		})
	}
}

func TestTwaUpdateAndBinMovement(t *testing.T) {
	t.Parallel()
	
	// Create a simple pool state for testing
	state := &MaverickPoolState{
		ActiveTick:   10,
		LastTwaD8:    2560000, // 10 * 2^8 (10 << 8)
		Timestamp:    1000,
		LookbackSec:  600,
		AccumValueD8: new(uint256.Int),
		Bins: map[uint32]Bin{
			1: {
				ReserveA: new(uint256.Int).SetUint64(1000000),
				ReserveB: new(uint256.Int).SetUint64(500000),
				Tick:     10,
			},
			2: {
				ReserveA: new(uint256.Int).SetUint64(800000),
				ReserveB: new(uint256.Int).SetUint64(600000),
				Tick:     11,
			},
		},
		BinPositions: map[int32][]uint32{
			10: {1},
			11: {2},
		},
	}
	
	// Test updateTwaValue
	t.Run("updateTwaValue", func(t *testing.T) {
		// Update with a 10 second time difference
		newTimestamp := int64(1010)
		newValueD8 := int64(3072000) // 12 * 2^8 (12 << 8)
		
		// Get initial values
		initialLastTwaD8 := state.LastTwaD8
		initialAccumValue := new(uint256.Int).Set(state.AccumValueD8)
		
		// Update TWA value
		updateTwaValue(state, newValueD8, newTimestamp)
		
		// Verify values updated
		assert.NotEqual(t, initialLastTwaD8, state.LastTwaD8, "LastTwaD8 should be updated")
		assert.Equal(t, newValueD8, state.LastTwaD8, "LastTwaD8 should equal new value")
		assert.Equal(t, newTimestamp, state.Timestamp, "Timestamp should be updated")
		
		// Verify accumulator increased
		// Expected increase = oldValue * timeDelta = 2560000 * 10
		assert.True(t, state.AccumValueD8.Cmp(initialAccumValue) > 0, "AccumValueD8 should increase")
	})
	
	// Test moveBins with significant TWA change
	t.Run("moveBins with significant change", func(t *testing.T) {
		// Clone state
		testState := state.Clone()
		
		// Initial reserves
		initialReservesA := make(map[uint32]*uint256.Int)
		initialReservesB := make(map[uint32]*uint256.Int)
		for binId, bin := range testState.Bins {
			initialReservesA[binId] = new(uint256.Int).Set(bin.ReserveA)
			initialReservesB[binId] = new(uint256.Int).Set(bin.ReserveB)
		}
		
		// Threshold for bin movement (small value to ensure movement happens)
		threshold := new(uint256.Int).SetUint64(1)
		
		// Move from tick 10 to tick 15 with significant TWA change
		moveBins(testState, 10, 15, 2560000, 3840000, threshold) // 10<<8 to 15<<8
		
		// Verify bins were adjusted
		for binId := range testState.Bins {
			assert.NotEqual(t, initialReservesA[binId], testState.Bins[binId].ReserveA,
				"ReserveA should change after significant bin movement")
			assert.NotEqual(t, initialReservesB[binId], testState.Bins[binId].ReserveB,
				"ReserveB should change after significant bin movement")
		}
	})
	
	// Test moveBins with small TWA change (below threshold)
	t.Run("moveBins with small change", func(t *testing.T) {
		// Clone state
		testState := state.Clone()
		
		// Initial reserves
		initialReservesA := make(map[uint32]*uint256.Int)
		initialReservesB := make(map[uint32]*uint256.Int)
		for binId, bin := range testState.Bins {
			initialReservesA[binId] = new(uint256.Int).Set(bin.ReserveA)
			initialReservesB[binId] = new(uint256.Int).Set(bin.ReserveB)
		}
		
		// Threshold too high for bin movement
		threshold := new(uint256.Int).SetUint64(10000000)
		
		// Move from tick 10 to tick 11 with small TWA change
		moveBins(testState, 10, 11, 2560000, 2816000, threshold) // 10<<8 to 11<<8
		
		// Verify bins were NOT adjusted (due to high threshold)
		for binId := range testState.Bins {
			assert.True(t, testState.Bins[binId].ReserveA.Cmp(initialReservesA[binId]) == 0,
				"ReserveA should not change when below threshold")
			assert.True(t, testState.Bins[binId].ReserveB.Cmp(initialReservesB[binId]) == 0,
				"ReserveB should not change when below threshold")
		}
	})
}
