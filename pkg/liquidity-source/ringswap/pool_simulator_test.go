package ringswap

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	uniswapv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v2"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		swapLimit         poolpkg.SwapLimit
		tokenAmountIn     poolpkg.TokenAmount
		tokenOut          string
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576"), big.NewInt(1), big.NewInt(1)},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("10089138480746"),
					Reserve1: utils.NewBig("10066716097576"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("10089138480746"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("10066716097576"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("125224746"),
				Token:  "0x25f233c3e3676f9e900a89644a3fe5404d643c84",
			},
			tokenOut:          "0x4300000000000000000000000000000000000004",
			expectedAmountOut: utils.NewBig("124570062"),
			expectedError:     nil,
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005"), big.NewInt(1), big.NewInt(1)},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("70361282326226590645832"),
					Reserve1: utils.NewBig("54150601005"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("54150601005"),
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("70361282326226590645832"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("124570062"),
				Token:  "0x4300000000000000000000000000000000000004",
			},
			tokenOut:          "0x25f233c3e3676f9e900a89644a3fe5404d643c84",
			expectedAmountOut: utils.NewBig("161006857684289764421"),
			expectedError:     nil,
		},
		{
			name: "[swapWrapped0toOriginal1] it should return correct amountOut when swapping from wrapped token 0 to original token 1",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("5000000000"),
					Reserve1: utils.NewBig("3000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("5000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("3000000000"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("5000000"),
				Token:  "0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
			},
			tokenOut:          "0x4300000000000000000000000000000000000004", // Original token 1
			expectedAmountOut: utils.NewBig("2988020"),
			expectedError:     nil,
		},
		{
			name: "[swapOriginal0toWrapped1] it should return correct amountOut when swapping from original token 0 to wrapped token 1",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("5000000000"),
					Reserve1: utils.NewBig("3000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("5000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("3000000000"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("2000000"),
				Token:  "0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
			},
			tokenOut:          "0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
			expectedAmountOut: utils.NewBig("1195923"),
			expectedError:     nil,
		},
		{
			name: "[swapWrapped0toOriginal0] it should return an error when trying to swap from wrapped token 0 to original token 0",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("5000000000"),
					Reserve1: utils.NewBig("3000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("5000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("3000000000"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("5000000"),
				Token:  "0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
			},
			tokenOut:          "0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
			expectedAmountOut: nil,
			expectedError:     ErrTokenSwapNotAllowed,
		},
		{
			name: "[swapOriginal0toWrapped0] it should return an error when trying to swap from original token 0 to wrapped token 0",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("5000000000"),
					Reserve1: utils.NewBig("3000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("5000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("3000000000"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("1000000"),
				Token:  "0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
			},
			tokenOut:          "0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
			expectedAmountOut: nil,
			expectedError:     ErrTokenSwapNotAllowed,
		},
		{
			name: "[swap0to1] it should return an error when amountOut exceed the original reserve",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("5000000000"),
					Reserve1: utils.NewBig("1000000"), // smaller than expected amountOut (1195923)
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("5000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("1000000"),
			}),
			tokenAmountIn: poolpkg.TokenAmount{
				Amount: utils.NewBig("2000000"),
				Token:  "0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
			},
			tokenOut:          "0x4300000000000000000000000000000000000004", // Original token 1
			expectedAmountOut: utils.NewBig("1195923"),
			expectedError:     uniswapv2.ErrInsufficientLiquidity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := testutil.MustConcurrentSafe[*poolpkg.CalcAmountOutResult](t, func() (any, error) {
				return tc.poolSimulator.CalcAmountOut(poolpkg.CalcAmountOutParams{
					TokenAmountIn: tc.tokenAmountIn,
					TokenOut:      tc.tokenOut,
					Limit:         tc.swapLimit,
				})
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountOut, result.TokenAmountOut.Amount)
			}
		})
	}
}

func TestPoolSimulator_CalcAmountIn(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		swapLimit        poolpkg.SwapLimit
		tokenAmountOut   poolpkg.TokenAmount
		tokenIn          string
		expectedAmountIn *big.Int
		expectedError    error
	}{
		{
			name: "[swap0to1] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{
							"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7",
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("100000000"), utils.NewBig("100000000"), utils.One, utils.One},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("100000000"),
					Reserve1: utils.NewBig("100000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("100000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("100000000"),
			}),
			tokenAmountOut: poolpkg.TokenAmount{
				Amount: utils.NewBig("20000000"),
				Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			tokenIn:          "0xdac17f958d2ee523a2206206994597c13d831ec7",
			expectedAmountIn: utils.NewBig("25075226"),
			expectedError:    nil,
		},
		{
			name: "[swap1to0] it should return correct amountIn and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{
							"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7",
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("100000000000000000000"), utils.NewBig("100000000000000000000"), utils.One, utils.One},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("100000000000000000000"),
					Reserve1: utils.NewBig("100000000000000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("100000000000000000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("100000000000000000000"),
			}),
			tokenAmountOut: poolpkg.TokenAmount{
				Amount: utils.NewBig("20000000"),
				Token:  "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
			tokenIn:          "0x32a7c02e79c4ea1008dd6564b35f131428673c41",
			expectedAmountIn: utils.NewBig("20060181"),
			expectedError:    nil,
		},
		{
			name: "[swap0to1] it should return an error when amountOut exceed the original reserve",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address:  "0x1234567890abcdef1234567890abcdef12345678",
						Reserves: []*big.Int{utils.NewBig("5000000000"), utils.NewBig("3000000000"), big.NewInt(1), big.NewInt(1)},
						Tokens: []string{
							"0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
							"0x4300000000000000000000000000000000000004", // Original token 1
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("1000000"), // smaller than expected amountOut (2000000)
					Reserve1: utils.NewBig("5000000000"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			swapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
				"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("1000000"),
				"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("5000000000"),
			}),
			tokenAmountOut: poolpkg.TokenAmount{
				Amount: utils.NewBig("2000000"),
				Token:  "0x25f233c3e3676f9e900a89644a3fe5404d643c84", // Original token 0
			},
			tokenIn:          "0x4300000000000000000000000000000000000004", // Original token 1
			expectedAmountIn: utils.NewBig("1195923"),
			expectedError:    uniswapv2.ErrInsufficientLiquidity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountIn(poolpkg.CalcAmountInParams{
				TokenAmountOut: tc.tokenAmountOut,
				TokenIn:        tc.tokenIn,
				Limit:          tc.swapLimit,
			})

			if tc.expectedError != nil {
				assert.ErrorIs(t, tc.expectedError, err)
			} else {
				assert.Equal(t, tc.expectedAmountIn, result.TokenAmountIn.Amount)
			}
		})
	}
}

func TestPoolSimulator_UpdateBalance(t *testing.T) {
	testCases := []struct {
		name                     string
		poolSimulator            PoolSimulator
		params                   poolpkg.UpdateBalanceParams
		expectedFwReserves       [4]*big.Int
		expectedOriginalReserves uniswapv2.ReserveData
	}{
		{
			name: "[swap0to1] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x3041cbd36888becc7bbcbc0045e3b1f144466f5f",
						Tokens: []string{
							"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
							"0xdac17f958d2ee523a2206206994597c13d831ec7",
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("10089138480746"), utils.NewBig("10066716097576"), utils.One, utils.One},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("10089138480746"),
					Reserve1: utils.NewBig("10066716097576"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: poolpkg.UpdateBalanceParams{
				SwapInfo: SwapInfo{
					IsWrapIn:    true,
					IsUnwrapOut: true,
					WTokenIn:    "0x18755d2cec785ab87680edb8e117615e4b005430",
					WTokenOut:   "0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1",
				},
				SwapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
					"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("10089138480746"),
					"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("10066716097576"),
				}),
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: utils.NewBig("125224746")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7", Amount: utils.NewBig("124570062")},
				Fee:            poolpkg.TokenAmount{Token: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Amount: utils.NewBig("375674")},
			},
			expectedFwReserves: [4]*big.Int{utils.NewBig("10089263705492"), utils.NewBig("10066591527514"), utils.One, utils.One},
			expectedOriginalReserves: uniswapv2.ReserveData{
				Reserve0: utils.NewBig("10089263705492"),
				Reserve1: utils.NewBig("10066591527514"),
			},
		},
		{
			name: "[swap1to0] it should return correct amountOut and fee",
			poolSimulator: PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Address: "0x576cea6d4461fcb3a9d43e922c9b54c0f791599a",
						Tokens: []string{
							"0x32a7c02e79c4ea1008dd6564b35f131428673c41",
							"0xdac17f958d2ee523a2206206994597c13d831ec7",
							"0x18755d2cec785ab87680edb8e117615e4b005430", // Wrapped token 0
							"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1", // Wrapped token 1
						},
						Reserves: []*big.Int{utils.NewBig("70361282326226590645832"), utils.NewBig("54150601005"), utils.One, utils.One},
					},
				},
				originalReserves: uniswapv2.ReserveData{
					Reserve0: utils.NewBig("70361282326226590645832"),
					Reserve1: utils.NewBig("54150601005"),
				},
				fee:          number.NewUint256("3"),
				feePrecision: number.NewUint256("1000"),
			},
			params: poolpkg.UpdateBalanceParams{
				SwapInfo: SwapInfo{
					IsWrapIn:    true,
					IsUnwrapOut: true,
					WTokenIn:    "0x18755d2cec785ab87680edb8e117615e4b005430",
					WTokenOut:   "0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1",
				},
				SwapLimit: swaplimit.NewInventory("ringswap", map[string]*big.Int{
					"0x18755d2cec785ab87680edb8e117615e4b005430": utils.NewBig("70361282326226590645832"),
					"0x66714db8f3397c767d0a602458b5b4e3c0fe7dd1": utils.NewBig("54150601005"),
				}),
				TokenAmountIn:  poolpkg.TokenAmount{Token: "0xdac17f958d2ee523a2206206994597c13d831ec7", Amount: utils.NewBig("124570062")},
				TokenAmountOut: poolpkg.TokenAmount{Token: "0x32a7c02e79c4ea1008dd6564b35f131428673c41", Amount: utils.NewBig("161006857684289764421")},
			},
			expectedFwReserves: [4]*big.Int{utils.NewBig("70200275468542300881411"), utils.NewBig("54275171067"), utils.One, utils.One},
			expectedOriginalReserves: uniswapv2.ReserveData{
				Reserve0: utils.NewBig("70200275468542300881411"),
				Reserve1: utils.NewBig("54275171067"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.poolSimulator.UpdateBalance(tc.params)

			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[0].Cmp(tc.expectedFwReserves[0]))
			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[1].Cmp(tc.expectedFwReserves[1]))
			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[2].Cmp(tc.expectedFwReserves[2]))
			assert.Equal(t, 0, tc.poolSimulator.Info.Reserves[3].Cmp(tc.expectedFwReserves[3]))

			assert.Equal(t, 0, tc.poolSimulator.originalReserves.Reserve0.Cmp(tc.expectedOriginalReserves.Reserve0))
			assert.Equal(t, 0, tc.poolSimulator.originalReserves.Reserve1.Cmp(tc.expectedOriginalReserves.Reserve1))
		})
	}
}

func TestPoolSimulator_getAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     PoolSimulator
		reserveIn         *uint256.Int
		reserveOut        *uint256.Int
		amountIn          *uint256.Int
		expectedAmountOut *uint256.Int
	}{
		{
			name:              "it should return correct amountOut",
			poolSimulator:     PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:         number.NewUint256("10089138480746"),
			reserveOut:        number.NewUint256("10066716097576"),
			amountIn:          number.NewUint256("125224746"),
			expectedAmountOut: number.NewUint256("124570062"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountOut := tc.poolSimulator.getAmountOut(tc.amountIn, tc.reserveIn, tc.reserveOut)

			assert.Equal(t, 0, tc.expectedAmountOut.Cmp(amountOut))
		})
	}
}

func TestPoolSimulator_getAmountIn(t *testing.T) {
	testCases := []struct {
		name             string
		poolSimulator    PoolSimulator
		reserveIn        *uint256.Int
		reserveOut       *uint256.Int
		amountOut        *uint256.Int
		expectedAmountIn *uint256.Int
		expectedErr      error
	}{
		{
			name:             "it should return correct amountIn",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("100000000"),
			reserveOut:       number.NewUint256("100000000000000000000"),
			amountOut:        number.NewUint256("20000000000000000000"),
			expectedAmountIn: number.NewUint256("25075226"),
			expectedErr:      nil,
		},
		{
			name:             "it should return correct ErrDSMathSubUnderflow error",
			poolSimulator:    PoolSimulator{fee: uint256.NewInt(3), feePrecision: uint256.NewInt(1000)},
			reserveIn:        number.NewUint256("1160689189059097452"),
			reserveOut:       number.NewUint256("1161607"),
			amountOut:        number.NewUint256("500000000"),
			expectedAmountIn: nil,
			expectedErr:      uniswapv2.ErrDSMathSubUnderflow,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			amountIn, err := tc.poolSimulator.getAmountIn(tc.amountOut, tc.reserveIn, tc.reserveOut)
			assert.ErrorIs(t, err, tc.expectedErr)

			if err == nil {
				fmt.Printf("amountIn: %s\n", amountIn.String())
				assert.Equal(t, 0, tc.expectedAmountIn.Cmp(amountIn))
			}
		})
	}
}
