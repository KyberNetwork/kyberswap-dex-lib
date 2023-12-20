package weighted

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func Test_CalcAmountOut(t *testing.T) {
	t.Run("1. should return OK", func(t *testing.T) {
		// input
		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0xac3E018457B222d93114458476f3E3416Abbe38F",
						"0xae78736Cd615f374D3085123A210448E74Fc6393",
						"0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84",
					},
					Reserves: []*big.Int{
						big.NewInt(331125),
						big.NewInt(320633),
						big.NewInt(348846),
					},
				},
			},

			swapFeePercentage: uint256.NewInt(3000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333300000000000000),
				uint256.NewInt(333400000000000000),
			},
			totalAmountsIn: []*uint256.Int{uint256.NewInt(0), uint256.NewInt(0), uint256.NewInt(0)},
			scaledMaxTotalAmountsIn: []*uint256.Int{
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
			poolTypeVer: 3,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xac3E018457B222d93114458476f3E3416Abbe38F",
			Amount: big.NewInt(3311),
		}
		tokenOut := "0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84"

		// expected
		amountOut := "3442"

		// calculation
		result, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("2. should return OK", func(t *testing.T) {
		// input
		reserve0, _ := new(big.Int).SetString("3360160080014532471350474", 10)
		reserve1, _ := new(big.Int).SetString("1112301324508754708737", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
						"0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
					},
					Reserves: []*big.Int{
						reserve0,
						reserve1,
					},
				},
			},

			swapFeePercentage: uint256.NewInt(1000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(800000000000000000),
				uint256.NewInt(200000000000000000),
			},
			totalAmountsIn: []*uint256.Int{uint256.NewInt(0), uint256.NewInt(0)},
			scaledMaxTotalAmountsIn: []*uint256.Int{
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
			poolTypeVer: 2,
		}

		amountIn, _ := new(big.Int).SetString("60160080014532471350474", 10)
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
			Amount: amountIn,
		}
		tokenOut := "0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223"

		// expected
		amountOut := "76143667376405160244"

		// calculation
		result, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return OK", func(t *testing.T) {
		// input
		reserve0, _ := new(big.Int).SetString("3360160080014532471350474", 10)
		reserve1, _ := new(big.Int).SetString("1112301324508754708737", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Tokens: []string{
						"0x5a8F45b943A7E6a4BEA463A98de68940A153c78a",
						"0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
					},
					Reserves: []*big.Int{
						reserve0,
						reserve1,
					},
				},
			},

			swapFeePercentage: uint256.NewInt(1000000000000000),
			scalingFactors: []*uint256.Int{
				uint256.NewInt(1000000000000000000),
				uint256.NewInt(1000000000000000000),
			},
			normalizedWeights: []*uint256.Int{
				uint256.NewInt(800000000000000000),
				uint256.NewInt(200000000000000000),
			},
			totalAmountsIn: []*uint256.Int{uint256.NewInt(0), uint256.NewInt(0)},
			scaledMaxTotalAmountsIn: []*uint256.Int{
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
				uint256.MustFromDecimal("115792089237316195423570985008687907853269984665640564039457584007913129639935"),
			},
			poolTypeVer: 4,
		}

		amountIn, _ := new(big.Int).SetString("6016008001453247", 10)
		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xbE666bB32a8e4B6b2f2D0fb053d965bdfA277223",
			Amount: amountIn,
		}
		tokenOut := "0x5a8F45b943A7E6a4BEA463A98de68940A153c78a"

		// expected
		amountOut := "4538893010907736440"

		// calculation
		result, err := s.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: tokenAmountIn,
			TokenOut:      tokenOut,
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, amountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("4. should return OK", func(t *testing.T) {
		// input
		// block 18783187
		p := `{
	"address": "0x5c6ee304399dbdb9c8ef030ab642b10820db8f56",
	"reserveUsd": 153314467.24136648,
	"amplifiedTvl": 153314467.24136648,
	"exchange": "balancer-v2-weighted",
	"type": "balancer-v2-weighted",
	"timestamp": 1702542461,
	"reserves": [
		"31686717298564222587034828",
		"14236767788701850247952"
	],
	"tokens": [
		{
			"address": "0xba100000625a3754423978a60c9317c58a424e3d",
			"name": "",
			"symbol": "",
			"decimals": 0,
			"weight": 0,
			"swappable": true
		},
		{
			"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			"name": "",
			"symbol": "",
			"decimals": 0,
			"weight": 0,
			"swappable": true
		}
	],
	"extra": "{\"swapFeePercentage\":\"0x2386f26fc10000\",\"paused\":false}",
	"staticExtra": "{\"poolId\":\"0x5c6ee304399dbdb9c8ef030ab642b10820db8f56000200000000000000000014\",\"poolType\":\"Weighted\",\"poolTypeVer\":1,\"scalingFactors\":[\"0x1\",\"0x1\"],\"normalizedWeights\":[\"0xb1a2bc2ec500000\",\"0x2c68af0bb140000\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "1014934149732776116160723"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("2000000000000000000000", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				Amount: amountIn,
			},
			TokenOut: "0xba100000625a3754423978a60c9317c58a424e3d",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})

	t.Run("5. should return OK", func(t *testing.T) {
		// input
		// polygon, block 51339771
		p := `{
			"address": "0x32fc95287b14eaef3afa92cccc48c285ee3a280a",
			"reserveUsd": 3454.483888331181,
			"amplifiedTvl": 3454.483888331181,
			"exchange": "balancer-v2-weighted",
			"type": "balancer-v2-weighted",
			"timestamp": 1703033832,
			"reserves": [
				"382259350067562080018",
				"563895201975090444069",
				"432276836",
				"415858931425966091248020",
				"198780894165507591",
				"9187067339281421763",
				"111172932376992452571",
				"1835599921140802978251"
			],
			"tokens": [
				{
					"address": "0x0b3f868e0be5597d5db7feb59e1cadbb0fdda50a",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x2791bca1f2de4661ed88a30c99a7a9449aa84174",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x580a84c73811e1839f75d86d75d88cca0c241ff4",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x7ceb23fd6bc0add59e62ac25578270cff1b9f619",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x831753dd7087cac61ab5644b308642cc1c33dc13",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0x9a71012b13ca4d3d0cdc72a177df3ef03b0e76a3",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				},
				{
					"address": "0xc3fdbadc7c795ef1d6ba111e06ff8f16a20ea539",
					"name": "",
					"symbol": "",
					"decimals": 0,
					"weight": 0,
					"swappable": true
				}
			],
			"extra": "{\"swapFeePercentage\":\"0x2386f26fc10000\",\"paused\":false}",
			"staticExtra": "{\"poolId\":\"0x32fc95287b14eaef3afa92cccc48c285ee3a280a000100000000000000000005\",\"poolType\":\"Weighted\",\"poolTypeVer\":1,\"scalingFactors\":[\"0x1\",\"0x1\",\"0xe8d4a51000\",\"0x1\",\"0x1\",\"0x1\",\"0x1\",\"0x1\"],\"normalizedWeights\":[\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\",\"0x1bc16d674ec8000\"],\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
		}`
		var pool entity.Pool
		err := json.Unmarshal([]byte(p), &pool)
		assert.Nil(t, err)

		// expected
		expectedAmountOut := "49523009318781117474536"

		// calculation
		simulator, err := NewPoolSimulator(pool)
		assert.Nil(t, err)
		amountIn, _ := new(big.Int).SetString("77000000000000000000", 10)
		result, err := simulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
			TokenAmountIn: poolpkg.TokenAmount{
				Token:  "0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270",
				Amount: amountIn,
			},
			TokenOut: "0x580a84c73811e1839f75d86d75d88cca0c241ff4",
		})

		// assert
		assert.Nil(t, err)
		assert.Equal(t, expectedAmountOut, result.TokenAmountOut.Amount.String())
	})
}
