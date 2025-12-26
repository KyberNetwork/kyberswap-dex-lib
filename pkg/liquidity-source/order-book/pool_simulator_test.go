package orderbook

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
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

var (
	eP2      entity.Pool
	_        = json.Unmarshal([]byte(`{"address":"0xf39c4fd5465ea2dd7b0756cebc48a258b34febf3","swapFee":0.0002,"exchange":"kuru-ob","type":"kuru-ob","reserves":["290848909497816279616061440","12392311602"],"tokens":[{"address":"0x3bd359c1119da7da1d913d1c4d2b7c461115433a","symbol":"WMON","decimals":18,"swappable":true},{"address":"0x00000000efe302beaa2b3e6e1b18d08d69a9012a","symbol":"AUSD","decimals":6,"swappable":true}],"extra":"{\"l\":[[[0,0],[592465.1388602164,0.02014],[218.9592034068,0.01996],[779.96323738151,0.01984],[3847.00809459866,0.01982],[221.06351542741,0.01977],[2084.55080406154,0.01976],[817.7459365594,0.01965],[4033.55775739468,0.01963],[223.20866700714,0.01958],[1678.07296371997,0.01957],[810.51495190528,0.01946],[225.39585869004,0.01939],[787.60143100702,0.01927],[227.62634375,0.0192],[200,0.01912],[780.06178197062,0.01908],[229.90140452393,0.01901],[441.17062962962,0.0189],[232.09908125331,0.01883],[445.41265491452,0.01872],[234.33917962465,0.01865],[449.73704962243,0.01854],[236.62293990253,0.01847],[454.14623638343,0.01836],[4546.73088365969,0.01683],[198.38474860762,0.001]],[[0,0],[80929.67891705604,49.164208456243855],[13650.013650013268,49.14004914004914],[186811.30846659088,48.99559039686428],[37097.570818009786,48.94762604013705],[444483.23210786714,48.685491723466406],[24071.647370323604,48.661800486618006],[183096.682209099,48.5201358563804],[56782.04209180998,48.47309743092583],[439423.9319777999,48.216007714561236],[13386.880856759999,48.19277108433735],[359397.5385006559,48.05382027871216],[56372.128201314925,48.00768122899664],[358123.4601157736,47.75549188156638],[13259.0824714926,47.7326968973747],[19986.81590685639,47.70992366412214],[355867.00507698715,47.596382674916704],[35219.60333953685,47.551117451260104],[354735.3479103264,47.30368968779565],[13133.70107696312,47.28132387706856],[54750.67808547356,47.214353163361665],[827797.4125947672,47.1253534401508],[50063.50822079331,47.080979284369114],[13004.57761131882,46.81647940074907],[19971.730316600377,46.79457182966776],[53916.840987715754,46.75081813931744],[814225.4546161676,46.66355576294914],[48780.98065971656,46.62004662004662],[313066.8742681206,46.3821892393321],[12877.968371709318,46.36068613815485],[52792.09311450231,46.2962962962963],[799144.1148715905,46.21072088724584],[32381.991802269626,46.16805170821792]]]}","staticExtra":"{\"p\":7,\"s\":11,\"n\":true}","blockNumber":42613084}`), &eP2)
	poolSim2 = lo.Must(NewPoolSimulatorWith(eP2, math.MaxInt64))
)

func TestCalcAmountIn_WithFee(t *testing.T) {
	testutil.TestCalcAmountIn(t, poolSim2)
}
