package susde

import (
	"errors"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	type fields struct {
		Pool        pool.Pool
		totalAssets *uint256.Int
		totalSupply *uint256.Int
	}
	type expected struct {
		calcAmountOutResult pool.CalcAmountOutResult
		totalAssets         *uint256.Int
		totalSupply         *uint256.Int
		error               error
	}
	tests := []struct {
		name     string
		fields   fields
		params   []pool.CalcAmountOutParams
		expected []expected
	}{
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/d4135224-503d-43af-a0dc-bcf95e7d3e38/simulation/83c41944-c23a-4085-9b9d-9ba1414277cb
			name: "should match expected result from a mock simulated tx for multi-path",
			fields: fields{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
						Tokens:   []string{"0x4c9edd5852cd905f086c759e8383e09bff1e68b3", "0x9d39a5de30e57443bff2a8307a4256c8797a3497"},
						Reserves: []*big.Int{utils.NewBig10("2006133174155182059108575912"), utils.NewBig10("1796588169625826666184796333")},
					},
				},
				totalAssets: uint256.MustFromDecimal("2006133174155182059108575912"),
				totalSupply: uint256.MustFromDecimal("1796588169625826666184796333"),
			},
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x4c9edd5852cd905f086c759e8383e09bff1e68b3",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
				},
			},
			expected: []expected{
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: utils.NewBig10("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133174155182059108575912"),
					totalSupply: uint256.MustFromDecimal("1796588169625826666184796333"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: utils.NewBig10("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133274155182059108575912"),
					totalSupply: uint256.MustFromDecimal("1796588259180607613486638472"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: utils.NewBig10("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133374155182059108575912"),
					totalSupply: uint256.MustFromDecimal("1796588348735388560788480611"),
					error:       nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &PoolSimulator{
				Pool:        tt.fields.Pool,
				totalAssets: tt.fields.totalAssets,
				totalSupply: tt.fields.totalSupply,
			}

			for i, param := range tt.params {
				got, err := s.CalcAmountOut(param)
				if !errors.Is(err, tt.expected[i].error) {
					t.Errorf("PoolSimulator.CalcAmountOut() error = %v, wantErr %v", err, tt.expected[i].error)
					return
				}

				require.Equal(t, tt.expected[i].calcAmountOutResult.TokenAmountOut.Amount, got.TokenAmountOut.Amount)
				require.Equal(t, tt.expected[i].totalAssets, s.totalAssets)
				require.Equal(t, tt.expected[i].totalSupply, s.totalSupply)

				s.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  param.TokenAmountIn,
					TokenAmountOut: *got.TokenAmountOut,
				})
			}
		})
	}
}
