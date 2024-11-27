package sfrxeth_convertor

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
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/3f1bcb50-fd2a-432b-8e95-f21b12d32112/simulation/3d0198cf-131f-487c-83d4-e79eea08cb7a
			name: "Test Deposit: should match expected result from a mock simulated tx for multi-path",
			fields: fields{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Tokens:   []string{"0x5e8422345238f34275888049021821e8e08caa1f", "0xac3e018457b222d93114458476f3e3416abbe38f"},
						Reserves: []*big.Int{utils.NewBig10("117954317618747599936548"), utils.NewBig10("106794914919235920539073")},
					},
				},
				totalAssets: uint256.MustFromDecimal("117954317618747599936548"),
				totalSupply: uint256.MustFromDecimal("106794914919235920539073"),
			},
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
						Amount: utils.NewBig10("100000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
						Amount: utils.NewBig10("100000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
						Amount: utils.NewBig10("100000000000000"),
					},
					TokenOut: "0xac3e018457b222d93114458476f3e3416abbe38f",
				},
			},
			expected: []expected{
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90539216431584"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: true,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954317618747599936548"),
					totalSupply: uint256.MustFromDecimal("106794914919235920539073"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90539216431584"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: true,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954317718747599936548"),
					totalSupply: uint256.MustFromDecimal("106794915009775136970657"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
							Amount: utils.NewBig10("90539216431584"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: true,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954317818747599936548"),
					totalSupply: uint256.MustFromDecimal("106794915100314353402241"),
					error:       nil,
				},
			},
		},
		{
			// https://dashboard.tenderly.co/tenderly_kyber/nhathm/fork/e48ba37e-29ff-4614-b3da-2a77cbeb9e36/simulation/2eb4b74a-7098-4f1a-b395-ab105928c6b1
			name: "Test Redeem: should match expected result from a mock simulated tx for multi-path",
			fields: fields{
				Pool: pool.Pool{
					Info: pool.PoolInfo{
						Address:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Tokens:   []string{"0x5e8422345238f34275888049021821e8e08caa1f", "0xac3e018457b222d93114458476f3e3416abbe38f"},
						Reserves: []*big.Int{utils.NewBig10("117954432267012196091328"), utils.NewBig10("106794914919235920539073")},
					},
				},
				totalAssets: uint256.MustFromDecimal("117954432267012196091328"),
				totalSupply: uint256.MustFromDecimal("106794914919235920539073"),
			},
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Amount: utils.NewBig10("1000000000000000"),
					},
					TokenOut: "0x5e8422345238f34275888049021821e8e08caa1f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Amount: utils.NewBig10("1000000000000000"),
					},
					TokenOut: "0x5e8422345238f34275888049021821e8e08caa1f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Amount: utils.NewBig10("1000000000000000"),
					},
					TokenOut: "0x5e8422345238f34275888049021821e8e08caa1f",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xac3e018457b222d93114458476f3e3416abbe38f",
						Amount: utils.NewBig10("1000000000000000"),
					},
					TokenOut: "0x5e8422345238f34275888049021821e8e08caa1f",
				},
			},
			expected: []expected{
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
							Amount: utils.NewBig10("1104494838131719"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: false,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954432267012196091328"),
					totalSupply: uint256.MustFromDecimal("106794914919235920539073"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
							Amount: utils.NewBig10("1104494838131719"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: false,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954431162517357959609"),
					totalSupply: uint256.MustFromDecimal("106794913919235920539073"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
							Amount: utils.NewBig10("1104494838131719"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: false,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954430058022519827890"),
					totalSupply: uint256.MustFromDecimal("106794912919235920539073"),
					error:       nil,
				},
				{
					calcAmountOutResult: pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x5e8422345238f34275888049021821e8e08caa1f",
							Amount: utils.NewBig10("1104494838131719"),
						},
						SwapInfo: SwapInfo{
							IsDeposit: false,
						},
					},
					totalAssets: uint256.MustFromDecimal("117954428953527681696171"),
					totalSupply: uint256.MustFromDecimal("106794911919235920539073"),
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
				require.Equal(t, tt.expected[i].calcAmountOutResult.SwapInfo, got.SwapInfo)

				s.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  param.TokenAmountIn,
					TokenAmountOut: *got.TokenAmountOut,
					SwapInfo:       got.SwapInfo,
				})
			}
		})
	}
}
