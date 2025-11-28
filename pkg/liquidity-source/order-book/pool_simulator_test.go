package orderbook

import (
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var entityPool = entity.Pool{
	Address:   "mx_trading_0xaf88d065e77c8cc2239327c5edb3a432268e5831_0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
	Exchange:  "pmm-1",
	Type:      "pmm-1",
	Timestamp: time.Now().Unix(),
	Reserves:  []string{"364190205979", "0"},
	Tokens: []*entity.PoolToken{
		{Address: "0xaf88d065e77c8cc2239327c5edb3a432268e5831", Decimals: 6, Swappable: true},
		{Address: "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", Decimals: 6, Swappable: true},
	},

	Extra: "{\"l\":[null,[[0,0.9991504223059631],[500.2249999999999,0.9991504223059631],[3501.5749999999994,0.9991504223059631],[9504.274999999998,0.9991504223059628],[18508.324999999997,0.9991504223059632],[30513.725,0.9990275390775962],[45520.474999999984,0.9988769463564154],[63528.57499999998,0.998190713968263],[84538.025,0.9987006696901404],[108548.82500000001,0.9987006696901407]]]}",
}

func TestPoolSimulator_NewPool(t *testing.T) {
	poolSimulator, err := NewPoolSimulator(entityPool)
	assert.NoError(t, err)
	assert.Equal(t, "0xaf88d065e77c8cc2239327c5edb3a432268e5831", poolSimulator.tokens[0].Address)
	assert.Equal(t, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", poolSimulator.tokens[1].Address)
	assert.Nil(t, poolSimulator.levelsFroms[0])
	assert.NotNil(t, poolSimulator.levelsFroms[1])
	assert.Equal(t, []string{"0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
		poolSimulator.CanSwapTo("0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"))
	assert.Equal(t, []string{"0xaf88d065e77c8cc2239327c5edb3a432268e5831"},
		poolSimulator.CanSwapFrom("0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"))
	assert.Equal(t, []string{"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"},
		poolSimulator.CanSwapTo("0xaf88d065e77c8cc2239327c5edb3a432268e5831"))
	assert.Equal(t, []string{"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9"},
		poolSimulator.CanSwapFrom("0xaf88d065e77c8cc2239327c5edb3a432268e5831"))
}

func TestPoolSimulator_GetAmountOut(t *testing.T) {
	// Create entity pool data for testing
	entityPoolStrData := `{
		"address": "mx_trading_0xaf88d065e77c8cc2239327c5edb3a432268e5831_0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9",
		"exchange": "pmm-1",
		"type": "pmm",
		"timestamp": 1999999999,
		"reserves": ["364190205979", "0"],
		"tokens": [
			{"address": "0xaf88d065e77c8cc2239327c5edb3a432268e5831", "decimals": 6, "swappable": true},
			{"address": "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", "decimals": 6, "swappable": true}
		],
		"extra": "{\"l\":[null,[[0,0.9991504223059631],[500.2249999999999,0.9991504223059631],[3501.5749999999994,0.9991504223059631],[9504.274999999998,0.9991504223059628],[18508.324999999997,0.9991504223059632],[30513.725,0.9990275390775962],[45520.474999999984,0.9988769463564154],[63528.57499999998,0.998190713968263],[84538.025,0.9987006696901404],[108548.82500000001,0.9987006696901407]]]}"
	}`

	token1 := "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9" // USDT
	token0 := "0xaf88d065e77c8cc2239327c5edb3a432268e5831" // USDC

	tests := []struct {
		name              string
		tokenIn, tokenOut string
		amountIn          *big.Int
		expectedAmountOut *big.Int
		expectedErr       error
	}{
		{
			name:              "swap USDT to USDC with small amount",
			tokenIn:           token1,
			tokenOut:          token0,
			amountIn:          bignumber.NewBig("1000000"), // 1 USDT
			expectedAmountOut: bignumber.NewBig("999150"),
		},
		{
			name:              "swap USDT to USDC with medium amount",
			tokenIn:           token1,
			tokenOut:          token0,
			amountIn:          bignumber.NewBig("10000000000"), // 10,000 USDT
			expectedAmountOut: bignumber.NewBig("9991504223"),
		},
		{
			name:              "swap USDT to USDC with large amount crossing multiple levels",
			tokenIn:           token1,
			tokenOut:          token0,
			amountIn:          bignumber.NewBig("50000000000"), // 50,000 USDT
			expectedAmountOut: bignumber.NewBig("49955310986"),
		},
		{
			name:              "swap USDT to USDC with amount crossing all levels",
			tokenIn:           token1,
			tokenOut:          token0,
			amountIn:          bignumber.NewBig("108548825000"), // 108,548.825 USDT (max from price levels)
			expectedAmountOut: bignumber.NewBig("108439925889"),
		},
		{
			name:        "it should return error when USDT amount exceeds available levels",
			tokenIn:     token1,
			tokenOut:    token0,
			amountIn:    bignumber.NewBig("364665000000"), // Just over the max amount
			expectedErr: ErrInsufficientLiquidity,
		},
		{
			name:        "it should return error when trying to swap from USDC to USDT",
			tokenIn:     token0,
			tokenOut:    token1,
			amountIn:    bignumber.NewBig("1000000"), // 1 USDC
			expectedErr: ErrEmptyLevels,              // Since 0to1 is null in the data
		},
		{
			name:              "it should respect swap limit",
			tokenIn:           token1,
			tokenOut:          token0,
			amountIn:          bignumber.NewBig("10000000000"), // 10,000 USDT
			expectedAmountOut: bignumber.NewBig("9991504223"),
		},
		{
			name:     "it should proceed when no swap limit provided",
			tokenIn:  token1,
			tokenOut: token0,
			amountIn: bignumber.NewBig("1000000"), // 1 USDT
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entityPool := entity.Pool{}
			err := json.Unmarshal([]byte(entityPoolStrData), &entityPool)
			require.NoError(t, err)

			poolSimulator, err := NewPoolSimulator(entityPool)
			require.NoError(t, err)

			var limit pool.SwapLimit
			if !errors.Is(tt.expectedErr, ErrNoSwapLimit) {
				limit = swaplimit.NewInventory("pmm-1", poolSimulator.CalculateLimit())
			}

			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tt.tokenIn, Amount: tt.amountIn},
				TokenOut:      tt.tokenOut,
				Limit:         limit,
			}

			result, err := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return poolSimulator.CalcAmountOut(params)
			})

			if tt.expectedErr != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedErr)
				} else if !strings.Contains(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("expected error to contain %v, got %v", tt.expectedErr, err)
				}
			} else {
				assert.NoError(t, err)
				if result != nil && tt.expectedAmountOut != nil {
					if tt.expectedAmountOut.Cmp(result.TokenAmountOut.Amount) != 0 {
						t.Errorf("Expected amount %s, got %s",
							tt.expectedAmountOut.String(),
							result.TokenAmountOut.Amount.String())
					}
				}
			}
		})
	}
}

func TestPoolSimulator_MultiSwap(t *testing.T) {
	entityPoolStrData := `{
		"address": "onebit_0x6982508145454ce325ddbe47a25d4ec3d2311933_0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		"reserveUsd": 180782.69169272823,
		"amplifiedTvl": 180782.69169272823,
		"exchange": "pmm-1",
		"type": "pmm-1",
		"timestamp": 1999999999,
		"reserves": [
			"19754164950060196220612116480",
			"0"
		],
		"tokens": [
			{
				"address": "0x6982508145454ce325ddbe47a25d4ec3d2311933",
				"symbol": "PEPE",
				"decimals": 18,
				"swappable": true
			},
			{
				"address": "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
				"symbol": "WETH",
				"decimals": 18,
				"swappable": true
			}
		],
		"extra": "{\"l\":[null,[[0,195570007.72401568],[0.15,195570007.72401568],[1.05,195570007.72401568],[2.8499999999999996,195570007.72401568],[5.55,195851142.38025662],[9.15,195457867.24975795],[13.649999999999999,195507959.31338486],[19.050000000000004,195350539.92769447],[25.349999999999994,195221825.9462004],[24.355532871620042,194933463.3847393]]]}"
	}`

	token0 := "0x6982508145454ce325ddbe47a25d4ec3d2311933" // PEPE
	token1 := "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" // WETH

	tests := []struct {
		name          string
		tokenIn       string
		tokenOut      string
		amountPerSwap *big.Int
		numberOfSwaps int
		expectedErr   error
	}{
		{
			name:          "swap 0.1 WETH x5",
			tokenIn:       token1,
			tokenOut:      token0,
			amountPerSwap: bignumber.NewBig("100000000000000000"), // 0.1 ETH
			numberOfSwaps: 5,
		},
		{
			name:          "swap 1 WETH x5",
			tokenIn:       token1,
			tokenOut:      token0,
			amountPerSwap: bignumber.NewBig("1000000000000000000"), // 1 ETH
			numberOfSwaps: 5,
		},
		{
			name:          "swap 21 WETH x5",
			tokenIn:       token1,
			tokenOut:      token0,
			amountPerSwap: bignumber.NewBig("21000000000000000000"), // 105 ETH
			numberOfSwaps: 5,
			expectedErr:   ErrInsufficientLiquidity, // maximum is ~ 101,15 ETH
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var entityPool entity.Pool
			require.NoError(t, json.Unmarshal([]byte(entityPoolStrData), &entityPool))

			sim, err := NewPoolSimulator(entityPool)
			require.NoError(t, err)

			limit := swaplimit.NewInventory("pmm-1", sim.CalculateLimit())

			var errSequential error
			for range tt.numberOfSwaps {
				params := pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: tt.tokenIn, Amount: tt.amountPerSwap},
					TokenOut:      tt.tokenOut,
					Limit:         limit,
				}

				_, errSequential = testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
					return sim.CalcAmountOut(params)
				})
				if errSequential != nil {
					break
				}

				sim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  params.TokenAmountIn,
					TokenAmountOut: pool.TokenAmount{},
					Fee:            pool.TokenAmount{},
					SwapInfo:       nil,
					SwapLimit:      limit,
				})
			}

			var entityPool2 entity.Pool
			require.NoError(t, json.Unmarshal([]byte(entityPoolStrData), &entityPool2))

			sim2, err := NewPoolSimulator(entityPool2)
			require.NoError(t, err)

			limit2 := swaplimit.NewInventory("pmm-1", sim2.CalculateLimit())

			totalAmount := new(big.Int).Mul(tt.amountPerSwap, big.NewInt(int64(tt.numberOfSwaps)))

			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tt.tokenIn, Amount: totalAmount},
				TokenOut:      tt.tokenOut,
				Limit:         limit2,
			}

			_, errBatch := testutil.MustConcurrentSafe(t, func() (*pool.CalcAmountOutResult, error) {
				return sim2.CalcAmountOut(params)
			})

			if tt.expectedErr != nil {
				assert.Error(t, errSequential)
				assert.Equal(t, tt.expectedErr, errSequential)
				assert.Error(t, errBatch)
				assert.Equal(t, tt.expectedErr, errBatch)
			} else {
				assert.NoError(t, errSequential)
				assert.NoError(t, errBatch)
			}
		})
	}
}
