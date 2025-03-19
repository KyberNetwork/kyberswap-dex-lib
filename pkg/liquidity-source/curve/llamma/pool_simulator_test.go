package llamma

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestStatefullCalcAmountOut(t *testing.T) {
	// Check using Python code from the repository: https://github.com/0xreviews/crvusdsim
	poolStr := "{\"address\":\"0xfa96ad0a9e64261db86950e2da362f5572c5c6fd\",\"exchange\":\"curve-llamma\",\"type\":\"curve-llamma\",\"timestamp\":0,\"reserves\":[\"0\",\"1001000000000150100000\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xac3e018457b222d93114458476f3e3416abbe38f\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"BasePrice\\\":\\\"2500000000000000000000\\\",\\\"Fee\\\":\\\"10000000000000000\\\",\\\"AdminFeesX\\\":\\\"0\\\",\\\"AdminFeesY\\\":\\\"0\\\",\\\"AdminFee\\\":\\\"0\\\",\\\"dynamicFee\\\":\\\"10000000000000000\\\",\\\"priceOracle\\\":\\\"2500000000000000000000\\\",\\\"ActiveBand\\\":0,\\\"MinBand\\\":0,\\\"MaxBand\\\":39,\\\"bands\\\":null}\",\"staticExtra\":\"{\\\"A\\\":\\\"100\\\",\\\"useDynamicFee\\\":true}\",\"blockNumber\":0}"

	var ep entity.Pool
	err := json.Unmarshal([]byte(poolStr), &ep)
	require.Nil(t, err)

	sim, err := NewPoolSimulator(ep)
	require.Nil(t, err)
	require.NotNil(t, sim)

	sim.BandsX = map[int64]*uint256.Int{}
	sim.BandsY = map[int64]*uint256.Int{
		4:  uint256.MustFromDecimal("52631578947368422"),
		5:  uint256.MustFromDecimal("52631578964035091"),
		6:  uint256.MustFromDecimal("52631578969590647"),
		7:  uint256.MustFromDecimal("52631578969590642"),
		8:  uint256.MustFromDecimal("52631578969590642"),
		9:  uint256.MustFromDecimal("52631578969590642"),
		10: uint256.MustFromDecimal("52631578969590642"),
		11: uint256.MustFromDecimal("52631578952923976"),
		12: uint256.MustFromDecimal("52631578952923976"),
		13: uint256.MustFromDecimal("52631578952923976"),
		14: uint256.MustFromDecimal("52631578952923976"),
		15: uint256.MustFromDecimal("52631578947368421"),
		16: uint256.MustFromDecimal("52631578947368421"),
		17: uint256.MustFromDecimal("52631578947368421"),
		18: uint256.MustFromDecimal("1000052631578947368421"),
		19: uint256.MustFromDecimal("52631578947373201"),
		20: uint256.MustFromDecimal("52631578947373182"),
		21: uint256.MustFromDecimal("52631578947373182"),
		22: uint256.MustFromDecimal("52631578947373182"),
		23: uint256.MustFromDecimal("4761"),
		24: uint256.MustFromDecimal("4761"),
		25: uint256.MustFromDecimal("4761"),
		26: uint256.MustFromDecimal("4761"),
		27: uint256.MustFromDecimal("4761"),
		28: uint256.MustFromDecimal("4761"),
		29: uint256.MustFromDecimal("4761"),
		30: uint256.MustFromDecimal("4761"),
		31: uint256.MustFromDecimal("4761"),
		32: uint256.MustFromDecimal("4761"),
		33: uint256.MustFromDecimal("4761"),
		34: uint256.MustFromDecimal("4761"),
		35: uint256.MustFromDecimal("4761"),
		36: uint256.MustFromDecimal("4761"),
		37: uint256.MustFromDecimal("4761"),
		38: uint256.MustFromDecimal("4761"),
		39: uint256.MustFromDecimal("4761"),
	}

	testCases := []struct {
		index             int
		pump              bool
		amountIn          string
		expectedAmountOut string
		err               error
	}{
		{
			index:    0,
			pump:     false,
			amountIn: "100000",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             1,
			pump:              true,
			amountIn:          "10000000000000",
			expectedAmountOut: "3619691851",
		},
		{
			index:             2,
			pump:              true,
			amountIn:          "20000000",
			expectedAmountOut: "7239",
		},
		{
			index:             3,
			pump:              false,
			amountIn:          "500000000",
			expectedAmountOut: "1328490669204",
		},
		{
			index:             4,
			pump:              true,
			amountIn:          "8000000000000",
			expectedAmountOut: "2895753478",
		},
		{
			index:             5,
			pump:              true,
			amountIn:          "90000000000",
			expectedAmountOut: "32577226",
		},
		{
			index:    6,
			pump:     true,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             7,
			pump:              false,
			amountIn:          "2",
			expectedAmountOut: "2709",
		},
		{
			index:             8,
			pump:              false,
			amountIn:          "999999999999999999999",
			expectedAmountOut: "16761529328087",
		},
		{
			index:    9,
			pump:     false,
			amountIn: "123456789",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             10,
			pump:              true,
			amountIn:          "987654321",
			expectedAmountOut: "357500",
		},
		{
			index:    11,
			pump:     true,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    12,
			pump:     true,
			amountIn: "10",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    13,
			pump:     true,
			amountIn: "100",
			err:      ErrZeroSwapAmount,
		},
		{
			index:    14,
			pump:     true,
			amountIn: "1000",
			err:      ErrZeroSwapAmount,
		},
		{
			index:             15,
			pump:              true,
			amountIn:          "10000",
			expectedAmountOut: "3",
		},
		{
			index:             16,
			pump:              true,
			amountIn:          "100000",
			expectedAmountOut: "36",
		},
		{
			index:             17,
			pump:              false,
			amountIn:          "100000",
			expectedAmountOut: "265696461",
		},
		{
			index:             18,
			pump:              false,
			amountIn:          "10000",
			expectedAmountOut: "26567478",
		},
		{
			index:             19,
			pump:              false,
			amountIn:          "1000",
			expectedAmountOut: "2655122",
		},
		{
			index:             20,
			pump:              false,
			amountIn:          "100",
			expectedAmountOut: "265512",
		},
		{
			index:             21,
			pump:              false,
			amountIn:          "10",
			expectedAmountOut: "24383",
		},
		{
			index:    22,
			pump:     false,
			amountIn: "1",
			err:      ErrZeroSwapAmount,
		},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("test %d", idx), func(t *testing.T) {
			tokenInIndex, tokenOutIndex := sim.getBorrowedIndex(), sim.getCollateralIndex()
			tokenIn, tokenOut := sim.GetTokens()[tokenInIndex], sim.GetTokens()[tokenOutIndex]
			if !tc.pump {
				tokenIn, tokenOut = tokenOut, tokenIn
			}

			got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig(tc.amountIn)},
				TokenOut:      tokenOut,
			})
			if tc.err != nil {
				require.Equal(t, tc.err, err)
			} else {
				require.Nil(t, err)
				assert.True(t,
					tc.expectedAmountOut == got.TokenAmountOut.Amount.String(),
					"expected: %s, got: %s", tc.expectedAmountOut, got.TokenAmountOut.Amount.String())

				sim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: bignumber.NewBig10(tc.amountIn)},
					TokenAmountOut: *got.TokenAmountOut,
					SwapInfo:       got.SwapInfo,
					SwapLimit:      nil,
				})
			}
		})
	}
}

func TestCalcAmountOut(t *testing.T) {
	pools := map[string]string{
		// "sfrxETH":  "",
		// "wstETH":   "",
		// "WBTC":     "",
		// "WETH":     "",
		// "sfrxETH2": "",
		"weETH": "{\"address\":\"0xed325262f54b2987e74436f4556a27f748146da1\",\"exchange\":\"curve-llamma\",\"type\":\"curve-llamma\",\"timestamp\":1742258672,\"reserves\":[\"0\",\"984536042801171920064\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee\",\"decimals\":18,\"swappable\":true}],\"extra\":\"{\\\"BasePrice\\\":\\\"1979589682107363089106\\\",\\\"Fee\\\":\\\"6000000000000000\\\",\\\"AdminFeesX\\\":\\\"2083\\\",\\\"AdminFeesY\\\":\\\"0\\\",\\\"AdminFee\\\":\\\"0\\\",\\\"priceOracle\\\":\\\"2020167854781601479421\\\",\\\"ActiveBand\\\":-2,\\\"MinBand\\\":7,\\\"MaxBand\\\":45,\\\"bands\\\":[{\\\"i\\\":36,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192010\\\"},{\\\"i\\\":37,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":38,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":39,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":40,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":41,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":42,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":43,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":44,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"},{\\\"i\\\":45,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"98453604280117192006\\\"}]}\",\"staticExtra\":\"{\\\"A\\\":\\\"70\\\",\\\"useDynamicFee\\\":true}\",\"blockNumber\":22065620}",
		"cbBTC": "{\"address\":\"0xb6e62aa178a5421d0a51d17e720a05de78d3137a\",\"exchange\":\"curve-llamma\",\"type\":\"curve-llamma\",\"timestamp\":1742258672,\"reserves\":[\"0\",\"3972951168\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"decimals\":18,\"swappable\":true},{\"address\":\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\",\"decimals\":8,\"swappable\":true}],\"extra\":\"{\\\"BasePrice\\\":\\\"79063866664790308236025\\\",\\\"Fee\\\":\\\"6000000000000000\\\",\\\"AdminFeesX\\\":\\\"0\\\",\\\"AdminFeesY\\\":\\\"0\\\",\\\"AdminFee\\\":\\\"0\\\",\\\"priceOracle\\\":\\\"83559304372859758418995\\\",\\\"ActiveBand\\\":10,\\\"MinBand\\\":10,\\\"MaxBand\\\":48,\\\"bands\\\":[{\\\"i\\\":14,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"9115390770000000000\\\"},{\\\"i\\\":15,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"9161840898000000000\\\"},{\\\"i\\\":16,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"9161840898000000000\\\"},{\\\"i\\\":17,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"9161840898000000000\\\"},{\\\"i\\\":18,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":19,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":20,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":21,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":22,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":23,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":24,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"46450128000000000\\\"},{\\\"i\\\":25,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":26,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":27,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":28,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":29,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":30,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":31,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":32,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":33,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":34,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"76164662000000000\\\"},{\\\"i\\\":36,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"117821788000000000\\\"},{\\\"i\\\":37,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"117821788000000000\\\"},{\\\"i\\\":38,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"117821788000000000\\\"},{\\\"i\\\":39,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":40,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":41,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":42,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":43,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":44,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":45,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"204180068000000000\\\"},{\\\"i\\\":46,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"86358280000000000\\\"},{\\\"i\\\":47,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"86358280000000000\\\"},{\\\"i\\\":48,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"86358280000000000\\\"}]}\",\"staticExtra\":\"{\\\"A\\\":\\\"75\\\",\\\"useDynamicFee\\\":true}\",\"blockNumber\":22065620}",
		"LBTC":  "{\"address\":\"0x9a2e6bb3114b1eeb5492d97188a3ecb09e39fac8\",\"exchange\":\"curve-llamma\",\"type\":\"curve-llamma\",\"timestamp\":1742258672,\"reserves\":[\"62206732003586843\",\"590230\"],\"tokens\":[{\"address\":\"0xf939e0a03fb07f59a73314e73794be0e57ac1b4e\",\"decimals\":18,\"swappable\":true},{\"address\":\"0x8236a87084f8b84306f72007f36f2618a5634494\",\"decimals\":8,\"swappable\":true}],\"extra\":\"{\\\"BasePrice\\\":\\\"79068068809793641225472\\\",\\\"Fee\\\":\\\"6000000000000000\\\",\\\"AdminFeesX\\\":\\\"0\\\",\\\"AdminFeesY\\\":\\\"0\\\",\\\"AdminFee\\\":\\\"0\\\",\\\"priceOracle\\\":\\\"83521526869103297961456\\\",\\\"ActiveBand\\\":-3,\\\"MinBand\\\":-3,\\\"MaxBand\\\":6,\\\"bands\\\":[{\\\"i\\\":-3,\\\"x\\\":\\\"62206732003586843\\\",\\\"y\\\":\\\"550659637527530\\\"},{\\\"i\\\":-2,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"578808445342571\\\"},{\\\"i\\\":-1,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":0,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":1,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":2,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":3,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":4,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":5,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"},{\\\"i\\\":6,\\\"x\\\":\\\"0\\\",\\\"y\\\":\\\"596602000000000\\\"}]}\",\"staticExtra\":\"{\\\"A\\\":\\\"75\\\",\\\"useDynamicFee\\\":true}\",\"blockNumber\":22065620}\n",
	}

	type testCase struct {
		pool     string
		name     string
		pump     bool
		tokenIn  string
		amountIn string

		expectedTokenOut  string
		expectedAmountOut string
		expectedError     error
	}

	testCases := []testCase{
		{
			pool:          "weETH",
			pump:          true,
			name:          "new crvUSD -> weETH",
			tokenIn:       "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:      "100",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:          "weETH",
			pump:          true,
			name:          "new crvUSD -> weETH",
			tokenIn:       "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:      "1000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "100000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "14087",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "1000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "140879",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "10000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "1408794",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "10000000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "1408794778",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "1000000000000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "140879474927682",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "new crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "1000000000000000000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "137946696457338693778",
		},
		{
			pool:              "weETH",
			pump:              true,
			name:              "pump all crvUSD -> weETH",
			tokenIn:           "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			amountIn:          "10000000000000000000000000",
			expectedTokenOut:  "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			expectedAmountOut: "984536042801171920064",
		},
		{
			pool:          "weETH",
			pump:          false,
			name:          "dump crvUSD <- weETH",
			tokenIn:       "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:      "1000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:          "weETH",
			pump:          false,
			name:          "dump crvUSD <- weETH",
			tokenIn:       "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:      "1000000000000000000000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:          "cbBTC",
			pump:          true,
			name:          "pump crvUSD -> cbBTC",
			tokenIn:       "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:      "1000000000000000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:              "cbBTC",
			pump:              true,
			name:              "pump crvUSD -> cbBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "10000000000000000",
			expectedTokenOut:  "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
			expectedAmountOut: "6",
		},
		{
			pool:              "cbBTC",
			pump:              true,
			name:              "pump crvUSD -> cbBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "100000000000000000",
			expectedTokenOut:  "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
			expectedAmountOut: "66",
		},
		{
			pool:              "cbBTC",
			pump:              true,
			name:              "pump crvUSD -> cbBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "1000000000000000000",
			expectedTokenOut:  "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
			expectedAmountOut: "664",
		},
		{
			pool:              "cbBTC",
			pump:              true,
			name:              "pump crvUSD -> cbBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "1000000000000000000000000",
			expectedTokenOut:  "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
			expectedAmountOut: "658527379",
		},
		{
			pool:          "cbBTC",
			pump:          true,
			name:          "dump crvUSD <- cbBTC",
			tokenIn:       "0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf",
			amountIn:      "10000000000000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:          "LBTC",
			pump:          true,
			name:          "pump crvUSD -> LBTC",
			tokenIn:       "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:      "100000000000000",
			expectedError: ErrZeroSwapAmount,
		},
		{
			pool:              "LBTC",
			pump:              true,
			name:              "pump crvUSD -> LBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "1000000000000000",
			expectedTokenOut:  "0x8236a87084f8b84306f72007f36f2618a5634494",
			expectedAmountOut: "1",
		},
		{
			pool:              "LBTC",
			pump:              true,
			name:              "pump crvUSD -> LBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "10000000000000000",
			expectedTokenOut:  "0x8236a87084f8b84306f72007f36f2618a5634494",
			expectedAmountOut: "11",
		},
		{
			pool:              "LBTC",
			pump:              true,
			name:              "pump crvUSD -> LBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "100000000000000000",
			expectedTokenOut:  "0x8236a87084f8b84306f72007f36f2618a5634494",
			expectedAmountOut: "115",
		},
		{
			pool:              "LBTC",
			pump:              true,
			name:              "pump crvUSD -> LBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "1000000000000000000",
			expectedTokenOut:  "0x8236a87084f8b84306f72007f36f2618a5634494",
			expectedAmountOut: "1154",
		},
		{
			pool:              "LBTC",
			pump:              true,
			name:              "pump crvUSD -> LBTC",
			tokenIn:           "0xcd5fe23c85820f7b72d0926fc9b05b43e359b7ee",
			amountIn:          "1000000000000000000000000",
			expectedTokenOut:  "0x8236a87084f8b84306f72007f36f2618a5634494",
			expectedAmountOut: "590228",
		},
		{
			pool:              "LBTC",
			pump:              false,
			name:              "dump crvUSD <- LBTC",
			tokenIn:           "0x8236a87084f8b84306f72007f36f2618a5634494",
			amountIn:          "1",
			expectedTokenOut:  "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			expectedAmountOut: "853699137884736",
		},
		{
			pool:              "LBTC",
			pump:              false,
			name:              "dump crvUSD <- LBTC",
			tokenIn:           "0x8236a87084f8b84306f72007f36f2618a5634494",
			amountIn:          "10",
			expectedTokenOut:  "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			expectedAmountOut: "8536972932083106",
		},
		{
			pool:              "LBTC",
			pump:              false,
			name:              "dump crvUSD <- LBTC",
			tokenIn:           "0x8236a87084f8b84306f72007f36f2618a5634494",
			amountIn:          "100",
			expectedTokenOut:  "0xf939e0a03fb07f59a73314e73794be0e57ac1b4e",
			expectedAmountOut: "62206732003586843",
		},
	}

	sims := lo.MapEntries(pools, func(k, v string) (string, *PoolSimulator) {
		var ep entity.Pool
		err := json.Unmarshal([]byte(v), &ep)
		require.Nil(t, err)

		sim, err := NewPoolSimulator(ep)
		require.Nil(t, err)
		require.NotNil(t, sim)

		return k, sim
	})

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("test %v", tc.name), func(t *testing.T) {
			sim := sims[tc.pool]
			require.NotNil(t, sim)

			inIdx, outIdx := sim.getBorrowedIndex(), sim.getCollateralIndex()
			if !tc.pump {
				inIdx, outIdx = outIdx, inIdx
			}

			got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: sim.GetTokens()[inIdx], Amount: bignumber.NewBig(tc.amountIn)},
				TokenOut:      sim.GetTokens()[outIdx],
			})
			require.Equal(t, tc.expectedError, err)
			if tc.expectedError == nil {
				require.Nil(t, err)
				require.Equal(t, tc.expectedTokenOut, got.TokenAmountOut.Token)

				expectedAmountOutFloat64, _ := strconv.ParseFloat(tc.expectedAmountOut, 64)
				gotAmountOutFloat64, _ := got.TokenAmountOut.Amount.Float64()
				t.Logf("diff %.6f%%", (gotAmountOutFloat64/expectedAmountOutFloat64-1)*100)
				require.True(t, approx(expectedAmountOutFloat64, gotAmountOutFloat64, 1e-4, 0))
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {

}

func approx(x1, x2, precision, absPrecision float64) bool {
	if precision >= 1 {
		return true
	}
	result := false
	if absPrecision != 0 {
		result = math.Abs(x2-x1) <= absPrecision
	} else {
		absPrecision = 0
	}
	if x2 == 0 {
		return math.Abs(x1) <= absPrecision
	} else if x1 == 0 {
		return math.Abs(x2) <= absPrecision
	}
	return result || (math.Abs(math.Log(x1/x2)) <= precision)
}
