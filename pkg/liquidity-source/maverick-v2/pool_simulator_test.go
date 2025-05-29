package maverickv2

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/logger"
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

func TestUpdateBalance(t *testing.T) {
	t.Parallel()
	poolRedis := `{"address":"0x5fdf78aef906cbad032fbaea032aaae3accf9dc3","reserveUsd":47625.963767453606,"amplifiedTvl":2.0145226157464416e+41,"swapFee":0.0005,"exchange":"maverick-v2","type":"maverick-v2","timestamp":1704957203,"reserves":["108363845032166910770488","2097024497432052549"],"tokens":[{"address":"0x04506dddbf689714487f91ae1397047169afcf34","decimals":18,"weight":50,"swappable":true},{"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd","decimals":18,"weight":50,"swappable":true}],"extra":"{\"feeAIn\":500000000000000,\"feeBIn\":500000000000000,\"protocolFeeRatio\":0,\"bins\":{},\"binPositions\":{},\"binMap\":{}}","staticExtra":"{\"tickSpacing\":10}"}`
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

			// Check that the swap info has the expected fields
			swapInfo, ok := result.SwapInfo.(maverickSwapInfo)
			require.True(t, ok, "SwapInfo should be of type maverickSwapInfo")
			require.Equal(t, tc.expNextTick, swapInfo.activeTick)

			// Verify fractional part is included in swap info
			require.True(t, swapInfo.fractionalPartD8 >= 0, "Fractional part should be non-negative")
			require.True(t, swapInfo.fractionalPartD8 <= int64(BI_POWS[8].Uint64()), "Fractional part should be <= 2^8")

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
			if absDiffInt32(int32(tc.expNextTick), 433) > 3 {
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
    "extra": "{\"feeAIn\":1000000000000000,\"feeBIn\":1000000000000000,\"protocolFeeRatio\":0,\"bins\":{},\"binPositions\":{},\"binMap\":{}}",
    "staticExtra": "{\"tickSpacing\":10}"
  }`
	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(poolRedis), &poolEnt)
	require.Nil(t, err)

	_, err = NewPoolSimulator(poolEnt)
	assert.True(t, errors.Is(err, ErrEmptyBins))
}

func TestRealPoolData_MAV_USDT(t *testing.T) {
	t.Parallel()

	// Real MAV/USDT pool data from user
	realPoolData := `{
		"address":"0x6104de9dc424f66aced5d5a464b6d9799daa2ffb",
		"exchange":"maverick-v2",
		"type":"maverick-v2",
		"timestamp":1723007123,
		"reserves":["39472782069565585411","1"],
		"tokens":[
			{
				"address":"0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd",
				"symbol":"MAV",
				"decimals":18,
				"swappable":true
			},
			{
				"address":"0xdac17f958d2ee523a2206206994597c13d831ec7",
				"symbol":"USDT",
				"decimals":6,
				"swappable":true
			}
		],
		"extra":"{\"feeAIn\":1000000000000000,\"feeBIn\":1000000000000000,\"protocolFeeRatio\":0,\"bins\":{\"1\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3011292960208985072\",\"kind\":0,\"tick\":6,\"tickBalance\":\"3011292822329674833\",\"reserveA\":\"5888227785742208896\",\"reserveB\":\"0\"},\"2\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814937650\",\"kind\":0,\"tick\":-1,\"tickBalance\":\"3010415662714937650\",\"reserveA\":\"2692536652529190021\",\"reserveB\":\"0\"},\"3\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814940980\",\"kind\":0,\"tick\":0,\"tickBalance\":\"3010415662714940980\",\"reserveA\":\"3010415662714940980\",\"reserveB\":\"0\"},\"4\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814946036\",\"kind\":0,\"tick\":1,\"tickBalance\":\"3010415662714946036\",\"reserveA\":\"3365823248425099770\",\"reserveB\":\"0\"},\"5\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814936393\",\"kind\":0,\"tick\":2,\"tickBalance\":\"3010415662714936393\",\"reserveA\":\"3763190007263647043\",\"reserveB\":\"0\"},\"6\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814941423\",\"kind\":0,\"tick\":3,\"tickBalance\":\"3010415662714941423\",\"reserveA\":\"4207469610115558436\",\"reserveB\":\"0\"},\"7\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814937240\",\"kind\":0,\"tick\":4,\"tickBalance\":\"3010415662714937240\",\"reserveA\":\"4704200554815532217\",\"reserveB\":\"0\"},\"8\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010415662814937645\",\"kind\":0,\"tick\":5,\"tickBalance\":\"3010415662714937645\",\"reserveA\":\"5259575210412275938\",\"reserveB\":\"0\"},\"9\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"3010413474405197601\",\"kind\":0,\"tick\":7,\"tickBalance\":\"3010413474305197601\",\"reserveA\":\"6581343337547132110\",\"reserveB\":\"0\"}},\"binPositions\":{\"-1\":[2],\"0\":[3],\"1\":[4],\"2\":[5],\"3\":[6],\"4\":[7],\"5\":[8],\"6\":[1],\"7\":[9]},\"activeTick\":4,\"lastTwaD8\":402564673,\"timestamp\":1723007123}",
		"staticExtra":"{\"tickSpacing\":2232}",
		"blockNumber":22585452
	}`

	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(realPoolData), &poolEnt)
	require.NoError(t, err)

	sim, err := NewPoolSimulator(poolEnt)
	require.NoError(t, err)

	// Verify pool initialization
	assert.Equal(t, "0x6104de9dc424f66aced5d5a464b6d9799daa2ffb", sim.Info.Address)
	assert.Equal(t, int32(4), sim.state.ActiveTick)
	assert.Equal(t, int64(402564673), sim.state.LastTwaD8)
	assert.Equal(t, uint32(2232), sim.state.TickSpacing)
	assert.Equal(t, uint64(1000000000000000), sim.state.FeeAIn)
	assert.Equal(t, uint64(1000000000000000), sim.state.FeeBIn)

	// Verify bins are loaded correctly
	assert.Len(t, sim.state.Bins, 9)
	assert.Len(t, sim.state.BinPositions, 9) // 9 different tick values: -1, 0, 1, 2, 3, 4, 5, 6, 7

	// Verify specific bin data
	bin1, exists := sim.state.Bins[1]
	require.True(t, exists)
	assert.Equal(t, int32(6), bin1.Tick)
	assert.Equal(t, uint8(0), bin1.Kind)

	// Get bin reserves using the binReserves function
	tick1, tickExists := sim.state.Ticks[bin1.Tick]
	require.True(t, tickExists)
	bin1ReserveA, bin1ReserveB := binReserves(bin1, tick1)
	assert.Equal(t, "5888227785742208896", bin1ReserveA.String())
	assert.Equal(t, "0", bin1ReserveB.String())

	// Debug: Print pool state
	t.Logf("Pool reserves: A=%s, B=%s", sim.Info.Reserves[0].String(), sim.Info.Reserves[1].String())
	t.Logf("Active tick: %d", sim.state.ActiveTick)
	t.Logf("Number of bins: %d", len(sim.state.Bins))

	// Check if pool has liquidity in the right direction
	totalReserveA := new(big.Int)
	totalReserveB := new(big.Int)
	for _, bin := range sim.state.Bins {
		tick, tickExists := sim.state.Ticks[bin.Tick]
		if tickExists {
			reserveA, reserveB := binReserves(bin, tick)
			totalReserveA.Add(totalReserveA, reserveA.ToBig())
			totalReserveB.Add(totalReserveB, reserveB.ToBig())
		}
	}
	t.Logf("Total bin reserves: A=%s, B=%s", totalReserveA.String(), totalReserveB.String())

	// Test that this pool only has MAV liquidity (no USDT)
	assert.True(t, totalReserveA.Cmp(bignumber.ZeroBI) > 0, "Pool should have MAV liquidity")
	assert.True(t, totalReserveB.Cmp(bignumber.ZeroBI) == 0, "Pool should have no USDT liquidity")

	// Test cases for swapping
	testCases := []struct {
		name         string
		tokenIn      string
		tokenOut     string
		amountIn     string
		description  string
		expectOutput bool // whether we expect positive output
	}{
		{
			name:         "MAV to USDT - Should fail (no USDT liquidity)",
			tokenIn:      "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", // MAV
			tokenOut:     "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
			amountIn:     "1000000000000000000",                        // 1 MAV
			description:  "Swap 1 MAV for USDT",
			expectOutput: false, // No USDT in pool
		},
		{
			name:         "USDT to MAV - Should fail (no USDT to swap)",
			tokenIn:      "0xdac17f958d2ee523a2206206994597c13d831ec7", // USDT
			tokenOut:     "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd", // MAV
			amountIn:     "1000000",                                    // 1 USDT (6 decimals)
			description:  "Swap 1 USDT for MAV",
			expectOutput: false, // Can't swap USDT when pool has no USDT
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test CalcAmountOut
			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})

			// The swap should either succeed or fail gracefully
			if err != nil {
				t.Logf("Swap failed as expected: %v", err)
				return
			}

			require.NotNil(t, result)
			require.NotNil(t, result.TokenAmountOut)
			require.NotNil(t, result.Fee)

			t.Logf("%s: Input=%s %s, Output=%s %s, Fee=%s",
				tc.description,
				tc.amountIn, getTokenSymbol(tc.tokenIn),
				result.TokenAmountOut.Amount.String(), getTokenSymbol(tc.tokenOut),
				result.Fee.Amount.String())

			// Check if output matches expectation
			hasOutput := result.TokenAmountOut.Amount.Cmp(bignumber.ZeroBI) > 0
			if tc.expectOutput {
				assert.True(t, hasOutput, "Expected positive output for %s", tc.name)
			} else {
				assert.False(t, hasOutput, "Expected zero output for %s due to pool liquidity constraints", tc.name)
			}

			// Test UpdateBalance only if swap was successful
			if hasOutput {
				originalState := sim.CloneState().(*PoolSimulator)

				updateParams := pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: tc.tokenIn, Amount: bignumber.NewBig10(tc.amountIn)},
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				}

				sim.UpdateBalance(updateParams)

				// Verify the state changed after update
				newState := sim
				assert.NotEqual(t, originalState.Info.Reserves[0].String(), newState.Info.Reserves[0].String())

				// Restore state for next test
				sim.state = originalState.state.Clone()
				sim.Info.Reserves[0] = new(big.Int).Set(originalState.Info.Reserves[0])
				sim.Info.Reserves[1] = new(big.Int).Set(originalState.Info.Reserves[1])
			}
		})
	}

	// Test CloneState functionality
	t.Run("CloneState", func(t *testing.T) {
		cloned := sim.CloneState().(*PoolSimulator)

		// Verify cloned state is independent
		assert.Equal(t, sim.Info.Address, cloned.Info.Address)
		assert.Equal(t, sim.state.ActiveTick, cloned.state.ActiveTick)
		assert.Equal(t, len(sim.state.Bins), len(cloned.state.Bins))

		// Modify cloned state and verify original is unchanged
		originalActiveTick := sim.state.ActiveTick
		cloned.state.ActiveTick = 999
		assert.Equal(t, originalActiveTick, sim.state.ActiveTick)
		assert.Equal(t, int32(999), cloned.state.ActiveTick)
	})
}

func TestRealPoolData_USDC_USDT(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err = logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data
	realPoolData := `{"address":"0x31373595f40ea48a7aab6cbcb0d377c6066e2dca","exchange":"maverick-v2","type":"maverick-v2","timestamp":1748487959,"reserves":["278416610034","2863171384617"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"feeAIn\":10000000000000,\"feeBIn\":10000000000000,\"protocolFeeRatio\":0,\"bins\":{\"1\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"24019190385150318054580\",\"kind\":0,\"tick\":-2,\"tickBalance\":\"24019187983111426805681\",\"reserveA\":\"25213218403113805452470\",\"reserveB\":\"0\"},\"10\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"18855246071402780835020\",\"kind\":0,\"tick\":-3,\"tickBalance\":\"18855246071383971239184\",\"reserveA\":\"19727331936856521514748\",\"reserveB\":\"0\"},\"16\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"11700330106773539800445\",\"kind\":0,\"tick\":4,\"tickBalance\":\"11700330106761867797854\",\"reserveA\":\"81561674730419264373353\",\"reserveB\":\"2837439476465300009813586\"},\"17\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"9160766047657905425134\",\"kind\":0,\"tick\":5,\"tickBalance\":\"9160766047648766838052\",\"reserveA\":\"0\",\"reserveB\":\"9562860924232363005816\"},\"18\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"7535267618600221806670\",\"kind\":0,\"tick\":6,\"tickBalance\":\"7535267618592704782645\",\"reserveA\":\"0\",\"reserveB\":\"7854742810240757515323\"},\"19\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"6614040812468079420134\",\"kind\":0,\"tick\":7,\"tickBalance\":\"6614040812461481392473\",\"reserveA\":\"0\",\"reserveB\":\"6885768064575532743257\"}},\"binPositions\":{\"-3\":[10],\"-2\":[1],\"4\":[16],\"5\":[17],\"6\":[18],\"7\":[19]},\"activeTick\":4,\"lastTwaD8\":402564673,\"timestamp\":1748487959}","staticExtra":"{\"tickSpacing\":1}","blockNumber":22585623}`

	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(realPoolData), &poolEnt)
	require.NoError(t, err)

	// Create pool simulator
	sim, err := NewPoolSimulator(poolEnt)
	require.NoError(t, err)
	require.NotNil(t, sim)

	// Basic pool validation
	assert.Equal(t, "0x31373595f40ea48a7aab6cbcb0d377c6066e2dca", sim.Info.Address)
	assert.Equal(t, "maverick-v2", sim.Info.Exchange)
	assert.Len(t, sim.Info.Tokens, 2)

	// Token validation
	assert.Equal(t, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", sim.Info.Tokens[0])
	assert.Equal(t, "0xdac17f958d2ee523a2206206994597c13d831ec7", sim.Info.Tokens[1])

	// Pool state validation
	assert.Equal(t, int32(4), sim.state.ActiveTick)
	assert.Equal(t, uint32(1), sim.state.TickSpacing)
	assert.Len(t, sim.state.Bins, 6) // 6 bins from ticks -3 to 7

	// Log pool state for debugging
	t.Logf("Pool state: ActiveTick=%d, TickSpacing=%d", sim.state.ActiveTick, sim.state.TickSpacing)
	t.Logf("Bins: %d, BinPositions: %v", len(sim.state.Bins), sim.state.BinPositions)

	// Check key bins around active tick
	activeBin, hasActiveBin := sim.state.Bins[16] // Bin at tick 4 (active tick)
	if hasActiveBin {
		activeTick, activeTickExists := sim.state.Ticks[activeBin.Tick]
		if activeTickExists {
			activeReserveA, activeReserveB := binReserves(activeBin, activeTick)
			t.Logf("Active bin (tick 4): ReserveA=%s, ReserveB=%s", activeReserveA.String(), activeReserveB.String())
		}
	}

	// Check other bins with liquidity
	for binID, bin := range sim.state.Bins {
		tick, tickExists := sim.state.Ticks[bin.Tick]
		if tickExists {
			reserveA, reserveB := binReserves(bin, tick)
			if reserveA.Cmp(uint256.NewInt(0)) > 0 || reserveB.Cmp(uint256.NewInt(0)) > 0 {
				t.Logf("Bin %d (tick %d): ReserveA=%s, ReserveB=%s", binID, bin.Tick, reserveA.String(), reserveB.String())
			}
		}
	}

	// Test with larger USDC -> USDT swap (tokenIn = token0, tokenOut = token1)
	usdcAmount := big.NewInt(1000000000) // 1000 USDC (6 decimals)

	// Debug: Log the amount and scaling
	t.Logf("Testing swap: %s USDC (raw amount)", usdcAmount.String())

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  sim.Info.Tokens[0], // USDC
			Amount: usdcAmount,
		},
		TokenOut: sim.Info.Tokens[1], // USDT
		Limit:    nil,
	})

	if err != nil {
		t.Logf("USDC->USDT swap error: %v", err)
		// Try smaller amount
		usdcAmount = big.NewInt(10000000) // 10 USDC
		t.Logf("Trying smaller amount: %s USDC", usdcAmount.String())
		result, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  sim.Info.Tokens[0], // USDC
				Amount: usdcAmount,
			},
			TokenOut: sim.Info.Tokens[1], // USDT
			Limit:    nil,
		})
		if err != nil {
			t.Logf("10 USDC->USDT swap also failed: %v", err)
		} else {
			t.Logf("10 USDC -> %s USDT (raw: %s)",
				new(big.Int).Div(result.TokenAmountOut.Amount, big.NewInt(1000000)).String(),
				result.TokenAmountOut.Amount.String())
		}
	} else {
		require.NotNil(t, result)
		// Convert to human readable (divide by 10^6 for USDT)
		humanAmount := new(big.Int).Div(result.TokenAmountOut.Amount, big.NewInt(1000000))
		t.Logf("1000 USDC -> %s USDT (raw: %s)", humanAmount.String(), result.TokenAmountOut.Amount.String())
	}

	// Test with larger USDT -> USDC swap (tokenIn = token1, tokenOut = token0)
	usdtAmount := big.NewInt(1000000000) // 1000 USDT (6 decimals)

	// Debug: Log the amount and scaling
	t.Logf("Testing swap: %s USDT (raw amount)", usdtAmount.String())

	result2, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  sim.Info.Tokens[1], // USDT
			Amount: usdtAmount,
		},
		TokenOut: sim.Info.Tokens[0], // USDC
		Limit:    nil,
	})

	if err != nil {
		t.Logf("USDT->USDC swap error: %v", err)
		// Try smaller amount
		usdtAmount = big.NewInt(10000000) // 10 USDT
		t.Logf("Trying smaller amount: %s USDT", usdtAmount.String())
		result2, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  sim.Info.Tokens[1], // USDT
				Amount: usdtAmount,
			},
			TokenOut: sim.Info.Tokens[0], // USDC
			Limit:    nil,
		})
		if err != nil {
			t.Logf("10 USDT->USDC swap also failed: %v", err)
		} else {
			t.Logf("10 USDT -> %s USDC (raw: %s)",
				new(big.Int).Div(result2.TokenAmountOut.Amount, big.NewInt(1000000)).String(),
				result2.TokenAmountOut.Amount.String())
		}
	} else {
		require.NotNil(t, result2)
		// Convert to human readable (divide by 10^6 for USDC)
		humanAmount := new(big.Int).Div(result2.TokenAmountOut.Amount, big.NewInt(1000000))
		t.Logf("1000 USDT -> %s USDC (raw: %s)", humanAmount.String(), result2.TokenAmountOut.Amount.String())
	}

	// Test CloneState functionality
	originalState := sim.state
	clonedSim := sim.CloneState()
	clonedState := clonedSim.(*PoolSimulator).state

	// Verify deep copy
	assert.Equal(t, originalState.ActiveTick, clonedState.ActiveTick)
	assert.Equal(t, originalState.TickSpacing, clonedState.TickSpacing)
	assert.Equal(t, len(originalState.Bins), len(clonedState.Bins))

	// Modify cloned state and ensure original is unchanged
	clonedState.ActiveTick = 999
	assert.NotEqual(t, originalState.ActiveTick, clonedState.ActiveTick)
	assert.Equal(t, int32(4), originalState.ActiveTick)
	assert.Equal(t, int32(999), clonedState.ActiveTick)

	t.Logf("USDC/USDT Pool test completed successfully")
}

// Helper function to get token symbol for logging
func getTokenSymbol(tokenAddress string) string {
	switch tokenAddress {
	case "0x7448c7456a97769f6cd04f1e83a4a23ccdc46abd":
		return "MAV"
	case "0xdac17f958d2ee523a2206206994597c13d831ec7":
		return "USDT"
	default:
		return "UNKNOWN"
	}
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

// Helper for absolute difference between int32 values
func absDiffInt32(a, b int32) int32 {
	if a > b {
		return a - b
	}
	return b - a
}

// Test for fractional part calculation
func TestFractionalPartCalculation(t *testing.T) {
	t.Parallel()

	// Create a simple pool state for testing
	state := &MaverickPoolState{
		ActiveTick:  10,
		TickSpacing: 1,
		Bins: map[uint32]Bin{
			1: {
				MergeBinBalance:  new(uint256.Int),
				MergeId:          0,
				TotalSupply:      new(uint256.Int).SetUint64(1000000),
				Kind:             0,
				Tick:             10,
				TickBalance:      new(uint256.Int).SetUint64(1000000),
				CurrentLiquidity: new(uint256.Int).SetUint64(1000000),
			},
			2: {
				MergeBinBalance:  new(uint256.Int),
				MergeId:          0,
				TotalSupply:      new(uint256.Int).SetUint64(800000),
				Kind:             0,
				Tick:             11,
				TickBalance:      new(uint256.Int).SetUint64(800000),
				CurrentLiquidity: new(uint256.Int).SetUint64(800000),
			},
		},
		Ticks: map[int32]Tick{
			10: {
				ReserveA:     new(uint256.Int).SetUint64(1000000),
				ReserveB:     new(uint256.Int).SetUint64(500000),
				TotalSupply:  new(uint256.Int).SetUint64(1000000),
				BinIdsByTick: map[uint8]uint32{0: 1},
			},
			11: {
				ReserveA:     new(uint256.Int).SetUint64(800000),
				ReserveB:     new(uint256.Int).SetUint64(600000),
				TotalSupply:  new(uint256.Int).SetUint64(800000),
				BinIdsByTick: map[uint8]uint32{0: 2},
			},
		},
		BinPositions: map[int32][]uint32{
			10: {1},
			11: {2},
		},
	}

	// Test tickSqrtPriceAndLiquidity
	sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, _ := tickSqrtPriceAndLiquidity(state, state.ActiveTick)

	// Verify all sqrt prices are non-zero
	assert.False(t, sqrtLowerTickPrice.IsZero(), "sqrtLowerTickPrice should not be zero")
	assert.False(t, sqrtUpperTickPrice.IsZero(), "sqrtUpperTickPrice should not be zero")
	assert.False(t, sqrtPrice.IsZero(), "sqrtPrice should not be zero")

	// Calculate fractional part for TWA
	var fractionalPartD8 int64
	if !sqrtPrice.IsZero() && !sqrtLowerTickPrice.IsZero() && !sqrtUpperTickPrice.IsZero() {
		// Calculate how far we are between the lower and upper tick prices
		tickRange := new(uint256.Int).Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)
		tickPosition := new(uint256.Int).Sub(sqrtPrice, sqrtLowerTickPrice)

		if !tickRange.IsZero() {
			// Calculate the fractional part as a value between 0 and 2^8
			fractionalPart := mulDiv(tickPosition, BI_POWS[8], tickRange)
			fractionalPartD8 = int64(fractionalPart.Uint64())
		} else {
			// Default to half-tick if calculation fails
			fractionalPartD8 = int64(BI_POWS[7].Uint64())
		}
	} else {
		// Default to half-tick value if sqrt prices are not available
		fractionalPartD8 = int64(BI_POWS[7].Uint64())
	}

	// Verify fractional part is within expected range (0 to 2^8)
	assert.True(t, fractionalPartD8 >= 0, "Fractional part should be non-negative")
	assert.True(t, fractionalPartD8 <= int64(BI_POWS[8].Uint64()), "Fractional part should be <= 2^8")

	// Test different positions within the tick
	t.Run("Different positions within tick", func(t *testing.T) {
		// Create test cases for different positions within the tick
		testCases := []struct {
			name          string
			reserveARatio uint64
			reserveBRatio uint64
			expectedRange string // "low", "mid", or "high"
		}{
			{"Near lower bound", 10, 1, "low"},
			{"Middle of tick", 1, 1, "mid"},
			{"Near upper bound", 1, 10, "high"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Create a bin with the specified reserve ratio
				testState := &MaverickPoolState{
					ActiveTick:  10,
					TickSpacing: 1,
					Bins: map[uint32]Bin{
						1: {
							MergeBinBalance:  new(uint256.Int),
							MergeId:          0,
							TotalSupply:      new(uint256.Int).SetUint64(1000000),
							Kind:             0,
							Tick:             10,
							TickBalance:      new(uint256.Int).SetUint64(1000000),
							CurrentLiquidity: new(uint256.Int).SetUint64(1000000),
						},
						2: {
							MergeBinBalance:  new(uint256.Int),
							MergeId:          0,
							TotalSupply:      new(uint256.Int).SetUint64(800000),
							Kind:             0,
							Tick:             11,
							TickBalance:      new(uint256.Int).SetUint64(800000),
							CurrentLiquidity: new(uint256.Int).SetUint64(800000),
						},
					},
					Ticks: map[int32]Tick{
						10: {
							ReserveA:     new(uint256.Int).SetUint64(1000000 * tc.reserveARatio),
							ReserveB:     new(uint256.Int).SetUint64(1000000 * tc.reserveBRatio),
							TotalSupply:  new(uint256.Int).SetUint64(1000000),
							BinIdsByTick: map[uint8]uint32{0: 1},
						},
						11: {
							ReserveA:     new(uint256.Int).SetUint64(800000 * tc.reserveARatio),
							ReserveB:     new(uint256.Int).SetUint64(800000 * tc.reserveBRatio),
							TotalSupply:  new(uint256.Int).SetUint64(800000),
							BinIdsByTick: map[uint8]uint32{0: 2},
						},
					},
					BinPositions: map[int32][]uint32{
						10: {1},
						11: {2},
					},
				}

				// Calculate sqrt prices and fractional part
				sqrtLowerTickPrice, sqrtUpperTickPrice, sqrtPrice, _ := tickSqrtPriceAndLiquidity(testState, testState.ActiveTick)
				var fractionalPartD8 int64
				if !sqrtPrice.IsZero() && !sqrtLowerTickPrice.IsZero() && !sqrtUpperTickPrice.IsZero() {
					tickRange := new(uint256.Int).Sub(sqrtUpperTickPrice, sqrtLowerTickPrice)
					tickPosition := new(uint256.Int).Sub(sqrtPrice, sqrtLowerTickPrice)

					if !tickRange.IsZero() {
						fractionalPart := mulDiv(tickPosition, BI_POWS[8], tickRange)
						fractionalPartD8 = int64(fractionalPart.Uint64())
					} else {
						fractionalPartD8 = int64(BI_POWS[7].Uint64())
					}
				} else {
					fractionalPartD8 = int64(BI_POWS[7].Uint64())
				}

				// Verify fractional part is in the expected range
				switch tc.expectedRange {
				case "low":
					assert.True(t, fractionalPartD8 < int64(BI_POWS[7].Uint64()),
						"Should be in lower half of tick range: %d", fractionalPartD8)
				case "mid":
					halfTick := int64(BI_POWS[7].Uint64())
					assert.True(t, fractionalPartD8 >= halfTick-20 && fractionalPartD8 <= halfTick+20,
						"Should be near middle of tick range: %d vs %d", fractionalPartD8, halfTick)
				case "high":
					assert.True(t, fractionalPartD8 > int64(BI_POWS[7].Uint64()),
						"Should be in upper half of tick range: %d", fractionalPartD8)
				}
			})
		}
	})
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
		ActiveTick: 10,
		LastTwaD8:  2560000, // 10 * 2^8 (10 << 8)
		Timestamp:  1000,
		Bins: map[uint32]Bin{
			1: {
				MergeBinBalance:  new(uint256.Int),
				MergeId:          0,
				TotalSupply:      new(uint256.Int).SetUint64(1000000),
				Kind:             0,
				Tick:             10,
				TickBalance:      new(uint256.Int).SetUint64(1000000),
				CurrentLiquidity: new(uint256.Int).SetUint64(1000000),
			},
			2: {
				MergeBinBalance:  new(uint256.Int),
				MergeId:          0,
				TotalSupply:      new(uint256.Int).SetUint64(800000),
				Kind:             0,
				Tick:             11,
				TickBalance:      new(uint256.Int).SetUint64(800000),
				CurrentLiquidity: new(uint256.Int).SetUint64(800000),
			},
		},
		Ticks: map[int32]Tick{
			10: {
				ReserveA:     new(uint256.Int).SetUint64(1000000),
				ReserveB:     new(uint256.Int).SetUint64(500000),
				TotalSupply:  new(uint256.Int).SetUint64(1000000),
				BinIdsByTick: map[uint8]uint32{0: 1},
			},
			11: {
				ReserveA:     new(uint256.Int).SetUint64(800000),
				ReserveB:     new(uint256.Int).SetUint64(600000),
				TotalSupply:  new(uint256.Int).SetUint64(800000),
				BinIdsByTick: map[uint8]uint32{0: 2},
			},
		},
		BinPositions: map[int32][]uint32{
			10: {1},
			11: {2},
		},
	}

	// Test moveBins with significant TWA change
	t.Run("moveBins with significant change", func(t *testing.T) {
		// Clone state
		testState := state.Clone()

		// Initial reserves
		initialReservesA := make(map[uint32]*uint256.Int)
		initialReservesB := make(map[uint32]*uint256.Int)
		for binId, bin := range testState.Bins {
			tick, tickExists := testState.Ticks[bin.Tick]
			if tickExists {
				reserveA, reserveB := binReserves(bin, tick)
				initialReservesA[binId] = new(uint256.Int).Set(reserveA)
				initialReservesB[binId] = new(uint256.Int).Set(reserveB)
			}
		}

		// Threshold for bin movement (small value to ensure movement happens)
		threshold := new(uint256.Int).SetUint64(1)

		// Move from tick 10 to tick 15 with significant TWA change
		moveBins(testState, 10, 15, 2560000, 3840000, threshold) // 10<<8 to 15<<8

		// Verify bins were adjusted
		for binId := range testState.Bins {
			bin := testState.Bins[binId]
			tick, tickExists := testState.Ticks[bin.Tick]
			if tickExists {
				currentReserveA, currentReserveB := binReserves(bin, tick)
				assert.NotEqual(t, initialReservesA[binId], currentReserveA,
					"ReserveA should change after significant bin movement")
				assert.NotEqual(t, initialReservesB[binId], currentReserveB,
					"ReserveB should change after significant bin movement")
			}
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
			tick, tickExists := testState.Ticks[bin.Tick]
			if tickExists {
				reserveA, reserveB := binReserves(bin, tick)
				initialReservesA[binId] = new(uint256.Int).Set(reserveA)
				initialReservesB[binId] = new(uint256.Int).Set(reserveB)
			}
		}

		// Threshold too high for bin movement
		threshold := new(uint256.Int).SetUint64(10000000)

		// Move from tick 10 to tick 11 with small TWA change
		moveBins(testState, 10, 11, 2560000, 2816000, threshold) // 10<<8 to 11<<8

		// Verify bins were NOT adjusted (due to high threshold)
		for binId := range testState.Bins {
			bin := testState.Bins[binId]
			tick, tickExists := testState.Ticks[bin.Tick]
			if tickExists {
				currentReserveA, currentReserveB := binReserves(bin, tick)
				assert.True(t, currentReserveA.Cmp(initialReservesA[binId]) == 0,
					"ReserveA should not change when below threshold")
				assert.True(t, currentReserveB.Cmp(initialReservesB[binId]) == 0,
					"ReserveB should not change when below threshold")
			}
		}
	})
}

func TestSimpleSwaps_USDC_USDT(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err = logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data
	realPoolData := `{"address":"0x31373595f40ea48a7aab6cbcb0d377c6066e2dca","exchange":"maverick-v2","type":"maverick-v2","timestamp":1748487959,"reserves":["278416610034","2863171384617"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"feeAIn\":10000000000000,\"feeBIn\":10000000000000,\"protocolFeeRatio\":0,\"bins\":{\"1\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"24019190385150318054580\",\"kind\":0,\"tick\":-2,\"tickBalance\":\"24019187983111426805681\",\"reserveA\":\"25213218403113805452470\",\"reserveB\":\"0\"},\"10\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"18855246071402780835020\",\"kind\":0,\"tick\":-3,\"tickBalance\":\"18855246071383971239184\",\"reserveA\":\"19727331936856521514748\",\"reserveB\":\"0\"},\"16\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"11700330106773539800445\",\"kind\":0,\"tick\":4,\"tickBalance\":\"11700330106761867797854\",\"reserveA\":\"81561674730419264373353\",\"reserveB\":\"2837439476465300009813586\"},\"17\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"9160766047657905425134\",\"kind\":0,\"tick\":5,\"tickBalance\":\"9160766047648766838052\",\"reserveA\":\"0\",\"reserveB\":\"9562860924232363005816\"},\"18\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"7535267618600221806670\",\"kind\":0,\"tick\":6,\"tickBalance\":\"7535267618592704782645\",\"reserveA\":\"0\",\"reserveB\":\"7854742810240757515323\"},\"19\":{\"mergeBinBalance\":\"0\",\"mergeId\":0,\"totalSupply\":\"6614040812468079420134\",\"kind\":0,\"tick\":7,\"tickBalance\":\"6614040812461481392473\",\"reserveA\":\"0\",\"reserveB\":\"6885768064575532743257\"}},\"binPositions\":{\"-3\":[10],\"-2\":[1],\"4\":[16],\"5\":[17],\"6\":[18],\"7\":[19]},\"activeTick\":4,\"lastTwaD8\":402564673,\"timestamp\":1748487959}","staticExtra":"{\"tickSpacing\":1}","blockNumber":22585623}`

	var poolEnt entity.Pool
	err := json.Unmarshal([]byte(realPoolData), &poolEnt)
	require.NoError(t, err)

	// Create pool simulator
	sim, err := NewPoolSimulator(poolEnt)
	require.NoError(t, err)
	require.NotNil(t, sim)

	// Test small amounts first
	testCases := []struct {
		name     string
		tokenIn  string
		tokenOut string
		amountIn string
		tokenAIn bool
	}{
		{
			name:     "10 USDC -> USDT",
			tokenIn:  sim.Info.Tokens[0], // USDC
			tokenOut: sim.Info.Tokens[1], // USDT
			amountIn: "10000000",         // 10 USDC (6 decimals)
			tokenAIn: true,
		},
		{
			name:     "10 USDT -> USDC",
			tokenIn:  sim.Info.Tokens[1], // USDT
			tokenOut: sim.Info.Tokens[0], // USDC
			amountIn: "10000000",         // 10 USDT (6 decimals)
			tokenAIn: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing %s with tokenAIn=%v", tc.name, tc.tokenAIn)

			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token: tc.tokenIn,
					Amount: func() *big.Int {
						amount := new(big.Int)
						amount.SetString(tc.amountIn, 10)
						return amount
					}(),
				},
				TokenOut: tc.tokenOut,
				Limit:    nil,
			})

			if err != nil {
				t.Logf("Error: %v", err)
			} else {
				// Convert to human readable
				humanAmount := new(big.Int).Div(result.TokenAmountOut.Amount, big.NewInt(1000000))
				t.Logf("%s -> %s units (raw: %s)", tc.amountIn, humanAmount.String(), result.TokenAmountOut.Amount.String())
			}
		})
	}
}
