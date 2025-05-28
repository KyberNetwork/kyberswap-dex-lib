package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	t.Run("1. should return error balance didnt converge", func(t *testing.T) {
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000000", 10)
		reserves[1], _ = new(big.Int).SetString("99999910000000000056", 10)
		reserves[2], _ = new(big.Int).SetString("8897791020011100123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(50000000000000),
			amp:               uint256.NewInt(5000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1), uint256.NewInt(100)},

			poolType:    poolTypeStable,
			poolTypeVer: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: new(big.Int).SetUint64(99999910000000),
		}
		tokenOut := "0x6b175474e89094c44da98b954eedeac495271d0f"
		_, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.ErrorIs(t, err, math.ErrStableGetBalanceDidntConverge)
	})

	t.Run("2. should return OK", func(t *testing.T) {
		// input
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000000000", 10)
		reserves[1], _ = new(big.Int).SetString("9999991000000000005613", 10)
		reserves[2], _ = new(big.Int).SetString("13288977911102200123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(50000000000000),
			amp:               uint256.NewInt(1390000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1), uint256.NewInt(100)},

			poolType:    poolTypeStable,
			poolTypeVer: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0x6b175474e89094c44da98b954eedeac495271d0f",
			Amount: new(big.Int).SetUint64(12000000000000000000),
		}
		tokenOut := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"

		// expected
		expected := "1000000000000000000"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.TokenAmountOut.Amount.String())
	})

	t.Run("3. should return OK", func(t *testing.T) {
		// input
		reserves := make([]*big.Int, 3)
		reserves[0], _ = new(big.Int).SetString("9999991000000000013314124321", 10)
		reserves[1], _ = new(big.Int).SetString("9999991000000123120010005613", 10)
		reserves[2], _ = new(big.Int).SetString("1328897131447911102200123456", 10)

		s := PoolSimulator{
			Pool: poolpkg.Pool{
				Info: poolpkg.PoolInfo{
					Reserves: reserves,
					Tokens: []string{
						"0xdac17f958d2ee523a2206206994597c13d831ec7",
						"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
						"0x6b175474e89094c44da98b954eedeac495271d0f",
					},
				},
			},
			swapFeePercentage: uint256.NewInt(53332221119995),
			amp:               uint256.NewInt(1390000),
			scalingFactors:    []*uint256.Int{uint256.NewInt(100), uint256.NewInt(1000), uint256.NewInt(100)},

			poolType:    poolTypeStable,
			poolTypeVer: 1,
		}

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			Amount: new(big.Int).SetUint64(12111222333444555666),
		}
		tokenOut := "0x6b175474e89094c44da98b954eedeac495271d0f"

		// expected
		expected := "590000000000000000"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.TokenAmountOut.Amount.String())
	})

	t.Run("4. should return OK", func(t *testing.T) {
		poolStr := `{
			"address": "0x851523a36690bf267bbfec389c823072d82921a9",
			"exchange": "balancer-v2-stable",
			"type": "balancer-v2-stable",
			"timestamp": 1703667290,
			"reserves": [
			  "1152882153159026494",
			  "873225053252443292"
			],
			"tokens": [
			  {
				"address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
				"name": "",
				"symbol": "",
				"decimals": 0,
				"weight": 1,
				"swappable": true
			  },
			  {
				"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"name": "",
				"symbol": "",
				"decimals": 0,
				"weight": 1,
				"swappable": true
			  }
			],
			"extra": "{\"amp\":\"0xf4240\",\"swapFeePercentage\":\"0x16bcc41e90000\",\"scalingFactors\":[\"0xFFB10F9BCF7D41A\",\"0xde0b6b3a7640000\"],\"paused\":false}",
			"staticExtra": "{\"poolId\":\"0x851523a36690bf267bbfec389c823072d82921a90002000000000000000001ed\",\"poolType\":\"MetaStable\",\"poolTypeVersion\":1,\"poolSpecialization\":2,\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
		  }`
		var pool entity.Pool
		err := json.Unmarshal([]byte(poolStr), &pool)
		assert.Nil(t, err)

		s, err := NewPoolSimulator(pool, nil)
		assert.Nil(t, err)

		tokenAmountIn := poolpkg.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: big.NewInt(73183418984294781),
		}
		tokenOut := "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"

		// expected
		expected := "63551050657042642"

		// actual
		result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
			return s.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: tokenAmountIn,
				TokenOut:      tokenOut,
			})
		})

		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.TokenAmountOut.Amount.String())

	})
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	t.Parallel()
	type fields struct {
		poolStr string
	}

	tests := []struct {
		name    string
		fields  fields
		params  poolpkg.CalcAmountInParams
		want    *poolpkg.CalcAmountInResult
		wantErr error
	}{
		{
			name: "1. should return error ErrStableGetBalanceDidntConverge",
			fields: fields{
				poolStr: `{
					"address": "0x851523a36690bf267bbfec389c823072d82921a9",
					"exchange": "balancer-v2-stable",
					"type": "balancer-v2-stable",
					"timestamp": 1703667290,
					"reserves": [
					  "9999991000000000000",
					  "99999910000000000056",
					  "8897791020011100123456"
					],
					"tokens": [
						{
							"address": "0xdac17f958d2ee523a2206206994597c13d831ec7",
							"name": "",
							"symbol": "",
							"decimals": 0,
							"weight": 1,
							"swappable": true
						},
						{
							"address": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"name": "",
							"symbol": "",
							"decimals": 0,
							"weight": 1,
							"swappable": true
						},
						{
							"address": "0x6b175474e89094c44da98b954eedeac495271d0f",
							"name": "",
							"symbol": "",
							"decimals": 0,
							"weight": 1,
							"swappable": true
						}
					],
					"extra": "{\"amp\":\"0x1388\",\"swapFeePercentage\":\"0x2D79883D2000\",\"scalingFactors\":[\"100\",\"1\",\"100\"],\"paused\":false}",
					"staticExtra": "{\"poolId\":\"0x851523a36690bf267bbfec389c823072d82921a90002000000000000000001ed\",\"poolType\":\"Stable\",\"poolTypeVersion\":1,\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
					}`,
			},
			params: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
					Amount: big.NewInt(999999100000),
				},
				TokenIn: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			want:    nil,
			wantErr: math.ErrStableGetBalanceDidntConverge,
		},
		{
			name: "2. should return OK",
			fields: fields{
				poolStr: `{
					"address": "0x851523a36690bf267bbfec389c823072d82921a9",
					"exchange": "balancer-v2-stable",
					"type": "balancer-v2-stable",
					"timestamp": 1703667290,
					"reserves": [
					  "1152882153159026494",
					  "873225053252443292"
					],
					"tokens": [
					  {
						"address": "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
						"name": "",
						"symbol": "",
						"decimals": 0,
						"weight": 1,
						"swappable": true
					  },
					  {
						"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						"name": "",
						"symbol": "",
						"decimals": 0,
						"weight": 1,
						"swappable": true
					  }
					],
					"extra": "{\"amp\":\"0xf4240\",\"swapFeePercentage\":\"0x16bcc41e90000\",\"scalingFactors\":[\"0xFFB10F9BCF7D41A\",\"0xde0b6b3a7640000\"],\"paused\":false}",
					"staticExtra": "{\"poolId\":\"0x851523a36690bf267bbfec389c823072d82921a90002000000000000000001ed\",\"poolType\":\"MetaStable\",\"poolTypeVersion\":1,\"poolSpecialization\":2,\"vault\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\"}"
					}`,
			},
			params: poolpkg.CalcAmountInParams{
				TokenAmountOut: poolpkg.TokenAmount{
					Token:  "0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0",
					Amount: big.NewInt(63551050657042642),
				},
				TokenIn: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			},
			want: &poolpkg.CalcAmountInResult{
				TokenAmountIn: &poolpkg.TokenAmount{
					Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
					Amount: big.NewInt(73154145616700748),
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pool entity.Pool
			err := json.Unmarshal([]byte(tt.fields.poolStr), &pool)
			assert.Nil(t, err)

			simulator, err := NewPoolSimulator(pool, nil)
			assert.Nil(t, err)

			got, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountInResult, error) {
				return simulator.CalcAmountIn(tt.params)
			})
			if err != nil {
				assert.ErrorIsf(t, err, tt.wantErr, "PoolSimulator.CalcAmountIn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equalf(t, tt.want.TokenAmountIn.Token, got.TokenAmountIn.Token, "tokenIn = %v, want %v", got.TokenAmountIn.Token, tt.want.TokenAmountIn.Token)
			assert.Equalf(t, tt.want.TokenAmountIn.Amount, got.TokenAmountIn.Amount, "amountIn = %v, want %v", got.TokenAmountIn.Amount.String(), tt.want.TokenAmountIn.Amount.String())
		})
	}
}

func TestCanSwapTo(t *testing.T) {
	t.Parallel()
	// Setup base pools
	basePool1 := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address: "pool1",
				Tokens:  []string{"pool1", "ETH", "USDT"},
			},
		},
	}

	basePool2 := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address: "pool2",
				Tokens:  []string{"pool2", "BTC", "USDT"},
			},
		},
	}

	// Setup main pool
	pool := &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address: "main_pool",
				Tokens:  []string{"pool1", "pool2", "USDC"},
			},
		},
		basePools: map[string]shared.IBasePool{
			"pool1": basePool1,
			"pool2": basePool2,
		},
	}

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Token exists in the main pool",
			input:    "USDC",
			expected: []string{"pool1", "pool2", "ETH", "BTC", "USDT"},
		},
		{
			name:     "Token exists in base pool 1",
			input:    "ETH",
			expected: []string{"pool1", "pool2", "BTC", "USDT", "USDC"},
		},
		{
			name:     "Token exists in multiple pools",
			input:    "USDT",
			expected: []string{"pool1", "pool2", "ETH", "BTC", "USDC"},
		},
		{
			name:     "Token does not exist in any pool",
			input:    "KNC",
			expected: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := pool.CanSwapTo(tc.input)
			assert.ElementsMatch(t, tc.expected, result)
		})
	}
}
