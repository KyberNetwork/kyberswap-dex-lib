package ambient

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// pool fetched with: pools u ethereum ambient (block 25339274)
const pepePepePoolJSON = `{
	"address": "0x8e008cd78b86e502a7f5daf71ca8efff2f0a6020101553ebdbd1fb8b01f129ca",
	"exchange": "ambient",
	"type":     "ambient",
	"reserves": ["62802594184157341831","7702999369001304682216852"],
	"tokens": [
		{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2","symbol":"WETH","decimals":18,"swappable":true},
		{"address":"0x6982508145454ce325ddbe47a25d4ec3d2311933","symbol":"PEPE","decimals":18,"swappable":true}
	],
	"extra": "{\"state\":{\"Base\":\"0x0000000000000000000000000000000000000000\",\"Quote\":\"0x6982508145454ce325ddbe47a25d4ec3d2311933\",\"PoolIdx\":420,\"PoolHash\":\"0x8e008cd78b86e502a7f5daf71ca8efff2f0a6020101553ebdbd1fb8b01f129ca\",\"Curve\":{\"PriceRoot\":767249695060131,\"AmbientSeeds\":287900299556547505950,\"ConcLiq\":0,\"SeedDeflator\":4951228715665,\"ConcGrowth\":3840188497074},\"PoolSpec\":{\"Schema\":1,\"FeeRate\":2700,\"ProtocolTake\":0,\"TickSize\":16,\"JitThresh\":3,\"KnockoutBits\":36,\"OracleFlags\":0},\"PoolParams\":{\"FeeRate\":2700,\"ProtocolTake\":0,\"TickSize\":16},\"ActiveTicks\":[-214608,-214416,-211120,-211024,-209104,-209008,-206592,-204400,-196944,-194736],\"Levels\":[{\"Tick\":-214608,\"Level\":{\"BidLots\":22030383990457222,\"AskLots\":0,\"FeeOdometer\":517895040700}},{\"Tick\":-214416,\"Level\":{\"BidLots\":356466428453858906,\"AskLots\":0,\"FeeOdometer\":539798191040}},{\"Tick\":-211120,\"Level\":{\"BidLots\":72426694961268976,\"AskLots\":0,\"FeeOdometer\":1107318714187}},{\"Tick\":-211024,\"Level\":{\"BidLots\":275074792625484608,\"AskLots\":0,\"FeeOdometer\":1114583527272}},{\"Tick\":-209104,\"Level\":{\"BidLots\":0,\"AskLots\":72426694961268976,\"FeeOdometer\":1307501876939}},{\"Tick\":-209008,\"Level\":{\"BidLots\":0,\"AskLots\":275074792625484608,\"FeeOdometer\":1313346280682}},{\"Tick\":-206592,\"Level\":{\"BidLots\":0,\"AskLots\":22030383990457222,\"FeeOdometer\":1426626934481}},{\"Tick\":-204400,\"Level\":{\"BidLots\":0,\"AskLots\":356466428453858906,\"FeeOdometer\":1340180871052}},{\"Tick\":-196944,\"Level\":{\"BidLots\":325710782961597566,\"AskLots\":0,\"FeeOdometer\":3826755276771}},{\"Tick\":-194736,\"Level\":{\"BidLots\":0,\"AskLots\":325710782961597566,\"FeeOdometer\":3263114987145}}],\"minTick\":-221762,\"maxTick\":-181762}}",
	"staticExtra": "{\"nT\":\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\",\"pI\":420,\"sD\":\"0xaaaaaaaaa24eeeb8d57d431224f73832bc34f688\",\"b\":\"0x0000000000000000000000000000000000000000\",\"q\":\"0x6982508145454ce325ddbe47a25d4ec3d2311933\"}"
}`

func TestCalcAmountOut_WETH_PEPE(t *testing.T) {
	t.Parallel()

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(pepePepePoolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	// tokens[0]=WETH tokens[1]=PEPE
	// idx 0→1: buy PEPE with WETH (inBaseQty=true, isBuy=true)
	// idx 1→0: sell PEPE for WETH (inBaseQty=false, isBuy=false)
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{
		0: {1: {
			// dust
			"1000000000000": "576442870012802354608",
			// small — linear region
			"100000000000000":  "57180844910767991763488",
			"1000000000000000": "532874702733362709783030",
			// mid — starts crossing concentrated-liq ticks
			"10000000000000000":  "3430948441567559587480289",
			"100000000000000000": "6920286647218451536522758",
			// large — exhausts most concentrated liq, runs on ambient
			"1000000000000000000":  "7617075642681900846090866",
			"10000000000000000000": "7693764345863403104012500",
		}},
		1: {0: {
			// dust
			"1000000000000": "1707",
			// small — linear region
			"100000000000000":  "172511",
			"1000000000000000": "1725265",
			// mid
			"10000000000000000":  "17252793",
			"100000000000000000": "172528074",
			// large — crossing ticks
			"1000000000000000000":    "1725280679",
			"1000000000000000000000": "1725036690819",
			// very large — deep into ambient
			"1000000000000000000000000": "1511268015700814",
		}},
	})
}

func TestCalcAmountOutWithUpdateBalance_WETH_PEPE(t *testing.T) {
	t.Parallel()

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(pepePepePoolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	testutil.TestCalcAmountOutWithUpdateBalance(t, sim, map[int]map[int][][][2]string{
		0: {1: {
			// three successive 1 ETH buys move price up; 10 wei then rounds to zero output
			{
				{"1000000000000000000", "7617075642681900846090866"},
				{"1000000000000000000", "42375677572960160303268"},
				{"1000000000000000000", "14232917791043158912132"},
				{"10", "zero amount"},
			},
		}},
		1: {0: {
			// three successive 1B PEPE sells: price moves down each time
			{
				{"1000000000000000000000000000", "18570677112940530"},
				{"1000000000000000000000000000", "43199398599336"},
				{"1000000000000000000000000000", "14401802524962"},
			},
		}},
	})
}

func TestCloneState_WETH_PEPE(t *testing.T) {
	t.Parallel()

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(pepePepePoolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	tokens := sim.GetTokens()
	testutil.TestCloneState(t, sim, calcParams(tokens[0], tokens[1], bignum.NewBig10("10000000000000000")), nil)
}

func TestCalcAmountOut_WETH_PEPE_ConcurrentSafe(t *testing.T) {
	t.Parallel()

	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(pepePepePoolJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	tokens := sim.GetTokens()
	_, err = testutil.MustConcurrentSafe(t, func() (any, error) {
		return sim.CalcAmountOut(calcParams(tokens[0], tokens[1], bignum.NewBig10("10000000000000000")))
	})
	require.NoError(t, err)
}
