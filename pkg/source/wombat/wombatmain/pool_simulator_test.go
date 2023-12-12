package wombatmain

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
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
