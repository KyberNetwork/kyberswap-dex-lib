package metavault

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	// test data from https://polygonscan.com/address/0x32848e2d3aecfa7364595609fb050a301050a6b4#readContract
	// need to set bufferAmounts to all 1 to allow swap
	// need to set lastUpdatedAt to Now so it will get same price as simulation
	// simulation blocknumber = 44626808
	testcases := []struct {
		in                string
		inAmount          int64
		out               string
		expectedOutAmount int64
	}{
		{"A0", 1000000000000000, "A2", 350403344400},
		{"A0", 1000000000000000, "A9", 687960496669452},
		{"A2", 1000000000000000, "A10", 2627365723178042393},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Tokens: []*entity.PoolToken{
			{Address: "A0"}, {Address: "A1"}, {Address: "A2"}, {Address: "A3"}, {Address: "A4"},
			{Address: "A5"}, {Address: "A6"}, {Address: "A7"}, {Address: "A8"}, {Address: "A9"}, {Address: "A10"},
		},
		Extra: fmt.Sprintf("{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":false,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":25,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"A0\",\"A1\",\"A2\",\"A3\",\"A4\",\"A5\",\"A6\",\"A7\",\"A8\",\"A9\",\"A10\"],\"poolAmounts\":{\"A0\": 351500182590784658632430,\"A1\": 2875486701,\"A2\": 582500526946365638607,\"A3\": 3504229637461742465916,\"A4\": 75279988308635845,\"A5\": 266311733343887271182,\"A6\": 519980012039,\"A7\": 328181486966,\"A8\": 226370519501761590614462,\"A9\": 33830006206808659773115,\"A10\": 54956975689863757124184},\"bufferAmounts\":{\"A0\": 1,\"A1\": 1,\"A2\": 1,\"A3\": 1,\"A4\": 1,\"A5\": 1,\"A6\": 1,\"A7\": 1,\"A8\": 1,\"A9\": 1,\"A10\": 1},\"reservedAmounts\":{ \"A0\": 10076951923665922051838, \"A1\": 208416437, \"A2\": 177431951598853399981, \"A3\": 1552896361580005197835, \"A4\": 0, \"A5\": 58800938232557607904, \"A6\": 84639554819, \"A7\": 165038595, \"A8\": 0, \"A9\": 0, \"A10\": 0},\"tokenDecimals\":{\"A0\":18,\"A1\":8,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":6,\"A10\":18,\"A2\":18,\"A8\":18,\"A9\":18,\"A3\":18,\"A4\":18,\"A7\":6,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":18},\"stableTokens\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":true,\"A10\":false,\"A2\":false,\"A8\":true,\"A9\":true,\"A3\":false,\"A4\":false,\"A7\":true,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"usdmAmounts\":{\"A0\": 253813004375598635984875,\"A1\": 878978374373698064907047,\"A2\": 1093840701705446620699791,\"A3\": 19627716924545680764881,\"A4\": 279248927102162245,\"A5\": 15330582106181439208372,\"A6\": 519880496669442097196432,\"A7\": 328341628943729286849908,\"A8\": 226727050104206471903959,\"A9\": 33832853226499968909637,\"A10\": 39966351629451563348533},\"maxUsdmAmounts\":{\"A0\":400000000000000000000000,\"A1\":1100000000000000000000000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":900000000000000000000000,\"A10\":40000000000000000000000,\"A2\":1400000000000000000000000,\"A8\":400000000000000000000000,\"A9\":50000000000000000000000,\"A3\":25000000000000000000000,\"A4\":1000000000000000000,\"A7\":650000000000000000000000,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":25000000000000000000000},\"tokenWeights\":{\"A0\":8000,\"A1\":22000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":18000,\"A10\":1000,\"A2\":28000,\"A8\":8000,\"A9\":1000,\"A3\":500,\"A4\":0,\"A7\":13000,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":500},\"priceFeed\":{\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":15000000000000000000000000000,\"priceSampleSpace\":1,\"priceDecimals\":{\"A0\":8,\"A1\":8,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":8,\"A10\":8,\"A2\":8,\"A8\":8,\"A9\":8,\"A3\":8,\"A4\":8,\"A7\":8,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":8},\"spreadBasisPoints\":{\"A0\":8,\"A1\":0,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":17,\"A2\":0,\"A8\":0,\"A9\":0,\"A3\":8,\"A4\":8,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":8},\"adjustmentBasisPoints\":{\"A0\":0,\"A1\":0,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":0,\"A2\":0,\"A8\":0,\"A9\":0,\"A3\":0,\"A4\":0,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":0},\"strictStableTokens\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":false,\"A10\":false,\"A2\":false,\"A8\":false,\"A9\":false,\"A3\":false,\"A4\":false,\"A7\":false,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"isAdjustmentAdditive\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":false,\"A10\":false,\"A2\":false,\"A8\":false,\"A9\":false,\"A3\":false,\"A4\":false,\"A7\":false,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":%v,\"maxDeviationBasisPoints\":100,\"minAuthorizations\":1,\"priceDuration\":300,\"maxPriceUpdateDelay\":3600,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"A0\": 690650000000000000000000000000,\"A1\": 30659449302960000000000000000000000,\"A2\": 1964120000000000000000000000000000,\"A3\": 6624179450000000000000000000000,\"A4\": 5656892920000000000000000000000,\"A5\": 70056383700000000000000000000000,\"A6\": 0,\"A7\": 0,\"A8\": 0,\"A9\": 0,\"A10\": 0},\"priceData\":{\"A0\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A1\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A10\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A2\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A8\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A9\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A3\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A4\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A7\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"A0\":50000,\"A1\":50000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":0,\"A2\":50000,\"A8\":0,\"A9\":0,\"A3\":50000,\"A4\":50000,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":50000}},\"secondaryPriceFeedVersion\":2,\"priceFeeds\":{\"A0\":{\"roundId\":36893488147424548362,\"answer\":69050000,\"answers\":{\"36893488147424548362\":69050000}}, \"A1\":{\"roundId\":36893488147424548362,\"answer\":3062888000000,\"answers\":{\"36893488147424548362\":3062888000000}}, \"A2\":{\"roundId\":36893488147424548362,\"answer\":196345780000,\"answers\":{\"36893488147424548362\":196345780000}}, \"A3\":{\"roundId\":36893488147424548362,\"answer\":662318856,\"answers\":{\"36893488147424548362\":662318856}}, \"A4\":{\"roundId\":36893488147424548362,\"answer\":565717943,\"answers\":{\"36893488147424548362\":565717943}}, \"A5\":{\"roundId\":36893488147424548362,\"answer\":7002000000,\"answers\":{\"36893488147424548362\":7002000000}}, \"A6\":{\"roundId\":36893488147424548362,\"answer\":100000000,\"answers\":{\"36893488147424548362\":100000000}}, \"A7\":{\"roundId\":36893488147424548362,\"answer\":99984814,\"answers\":{\"36893488147424548362\":99984814}}, \"A8\":{\"roundId\":36893488147424548362,\"answer\":99975733,\"answers\":{\"36893488147424548362\":99975733}}, \"A9\":{\"roundId\":36893488147424548362,\"answer\":100009694,\"answers\":{\"36893488147424548362\":100009694}}, \"A10\":{\"roundId\":36893488147424548362,\"answer\":74353248,\"answers\":{\"36893488147424548362\":74353248}}}},\"usdm\":{\"address\":\"0x533403a3346cA31D67c380917ffaF185c24e7333\",\"totalSupply\":3389201190535341442726377}}}", time.Now().Unix()),
	})
	require.Nil(t, err)

	assert.Equal(t, []string{"A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9", "A10"}, p.CanSwapTo("A0"))
	assert.Equal(t, []string{"A0", "A1", "A2", "A3", "A4", "A5", "A6", "A7", "A8", "A9"}, p.CanSwapTo("A10"))
	assert.Equal(t, 0, len(p.CanSwapTo("B")))

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			out, err := p.CalcAmountOut(pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}, tc.out)
			require.Nil(t, err)
			assert.Equal(t, big.NewInt(tc.expectedOutAmount), out.TokenAmountOut.Amount)
			assert.Equal(t, tc.out, out.TokenAmountOut.Token)
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	// test data from https://polygonscan.com/address/0x32848e2d3aecfa7364595609fb050a301050a6b4#readContract
	// need to set bufferAmounts to all 1 to allow swap
	// need to set lastUpdatedAt to Now so it will get same price as simulation
	testcases := []struct {
		in             string
		inAmount       int64
		out            string
		expPoolAmounts []string
		expUsdm        []string
	}{
		{"A0", 1000000000000000, "A2",
			[]string{"351500183590784658632430", "2875486701", "582500526595013643821", "3504229637461742465916", "75279988308635845", "266311733343887271182", "519980012039", "328181486966", "226370519501761590614462", "33830006206808659773115", "54956975689863757124184"},
			[]string{"253813005065696115984875", "878978374373698064907047", "1093840701015349140699791", "19627716924545680764881", "279248927102162245", "15330582106181439208372", "519880496669442097196432", "328341628943729286849908", "226727050104206471903959", "33832853226499968909637", "39966351629451563348533"},
		},
		{"A0", 1000000000000000, "A9",
			[]string{"351500184590784658632430", "2875486701", "582500526595013643821", "3504229637461742465916", "75279988308635845", "266311733343887271182", "519980012039", "328181486966", "226370519501761590614462", "33830005516778071338358", "54956975689863757124184"},
			[]string{"253813005755793595984875", "878978374373698064907047", "1093840701015349140699791", "19627716924545680764881", "279248927102162245", "15330582106181439208372", "519880496669442097196432", "328341628943729286849908", "226727050104206471903959", "33832852536402488909637", "39966351629451563348533"},
		},
		{"A2", 1000000000000000, "A10",
			[]string{"351500184590784658632430", "2875486701", "582501526595013643821", "3504229637461742465916", "75279988308635845", "266311733343887271182", "519980012039", "328181486966", "226370519501761590614462", "33830005516778071338358", "54954338566785188377428"},
			[]string{"253813005755793595984875", "878978374373698064907047", "1093842665135349140699791", "19627716924545680764881", "279248927102162245", "15330582106181439208372", "519880496669442097196432", "328341628943729286849908", "226727050104206471903959", "33832852536402488909637", "39964387509451563348533"},
		},
	}
	p, err := NewPoolSimulator(entity.Pool{
		Exchange: "",
		Type:     "",
		Tokens: []*entity.PoolToken{
			{Address: "A0"}, {Address: "A1"}, {Address: "A2"}, {Address: "A3"}, {Address: "A4"},
			{Address: "A5"}, {Address: "A6"}, {Address: "A7"}, {Address: "A8"}, {Address: "A9"}, {Address: "A10"},
		},
		Extra: fmt.Sprintf("{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":false,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":25,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"A0\",\"A1\",\"A2\",\"A3\",\"A4\",\"A5\",\"A6\",\"A7\",\"A8\",\"A9\",\"A10\"],\"poolAmounts\":{\"A0\": 351500182590784658632430,\"A1\": 2875486701,\"A2\": 582500526946365638607,\"A3\": 3504229637461742465916,\"A4\": 75279988308635845,\"A5\": 266311733343887271182,\"A6\": 519980012039,\"A7\": 328181486966,\"A8\": 226370519501761590614462,\"A9\": 33830006206808659773115,\"A10\": 54956975689863757124184},\"bufferAmounts\":{\"A0\": 1,\"A1\": 1,\"A2\": 1,\"A3\": 1,\"A4\": 1,\"A5\": 1,\"A6\": 1,\"A7\": 1,\"A8\": 1,\"A9\": 1,\"A10\": 1},\"reservedAmounts\":{ \"A0\": 10076951923665922051838, \"A1\": 208416437, \"A2\": 177431951598853399981, \"A3\": 1552896361580005197835, \"A4\": 0, \"A5\": 58800938232557607904, \"A6\": 84639554819, \"A7\": 165038595, \"A8\": 0, \"A9\": 0, \"A10\": 0},\"tokenDecimals\":{\"A0\":18,\"A1\":8,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":6,\"A10\":18,\"A2\":18,\"A8\":18,\"A9\":18,\"A3\":18,\"A4\":18,\"A7\":6,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":18},\"stableTokens\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":true,\"A10\":false,\"A2\":false,\"A8\":true,\"A9\":true,\"A3\":false,\"A4\":false,\"A7\":true,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"usdmAmounts\":{\"A0\": 253813004375598635984875,\"A1\": 878978374373698064907047,\"A2\": 1093840701705446620699791,\"A3\": 19627716924545680764881,\"A4\": 279248927102162245,\"A5\": 15330582106181439208372,\"A6\": 519880496669442097196432,\"A7\": 328341628943729286849908,\"A8\": 226727050104206471903959,\"A9\": 33832853226499968909637,\"A10\": 39966351629451563348533},\"maxUsdmAmounts\":{\"A0\":400000000000000000000000,\"A1\":1100000000000000000000000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":900000000000000000000000,\"A10\":40000000000000000000000,\"A2\":1400000000000000000000000,\"A8\":400000000000000000000000,\"A9\":50000000000000000000000,\"A3\":25000000000000000000000,\"A4\":1000000000000000000,\"A7\":650000000000000000000000,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":25000000000000000000000},\"tokenWeights\":{\"A0\":8000,\"A1\":22000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":18000,\"A10\":1000,\"A2\":28000,\"A8\":8000,\"A9\":1000,\"A3\":500,\"A4\":0,\"A7\":13000,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":500},\"priceFeed\":{\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":15000000000000000000000000000,\"priceSampleSpace\":1,\"priceDecimals\":{\"A0\":8,\"A1\":8,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":8,\"A10\":8,\"A2\":8,\"A8\":8,\"A9\":8,\"A3\":8,\"A4\":8,\"A7\":8,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":8},\"spreadBasisPoints\":{\"A0\":8,\"A1\":0,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":17,\"A2\":0,\"A8\":0,\"A9\":0,\"A3\":8,\"A4\":8,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":8},\"adjustmentBasisPoints\":{\"A0\":0,\"A1\":0,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":0,\"A2\":0,\"A8\":0,\"A9\":0,\"A3\":0,\"A4\":0,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":0},\"strictStableTokens\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":false,\"A10\":false,\"A2\":false,\"A8\":false,\"A9\":false,\"A3\":false,\"A4\":false,\"A7\":false,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"isAdjustmentAdditive\":{\"A0\":false,\"A1\":false,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":false,\"A10\":false,\"A2\":false,\"A8\":false,\"A9\":false,\"A3\":false,\"A4\":false,\"A7\":false,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":%v,\"maxDeviationBasisPoints\":100,\"minAuthorizations\":1,\"priceDuration\":300,\"maxPriceUpdateDelay\":3600,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"A0\": 690650000000000000000000000000,\"A1\": 30659449302960000000000000000000000,\"A2\": 1964120000000000000000000000000000,\"A3\": 6624179450000000000000000000000,\"A4\": 5656892920000000000000000000000,\"A5\": 70056383700000000000000000000000,\"A6\": 0,\"A7\": 0,\"A8\": 0,\"A9\": 0,\"A10\": 0},\"priceData\":{\"A0\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A1\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A10\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A2\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A8\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A9\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A3\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A4\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"A7\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"A0\":50000,\"A1\":50000,\"0x2791bca1f2de4661ed88a30c99a7a9449aa84174\":0,\"A10\":0,\"A2\":50000,\"A8\":0,\"A9\":0,\"A3\":50000,\"A4\":50000,\"A7\":0,\"0xd6df932a45c0f255f85145f286ea0b292b21c90b\":50000}},\"secondaryPriceFeedVersion\":2,\"priceFeeds\":{\"A0\":{\"roundId\":36893488147424548362,\"answer\":69050000,\"answers\":{\"36893488147424548362\":69050000}}, \"A1\":{\"roundId\":36893488147424548362,\"answer\":3062888000000,\"answers\":{\"36893488147424548362\":3062888000000}}, \"A2\":{\"roundId\":36893488147424548362,\"answer\":196345780000,\"answers\":{\"36893488147424548362\":196345780000}}, \"A3\":{\"roundId\":36893488147424548362,\"answer\":662318856,\"answers\":{\"36893488147424548362\":662318856}}, \"A4\":{\"roundId\":36893488147424548362,\"answer\":565717943,\"answers\":{\"36893488147424548362\":565717943}}, \"A5\":{\"roundId\":36893488147424548362,\"answer\":7002000000,\"answers\":{\"36893488147424548362\":7002000000}}, \"A6\":{\"roundId\":36893488147424548362,\"answer\":100000000,\"answers\":{\"36893488147424548362\":100000000}}, \"A7\":{\"roundId\":36893488147424548362,\"answer\":99984814,\"answers\":{\"36893488147424548362\":99984814}}, \"A8\":{\"roundId\":36893488147424548362,\"answer\":99975733,\"answers\":{\"36893488147424548362\":99975733}}, \"A9\":{\"roundId\":36893488147424548362,\"answer\":100009694,\"answers\":{\"36893488147424548362\":100009694}}, \"A10\":{\"roundId\":36893488147424548362,\"answer\":74353248,\"answers\":{\"36893488147424548362\":74353248}}}},\"usdm\":{\"address\":\"0x533403a3346cA31D67c380917ffaF185c24e7333\",\"totalSupply\":3389201190535341442726377}}}", time.Now().Unix()),
	})
	require.Nil(t, err)

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			amountIn := pool.TokenAmount{Token: tc.in, Amount: big.NewInt(tc.inAmount)}
			out, err := p.CalcAmountOut(amountIn, tc.out)
			require.Nil(t, err)

			fmt.Println(out.TokenAmountOut)
			p.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  amountIn,
				TokenAmountOut: *out.TokenAmountOut,
				Fee:            *out.Fee,
				SwapInfo:       out.SwapInfo,
			})

			fmt.Println(p.Info.Reserves)
			for i, expPoolAmount := range tc.expPoolAmounts {
				tok := p.vault.WhitelistedTokens[i]
				assert.Equal(t, bignumber.NewBig10(expPoolAmount), p.vault.PoolAmounts[tok])
			}
			for i, expUsdm := range tc.expUsdm {
				tok := p.vault.WhitelistedTokens[i]
				assert.Equal(t, bignumber.NewBig10(expUsdm), p.vault.USDMAmounts[tok])
			}
		})
	}
}
