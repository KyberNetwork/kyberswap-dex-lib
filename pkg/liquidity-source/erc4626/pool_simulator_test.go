package erc4626

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCalcAmountOut(t *testing.T) {
	t.Parallel()
	type expected struct {
		out         *pool.CalcAmountOutResult
		totalShares *uint256.Int
		totalAssets *uint256.Int
		error       error
	}
	tests := []struct {
		name        string
		Pool        pool.Pool
		totalAssets *uint256.Int
		totalShares *uint256.Int
		entryFee    uint64
		exitFee     uint64
		params      []pool.CalcAmountOutParams
		expected    []expected
	}{
		{
			name: "should match expected result",
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address: "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
					Tokens: []string{"0x9d39a5de30e57443bff2a8307a4256c8797a3497",
						"0x4c9edd5852cd905f086c759e8383e09bff1e68b3"},
					Reserves: []*big.Int{bignum.NewBig("2006133174155182059108575912"),
						bignum.NewBig("1796588169625826666184796333")},
				},
			},
			totalAssets: uint256.MustFromDecimal("2006133174155182059108575912"),
			totalShares: uint256.MustFromDecimal("1796588169625826666184796333"),
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
					out: &pool.CalcAmountOutResult{
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
					out: &pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: bignum.NewBig("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133374155182059108575912"),
					totalShares: uint256.MustFromDecimal("1796588348735388560788480611"),
					error:       nil,
				},
				{
					out: &pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x9d39a5de30e57443bff2a8307a4256c8797a3497",
							Amount: bignum.NewBig("89554780947301842139"),
						},
					},
					totalAssets: uint256.MustFromDecimal("2006133474155182059108575912"),
					totalShares: uint256.MustFromDecimal("1796588438290169508090322750"),
					error:       nil,
				},
			},
		},
		{
			name: "with fee",
			Pool: pool.Pool{
				Info: pool.PoolInfo{
					Address: "0xffffff9936bd58a008855b0812b44d2c8dffe2aa",
					Tokens: []string{"0xffffff9936bd58a008855b0812b44d2c8dffe2aa",
						"0x00000000efe302beaa2b3e6e1b18d08d69a9012a"},
					Reserves: []*big.Int{bignum.NewBig("1050030703988"),
						bignum.NewBig("1050030703988")},
				},
			},
			totalShares: uint256.MustFromDecimal("1050030703988"),
			totalAssets: uint256.MustFromDecimal("1050030703988"),
			entryFee:    10,
			exitFee:     50,
			params: []pool.CalcAmountOutParams{
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0x00000000efe302beaa2b3e6e1b18d08d69a9012a",
						Amount: bignum.NewBig("5000000000"),
					},
					TokenOut: "0xffffff9936bd58a008855b0812b44d2c8dffe2aa",
				},
				{
					TokenAmountIn: pool.TokenAmount{
						Token:  "0xffffff9936bd58a008855b0812b44d2c8dffe2aa",
						Amount: bignum.NewBig("4995000000"),
					},
					TokenOut: "0x00000000efe302beaa2b3e6e1b18d08d69a9012a",
				},
			},
			expected: []expected{
				{
					out: &pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0xffffff9936bd58a008855b0812b44d2c8dffe2aa",
							Amount: bignum.NewBig("4995000000"),
						},
					},
					totalAssets: uint256.MustFromDecimal("1055025703988"),
					totalShares: uint256.MustFromDecimal("1055025703988"),
					error:       nil,
				},
				{
					out: &pool.CalcAmountOutResult{
						TokenAmountOut: &pool.TokenAmount{
							Token:  "0x00000000efe302beaa2b3e6e1b18d08d69a9012a",
							Amount: bignum.NewBig("4970025000"),
						},
					},
					totalAssets: uint256.MustFromDecimal("1050030703988"),
					totalShares: uint256.MustFromDecimal("1050030703988"),
					error:       nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := &PoolSimulator{
				Pool:                tt.Pool,
				TotalSupply:         tt.totalShares,
				TotalAssets:         tt.totalAssets,
				EntryFeeBasisPoints: tt.entryFee,
				ExitFeeBasisPoints:  tt.exitFee,
				supportedSwapType:   Both,
			}

			for i, param := range tt.params {
				t.Run(tt.name+"#"+strconv.Itoa(i), func(t *testing.T) {
					out, err := sim.CalcAmountOut(param)
					assert.ErrorIs(t, err, tt.expected[i].error)
					assert.Equal(t, tt.expected[i].out.TokenAmountOut, out.TokenAmountOut, "TokenAmountOut")

					in, err := sim.CalcAmountIn(pool.CalcAmountInParams{
						TokenAmountOut: pool.TokenAmount{
							Token:  out.TokenAmountOut.Token,
							Amount: out.TokenAmountOut.Amount,
						},
						TokenIn: param.TokenAmountIn.Token,
					})
					assert.NoError(t, err)
					assert.Equal(t, param.TokenAmountIn, *in.TokenAmountIn, "TokenAmountIn")

					sim.UpdateBalance(pool.UpdateBalanceParams{
						TokenAmountIn:  param.TokenAmountIn,
						TokenAmountOut: *out.TokenAmountOut,
						SwapInfo:       out.SwapInfo,
					})
					assert.Equal(t, tt.expected[i].totalAssets, sim.TotalAssets, "TotalAssets")
					assert.Equal(t, tt.expected[i].totalShares, sim.TotalSupply, "TotalSupply")
				})
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	t.Parallel()
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
