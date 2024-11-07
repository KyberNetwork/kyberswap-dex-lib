package staderethx

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	testCases := []struct {
		name              string
		poolSimulator     *PoolSimulator
		param             poolpkg.CalcAmountOutParams
		expectedAmountOut *big.Int
		expectedError     error
	}{
		{
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10000000000000000000000"),
					Token:  WETH,
				},
				TokenOut: ETHx,
			},
			expectedAmountOut: bignumber.NewBig("9593905841213731393939"),
		},
		{
			name: "it should return correct amount",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10000000023132132100"),
					Token:  WETH,
				},
				TokenOut: ETHx,
			},
			expectedAmountOut: bignumber.NewBig("9593905863406481121"),
		},
		{
			name: "it should return ErrInvalidDepositAmount when amountIn > maxDeposit",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("10000000000000000000001"),
					Token:  WETH,
				},
				TokenOut: ETHx,
			},
			expectedError: ErrInvalidDepositAmount,
		},
		{
			name: "it should return ErrInvalidDepositAmount when amountIn < minDeposit",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("99999999999999"),
					Token:  WETH,
				},
				TokenOut: ETHx,
			},
			expectedError: ErrInvalidDepositAmount,
		},
		{
			name: "it should return ErrInvalidTokenIn",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("99999999999999"),
					Token:  ETHx,
				},
				TokenOut: WETH,
			},
			expectedError: ErrInvalidTokenIn,
		},
		{
			name: "it should return ErrInvalidTokenOut",
			poolSimulator: &PoolSimulator{
				Pool: poolpkg.Pool{
					Info: poolpkg.PoolInfo{
						Tokens: []string{WETH, ETHx},
					},
				},
				paused:          false,
				minDeposit:      uint256.MustFromDecimal("100000000000000"),
				maxDeposit:      uint256.MustFromDecimal("10000000000000000000000"),
				exchangeRate:    uint256.MustFromDecimal("1042328345254521839"),
				totalETHXSupply: uint256.MustFromDecimal("118600315516947203976686"),
				totalETHBalance: uint256.MustFromDecimal("123620470619443769071059"),
			},
			param: poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{
					Amount: bignumber.NewBig("99999999999999"),
					Token:  WETH,
				},
				TokenOut: WETH,
			},
			expectedError: ErrInvalidTokenOut,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.poolSimulator.CalcAmountOut(tc.param)

			assert.Equal(t, tc.poolSimulator.CanSwapTo(tc.poolSimulator.Info.Tokens[0]), []string{})
			assert.Equal(t, tc.poolSimulator.CanSwapTo(tc.poolSimulator.Info.Tokens[1]), []string{WETH})

			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			}

			if tc.expectedAmountOut != nil {
				assert.Equal(t, tc.expectedAmountOut.String(), result.TokenAmountOut.Amount.String())
			}
		})
	}
}
