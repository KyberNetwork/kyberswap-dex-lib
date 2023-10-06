package gmxglp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPoolSimulator_MintAndStakeGlp(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut string
		expectedErr       error
	}{
		{
			name: "it should return correct stake amount",
			entityPool: entity.Pool{
				Address:  "0xec8d8d4b215727f3476ff0ab41c406fa99b4272c",
				Exchange: "gmx-glp",
				Type:     "gmx-glp",
				Reserves: []string{
					"102520598156912802634",
					"85479038",
					"52934692750",
					"183945726570303834141294",
					"25122196790",
					"2122309542211585107",
					"4493451845739631563",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad",
						Swappable: true,
					},
					{
						Address:   "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
						Swappable: true,
					},
					{
						Address:   "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
						Swappable: true,
					},
					{
						Address:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
						Swappable: true,
					},
					{
						Address:   "0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22",
						Swappable: true,
					},
					{
						Address:   "0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239",
						Swappable: true,
					},
				},
				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"totalTokenWeights\":100000,\"taxBasisPoints\":50,\"mintBurnFeeBasicPoints\":20,\"whitelistedTokens\":[\"0x4200000000000000000000000000000000000006\",\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\",\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\",\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\",\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\",\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\",\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\"],\"poolAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":85479038,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1302073542211585107,\"0x4200000000000000000000000000000000000006\":102520598156912802634,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":184004659405528985510676,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25063263955,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":4493451845739631563,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":54284692750},\"bufferAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":100000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":20000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":5000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25000000000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":1000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":25000000000},\"reservedAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":25836406613068438164,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"tokenDecimals\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":18,\"0x4200000000000000000000000000000000000006\":18,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":18,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":6,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":18,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":6},\"stableTokens\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"usdgAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":23811042950934211834902,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":2213444076089850916117,\"0x4200000000000000000000000000000000000006\":160397684261193008925962,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":184163468406164606125144,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25059579892136637207042,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":24229615915606520555357,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":54270724147763950408093},\"maxUsdgAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":2000000000000000000000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":500000000000000000000000,\"0x4200000000000000000000000000000000000006\":2000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":185000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":3000000000000000000000000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":40000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":3000000000000000000000000},\"tokenWeights\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1000,\"0x4200000000000000000000000000000000000006\":39000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":8000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":20000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":4000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":20000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":8,\"0x4200000000000000000000000000000000000006\":8,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":8,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":8,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":8,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":8},\"spreadBasisPoints\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"adjustmentBasisPoints\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"strictStableTokens\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"isAdjustmentAdditive\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":false,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":false,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1697019943,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"maxPriceUpdateDelay\":3600,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":27299806000000000000000000000000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1651686000000000000000000000000000,\"0x4200000000000000000000000000000000000006\":1576639000000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":5113000000000000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"priceData\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":{\"refPrice\":2728533697992,\"refTime\":1697019945,\"cumulativeRefDelta\":7970,\"cumulativeFastDelta\":3700},\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"refPrice\":165301736450,\"refTime\":1697019945,\"cumulativeRefDelta\":3541,\"cumulativeFastDelta\":4415},\"0x4200000000000000000000000000000000000006\":{\"refPrice\":157500716537,\"refTime\":1697019945,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":6213},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":{\"refPrice\":510764598692,\"refTime\":1697019945,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":2883},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":1000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":1000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0}},\"secondaryPriceFeedVersion\":2,\"priceFeeds\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":{\"roundId\":18446744073709556311,\"answer\":2728533697992,\"answers\":{\"18446744073709556311\":2728533697992}},\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"roundId\":18446744073709552288,\"answer\":165301736450,\"answers\":{\"18446744073709552288\":165301736450}},\"0x4200000000000000000000000000000000000006\":{\"roundId\":18446744073709554396,\"answer\":157814759100,\"answers\":{\"18446744073709554396\":157814759100}},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"roundId\":18446744073709551689,\"answer\":99990000,\"answers\":{\"18446744073709551689\":99990000}},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"roundId\":18446744073709551689,\"answer\":99993210,\"answers\":{\"18446744073709551689\":99993210}},\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":{\"roundId\":18446744073709551760,\"answer\":510764598692,\"answers\":{\"18446744073709551760\":510764598692}},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"roundId\":18446744073709551689,\"answer\":99993210,\"answers\":{\"18446744073709551689\":99993210}}}},\"usdg\":{\"address\":\"0xE974A88385935CB8846482F3Ab01b6c0f70fa5f3\",\"totalSupply\":478227757802068867158645},\"UseSwapPricing\":false},\"glpManager\":{\"maximiseAumInUsdg\":468406105166267947996275,\"notMaximiseAumInUsdg\":468380313631821018110712,\"glpSupply\":473830145011703517364088,\"glp\":\"0xe771b4e273df31b85d7a7ae0efd22fb44bdd0633\"}}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad",
				Amount: bignumber.NewBig10("102520598156912"),
			},
			tokenOut:          "0xe771b4e273df31b85d7a7ae0efd22fb44bdd0633",
			expectedAmountOut: "530263907448717",
			expectedErr:       nil,
		},
		{
			name: "it should return correct unStake amount",
			entityPool: entity.Pool{
				Address:  "0xec8d8d4b215727f3476ff0ab41c406fa99b4272c",
				Exchange: "gmx-glp",
				Type:     "gmx-glp",
				Reserves: []string{
					"102520598156912802634",
					"85479038",
					"52934692750",
					"183945726570303834141294",
					"25122196790",
					"2122309542211585107",
					"4493451845739631563",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad",
						Swappable: true,
					},
					{
						Address:   "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
						Swappable: true,
					},
					{
						Address:   "0x50c5725949a6f0c72e6c4a641f24049a917db0cb",
						Swappable: true,
					},
					{
						Address:   "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
						Swappable: true,
					},
					{
						Address:   "0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22",
						Swappable: true,
					},
					{
						Address:   "0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239",
						Swappable: true,
					},
				},
				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"totalTokenWeights\":100000,\"taxBasisPoints\":50,\"mintBurnFeeBasicPoints\":20,\"whitelistedTokens\":[\"0x4200000000000000000000000000000000000006\",\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\",\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\",\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\",\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\",\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\",\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\"],\"poolAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":85479038,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1302073542211585107,\"0x4200000000000000000000000000000000000006\":102520598156912802634,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":184004659405528985510676,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25063263955,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":4493451845739631563,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":54284692750},\"bufferAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":100000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":20000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":5000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25000000000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":1000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":25000000000},\"reservedAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":25836406613068438164,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"tokenDecimals\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":18,\"0x4200000000000000000000000000000000000006\":18,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":18,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":6,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":18,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":6},\"stableTokens\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"usdgAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":23811042950934211834902,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":2213444076089850916117,\"0x4200000000000000000000000000000000000006\":160397684261193008925962,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":184163468406164606125144,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":25059579892136637207042,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":24229615915606520555357,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":54270724147763950408093},\"maxUsdgAmounts\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":2000000000000000000000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":500000000000000000000000,\"0x4200000000000000000000000000000000000006\":2000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":185000000000000000000000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":3000000000000000000000000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":40000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":3000000000000000000000000},\"tokenWeights\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1000,\"0x4200000000000000000000000000000000000006\":39000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":8000,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":20000,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":4000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":20000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":8,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":8,\"0x4200000000000000000000000000000000000006\":8,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":8,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":8,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":8,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":8},\"spreadBasisPoints\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"adjustmentBasisPoints\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":0,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":0,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"strictStableTokens\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":true,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":true,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":true},\"isAdjustmentAdditive\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":false,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":false,\"0x4200000000000000000000000000000000000006\":false,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":false,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":false,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":false,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":false},\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1697019943,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"maxPriceUpdateDelay\":3600,\"spreadBasisPointsIfChainError\":500,\"spreadBasisPointsIfInactive\":50,\"prices\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":27299806000000000000000000000000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":1651686000000000000000000000000000,\"0x4200000000000000000000000000000000000006\":1576639000000000000000000000000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":5113000000000000000000000000000000,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0},\"priceData\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":{\"refPrice\":2728533697992,\"refTime\":1697019945,\"cumulativeRefDelta\":7970,\"cumulativeFastDelta\":3700},\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"refPrice\":165301736450,\"refTime\":1697019945,\"cumulativeRefDelta\":3541,\"cumulativeFastDelta\":4415},\"0x4200000000000000000000000000000000000006\":{\"refPrice\":157500716537,\"refTime\":1697019945,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":6213},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0},\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":{\"refPrice\":510764598692,\"refTime\":1697019945,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":2883},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"refPrice\":0,\"refTime\":0,\"cumulativeRefDelta\":0,\"cumulativeFastDelta\":0}},\"maxCumulativeDeltaDiffs\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":1000000,\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":0,\"0x4200000000000000000000000000000000000006\":1000000,\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":0,\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":0,\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":0,\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":0}},\"secondaryPriceFeedVersion\":2,\"priceFeeds\":{\"0x1a35ee4640b0a3b87705b0a4b45d227ba60ca2ad\":{\"roundId\":18446744073709556311,\"answer\":2728533697992,\"answers\":{\"18446744073709556311\":2728533697992}},\"0x2ae3f1ec7f1f5012cfeab0185bfc7aa3cf0dec22\":{\"roundId\":18446744073709552288,\"answer\":165301736450,\"answers\":{\"18446744073709552288\":165301736450}},\"0x4200000000000000000000000000000000000006\":{\"roundId\":18446744073709554396,\"answer\":157814759100,\"answers\":{\"18446744073709554396\":157814759100}},\"0x50c5725949a6f0c72e6c4a641f24049a917db0cb\":{\"roundId\":18446744073709551689,\"answer\":99990000,\"answers\":{\"18446744073709551689\":99990000}},\"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913\":{\"roundId\":18446744073709551689,\"answer\":99993210,\"answers\":{\"18446744073709551689\":99993210}},\"0x9eaf8c1e34f05a589eda6bafdf391cf6ad3cb239\":{\"roundId\":18446744073709551760,\"answer\":510764598692,\"answers\":{\"18446744073709551760\":510764598692}},\"0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca\":{\"roundId\":18446744073709551689,\"answer\":99993210,\"answers\":{\"18446744073709551689\":99993210}}}},\"usdg\":{\"address\":\"0xE974A88385935CB8846482F3Ab01b6c0f70fa5f3\",\"totalSupply\":478227757802068867158645},\"UseSwapPricing\":false},\"glpManager\":{\"maximiseAumInUsdg\":468406105166267947996275,\"notMaximiseAumInUsdg\":468380313631821018110712,\"glpSupply\":473830145011703517364088,\"glp\":\"0xe771b4e273df31b85d7a7ae0efd22fb44bdd0633\"}}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0xe771b4e273df31b85d7a7ae0efd22fb44bdd0633",
				Amount: bignumber.NewBig10("27958439169018274391171424"),
			},
			tokenOut:          "0xd9aaec86b65d86f6a7b5b1b0c42ffa531710b6ca",
			expectedAmountOut: "530263907448717",
			expectedErr:       nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pool, _ := NewPoolSimulator(tc.entityPool)

			calcAmountOutResult, err := pool.CalcAmountOut(tc.tokenAmountIn, tc.tokenOut)

			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut.Amount.String())
		})
	}
}
