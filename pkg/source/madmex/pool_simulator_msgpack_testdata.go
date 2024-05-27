package madmex

import (
	"fmt"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}, {Address: "D"}},
			Extra:    fmt.Sprintf("{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"A\",\"B\",\"C\",\"D\"],\"poolAmounts\":{\"C\":176522685577037266873231,\"A\":1640777763,\"D\":417621596032,\"B\":47192917723885198852},\"bufferAmounts\":{\"C\":1,\"A\":1,\"D\":1,\"B\":1},\"reservedAmounts\":{\"C\":14388220683939025001572,\"A\":227978222,\"D\":4210850176,\"B\":2337719678950856595},\"tokenDecimals\":{\"C\":18,\"A\":8,\"D\":6,\"B\":18},\"stableTokens\":{\"C\":false,\"A\":false,\"D\":true,\"B\":false},\"usdgAmounts\":{\"C\":226991552742006728124154,\"A\":370249303703403946435521,\"D\":407271566307761703548011,\"B\":108601943211855065272548},\"maxUsdgAmounts\":{\"C\":30000000000000000000000000,\"A\":30000000000000000000000000,\"D\":50000000000000000000000000,\"B\":30000000000000000000000000},\"tokenWeights\":{\"C\":20000,\"A\":20000,\"D\":40000,\"B\":20000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":50000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"C\":8,\"A\":8,\"D\":8,\"B\":8},\"spreadBasisPoints\":{\"C\":0,\"A\":0,\"D\":0,\"B\":0},\"adjustmentBasisPoints\":{\"C\":0,\"A\":0,\"D\":0,\"B\":0},\"strictStableTokens\":{\"C\":false,\"A\":false,\"D\":true,\"B\":false},\"isAdjustmentAdditive\":{\"C\":false,\"A\":false,\"D\":false,\"B\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":%v,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"volBasisPoints\":0,\"prices\":{\"C\":619500000000000000000000000000,\"A\":30168000000000000000000000000000000,\"D\":0,\"B\":1838730000000000000000000000000000}},\"secondaryPriceFeedVersion\":1,\"priceFeeds\":{\"C\":{\"roundId\":36893488147424514663,\"answer\":61931328,\"answers\":{\"36893488147424514663\":61931328}},\"A\":{\"roundId\":36893488147424540380,\"answer\":3016364000000,\"answers\":{\"36893488147424540380\":3016364000000}},\"D\":{\"roundId\":36893488147424479896,\"answer\":100007315,\"answers\":{\"36893488147424479896\":100007315}},\"B\":{\"roundId\":36893488147424540351,\"answer\":183824000000,\"answers\":{\"36893488147424540351\":183824000000}}}},\"usdg\":{\"address\":\"0x06eaaEa0b37bADF17E33B0DD99e97C000808B304\",\"totalSupply\":3119702491113301501233193}}}", time.Now().Unix()),
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
