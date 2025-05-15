package wombatmain

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulatorDeepUpdateBalance(t *testing.T) {
	p := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:  "0xe3abc29b035874a9f6dcdb06f8f20d9975069d87",
				Exchange: "wombat-main",
				Type:     "wombat-main",
				Tokens:   []string{"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be"},
				Reserves: []*big.Int{bignumber.NewBig10("547370669405545596073"), bignumber.NewBig10("301255868324403411564")},
			},
		},
		paused:        false,
		haircutRate:   bignumber.NewBig10("100000000000000"),
		ampFactor:     bignumber.NewBig10("2000000000000000"),
		startCovRatio: bignumber.NewBig10("1500000000000000000"),
		endCovRatio:   bignumber.NewBig10("1800000000000000000"),
		assets: map[string]wombat.Asset{
			"0xA35b1B31Ce002FBF2058D22F30f95D405200A15b": {
				Cash:                    bignumber.NewBig10("547370669405545596073"),
				Liability:               bignumber.NewBig10("516213215951692583758"),
				UnderlyingTokenDecimals: 18,
				RelativePrice:           bignumber.NewBig10("1007193313818254424"),
			},
			"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2": {
				Cash:                    bignumber.NewBig10("301255868324403411564"),
				Liability:               bignumber.NewBig10("332480258276764034667"),
				UnderlyingTokenDecimals: 18,
				RelativePrice:           bignumber.NewBig10("1000000000000000000"),
			},
		},
	}

	params := pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
			Amount: bignumber.NewBig10("1000000000"),
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			Amount: bignumber.NewBig10("1006432688"),
		},
		Fee: pool.TokenAmount{
			Token:  "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
			Amount: bignumber.NewBig10("100653"),
		},
		SwapInfo: wombatSwapInfo{
			newFromAssetCash: bignumber.NewBig10("1100000000"),
			newToAssetCash:   bignumber.NewBig10("1006432688"),
		},
	}

	p.UpdateBalance(params)
	fromAsset1 := p.assets[params.TokenAmountIn.Token]

	params2 := pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
			Amount: bignumber.NewBig10("1000000000"),
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
			Amount: bignumber.NewBig10("1006432688"),
		},
		Fee: pool.TokenAmount{
			Token:  "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b",
			Amount: bignumber.NewBig10("100653"),
		},
		SwapInfo: wombatSwapInfo{
			newFromAssetCash: bignumber.NewBig10("1200000000"),
			newToAssetCash:   bignumber.NewBig10("1006432688"),
		},
	}

	p.UpdateBalance(params2)
	fromAsset2 := p.assets[params.TokenAmountIn.Token]

	assert.NotEqual(t, fromAsset1.Cash.String(), fromAsset2.Cash.String())
}

func TestCalcAmountOutConcurrentSafe(t *testing.T) {
	type testcase struct {
		name        string
		poolEncoded string
		tokenIn     string
		amountIn    string
		tokenOut    string
	}
	testcases := []testcase{
		{
			name: "swap USDC for USDT",
			poolEncoded: `{
				"address": "0xa45c0abeef67c363364e0e73832df9986aba3800",
				"type": "wombat-main",
				"timestamp": 1705358001,
				"reserves": [
					"27437517755",
					"104442256607"
				],
				"tokens": [
					{
						"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
						"decimals": 6,
						"weight": 50,
						"swappable": true
					}
				],
				"extra": "{\"paused\":false,\"haircutRate\":20000000000000,\"ampFactor\":250000000000000,\"startCovRatio\":1500000000000000000,\"endCovRatio\":1800000000000000000,\"assetMap\":{\"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48\":{\"isPause\":false,\"address\":\"0x6966553568634F4225330D559a8783DE7649C7D3\",\"cash\":27437517755009846067811,\"liability\":46154718915224891070477,\"underlyingTokenDecimals\":6,\"relativePrice\":null},\"0xdac17f958d2ee523a2206206994597c13d831ec7\":{\"isPause\":false,\"address\":\"0x752945079a0446AA7efB6e9E1789751cDD601c95\",\"cash\":104442256607693995284288,\"liability\":69617497322874416078864,\"underlyingTokenDecimals\":6,\"relativePrice\":null}}}"
			}`,
			tokenIn:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			amountIn: "1000000000", // 1000
			tokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			poolEntity := new(entity.Pool)
			err := json.Unmarshal([]byte(tc.poolEncoded), poolEntity)
			require.NoError(t, err)

			poolSim, err := NewPoolSimulator(*poolEntity)
			require.NoError(t, err)

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
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
