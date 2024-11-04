package pancakev3

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPool_CalcAmountOut(t *testing.T) {
	test1AmountInBI, _ := new(big.Int).SetString("399888202451311482718477", 10)
	test2AmountInBI, _ := new(big.Int).SetString("399888202451311482718477", 10)

	tests := []struct {
		name            string
		entityPool      entity.Pool
		chainId         valueobject.ChainID
		tokenAmountIn   pool.TokenAmount
		tokenOut        string
		expectAmountOut *big.Int
		expectedErr     error
	}{
		{
			name: "it should return correct amount out OPENAI -> WBNB via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountIn: pool.TokenAmount{
				Token:     "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
				Amount:    test1AmountInBI,
				AmountUsd: 0,
			},
			tokenOut:        "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
			expectAmountOut: big.NewInt(10997482374483461),
			expectedErr:     nil,
		},
		{
			name: "it should return correct amount out WBNB -> OPENAI via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountIn: pool.TokenAmount{
				Token:     "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
				Amount:    test2AmountInBI,
				AmountUsd: 0,
			},
			tokenOut:        "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
			expectAmountOut: big.NewInt(90929739),
			expectedErr:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewPoolSimulatorBigInt(tt.entityPool, tt.chainId)
			calcAmountOutResult, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
				return p.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: tt.tokenAmountIn,
					TokenOut:      tt.tokenOut,
					Limit:         nil,
				})
			})

			assert.Equal(t, tt.expectAmountOut, calcAmountOutResult.TokenAmountOut.Amount)
			assert.Equal(t, big.NewInt(0), calcAmountOutResult.RemainingTokenAmountIn.Amount)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	test1AmountOutBI, _ := new(big.Int).SetString("399888202451311482718477", 10)
	test2AmountOutBI, _ := new(big.Int).SetString("399888202451311482718477", 10)
	test1ExpectedAmountInBI, _ := new(big.Int).SetString("18471935825790437235583368059210", 10)
	test2ExpectedAmountInBI, _ := new(big.Int).SetString("18471935824815090321109555072782", 10)
	test1ExpectedRemainingAmountOutBI, _ := new(big.Int).SetString("-399888202451311391788735", 10)
	test2ExpectedRemainingAmountOutBI, _ := new(big.Int).SetString("-399888191453829108235014", 10)

	tests := []struct {
		name                       string
		entityPool                 entity.Pool
		chainId                    valueobject.ChainID
		tokenAmountOut             pool.TokenAmount
		tokenIn                    string
		expectAmountIn             *big.Int
		expectedRemainingAmountOut *big.Int
		expectedErr                error
	}{
		{
			name: "it should return correct amount out OPENAI -> WBNB via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountOut: pool.TokenAmount{
				Token:     "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
				Amount:    test1AmountOutBI,
				AmountUsd: 0,
			},
			tokenIn:                    "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
			expectAmountIn:             test1ExpectedAmountInBI,
			expectedRemainingAmountOut: test1ExpectedRemainingAmountOutBI,
			expectedErr:                nil,
		},
		{
			name: "it should return correct amount out WBNB -> OPENAI via pool 0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
			entityPool: entity.Pool{
				Address:   "0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184",
				SwapFee:   2500,
				Exchange:  "pancake-v3",
				Type:      "pancake-v3",
				Timestamp: 1689072352,
				Reserves: entity.PoolReserves{
					"90929743",
					"10999982374483464",
				},
				Tokens: entity.PoolTokens{
					{
						Address:   "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
						Name:      "OPENAI",
						Symbol:    "CGPT",
						Decimals:  4,
						Weight:    50,
						Swappable: true,
					},
					{
						Address:   "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
						Name:      "Wrapped BNB",
						Symbol:    "WBNB",
						Decimals:  18,
						Weight:    50,
						Swappable: true,
					},
				},
				Extra:       "{\"liquidity\":999999118723,\"sqrtPriceX96\":871311088679755827947222956518526,\"tick\":186117,\"ticks\":[{\"index\":-887250,\"liquidityGross\":999999118723,\"liquidityNet\":999999118723},{\"index\":887250,\"liquidityGross\":999999118723,\"liquidityNet\":-999999118723}]}",
				StaticExtra: "{\"poolId\":\"0xe65fddb2b65451d73b6240e0e2b0cb34df0d9184\"}",
			},
			chainId: valueobject.ChainIDBSC,
			tokenAmountOut: pool.TokenAmount{
				Token:     "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
				Amount:    test2AmountOutBI,
				AmountUsd: 0,
			},
			tokenIn:                    "0x2c30f4bdb0191b82b5e57c629a5021f96f7375d8",
			expectAmountIn:             test2ExpectedAmountInBI,
			expectedRemainingAmountOut: test2ExpectedRemainingAmountOutBI,
			expectedErr:                nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, _ := NewPoolSimulatorBigInt(tt.entityPool, tt.chainId)
			calcAmountOutResult, err := testutil.MustConcurrentSafe[*pool.CalcAmountInResult](t, func() (any, error) {
				return p.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: tt.tokenAmountOut,
					TokenIn:        tt.tokenIn,
					Limit:          nil,
				})
			})

			assert.Equal(t, tt.expectAmountIn, calcAmountOutResult.TokenAmountIn.Amount)
			assert.Equal(t, tt.expectedRemainingAmountOut, calcAmountOutResult.RemainingTokenAmountOut.Amount)
			assert.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
