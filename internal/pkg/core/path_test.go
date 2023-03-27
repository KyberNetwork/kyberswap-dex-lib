package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/uni"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/entity"
)

func TestNewPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		pools            []poolPkg.IPool
		tokens           []entity.Token
		tokenAmountIn    poolPkg.TokenAmount
		tokenOut         string
		gasOption        GasOption
		tokenOutPrice    float64
		tokenOutDecimals uint8
		expectedPath     *Path
		expectedError    error
	}{
		{
			name: "it should return path successfully",
			pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool1",
							Tokens:   []string{"token1", "token2"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool2",
							Tokens:   []string{"token2", "token3"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
			},
			tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			tokenAmountIn:    poolPkg.TokenAmount{Token: "token1", Amount: big.NewInt(100), AmountUsd: 100},
			tokenOut:         "token3",
			tokenOutPrice:    10000,
			tokenOutDecimals: 6,
			gasOption: GasOption{
				GasFeeInclude: false,
				Price:         big.NewFloat(0),
				TokenPrice:    0,
			},
			expectedPath: &Path{
				Input: poolPkg.TokenAmount{
					Token:     "token1",
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				Output: poolPkg.TokenAmount{
					Token:     "token3",
					Amount:    big.NewInt(98),
					AmountUsd: 0.98,
				},
				TotalGas: 0,
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address:  "pool1",
								Tokens:   []string{"token1", "token2"},
								Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
								SwapFee:  big.NewInt(0),
							},
						},
						Weights: []uint{50, 50},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address:  "pool2",
								Tokens:   []string{"token2", "token3"},
								Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
								SwapFee:  big.NewInt(0),
							},
						},
						Weights: []uint{50, 50},
					},
				},
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				PriceImpact: big.NewInt(20000000000000000),
			},
			expectedError: nil,
		},
		{
			name: "it should return ErrInvalidTokenLength when token length < 2",
			tokens: []entity.Token{
				{
					Address: "token1",
				},
			},
			expectedPath:  nil,
			expectedError: ErrInvalidTokenLength,
		},
		{
			name: "it should return ErrInvalidPoolLength when token length != pool length + 1",
			pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool1",
							Tokens:   []string{"token1", "token2"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool2",
							Tokens:   []string{"token2", "token3"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
			},
			tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
			},
			expectedPath:  nil,
			expectedError: ErrInvalidPoolLength,
		},
		{
			name: "it should return ErrInvalidTokenIn when tokenIn different than the first token",
			pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool1",
							Tokens:   []string{"token1", "token2"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool2",
							Tokens:   []string{"token2", "token3"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
			},
			tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			tokenAmountIn:    poolPkg.TokenAmount{Token: "token2", Amount: big.NewInt(100)},
			tokenOut:         "token3",
			tokenOutPrice:    10000,
			tokenOutDecimals: 6,
			gasOption: GasOption{
				GasFeeInclude: false,
				Price:         big.NewFloat(0),
				TokenPrice:    0,
			},
			expectedPath:  nil,
			expectedError: ErrInvalidTokenIn,
		},
		{
			name: "it should return ErrInvalidTokenOut when tokenOut different than the last token",
			pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool1",
							Tokens:   []string{"token1", "token2"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address:  "pool2",
							Tokens:   []string{"token2", "token3"},
							Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
							SwapFee:  big.NewInt(0),
						},
					},
					Weights: []uint{50, 50},
				},
			},
			tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			tokenAmountIn:    poolPkg.TokenAmount{Token: "token1", Amount: big.NewInt(100)},
			tokenOut:         "token2",
			tokenOutPrice:    10000,
			tokenOutDecimals: 6,
			gasOption: GasOption{
				GasFeeInclude: false,
				Price:         big.NewFloat(0),
				TokenPrice:    0,
			},
			expectedPath:  nil,
			expectedError: ErrInvalidTokenOut,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := NewPath(
				tc.pools,
				tc.tokens,
				tc.tokenAmountIn,
				tc.tokenOut,
				tc.tokenOutPrice,
				tc.tokenOutDecimals,
				tc.gasOption,
			)

			assert.Equal(t, tc.expectedPath, path)
			assert.ErrorIs(t, tc.expectedError, err)
		})
	}
}

func TestPath_TrySwap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                   string
		path                   *Path
		tokenAmountIn          poolPkg.TokenAmount
		expectedTokenAmountOut poolPkg.TokenAmount
		expectedErr            error
	}{
		{
			name: "it should swap and return amount correctly",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address:  "pool1",
								Tokens:   []string{"token1", "token2"},
								Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
								SwapFee:  big.NewInt(0),
							},
						},
						Weights: []uint{50, 50},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address:  "pool2",
								Tokens:   []string{"token2", "token3"},
								Reserves: []*big.Int{big.NewInt(10000), big.NewInt(10000)},
								SwapFee:  big.NewInt(0),
							},
						},
						Weights: []uint{50, 50},
					},
				},
			},
			tokenAmountIn:          poolPkg.TokenAmount{Token: "token1", Amount: big.NewInt(100)},
			expectedTokenAmountOut: poolPkg.TokenAmount{Token: "token3", Amount: big.NewInt(98)},
			expectedErr:            nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenAmountOut, err := tc.path.TrySwap(tc.tokenAmountIn)

			assert.Equal(t, tc.expectedTokenAmountOut, tokenAmountOut)
			assert.ErrorIs(t, tc.expectedErr, err)
		})
	}
}

func TestPath_Equals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		path           *Path
		otherPath      *Path
		expectedResult bool
	}{
		{
			name: "it should return false when pool length is different",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			otherPath: &Path{
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "it should return false when token length is different",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "it should return false when input token is different",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
				Input: poolPkg.TokenAmount{Token: "token1"},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token2",
					},
					{
						Address: "token1",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
				Input: poolPkg.TokenAmount{Token: "token2"},
			},
			expectedResult: false,
		},
		{
			name: "it should return false when output token is different",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
				Input:  poolPkg.TokenAmount{Token: "token1"},
				Output: poolPkg.TokenAmount{Token: "token2"},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
				Input:  poolPkg.TokenAmount{Token: "token1"},
				Output: poolPkg.TokenAmount{Token: "token3"},
			},
			expectedResult: false,
		},
		{
			name: "it should return false when token list are not equal",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token2",
					},
					{
						Address: "token1",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "it should return false when pool list are not equal",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		{
			name: "it should return true when two paths are equal",
			path: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{
						Address: "token1",
					},
					{
						Address: "token2",
					},
					{
						Address: "token3",
					},
				},
				Pools: []poolPkg.IPool{
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool1",
							},
						},
					},
					&uni.Pool{
						Pool: poolPkg.Pool{
							Info: poolPkg.PoolInfo{
								Address: "pool2",
							},
						},
					},
				},
			},
			expectedResult: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.path.Equals(tc.otherPath)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestPath_Merge(t *testing.T) {
	t.Parallel()

	t.Run("it should return false when two paths are not equal", func(t *testing.T) {
		path := &Path{
			Tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			Pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool1",
						},
					},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool2",
						},
					},
				},
			},
		}
		otherPath := &Path{
			Pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool1",
						},
					},
				},
			},
		}

		result := path.Merge(otherPath)

		assert.False(t, result)
	})

	t.Run("it should return true when two paths are equal", func(t *testing.T) {
		path := &Path{
			Tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			Pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool1",
						},
					},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool2",
						},
					},
				},
			},
			Input: poolPkg.TokenAmount{
				Token:     "token1",
				Amount:    big.NewInt(100),
				AmountUsd: 100,
			},
			Output: poolPkg.TokenAmount{
				Token:     "token3",
				Amount:    big.NewInt(98),
				AmountUsd: 98,
			},
		}
		otherPath := &Path{
			Tokens: []entity.Token{
				{
					Address: "token1",
				},
				{
					Address: "token2",
				},
				{
					Address: "token3",
				},
			},
			Pools: []poolPkg.IPool{
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool1",
						},
					},
				},
				&uni.Pool{
					Pool: poolPkg.Pool{
						Info: poolPkg.PoolInfo{
							Address: "pool2",
						},
					},
				},
			},
			Input: poolPkg.TokenAmount{
				Token:     "token1",
				Amount:    big.NewInt(100),
				AmountUsd: 100,
			},
			Output: poolPkg.TokenAmount{
				Token:     "token3",
				Amount:    big.NewInt(98),
				AmountUsd: 98,
			},
		}

		result := path.Merge(otherPath)

		assert.True(t, result)
		assert.Equal(t, big.NewInt(200), path.Input.Amount)
		assert.Equal(t, big.NewInt(196), path.Output.Amount)
		assert.EqualValues(t, 200, path.Input.AmountUsd)
		assert.EqualValues(t, 196, path.Output.AmountUsd)
	})
}

func TestPath_CompareTo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		path           *Path
		otherPath      *Path
		gasInclude     bool
		expectedResult int
	}{
		{
			name:           "it should return -1 when other path is nil",
			path:           &Path{},
			otherPath:      nil,
			gasInclude:     true,
			expectedResult: -1,
		},
		{
			name: "it should return -1 when gasInclude is true and the path has greater output amountUSD",
			path: &Path{
				Output: poolPkg.TokenAmount{
					AmountUsd: 100,
				},
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					AmountUsd: 99,
				},
			},
			gasInclude:     true,
			expectedResult: -1,
		},
		{
			name: "it should return 1 when gasInclude is true and the path has less output amountUSD",
			path: &Path{
				Output: poolPkg.TokenAmount{
					AmountUsd: 99,
				},
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					AmountUsd: 100,
				},
			},
			gasInclude:     true,
			expectedResult: 1,
		},
		{
			name: "it should return -1 when gasInclude is true, both path has same output amountUSD and the path has greater output amount",
			path: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(99),
					AmountUsd: 100,
				},
			},
			gasInclude:     true,
			expectedResult: -1,
		},
		{
			name: "it should return 1 when gasInclude is true, both path has same output amountUSD and the path has less output amount",
			path: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(99),
					AmountUsd: 100,
				},
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
			},
			gasInclude:     true,
			expectedResult: 1,
		},
		{
			name: "it should return 1 when both path has same output amounts and the path has greater price impact",
			path: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(2),
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			gasInclude:     false,
			expectedResult: 1,
		},
		{
			name: "it should return -1 when both path has same output amounts and the path has less price impact",
			path: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			otherPath: &Path{
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(2),
			},
			gasInclude:     false,
			expectedResult: -1,
		},
		{
			name: "it should return -1 when both path has same output amounts, price impact and the path has less token",
			path: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
					{Address: "token3"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			gasInclude:     false,
			expectedResult: -1,
		},
		{
			name: "it should return 1 when both path has same output amounts, price impact and the path has more token",
			path: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
					{Address: "token3"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			gasInclude:     false,
			expectedResult: 1,
		},
		{
			name: "it should return 0 when both path has same output amounts, price impact and token len",
			path: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
					{Address: "token3"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			otherPath: &Path{
				Tokens: []entity.Token{
					{Address: "token1"},
					{Address: "token2"},
					{Address: "token3"},
				},
				Output: poolPkg.TokenAmount{
					Amount:    big.NewInt(100),
					AmountUsd: 100,
				},
				PriceImpact: big.NewInt(1),
			},
			gasInclude:     false,
			expectedResult: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.path.CompareTo(tc.otherPath, tc.gasInclude)

			assert.Equal(t, tc.expectedResult, result)
		})
	}
}
