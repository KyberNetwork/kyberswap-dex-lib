package wombatlsd

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
				Exchange: "wombat-lsd",
				Type:     "wombat-lsd",
				Tokens:   []string{"0xb31f66aa3c1e785363f0875a1b74e27b85fd66c7", "0x2b2c81e08f1af8835a78bb2a90ae924ace0ea4be"},
				Reserves: []*big.Int{bignumber.NewBig10("547370669405545596073"), bignumber.NewBig10("301255868324403411564")},
				Checked:  false,
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
			name: "swap sfrxETH for frxETH",
			poolEncoded: `{
				"address": "0x3161f40ea6c0c4cc8b2433d6d530ef255816e854",
				"type": "wombat-lsd",
				"timestamp": 1705357248,
				"reserves": [
					"38310717687612156529",
					"60557257422784379622",
					"31480055744644999606"
				],
				"tokens": [
					{
						"address": "0xac3e018457b222d93114458476f3e3416abbe38f",
						"decimals": 18,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0x5e8422345238f34275888049021821e8e08caa1f",
						"decimals": 18,
						"weight": 50,
						"swappable": true
					},
					{
						"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"decimals": 18,
						"weight": 50,
						"swappable": true
					}
				],
				"extra": "{\"paused\":false,\"haircutRate\":100000000000000,\"ampFactor\":2000000000000000,\"startCovRatio\":1500000000000000000,\"endCovRatio\":1800000000000000000,\"assetMap\":{\"0x5e8422345238f34275888049021821e8e08caa1f\":{\"isPause\":false,\"address\":\"0x724515010904518eCF638Cc6d693046B82548068\",\"cash\":60557257422784379622,\"liability\":52162794293656098535,\"underlyingTokenDecimals\":18,\"relativePrice\":1000000000000000000},\"0xac3e018457b222d93114458476f3e3416abbe38f\":{\"isPause\":false,\"address\":\"0x51E073D92b0c226F7B0065909440b18A85769606\",\"cash\":38310717687612156529,\"liability\":34435738368623317194,\"underlyingTokenDecimals\":18,\"relativePrice\":1071887273891919214},\"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2\":{\"isPause\":false,\"address\":\"0xC096FF2606152eD2A06dd12F15A3c0466Aa5A9fa\",\"cash\":31480055744644999606,\"liability\":43968771821191291731,\"underlyingTokenDecimals\":18,\"relativePrice\":1000000000000000000}}}"
			}`,
			tokenIn:  "0xac3e018457b222d93114458476f3e3416abbe38f",
			amountIn: "1000000000000000000", // 1
			tokenOut: "0x5e8422345238f34275888049021821e8e08caa1f",
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
