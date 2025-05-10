package bin

import (
	_ "embed"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	//go:embed sample_pool.json
	poolData string
	chainID  = 1
)

func TestCalcAmountOut(t *testing.T) {
	var poolEnt entity.Pool
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	tests := []struct {
		name            string
		tokenIn         string
		tokenOut        string
		amountIn        string
		expectAmountOut string
		expectError     error
	}{
		{
			name:            "small amount in",
			tokenIn:         "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			tokenOut:        "0x55d398326f99059ff775485246999027b3197955", // BUSD
			amountIn:        "1000000000000000",                           // 0.001 USDC
			expectAmountOut: "999801019898010",                            // ~0.00099 BUSD
			expectError:     nil,
		},
		{
			name:            "normal amount in",
			tokenIn:         "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			tokenOut:        "0x55d398326f99059ff775485246999027b3197955", // BUSD
			amountIn:        "1000000000000000000",                        // 1 USDC
			expectAmountOut: "999801019898010198",                         // ~0.99 BUSD
			expectError:     nil,
		},
		{
			name:        "large amount in",
			tokenIn:     "0x55d398326f99059ff775485246999027b3197955", // BUSD
			tokenOut:    "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", // USDC
			amountIn:    "1000000000000000000000000000",               // 1M USDC (USDC reserve is lower than 1M)
			expectError: ErrBinIDNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  tc.tokenIn,
					Amount: utils.NewBig10(tc.amountIn),
				},
				TokenOut: tc.tokenOut,
			})

			if tc.expectError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, out.TokenAmountOut.Amount.String(), tc.expectAmountOut)
			}
		})
	}
}

func TestCalcAmountIn(t *testing.T) {
	var poolEnt entity.Pool

	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	res, err := pSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  "0x55d398326f99059ff775485246999027b3197955",
			Amount: utils.NewBig10("999801019898010198"),
		},
		TokenIn: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
	})
	assert.NoError(t, err)
	assert.Equal(t, utils.NewBig10("1000000000000000000"), res.TokenAmountIn.Amount)
}

func TestMergeSwap(t *testing.T) {
	var poolEnt entity.Pool
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	const (
		loop      = 20
		tokenIn   = "0x55d398326f99059ff775485246999027b3197955"
		tokenOut  = "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"
		amountRaw = "1000000000000000000000" // 1_000 BUSD
	)

	amountIn := utils.NewBig10(amountRaw)
	amountInTotal := new(big.Int).Mul(amountIn, big.NewInt(int64(loop)))

	var amountOutSingle *big.Int
	t.Run("single large swap", func(t *testing.T) {
		pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
		assert.NoError(t, err)

		result, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountInTotal},
			TokenOut:      tokenOut,
		})
		require.NoError(t, err)

		amountOutSingle = result.TokenAmountOut.Amount
	})

	var totalAmountOutMulti = big.NewInt(0)
	t.Run("multiple smaller swaps", func(t *testing.T) {
		pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
		assert.NoError(t, err)

		for range loop {
			result, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenOut:      tokenOut,
			})
			require.NoError(t, err)

			pSim.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  pool.TokenAmount{Token: tokenIn, Amount: amountIn},
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: result.TokenAmountOut.Amount},
				Fee:            *result.Fee,
				SwapInfo:       result.SwapInfo,
			})

			totalAmountOutMulti.Add(totalAmountOutMulti, result.TokenAmountOut.Amount)
		}
	})

	t.Run("compare results", func(t *testing.T) {
		diff := new(big.Int).Sub(amountOutSingle, totalAmountOutMulti)
		ratio := new(big.Float).Quo(new(big.Float).SetInt(diff), new(big.Float).SetInt(amountOutSingle))

		maxAllowedDiff := big.NewFloat(0.005) // 0.5%
		assert.True(t, ratio.Cmp(maxAllowedDiff) < 0, "output mismatch too large: %.6f", ratio)
	})
}
