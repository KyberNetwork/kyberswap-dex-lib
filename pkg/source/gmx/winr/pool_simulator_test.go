package winr

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	poolPkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func getPool(isSwapEnabled bool) entity.Pool {
	return entity.Pool{
		Address:  "0x489ee077994b6658eafa855c308275ead8097c4a",
		Exchange: "winr",
		Type:     "winr",
		Reserves: []string{
			"167076861135",
			"43017196799106911057528",
			"102386518696054",
			"565590490613956392825536",
			"306644459880480991236045",
			"2341824812754",
			"575853493761361399",
			"5883596810011698955188172",
			"15080772970488647125188999",
		},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
				Swappable: true,
			},
			{
				Address:   "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
				Swappable: true,
			},
			{
				Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
				Swappable: true,
			},
			{
				Address:   "0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
				Swappable: true,
			},
			{
				Address:   "0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
				Swappable: true,
			},
			{
				Address:   "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
				Swappable: true,
			},
			{
				Address:   "0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
				Swappable: true,
			},
			{
				Address:   "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Swappable: true,
			},
			{
				Address:   "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
				Swappable: true,
			},
		},
		Extra: fmt.Sprintf("{\"vault\":{\"hasDynamicFees\":true,\"isSwapEnabled\":%t,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100001,\"bufferAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":150000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":38000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":6000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":100000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":85000000000000},\"whitelistedTokens\":[\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\",\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\",\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\",\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\"],\"poolAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6519788682577332118251092,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":219815695089,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":49260098176278584480106,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15992252153126931909711849,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":639479769164077825433768,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":298029962360974882529804,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3429458903551,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":757712078649433621,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":103726704414885},\"tokenDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":18,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":18,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":18,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":18,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":18,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":6,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":18,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":6},\"stableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"usdwAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":5848526070946065485831073,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":35992305182501199876113159,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":61622981434523338602970751,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":14959945068283502625618892,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":3365878830264306289250099,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2051986511691393819746061,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2345972841404642490763341,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":575853493761361399,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":100654458313698251269013031},\"maxUsdwAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6500000000000000000000000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":50000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":120000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":6000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2500000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3500000000000000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1000000000000000000,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":120000000000000000000000000},\"tokenWeights\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":2000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":25000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":28000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":5000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":1000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":1000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":36000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":8,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":8,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":8,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"spreadBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":20,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"adjustmentBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":0,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":0,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"strictStableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"isAdjustmentAdditive\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":false,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":false,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":false,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":false,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":false},\"chainlinkFlags\":{\"flags\":{\"0xa438451d6458044c3c8cd2f6f31c91ac882a6d91\":false}},\"secondaryPriceFeedVersion\":1,\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1660186564,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"volBasisPoints\":0,\"prices\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":24274290000000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":1877570000000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":9119000000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":9287000000000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0}},\"priceFeeds\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":{\"roundId\":18446744073709552645,\"answer\":100024010,\"answers\":{\"18446744073709552645\":100024010}},\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"roundId\":18446744073709629883,\"answer\":2428233038195,\"answers\":{\"18446744073709629883\":2428233038195}},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"roundId\":18446744073709766709,\"answer\":187831000000,\"answers\":{\"18446744073709766709\":187831000000}},\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":{\"roundId\":18446744073709559243,\"answer\":100090564,\"answers\":{\"18446744073709559243\":100090564}},\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":{\"roundId\":18446744073709599361,\"answer\":911661972,\"answers\":{\"18446744073709599361\":911661972}},\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":{\"roundId\":18446744073709604372,\"answer\":927926606,\"answers\":{\"18446744073709604372\":927926606}},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"roundId\":18446744073709553269,\"answer\":100000000,\"answers\":{\"18446744073709553269\":100000000}},\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":{\"roundId\":18446744073709552597,\"answer\":99751504,\"answers\":{\"18446744073709552597\":99751504}},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"roundId\":18446744073709553457,\"answer\":99991237,\"answers\":{\"18446744073709553457\":99991237}}}},\"usdw\":{\"address\":\"0x45096e7aA921f27590f8F19e457794EB09678141\",\"totalSupply\":282098184855476286376531249}}}", isSwapEnabled),
	}
}
func TestPool_CalcAmountOut(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name              string
		entityPool        entity.Pool
		tokenAmountIn     poolPkg.TokenAmount
		tokenOut          string
		expectedAmountOut *poolPkg.TokenAmount
		expectedFee       *poolPkg.TokenAmount
		expectedGas       int64
		expectedErr       error
	}{
		{
			name:       "it should return correct amount using getPriceV1",
			entityPool: getPool(true),
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: &poolPkg.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: new(big.Int).SetInt64(530263907448717)},
			expectedFee:       &poolPkg.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: new(big.Int).SetInt64(2129573925497)},
			expectedGas:       165000,
			expectedErr:       nil,
		},
		{
			name:       "it should return correct amount using getPriceV2",
			entityPool: getPool(true),
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: &poolPkg.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: new(big.Int).SetInt64(530263907448717)},
			expectedFee:       &poolPkg.TokenAmount{Token: "0x82af49447d8a07e3bd95bd0d56f35241523fbab1", Amount: new(big.Int).SetInt64(2129573925497)},
			expectedGas:       165000,
			expectedErr:       nil,
		},
		{
			name:       "it should return ErrVaultSwapsNotEnabled when vault is disable swap",
			entityPool: getPool(false),
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: nil,
			expectedFee:       nil,
			expectedGas:       0,
			expectedErr:       gmx.ErrVaultSwapsNotEnabled,
		},
		{
			name: "it should return ErrVaultPriceFeedInvalidPriceFeed when price feed is invalid v1",
			entityPool: entity.Pool{
				Address:  "0x489ee077994b6658eafa855c308275ead8097c4a",
				Exchange: "gmx",
				Type:     "gmx",
				Reserves: []string{
					"167076861135",
					"43017196799106911057528",
					"102386518696054",
					"565590490613956392825536",
					"306644459880480991236045",
					"2341824812754",
					"575853493761361399",
					"5883596810011698955188172",
					"15080772970488647125188999",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
						Swappable: true,
					},
					{
						Address:   "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						Swappable: true,
					},
					{
						Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						Swappable: true,
					},
					{
						Address:   "0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
						Swappable: true,
					},
					{
						Address:   "0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
						Swappable: true,
					},
					{
						Address:   "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
						Swappable: true,
					},
					{
						Address:   "0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
						Swappable: true,
					},
					{
						Address:   "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
						Swappable: true,
					},
					{
						Address:   "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
						Swappable: true,
					},
				},
				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100001,\"bufferAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":150000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":38000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":6000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":100000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":85000000000000},\"whitelistedTokens\":[\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\",\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\",\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\",\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\"],\"poolAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6519788682577332118251092,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":219815695089,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":49260098176278584480106,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15992252153126931909711849,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":639479769164077825433768,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":298029962360974882529804,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3429458903551,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":757712078649433621,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":103726704414885},\"reservedAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":303782519145927671527588,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":20157424075,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":14211256424348089508681,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":325216808461824176853526,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":71980988686260872025702,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":11856899719477956520764,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1409426517465,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":27985830646075},\"tokenDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":18,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":18,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":18,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":18,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":18,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":6,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":18,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":6},\"stableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"usdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":5848526070946065485831073,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":35992305182501199876113159,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":61622981434523338602970751,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":14959945068283502625618892,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":3365878830264306289250099,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2051986511691393819746061,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2345972841404642490763341,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":575853493761361399,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":100654458313698251269013031},\"maxUsdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6500000000000000000000000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":50000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":120000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":6000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2500000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3500000000000000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1000000000000000000,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":120000000000000000000000000},\"tokenWeights\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":2000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":25000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":28000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":5000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":1000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":1000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":36000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":true,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":8,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":8,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":8,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"spreadBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":20,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"adjustmentBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":0,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":0,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"strictStableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"isAdjustmentAdditive\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":false,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":false,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":false,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":false,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":false},\"chainlinkFlags\":{\"flags\":{\"0xa438451d6458044c3c8cd2f6f31c91ac882a6d91\":false}},\"secondaryPriceFeedVersion\":1,\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1660186564,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"volBasisPoints\":0,\"prices\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":24274290000000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":1877570000000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":9119000000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":9287000000000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0}},\"priceFeeds\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":{\"roundId\":18446744073709552645,\"answer\":100024010,\"answers\":{\"18446744073709552645\":100024010}},\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"roundId\":18446744073709629883,\"answer\":2428233038195,\"answers\":{\"18446744073709629883\":2428233038195}},\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":{\"roundId\":18446744073709559243,\"answer\":100090564,\"answers\":{\"18446744073709559243\":100090564}},\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":{\"roundId\":18446744073709599361,\"answer\":911661972,\"answers\":{\"18446744073709599361\":911661972}},\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":{\"roundId\":18446744073709604372,\"answer\":927926606,\"answers\":{\"18446744073709604372\":927926606}},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"roundId\":18446744073709553269,\"answer\":100000000,\"answers\":{\"18446744073709553269\":100000000}},\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":{\"roundId\":18446744073709552597,\"answer\":99751504,\"answers\":{\"18446744073709552597\":99751504}},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"roundId\":18446744073709553457,\"answer\":99991237,\"answers\":{\"18446744073709553457\":99991237}}}},\"usdg\":{\"address\":\"0x45096e7aA921f27590f8F19e457794EB09678141\",\"totalSupply\":282098184855476286376531249}}}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: nil,
			expectedFee:       nil,
			expectedGas:       0,
			expectedErr:       gmx.ErrVaultPriceFeedInvalidPriceFeed,
		},
		{
			name: "it should return ErrVaultPriceFeedInvalidPriceFeed when price feed is invalid v2",
			entityPool: entity.Pool{
				Address:  "0x489ee077994b6658eafa855c308275ead8097c4a",
				Exchange: "gmx",
				Type:     "gmx",
				Reserves: []string{
					"167076861135",
					"43017196799106911057528",
					"102386518696054",
					"565590490613956392825536",
					"306644459880480991236045",
					"2341824812754",
					"575853493761361399",
					"5883596810011698955188172",
					"15080772970488647125188999",
				},
				Tokens: []*entity.PoolToken{
					{
						Address:   "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
						Swappable: true,
					},
					{
						Address:   "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
						Swappable: true,
					},
					{
						Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
						Swappable: true,
					},
					{
						Address:   "0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
						Swappable: true,
					},
					{
						Address:   "0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
						Swappable: true,
					},
					{
						Address:   "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
						Swappable: true,
					},
					{
						Address:   "0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
						Swappable: true,
					},
					{
						Address:   "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
						Swappable: true,
					},
					{
						Address:   "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
						Swappable: true,
					},
				},
				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":true,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100001,\"bufferAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":150000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":38000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":6000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":100000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":85000000000000},\"whitelistedTokens\":[\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\",\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\",\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\",\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\"],\"poolAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6519788682577332118251092,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":219815695089,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":49260098176278584480106,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15992252153126931909711849,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":639479769164077825433768,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":298029962360974882529804,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3429458903551,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":757712078649433621,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":103726704414885},\"reservedAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":303782519145927671527588,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":20157424075,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":14211256424348089508681,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":325216808461824176853526,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":71980988686260872025702,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":11856899719477956520764,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1409426517465,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":27985830646075},\"tokenDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":18,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":18,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":18,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":18,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":18,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":6,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":18,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":6},\"stableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"usdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":5848526070946065485831073,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":35992305182501199876113159,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":61622981434523338602970751,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":14959945068283502625618892,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":3365878830264306289250099,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2051986511691393819746061,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2345972841404642490763341,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":575853493761361399,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":100654458313698251269013031},\"maxUsdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6500000000000000000000000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":50000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":120000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":6000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2500000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3500000000000000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1000000000000000000,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":120000000000000000000000000},\"tokenWeights\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":2000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":25000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":28000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":5000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":1000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":1000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":36000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":true,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":true,\"priceDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":8,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":8,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":8,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"spreadBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":20,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"adjustmentBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":0,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":0,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"strictStableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"isAdjustmentAdditive\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":false,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":false,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":false,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":false,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":false},\"chainlinkFlags\":{\"flags\":{\"0xa438451d6458044c3c8cd2f6f31c91ac882a6d91\":false}},\"secondaryPriceFeedVersion\":1,\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1660186564,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":0,\"priceDuration\":999999999999999999,\"volBasisPoints\":0,\"prices\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":24274290000000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":1877570000000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":9119000000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":9287000000000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0}},\"priceFeeds\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":{\"roundId\":18446744073709552645,\"answer\":100024010,\"answers\":{\"18446744073709552645\":100024010}},\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"roundId\":18446744073709629883,\"answer\":2428233038195,\"answers\":{\"18446744073709629883\":2428233038195}},\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":{\"roundId\":18446744073709559243,\"answer\":100090564,\"answers\":{\"18446744073709559243\":100090564}},\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":{\"roundId\":18446744073709599361,\"answer\":911661972,\"answers\":{\"18446744073709599361\":911661972}},\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":{\"roundId\":18446744073709604372,\"answer\":927926606,\"answers\":{\"18446744073709604372\":927926606}},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"roundId\":18446744073709553269,\"answer\":100000000,\"answers\":{\"18446744073709553269\":100000000}},\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":{\"roundId\":18446744073709552597,\"answer\":99751504,\"answers\":{\"18446744073709552597\":99751504}},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"roundId\":18446744073709553457,\"answer\":99991237,\"answers\":{\"18446744073709553457\":99991237}}}},\"usdg\":{\"address\":\"0x45096e7aA921f27590f8F19e457794EB09678141\",\"totalSupply\":282098184855476286376531249}}}",
			},
			tokenAmountIn: poolPkg.TokenAmount{
				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
			},
			tokenOut:          "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
			expectedAmountOut: nil,
			expectedFee:       nil,
			expectedGas:       0,
			expectedErr:       gmx.ErrVaultPriceFeedInvalidPriceFeed,
		},
	}

	for _, tc := range testCases[2:3] {
		t.Run(tc.name, func(t *testing.T) {
			pool, _ := NewPoolSimulator(tc.entityPool)

			calcAmountOutResult, err := testutil.MustConcurrentSafe(t, func() (*poolPkg.CalcAmountOutResult, error) {
				return pool.CalcAmountOut(poolPkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tc.expectedAmountOut, calcAmountOutResult.TokenAmountOut)
			assert.Equal(t, tc.expectedFee, calcAmountOutResult.Fee)
			assert.Equal(t, tc.expectedGas, calcAmountOutResult.Gas)
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

// func TestPool_UpdateBalance(t *testing.T) {
// 	t.Parallel()

// 	setBigIntFromStr := func(str string) *big.Int {
// 		value, _ := new(big.Int).SetString(str, 10)
// 		return value
// 	}

// 	testCases := []struct {
// 		name                string
// 		entityPool          entity.Pool
// 		tokenAmountIn       poolPkg.TokenAmount
// 		tokenAmountOut      poolPkg.TokenAmount
// 		expectedPool        PoolSimulator
// 		expectedUSDGAmounts map[string]*big.Int
// 	}{
// 		{
// 			name: "it should update balance correctly",
// 			entityPool: entity.Pool{
// 				Address:  "0x489ee077994b6658eafa855c308275ead8097c4a",
// 				Exchange: "gmx",
// 				Type:     "gmx",
// 				Reserves: []string{
// 					"167076861135",
// 					"43017196799106911057528",
// 					"102386518696054",
// 					"565590490613956392825536",
// 					"306644459880480991236045",
// 					"2341824812754",
// 					"575853493761361399",
// 					"5883596810011698955188172",
// 					"15080772970488647125188999",
// 				},
// 				Tokens: []*entity.PoolToken{
// 					{
// 						Address:   "0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
// 						Swappable: true,
// 					},
// 					{
// 						Address:   "0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
// 						Swappable: true,
// 					},
// 				},
// 				Extra: "{\"vault\":{\"hasDynamicFees\":true,\"includeAmmPrice\":false,\"isSwapEnabled\":true,\"stableSwapFeeBasisPoints\":1,\"stableTaxBasisPoints\":5,\"swapFeeBasisPoints\":30,\"taxBasisPoints\":50,\"totalTokenWeights\":100001,\"bufferAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":150000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":38000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":6000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":100000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":85000000000000},\"whitelistedTokens\":[\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\",\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\",\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\",\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\",\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\",\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\"],\"poolAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6519788682577332118251092,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":219815695089,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":49260098176278584480106,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15992252153126931909711849,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":639479769164077825433768,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":298029962360974882529804,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3429458903551,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":757712078649433621,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":103726704414885},\"reservedAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":303782519145927671527588,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":20157424075,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":14211256424348089508681,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":325216808461824176853526,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":71980988686260872025702,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":11856899719477956520764,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":1409426517465,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":27985830646075},\"tokenDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":18,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":18,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":18,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":18,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":18,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":6,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":18,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":6},\"stableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"usdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":5848526070946065485831073,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":35992305182501199876113159,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":61622981434523338602970751,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":14959945068283502625618892,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":3365878830264306289250099,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2051986511691393819746061,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2345972841404642490763341,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":575853493761361399,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":100654458313698251269013031},\"maxUsdgAmounts\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":6500000000000000000000000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":50000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":120000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":15000000000000000000000000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":6000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":2500000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":3500000000000000000000000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1000000000000000000,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":120000000000000000000000000},\"tokenWeights\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":2000,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":25000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":28000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":5000,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":1000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":1000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":2000,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":1,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":36000},\"priceFeed\":{\"bnb\":\"0x0000000000000000000000000000000000000000\",\"btc\":\"0x0000000000000000000000000000000000000000\",\"eth\":\"0x0000000000000000000000000000000000000000\",\"favorPrimaryPrice\":false,\"isAmmEnabled\":false,\"isSecondaryPriceEnabled\":true,\"maxStrictPriceDeviation\":10000000000000000000000000000,\"priceSampleSpace\":1,\"spreadThresholdBasisPoints\":30,\"useV2Pricing\":false,\"priceDecimals\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":8,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":8,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":8,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":8,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":8,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":8,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":8,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":8,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":8},\"spreadBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":20,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":20,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"adjustmentBasisPoints\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":0,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":0,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":0,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":0,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0},\"strictStableTokens\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":true,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":true,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":true,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":true,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":true},\"isAdjustmentAdditive\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":false,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":false,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":false,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":false,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":false,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":false,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":false,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":false,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":false},\"chainlinkFlags\":{\"flags\":{\"0xa438451d6458044c3c8cd2f6f31c91ac882a6d91\":false}},\"secondaryPriceFeedVersion\":1,\"secondaryPriceFeed\":{\"disableFastPriceVoteCount\":0,\"isSpreadEnabled\":false,\"lastUpdatedAt\":1660186564,\"maxDeviationBasisPoints\":250,\"minAuthorizations\":1,\"priceDuration\":300,\"volBasisPoints\":0,\"prices\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":0,\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":24274290000000000000000000000000000,\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":1877570000000000000000000000000000,\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":0,\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":9119000000000000000000000000000,\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":9287000000000000000000000000000,\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":0,\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":0,\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":0}},\"priceFeeds\":{\"0x17fc002b466eec40dae837fc4be5c67993ddbd6f\":{\"roundId\":18446744073709552645,\"answer\":100024010,\"answers\":{\"18446744073709552645\":100024010}},\"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f\":{\"roundId\":18446744073709629883,\"answer\":2428233038195,\"answers\":{\"18446744073709629883\":2428233038195}},\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\":{\"roundId\":18446744073709766709,\"answer\":187831000000,\"answers\":{\"18446744073709766709\":187831000000}},\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\":{\"roundId\":18446744073709559243,\"answer\":100090564,\"answers\":{\"18446744073709559243\":100090564}},\"0xf97f4df75117a78c1a5a0dbb814af92458539fb4\":{\"roundId\":18446744073709599361,\"answer\":911661972,\"answers\":{\"18446744073709599361\":911661972}},\"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0\":{\"roundId\":18446744073709604372,\"answer\":927926606,\"answers\":{\"18446744073709604372\":927926606}},\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\":{\"roundId\":18446744073709553269,\"answer\":100000000,\"answers\":{\"18446744073709553269\":100000000}},\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\":{\"roundId\":18446744073709552597,\"answer\":99751504,\"answers\":{\"18446744073709552597\":99751504}},\"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8\":{\"roundId\":18446744073709553457,\"answer\":99991237,\"answers\":{\"18446744073709553457\":99991237}}}},\"usdg\":{\"address\":\"0x45096e7aA921f27590f8F19e457794EB09678141\",\"totalSupply\":282098184855476286376531249}}}",
// 			},
// 			tokenAmountIn: poolPkg.TokenAmount{
// 				Token:  "0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
// 				Amount: new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil),
// 			},
// 			tokenAmountOut: poolPkg.TokenAmount{
// 				Token:  "0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
// 				Amount: new(big.Int).SetInt64(530263907448717),
// 			},
// 			expectedUSDGAmounts: map[string]*big.Int{
// 				"0x17fc002b466eec40dae837fc4be5c67993ddbd6f": setBigIntFromStr("5848527070946065485831073"),
// 				"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f": setBigIntFromStr("35992305182501199876113159"),
// 				"0x82af49447d8a07e3bd95bd0d56f35241523fbab1": setBigIntFromStr("61622980434523338602970751"),
// 				"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1": setBigIntFromStr("14959945068283502625618892"),
// 				"0xf97f4df75117a78c1a5a0dbb814af92458539fb4": setBigIntFromStr("3365878830264306289250099"),
// 				"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0": setBigIntFromStr("2051986511691393819746061"),
// 				"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9": setBigIntFromStr("2345972841404642490763341"),
// 				"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a": setBigIntFromStr("575853493761361399"),
// 				"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8": setBigIntFromStr("100654458313698251269013031"),
// 			},
// 		},
// 	}

// 	for _, tc := range testCases {
// 		t.Run(tc.name, func(t *testing.T) {
// 			pool, _ := NewPoolSimulator(tc.entityPool)
// 			params := poolPkg.UpdateBalanceParams{
// 				TokenAmountIn:  tc.tokenAmountIn,
// 				TokenAmountOut: tc.tokenAmountOut,
// 				Fee: poolPkg.TokenAmount{
// 					Amount: bignumber.ZeroBI,
// 				},
// 				SwapInfo: nil,
// 			}
// 			pool.UpdateBalance(params)

// 			assert.Equal(t, tc.expectedUSDGAmounts, pool.vault.USDGAmounts)
// 		})
// 	}
// }

// func TestPool_CanSwapTo(t *testing.T) {
// 	t.Run("it should return correct swappable tokens", func(t *testing.T) {
// 		pool := PoolSimulator{
// 			vault: &Vault{
// 				WhitelistedTokens: []string{
// 					"0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
// 					"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
// 					"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
// 					"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
// 					"0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
// 					"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
// 					"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
// 					"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
// 					"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8",
// 				},
// 			},
// 		}

// 		tokens := pool.CanSwapTo("0xff970a61a04b1ca14834a43f5de4533ebddb5cc8")

// 		assert.Equal(t, []string{
// 			"0x17fc002b466eec40dae837fc4be5c67993ddbd6f",
// 			"0x2f2a2543b76a4166549f7aab2e75bef0aefc5b0f",
// 			"0x82af49447d8a07e3bd95bd0d56f35241523fbab1",
// 			"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1",
// 			"0xf97f4df75117a78c1a5a0dbb814af92458539fb4",
// 			"0xfa7f8980b0f1e64a2062791cc3b0871572f1f7f0",
// 			"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
// 			"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a",
// 		}, tokens)
// 	})
// }

// func TestPool_GetMetaInfo(t *testing.T) {
// 	t.Run("it should return nil", func(t *testing.T) {
// 		pool := PoolSimulator{}

// 		assert.Nil(t, pool.GetMetaInfo("0xda10009cbd5d07dd0cecc66161fc93d7c9000da1", "0xff970a61a04b1ca14834a43f5de4533ebddb5cc8"))
// 	})
// }
