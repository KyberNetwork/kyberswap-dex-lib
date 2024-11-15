package sfrxeth

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
			name: "should match expected result from a mock simulated tx for multi-path",
			fields: fields{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0xbafa44efe7901e04e39dad13167d089c559c1138",
						Tokens:   []string{"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", "0xac3e018457b222d93114458476f3e3416abbe38f"},
						Reserves: []*big.Int{utils.NewBig10("118146441674159654557167"), utils.NewBig10("106975517640850176664420")},
					},
				},
				totalAssets: uint256.MustFromDecimal("118146441674159654557167"),
				totalSupply: uint256.MustFromDecimal("106975517640850176664420"),
			},
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Amount: utils.NewBig10("100000000000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
			},
			expected: []expected{
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90544849362354751747"),
						},
					},
					totalAssets: uint256.MustFromDecimal("118146441674159654557167"),
					totalSupply: uint256.MustFromDecimal("106975517640850176664420"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90544849362354751747"),
						},
					},
					totalAssets: uint256.MustFromDecimal("118246441674159654557167"),
					totalSupply: uint256.MustFromDecimal("107066062490212531416167"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90544849362354751747"),
						},
					},
					totalAssets: uint256.MustFromDecimal("118346441674159654557167"),
					totalSupply: uint256.MustFromDecimal("107156607339574886167914"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90544849362354751747"),
						},
					},
					totalAssets: uint256.MustFromDecimal("118446441674159654557167"),
					totalSupply: uint256.MustFromDecimal("107247152188937240919661"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90544849362354751747"),
						},
					},
					totalAssets: uint256.MustFromDecimal("118546441674159654557167"),
					totalSupply: uint256.MustFromDecimal("107337697038299595671408"),
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
