package uniswapv4

import (
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

var tokens = map[string]struct {
	Address string
}{
	"WETH":   {"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"},
	"UNI":    {"0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"},
	"fwWETH": {"0xa250cc729bb3323e7933022a67b52200fe354767"},
	"fwUNI":  {"0xe8e1f50392bd61d0f8f48e8e7af51d3b8a52090a"},
	"KNC":    {"0xdefa4e8a7bcba345f687a2f1456f5edd9ce97202"},
}

func TestCanSwapFromUniV4(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name           string
		poolTokens     []string
		swapToken      string
		isTokenIn      bool
		expectedResult []string
	}{
		{
			name:           "should use normal behavior if pool has no FEW token",
			poolTokens:     []string{"WETH", "KNC"},
			swapToken:      "KNC",
			isTokenIn:      true,
			expectedResult: []string{"WETH"},
		},
		{
			name:           "should still use normal behavior if pool has only one FEW token, case normal token is tokenIn",
			poolTokens:     []string{"UNI", "fwUNI"},
			swapToken:      "UNI",
			isTokenIn:      true,
			expectedResult: []string{"fwUNI"},
		},
		{
			name:           "should still use normal behavior if pool has only one FEW token, case normal token is tokenOut",
			poolTokens:     []string{"UNI", "fwUNI"},
			swapToken:      "UNI",
			isTokenIn:      false,
			expectedResult: []string{"fwUNI"},
		},
		{
			name:           "should still use normal behavior if pool has only one FEW token, case few token is tokenIn",
			poolTokens:     []string{"UNI", "fwUNI"},
			swapToken:      "fwUNI",
			isTokenIn:      true,
			expectedResult: []string{"UNI"},
		},
		{
			name:           "should still use normal behavior if pool has only one FEW token, case few token is tokenOut",
			poolTokens:     []string{"UNI", "fwUNI"},
			swapToken:      "fwUNI",
			isTokenIn:      false,
			expectedResult: []string{"UNI"},
		},
		{
			name:           "should swap from FEW token to both normal and FEW token",
			poolTokens:     []string{"fwWETH", "fwUNI"},
			swapToken:      "fwWETH",
			isTokenIn:      true,
			expectedResult: []string{"fwUNI", "UNI"},
		},
		{
			name:           "should swap from normal token to both normal and FEW token",
			poolTokens:     []string{"fwWETH", "fwUNI"},
			swapToken:      "WETH",
			isTokenIn:      true,
			expectedResult: []string{"fwUNI", "UNI"},
		},
		{
			name:           "should swap to FEW token from both normal token and FEW token",
			poolTokens:     []string{"fwWETH", "fwUNI"},
			swapToken:      "fwUNI",
			isTokenIn:      false,
			expectedResult: []string{"WETH", "fwWETH"},
		},
		{
			name:           "should swap to normal token from both normal token and FEW token",
			poolTokens:     []string{"fwWETH", "fwUNI"},
			swapToken:      "UNI",
			isTokenIn:      false,
			expectedResult: []string{"WETH", "fwWETH"},
		},
		{
			name:           "should not swap from unrelated token",
			poolTokens:     []string{"WETH", "KNC"},
			swapToken:      "fwWETH",
			isTokenIn:      true,
			expectedResult: []string{},
		},
		{
			name:           "should not swap from unrelated token",
			poolTokens:     []string{"fwWETH", "fwUNI"},
			swapToken:      "KNC",
			isTokenIn:      true,
			expectedResult: []string{},
		},
	}

	for idx, testcase := range testcases {
		if idx != 5 {
			continue
		}
		t.Run(testcase.name, func(t *testing.T) {
			entityPool := entity.Pool{
				Address:  "uniswapV4PoolAddress",
				Exchange: "uniswap-v4",
				Type:     "uniswap-v4",
				Reserves: []string{"1000000000000000000", "1000000000000000000"},
				Tokens: []*entity.PoolToken{
					{
						Address:   tokens[testcase.poolTokens[0]].Address,
						Symbol:    testcase.poolTokens[0],
						Swappable: true,
					},
					{
						Address:   tokens[testcase.poolTokens[1]].Address,
						Symbol:    testcase.poolTokens[1],
						Swappable: true,
					},
				},
				StaticExtra: "{\"0x0\":[true,false],\"fee\":100,\"tS\":1,\"hooks\":\"0x0000000000000000000000000000000000000000\",\"uR\":\"0x66a9893cc07d91d95644aedd05d03f95e1dba8af\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}",
				Extra:       "{\"liquidity\":153714578683118133,\"sqrtPriceX96\":4941858384980104956506206,\"tickSpacing\":1,\"tick\":-193657,\"ticks\":[{\"index\":-887272,\"liquidityGross\":12496692782041,\"liquidityNet\":12496692782041},{\"index\":-207244,\"liquidityGross\":561268407024557,\"liquidityNet\":561268407024557},{\"index\":-197876,\"liquidityGross\":266948242826266,\"liquidityNet\":266948242826266},{\"index\":-197311,\"liquidityGross\":766974383038070,\"liquidityNet\":766974383038070},{\"index\":-196561,\"liquidityGross\":5984254174879,\"liquidityNet\":5984254174879},{\"index\":-194770,\"liquidityGross\":515657539954493,\"liquidityNet\":515657539954493},{\"index\":-194655,\"liquidityGross\":12125701395558971,\"liquidityNet\":12125701395558971},{\"index\":-194461,\"liquidityGross\":5984254174879,\"liquidityNet\":-5984254174879},{\"index\":-193950,\"liquidityGross\":101875313477277975,\"liquidityNet\":101875313477277975},{\"index\":-193752,\"liquidityGross\":37590218544655760,\"liquidityNet\":37590218544655760},{\"index\":-193285,\"liquidityGross\":101875313477277975,\"liquidityNet\":-101875313477277975},{\"index\":-193130,\"liquidityGross\":266948242826266,\"liquidityNet\":-266948242826266},{\"index\":-193012,\"liquidityGross\":37590218544655760,\"liquidityNet\":-37590218544655760},{\"index\":-192892,\"liquidityGross\":766974383038070,\"liquidityNet\":-766974383038070},{\"index\":-192608,\"liquidityGross\":12125701395558971,\"liquidityNet\":-12125701395558971},{\"index\":-192483,\"liquidityGross\":515657539954493,\"liquidityNet\":-515657539954493},{\"index\":-191148,\"liquidityGross\":561268407024557,\"liquidityNet\":-561268407024557},{\"index\":-115136,\"liquidityGross\":462452451821,\"liquidityNet\":462452451821},{\"index\":-92109,\"liquidityGross\":462452451821,\"liquidityNet\":-462452451821},{\"index\":887272,\"liquidityGross\":12496692782041,\"liquidityNet\":-12496692782041}]}",
			}

			simulator, err := NewPoolSimulator(entityPool, valueobject.ChainIDEthereum)
			assert.NoError(t, err)

			var result []string
			if testcase.isTokenIn {
				result = simulator.CanSwapFrom(tokens[testcase.swapToken].Address)
			} else {
				result = simulator.CanSwapTo(tokens[testcase.swapToken].Address)
			}

			resultSymbol := lo.Map(result, func(canSwap string, _ int) string {
				for tokenSymbol, token := range tokens {
					if canSwap == token.Address {
						return tokenSymbol
					}
				}
				return ""
			})

			assert.ElementsMatch(t, testcase.expectedResult, resultSymbol)
		})
	}
}
