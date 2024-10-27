package pancakev3

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var poolEncoded = `{
  "address": "0x1445f32d1a74872ba41f3d8cf4022e9996120b31",
  "swapFee": 100,
  "exchange": "pancake-v3",
  "type": "pancake-v3",
  "timestamp": 1730057750,
  "reserves": [
    "23276143922",
    "23534462821746224455"
  ],
  "tokens": [
    {
      "address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
      "name": "USD Coin",
      "symbol": "USDC",
      "decimals": 6,
      "weight": 0,
      "swappable": true
    },
    {
      "address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
      "name": "Wrapped Ether",
      "symbol": "WETH",
      "decimals": 18,
      "weight": 0,
      "swappable": true
    }
  ],
  "extra": "{\"liquidity\":12542819400283132,\"sqrtPriceX96\":1588522411036495101863810620899499,\"tickSpacing\":1,\"tick\":198129,\"ticks\":[{\"index\":-887272,\"liquidityGross\":352898121260,\"liquidityNet\":352898121260},{\"index\":193379,\"liquidityGross\":12976195212048,\"liquidityNet\":12976195212048},{\"index\":193577,\"liquidityGross\":1827309675487154,\"liquidityNet\":1827309675487154},{\"index\":193631,\"liquidityGross\":12034521832820,\"liquidityNet\":12034521832820},{\"index\":193946,\"liquidityGross\":261999638637173,\"liquidityNet\":261999638637173},{\"index\":193966,\"liquidityGross\":261999638637173,\"liquidityNet\":-261999638637173},{\"index\":194597,\"liquidityGross\":68281715054597,\"liquidityNet\":68281715054597},{\"index\":194669,\"liquidityGross\":12822500049668,\"liquidityNet\":12822500049668},{\"index\":194673,\"liquidityGross\":965433033416053,\"liquidityNet\":965433033416053},{\"index\":194714,\"liquidityGross\":13520989065645,\"liquidityNet\":13520989065645},{\"index\":194767,\"liquidityGross\":1376519583754822,\"liquidityNet\":1376519583754822},{\"index\":194773,\"liquidityGross\":965433033416053,\"liquidityNet\":-965433033416053},{\"index\":194786,\"liquidityGross\":1376519583754822,\"liquidityNet\":-1376519583754822},{\"index\":195004,\"liquidityGross\":5802171746131,\"liquidityNet\":5802171746131},{\"index\":195017,\"liquidityGross\":55279136776937,\"liquidityNet\":55279136776937},{\"index\":195040,\"liquidityGross\":55279136776937,\"liquidityNet\":-55279136776937},{\"index\":195045,\"liquidityGross\":1237068111047882,\"liquidityNet\":1237068111047882},{\"index\":195064,\"liquidityGross\":1237068111047882,\"liquidityNet\":-1237068111047882},{\"index\":195232,\"liquidityGross\":219545554902570,\"liquidityNet\":219545554902570},{\"index\":195245,\"liquidityGross\":101285368474929,\"liquidityNet\":101285368474929},{\"index\":195255,\"liquidityGross\":101285368474929,\"liquidityNet\":-101285368474929},{\"index\":195262,\"liquidityGross\":219545554902570,\"liquidityNet\":-219545554902570},{\"index\":195370,\"liquidityGross\":3144178413795993,\"liquidityNet\":3144178413795993},{\"index\":195453,\"liquidityGross\":149923876684256,\"liquidityNet\":149923876684256},{\"index\":195458,\"liquidityGross\":149923876684256,\"liquidityNet\":-149923876684256},{\"index\":195493,\"liquidityGross\":68281715054597,\"liquidityNet\":-68281715054597},{\"index\":195499,\"liquidityGross\":94678267281393,\"liquidityNet\":94678267281393},{\"index\":195507,\"liquidityGross\":94678267281393,\"liquidityNet\":-94678267281393},{\"index\":195513,\"liquidityGross\":222470083089442,\"liquidityNet\":222470083089442},{\"index\":195536,\"liquidityGross\":35772370059051,\"liquidityNet\":35772370059051},{\"index\":195575,\"liquidityGross\":119431925191797,\"liquidityNet\":119431925191797},{\"index\":195582,\"liquidityGross\":119431925191797,\"liquidityNet\":-119431925191797},{\"index\":195641,\"liquidityGross\":312150070866787,\"liquidityNet\":312150070866787},{\"index\":195673,\"liquidityGross\":87723000368388,\"liquidityNet\":87723000368388},{\"index\":195674,\"liquidityGross\":273519714413407,\"liquidityNet\":273519714413407},{\"index\":195677,\"liquidityGross\":445414707911037,\"liquidityNet\":445414707911037},{\"index\":195679,\"liquidityGross\":718934422324444,\"liquidityNet\":-718934422324444},{\"index\":195690,\"liquidityGross\":458255612184377,\"liquidityNet\":458255612184377},{\"index\":195692,\"liquidityGross\":458255612184377,\"liquidityNet\":-458255612184377},{\"index\":195694,\"liquidityGross\":302371544104097,\"liquidityNet\":302371544104097},{\"index\":195696,\"liquidityGross\":87723000368388,\"liquidityNet\":-87723000368388},{\"index\":195697,\"liquidityGross\":302371544104097,\"liquidityNet\":-302371544104097},{\"index\":195698,\"liquidityGross\":303799148280159,\"liquidityNet\":303799148280159},{\"index\":195701,\"liquidityGross\":303799148280159,\"liquidityNet\":-303799148280159},{\"index\":195713,\"liquidityGross\":222470083089442,\"liquidityNet\":-222470083089442},{\"index\":195724,\"liquidityGross\":175469722190515,\"liquidityNet\":175469722190515},{\"index\":195726,\"liquidityGross\":1440970212845026,\"liquidityNet\":1440970212845026},{\"index\":195729,\"liquidityGross\":1038989744762225,\"liquidityNet\":-1038989744762225},{\"index\":195730,\"liquidityGross\":577450190273316,\"liquidityNet\":-577450190273316},{\"index\":195735,\"liquidityGross\":99641290507077,\"liquidityNet\":99641290507077},{\"index\":195736,\"liquidityGross\":35772370059051,\"liquidityNet\":-35772370059051},{\"index\":195738,\"liquidityGross\":6428814811640493,\"liquidityNet\":6428814811640493},{\"index\":195740,\"liquidityGross\":99641290507077,\"liquidityNet\":-99641290507077},{\"index\":195750,\"liquidityGross\":6428814811640493,\"liquidityNet\":-6428814811640493},{\"index\":195757,\"liquidityGross\":753062890732320,\"liquidityNet\":753062890732320},{\"index\":195759,\"liquidityGross\":537033634617358,\"liquidityNet\":-537033634617358},{\"index\":195762,\"liquidityGross\":236499412274941,\"liquidityNet\":236499412274941},{\"index\":195764,\"liquidityGross\":41203673245239,\"liquidityNet\":41203673245239},{\"index\":195767,\"liquidityGross\":216029256114962,\"liquidityNet\":-216029256114962},{\"index\":195773,\"liquidityGross\":341966189074172,\"liquidityNet\":341966189074172},{\"index\":195774,\"liquidityGross\":41203673245239,\"liquidityNet\":-41203673245239},{\"index\":195779,\"liquidityGross\":341966189074172,\"liquidityNet\":341966189074172},{\"index\":195782,\"liquidityGross\":1142709112688336,\"liquidityNet\":-225155643608352},{\"index\":195784,\"liquidityGross\":458776734539992,\"liquidityNet\":-458776734539992},{\"index\":195789,\"liquidityGross\":236499412274941,\"liquidityNet\":-236499412274941},{\"index\":195790,\"liquidityGross\":104424348734220,\"liquidityNet\":104424348734220},{\"index\":195795,\"liquidityGross\":319535642145353,\"liquidityNet\":319535642145353},{\"index\":195797,\"liquidityGross\":1545206891382690,\"liquidityNet\":1545206891382690},{\"index\":195798,\"liquidityGross\":319535642145353,\"liquidityNet\":-319535642145353},{\"index\":195799,\"liquidityGross\":1361580255083803,\"liquidityNet\":-1361580255083803},{\"index\":195800,\"liquidityGross\":330848383593483,\"liquidityNet\":330848383593483},{\"index\":195802,\"liquidityGross\":104424348734220,\"liquidityNet\":-104424348734220},{\"index\":195805,\"liquidityGross\":183626636298887,\"liquidityNet\":-183626636298887},{\"index\":195813,\"liquidityGross\":163464692513391,\"liquidityNet\":163464692513391},{\"index\":195818,\"liquidityGross\":329947774477080,\"liquidityNet\":329947774477080},{\"index\":195821,\"liquidityGross\":329947774477080,\"liquidityNet\":-329947774477080},{\"index\":195823,\"liquidityGross\":163464692513391,\"liquidityNet\":-163464692513391},{\"index\":195835,\"liquidityGross\":330848383593483,\"liquidityNet\":-330848383593483},{\"index\":195852,\"liquidityGross\":209354485525252,\"liquidityNet\":209354485525252},{\"index\":195855,\"liquidityGross\":209354485525252,\"liquidityNet\":-209354485525252},{\"index\":195864,\"liquidityGross\":58036145597773,\"liquidityNet\":58036145597773},{\"index\":195866,\"liquidityGross\":342341608760710,\"liquidityNet\":342341608760710},{\"index\":195869,\"liquidityGross\":342341608760710,\"liquidityNet\":-342341608760710},{\"index\":195872,\"liquidityGross\":18095898717731,\"liquidityNet\":18095898717731},{\"index\":195880,\"liquidityGross\":58036145597773,\"liquidityNet\":-58036145597773},{\"index\":195887,\"liquidityGross\":676760942398354,\"liquidityNet\":676760942398354},{\"index\":195888,\"liquidityGross\":177615638760782,\"liquidityNet\":177615638760782},{\"index\":195892,\"liquidityGross\":177615638760782,\"liquidityNet\":-177615638760782},{\"index\":195893,\"liquidityGross\":18095898717731,\"liquidityNet\":-18095898717731},{\"index\":195897,\"liquidityGross\":676760942398354,\"liquidityNet\":-676760942398354},{\"index\":195913,\"liquidityGross\":628083987494657,\"liquidityNet\":628083987494657},{\"index\":195916,\"liquidityGross\":628083987494657,\"liquidityNet\":-628083987494657},{\"index\":195949,\"liquidityGross\":2542228670238125,\"liquidityNet\":2542228670238125},{\"index\":195956,\"liquidityGross\":1166131769864330,\"liquidityNet\":1166131769864330},{\"index\":195961,\"liquidityGross\":2542228670238125,\"liquidityNet\":-2542228670238125},{\"index\":195965,\"liquidityGross\":298814267872188,\"liquidityNet\":298814267872188},{\"index\":195966,\"liquidityGross\":1166131769864330,\"liquidityNet\":-1166131769864330},{\"index\":196006,\"liquidityGross\":109136423802184,\"liquidityNet\":109136423802184},{\"index\":196010,\"liquidityGross\":222275930753471,\"liquidityNet\":222275930753471},{\"index\":196012,\"liquidityGross\":109136423802184,\"liquidityNet\":-109136423802184},{\"index\":196043,\"liquidityGross\":12822500049668,\"liquidityNet\":-12822500049668},{\"index\":196133,\"liquidityGross\":165151248660867,\"liquidityNet\":165151248660867},{\"index\":196144,\"liquidityGross\":3649554872943623,\"liquidityNet\":3649554872943623},{\"index\":196154,\"liquidityGross\":3649554872943623,\"liquidityNet\":-3649554872943623},{\"index\":196159,\"liquidityGross\":12306532940022,\"liquidityNet\":12306532940022},{\"index\":196176,\"liquidityGross\":1827309675487154,\"liquidityNet\":-1827309675487154},{\"index\":196185,\"liquidityGross\":678685704012,\"liquidityNet\":678685704012},{\"index\":196213,\"liquidityGross\":2728151306835273,\"liquidityNet\":2728151306835273},{\"index\":196233,\"liquidityGross\":2728151306835273,\"liquidityNet\":-2728151306835273},{\"index\":196246,\"liquidityGross\":298814267872188,\"liquidityNet\":-298814267872188},{\"index\":196256,\"liquidityGross\":4519844597927358,\"liquidityNet\":4519844597927358},{\"index\":196282,\"liquidityGross\":35441161371335,\"liquidityNet\":35441161371335},{\"index\":196289,\"liquidityGross\":173138038282742,\"liquidityNet\":173138038282742},{\"index\":196312,\"liquidityGross\":173138038282742,\"liquidityNet\":-173138038282742},{\"index\":196333,\"liquidityGross\":165151248660867,\"liquidityNet\":-165151248660867},{\"index\":196344,\"liquidityGross\":202456886722818,\"liquidityNet\":202456886722818},{\"index\":196349,\"liquidityGross\":451425142620686,\"liquidityNet\":451425142620686},{\"index\":196359,\"liquidityGross\":451425142620686,\"liquidityNet\":-451425142620686},{\"index\":196364,\"liquidityGross\":202456886722818,\"liquidityNet\":-202456886722818},{\"index\":196482,\"liquidityGross\":35441161371335,\"liquidityNet\":-35441161371335},{\"index\":196489,\"liquidityGross\":1,\"liquidityNet\":1},{\"index\":196566,\"liquidityGross\":12306532940022,\"liquidityNet\":-12306532940022},{\"index\":196595,\"liquidityGross\":18986102972227,\"liquidityNet\":-18986102972227},{\"index\":196632,\"liquidityGross\":1,\"liquidityNet\":1},{\"index\":197037,\"liquidityGross\":3144178413795993,\"liquidityNet\":-3144178413795993},{\"index\":197040,\"liquidityGross\":1167146412458356,\"liquidityNet\":1167146412458356},{\"index\":197093,\"liquidityGross\":25979717184444,\"liquidityNet\":25979717184444},{\"index\":197193,\"liquidityGross\":25979717184444,\"liquidityNet\":-25979717184444},{\"index\":197216,\"liquidityGross\":1,\"liquidityNet\":1},{\"index\":197218,\"liquidityGross\":1447101119962,\"liquidityNet\":1447101119962},{\"index\":197234,\"liquidityGross\":1,\"liquidityNet\":1},{\"index\":197308,\"liquidityGross\":439594707586211,\"liquidityNet\":439594707586211},{\"index\":197550,\"liquidityGross\":130528459186074,\"liquidityNet\":130528459186074},{\"index\":197570,\"liquidityGross\":130528459186074,\"liquidityNet\":-130528459186074},{\"index\":197626,\"liquidityGross\":439594707586211,\"liquidityNet\":-439594707586211},{\"index\":197687,\"liquidityGross\":1088130807041084,\"liquidityNet\":1088130807041084},{\"index\":197689,\"liquidityGross\":1,\"liquidityNet\":-1},{\"index\":197751,\"liquidityGross\":1170200415560949,\"liquidityNet\":1170200415560949},{\"index\":197760,\"liquidityGross\":37482051220930783,\"liquidityNet\":37482051220930783},{\"index\":197771,\"liquidityGross\":1170200415560949,\"liquidityNet\":-1170200415560949},{\"index\":197780,\"liquidityGross\":37482051220930783,\"liquidityNet\":-37482051220930783},{\"index\":197791,\"liquidityGross\":5205445121406423,\"liquidityNet\":5205445121406423},{\"index\":197832,\"liquidityGross\":1,\"liquidityNet\":-1},{\"index\":198159,\"liquidityGross\":996239029932805,\"liquidityNet\":996239029932805},{\"index\":198165,\"liquidityGross\":245205847465395,\"liquidityNet\":245205847465395},{\"index\":198169,\"liquidityGross\":996239029932805,\"liquidityNet\":-996239029932805},{\"index\":198327,\"liquidityGross\":4518599274173229,\"liquidityNet\":-4518599274173229},{\"index\":198343,\"liquidityGross\":1167146412458356,\"liquidityNet\":-1167146412458356},{\"index\":198365,\"liquidityGross\":245205847465395,\"liquidityNet\":-245205847465395},{\"index\":198416,\"liquidityGross\":1,\"liquidityNet\":-1},{\"index\":198434,\"liquidityGross\":1,\"liquidityNet\":-1},{\"index\":199226,\"liquidityGross\":5205445121406423,\"liquidityNet\":-5205445121406423},{\"index\":199321,\"liquidityGross\":312150070866787,\"liquidityNet\":-312150070866787},{\"index\":199358,\"liquidityGross\":1088130807041084,\"liquidityNet\":-1088130807041084},{\"index\":199482,\"liquidityGross\":1447101119962,\"liquidityNet\":-1447101119962},{\"index\":200311,\"liquidityGross\":14221518966177,\"liquidityNet\":-14221518966177},{\"index\":200336,\"liquidityGross\":222275930753471,\"liquidityNet\":-222275930753471},{\"index\":200646,\"liquidityGross\":678685704012,\"liquidityNet\":-678685704012},{\"index\":204619,\"liquidityGross\":337057839549,\"liquidityNet\":-337057839549},{\"index\":205419,\"liquidityGross\":12034521832820,\"liquidityNet\":-12034521832820},{\"index\":414481,\"liquidityGross\":5236537175,\"liquidityNet\":-5236537175},{\"index\":887272,\"liquidityGross\":347661584085,\"liquidityNet\":-347661584085}]}",
  "staticExtra": "{\"poolId\":\"0x1445f32d1a74872ba41f3d8cf4022e9996120b31\"}"
}`

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name     string
		tokenIn  string
		amountIn string
		tokenOut string
	}
	testcases := []testcase{
		{
			name:     "swap WETH for USDC",
			tokenIn:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			amountIn: "1000000000000000000", // 1
			tokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})

		t.Run(tc.name+"new sim", func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return poolSim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{
						Token:  tc.tokenIn,
						Amount: bignumber.NewBig10(tc.amountIn),
					},
					TokenOut: tc.tokenOut,
				})
			})
			require.NoError(t, err)
			_ = result
		})
	}
}

func TestComparePoolSimulatorV2(t *testing.T) {
	poolEntity := new(entity.Pool)
	err := json.Unmarshal([]byte(poolEncoded), poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulatorBigInt(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	poolSimV2, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	for i := 0; i < 500; i++ {
		amt := RandNumberString(24)

		t.Run(fmt.Sprintf("test %s WETH -> USDC %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s WETH -> USDC (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Limit:   nil,
			})

			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s USDC -> WETH %d", amt, i), func(t *testing.T) {
			in := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig10(amt),
				},
				TokenOut: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			}
			result, err := poolSim.CalcAmountOut(in)
			resultV2, errV2 := poolSimV2.CalcAmountOut(in)

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountOut, resultV2.TokenAmountOut)
				assert.Equal(t, result.Fee, resultV2.Fee)
				assert.Equal(t, result.RemainingTokenAmountIn.Amount.String(), resultV2.RemainingTokenAmountIn.Amount.String())

				poolSim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *result.TokenAmountOut,
					Fee:            *result.Fee,
					SwapInfo:       result.SwapInfo,
				})
				poolSimV2.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  in.TokenAmountIn,
					TokenAmountOut: *resultV2.TokenAmountOut,
					Fee:            *resultV2.Fee,
					SwapInfo:       resultV2.SwapInfo,
				})
			} else {
				fmt.Println(err)
			}
		})

		t.Run(fmt.Sprintf("test %s USDC -> WETH (reversed) %d", amt, i), func(t *testing.T) {
			result, err := poolSim.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Limit:   nil,
			})
			resultV2, errV2 := poolSimV2.CalcAmountIn(pool.CalcAmountInParams{
				TokenAmountOut: pool.TokenAmount{
					Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
					Amount: bignumber.NewBig10(amt),
				},
				TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Limit:   nil,
			})

			require.Equal(t, err, errV2)
			if err == nil {
				assert.Equal(t, result.TokenAmountIn.Amount, resultV2.TokenAmountIn.Amount)
				assert.Equal(t, result.Fee.Amount, resultV2.Fee.Amount)
			} else {
				fmt.Println(err)
			}
		})
	}
}

// not really random but should be enough for testing
func RandNumberString(maxLen int) string {
	sLen := rand.Intn(maxLen-1) + 1
	var s string
	for i := 0; i < sLen; i++ {
		var c int
		if i == 0 {
			c = rand.Intn(9) + 1
		} else {
			c = rand.Intn(10)
		}
		s = fmt.Sprintf("%s%d", s, c)
	}
	return s
}
