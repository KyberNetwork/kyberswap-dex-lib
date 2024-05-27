package fxdx

import (
	"encoding/json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	rawPools := []string{
		`{
			"address": "0x1ce0ebd2b95221b924765456fde017b076e79dbe",
			"type": "fxdx",
			"timestamp": 1705353097,
			"reserves": [
				"25043681537564780603",
				"6313740770058370935",
				"72284603421",
				"14683596252646794547903",
				"26974696715"
			],
			"tokens": [
				{
					"address": "0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f",
					"swappable": true
				},
				{
					"address": "0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22",
					"swappable": true
				},
				{
					"address": "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
					"swappable": true
				},
				{
					"address": "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
					"swappable": true
				},
				{
					"address": "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
					"swappable": true
				}
			],
			"extra": "{\"vault\":{\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"totalTokenWeights\":100000,\"whitelistedTokens\":[\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\",\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\",\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\",\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\",\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\"],\"poolAmounts\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":6313740770058370935,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":14683596252646794547903,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":26974696715,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":25043681537564780603,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":72284603421},\"bufferAmounts\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"reservedAmounts\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":24665993983186750,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":233199189,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":19766895376688956827,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":6227909107},\"tokenDecimals\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":18,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":18,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":6,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":18,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":6},\"stableTokens\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"usdfAmounts\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":12555087948177239310937,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":13958048328408935288990,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":27013671334811285837354,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":27526492903901124005110,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":72576961222501961304745},\"maxUsdfAmounts\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":24000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":96000000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":120000000000000000000000000,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":120000000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":120000000000000000000000000},\"tokenWeights\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":5000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":20000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25000,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":25000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":25000},\"priceFeed\":{\"address\":\"0xDA6E43c3b5Fb0D3Ba67F23Ab17C7F76A277e1A9e\",\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":8,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":8,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":8,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":8,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":8},\"spreadBasisPoints\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"adjustmentBasisPoints\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"strictStableTokens\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"isAdjustmentAdditive\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":false,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":false,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1705311603,\"maxDeviationBasisPoints\":750,\"minAuthorizations\":3,\"priceDuration\":120,\"maxPriceUpdateDelay\":46800,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":2663940000000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":1000000000000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":2525968000000000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":1000000000000000000000000000000},\"priceData\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"refPrice\":265623521228,\"refTime\":1705311605,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":6761},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"refPrice\":100005500,\"refTime\":1691897495,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":{\"refPrice\":252289000000,\"refTime\":1705311605,\"cumulativeRefDelta\":6782,\"cumulativeFastDelta\":17767},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"refPrice\":100006760,\"refTime\":1691897495,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":10000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":10000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0}},\"priceFeeds\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"roundId\":18446744073709564485,\"answer\":267017877220,\"answers\":{\"18446744073709564485\":267017877220}},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"roundId\":18446744073709551789,\"answer\":100004860,\"answers\":{\"18446744073709551789\":100004860}},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"roundId\":18446744073709551788,\"answer\":100022977,\"answers\":{\"18446744073709551788\":100022977}},\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":{\"roundId\":18446744073709570616,\"answer\":252530487042,\"answers\":{\"18446744073709570616\":252530487042}},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"roundId\":18446744073709551788,\"answer\":100022977,\"answers\":{\"18446744073709551788\":100022977}}}},\"usdf\":{\"address\":\"0xfe4DFb5789f6FD2c2bc3C3B8D1a13025B55756B1\",\"totalSupply\":153630261737800545747136},\"useSwapPricing\":false},\"feeUtils\":{\"address\":\"0xd2CEDbf8089d521F9573625C4FA27FdC48870907\",\"isInitialized\":true,\"isActive\":false,\"feeMultiplierIfInactive\":10,\"hasDynamicFees\":true,\"taxBasisPoints\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":25,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":25,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":25,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":25},\"swapFeeBasisPoints\":{\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":25,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":25,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25,\"0xd6c5469a7cc587e1e89a841fb7c102ff1370c05f\":25,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":25}}}"
		}`,
	}
	poolEntites := make([]entity.Pool, len(rawPools))
	for i, rawPool := range rawPools {
		err := json.Unmarshal([]byte(rawPool), &poolEntites[i])
		if err != nil {
			panic(err)
		}
	}
	var err error
	pools := make([]*PoolSimulator, len(rawPools))
	for i, poolEntity := range poolEntites {
		pools[i], err = NewPoolSimulator(poolEntity)
		if err != nil {
			panic(err)
		}
	}
	return pools
}
