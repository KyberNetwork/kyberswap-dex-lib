package erc4626

import (
	"errors"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	type expected struct {
		out         pool.CalcAmountOutResult
		totalShares *uint256.Int
		totalAssets *uint256.Int
		error       error
	}
	tests := []struct {
		name        string
		Pool        pool.Pool
		totalAssets *uint256.Int
		totalShares *uint256.Int
		params      []pool.CalcAmountOutParams
		expected    []expected
	}{
		{
			name: "should match expected result",
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
					Tokens:   []string{"0x9d39a5de30e57443bff2a8307a4256c8797a3497", "0x4c9edd5852cd905f086c759e8383e09bff1e68b3"},
					Reserves: []*big.Int{bignum.NewBig("2006133174155182059108575912"), bignum.NewBig("1796588169625826666184796333")},
				},
			},
			totalShares: uint256.MustFromDecimal("1796588169625826666184796333"),
			totalAssets: uint256.MustFromDecimal("2006133174155182059108575912"),
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: bignum.NewBig("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: bignum.NewBig("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: bignum.NewBig("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
			},
			expected: []expected{
				{
					out: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: bignum.NewBig("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133174155182059108575912"),
					totalShares: uint256.MustFromDecimal("1796588169625826666184796333"),
					error:       nil,
				},
				{
					out: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: bignum.NewBig("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133274155182059108575912"),
					totalShares: uint256.MustFromDecimal("1796588259180607613486638472"),
					error:       nil,
				},
				{
					out: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: bignum.NewBig("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133374155182059108575912"),
					totalShares: uint256.MustFromDecimal("1796588348735388560788480611"),
					error:       nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := &PoolSimulator{
				Pool:              tt.Pool,
				TotalSupply:       tt.totalShares,
				TotalAssets:       tt.totalAssets,
				supportedSwapType: Deposit,
			}

			for i, param := range tt.params {
				out, err := sim.CalcAmountOut(param)
				if !errors.Is(err, tt.expected[i].error) {
					t.Errorf("PoolSimulator.CalcAmountOut() error = %v, wantErr %v", err, tt.expected[i].error)
					return
				}

				assert.Equal(t, tt.expected[i].out.TokenAmountOut.Amount, out.TokenAmountOut.Amount)
				assert.Equal(t, tt.expected[i].totalShares, sim.TotalSupply)
				assert.Equal(t, tt.expected[i].totalAssets, sim.TotalAssets)

				sim.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  param.TokenAmountIn,
					TokenAmountOut: *out.TokenAmountOut,
					SwapInfo:       out.SwapInfo,
				})

				in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
					TokenAmountOut: pool.TokenAmount{
						Token:  out.TokenAmountOut.Token,
						Amount: out.TokenAmountOut.Amount,
					},
					TokenIn: param.TokenAmountIn.Token,
				})
				assert.NoError(t, err)
				assert.Equal(t, param.TokenAmountIn.Amount, in.TokenAmountIn.Amount)
				assert.Equal(t, param.TokenAmountIn.Token, in.TokenAmountIn.Token)
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	tests := []struct {
		totalAssets       *uint256.Int
		totalShares       *uint256.Int
		supportedSwapType SwapType
	}{
		{
			totalShares:       uint256.MustFromDecimal("4322261860101847297422878"),
			totalAssets:       uint256.MustFromDecimal("4328501428476501398713219"),
			supportedSwapType: Deposit,
		},
		{
			totalShares:       uint256.MustFromDecimal("4322261860101847297422878"),
			totalAssets:       uint256.MustFromDecimal("4328501428476501398713219"),
			supportedSwapType: Redeem,
		},
		{
			totalShares:       uint256.MustFromDecimal("4322261860101847297422878"),
			totalAssets:       uint256.MustFromDecimal("4328501428476501398713219"),
			supportedSwapType: Both,
		},
		{
			totalShares:       uint256.MustFromDecimal("88888888297422878"),
			totalAssets:       uint256.MustFromDecimal("4328501428476501398713219"),
			supportedSwapType: Both,
		},
		{
			totalShares:       uint256.MustFromDecimal("88888888297422878"),
			totalAssets:       uint256.MustFromDecimal("99476501398713219"),
			supportedSwapType: Both,
		},
		{
			totalShares:       uint256.MustFromDecimal("1000000000000000000000"),
			totalAssets:       uint256.MustFromDecimal("9999999999999999999999"),
			supportedSwapType: Deposit,
		},
		{
			totalShares:       uint256.MustFromDecimal("99999999999999"),
			totalAssets:       uint256.MustFromDecimal("1000000000000000000000"),
			supportedSwapType: Deposit,
		},
		{
			totalShares:       uint256.MustFromDecimal("99999999999999"),
			totalAssets:       uint256.MustFromDecimal("1000000000000000000000"),
			supportedSwapType: Redeem,
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			s := &PoolSimulator{
				Pool:              pool.Pool{Info: pool.PoolInfo{Tokens: []string{"A", "B"}}},
				TotalSupply:       tt.totalShares,
				TotalAssets:       tt.totalAssets,
				supportedSwapType: tt.supportedSwapType,
			}

			testutil.TestCalcAmountIn(t, s)
		})
	}
}
