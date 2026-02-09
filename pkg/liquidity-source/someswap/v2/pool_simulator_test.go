package someswapv2

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestHighPrecisionBPS tests the new high-precision BPS format (10^9)
// where: 7500 = 0.00075%, 300000 = 0.03%, 190000 = 0.019%
func TestHighPrecisionBPS(t *testing.T) {
	BPS_DEN := mustFromDecimal("1000000000") // 10^9

	testCases := []struct {
		name            string
		amountIn        *uint256.Int
		reserveIn       *uint256.Int
		reserveOut      *uint256.Int
		baseFee         *uint256.Int // high-precision BPS (10^9 = 100%)
		wToken0         *uint256.Int // weight for token0 (10^9 scale)
		wToken1         *uint256.Int // weight for token1 (10^9 scale)
		tokenIn         int
		expectedFeePerc float64 // expected fee in %
	}{
		{
			name:            "0.03% fee (baseFee=300000), 100% on input",
			amountIn:        uint256.NewInt(1_000_000_000), // 1000 tokens
			reserveIn:       uint256.NewInt(100_000_000_000),
			reserveOut:      uint256.NewInt(100_000_000_000),
			baseFee:         uint256.NewInt(300000),        // 0.03%
			wToken0:         mustFromDecimal("1000000000"), // 100%
			wToken1:         uint256.NewInt(0),
			tokenIn:         0,
			expectedFeePerc: 0.03,
		},
		{
			name:            "0.00075% fee (baseFee=7500), 100% on input",
			amountIn:        uint256.NewInt(1_000_000_000),
			reserveIn:       uint256.NewInt(100_000_000_000),
			reserveOut:      uint256.NewInt(100_000_000_000),
			baseFee:         uint256.NewInt(7500), // 0.00075%
			wToken0:         mustFromDecimal("1000000000"),
			wToken1:         uint256.NewInt(0),
			tokenIn:         0,
			expectedFeePerc: 0.00075,
		},
		{
			name:            "0.019% fee (baseFee=190000), 100% on input",
			amountIn:        uint256.NewInt(1_000_000_000),
			reserveIn:       uint256.NewInt(100_000_000_000),
			reserveOut:      uint256.NewInt(100_000_000_000),
			baseFee:         uint256.NewInt(190000), // 0.019%
			wToken0:         mustFromDecimal("1000000000"),
			wToken1:         uint256.NewInt(0),
			tokenIn:         0,
			expectedFeePerc: 0.019,
		},
		{
			name:            "0.03% fee, 50/50 split",
			amountIn:        uint256.NewInt(1_000_000_000),
			reserveIn:       uint256.NewInt(100_000_000_000),
			reserveOut:      uint256.NewInt(100_000_000_000),
			baseFee:         uint256.NewInt(300000),       // 0.03%
			wToken0:         mustFromDecimal("500000000"), // 50%
			wToken1:         mustFromDecimal("500000000"), // 50%
			tokenIn:         0,
			expectedFeePerc: 0.03,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate using the correct formula
			weightIn := tc.wToken0
			if tc.tokenIn == 1 {
				weightIn = tc.wToken1
			}

			// inFee = baseFee * weightIn / BPS_DEN
			inFee := new(uint256.Int).Mul(tc.baseFee, weightIn)
			inFee.Div(inFee, BPS_DEN)

			// outFee = baseFee - inFee
			outFee := new(uint256.Int).Sub(tc.baseFee, inFee)

			// amountInAfterFee = amountIn * (BPS_DEN - inFee) / BPS_DEN
			feeMultiplier := new(uint256.Int).Sub(BPS_DEN, inFee)
			amountInAfterFee := new(uint256.Int).Mul(tc.amountIn, feeMultiplier)
			amountInAfterFee.Div(amountInAfterFee, BPS_DEN)

			// grossOut = amountInAfterFee * reserveOut / (reserveIn + amountInAfterFee)
			grossOut := new(uint256.Int).Mul(amountInAfterFee, tc.reserveOut)
			denom := new(uint256.Int).Add(tc.reserveIn, amountInAfterFee)
			grossOut.Div(grossOut, denom)

			// netOut = grossOut * (BPS_DEN - outFee) / BPS_DEN
			outFeeMultiplier := new(uint256.Int).Sub(BPS_DEN, outFee)
			expectedNetOut := new(uint256.Int).Mul(grossOut, outFeeMultiplier)
			expectedNetOut.Div(expectedNetOut, BPS_DEN)

			actualFeePerc := float64(tc.baseFee.Uint64()) / float64(BPS_DEN.Uint64()) * 100
			assert.InDelta(t, tc.expectedFeePerc, actualFeePerc, 0.0001)

			sim := createTestSimulator(t, tc.reserveIn, tc.reserveOut, tc.baseFee, tc.wToken0, tc.wToken1)
			result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{
					Token:  sim.Info.Tokens[tc.tokenIn],
					Amount: tc.amountIn.ToBig(),
				},
				TokenOut: sim.Info.Tokens[1-tc.tokenIn],
			})
			require.NoError(t, err)

			actualNetOut := uint256.MustFromBig(result.TokenAmountOut.Amount)
			assert.Equal(t, expectedNetOut.String(), actualNetOut.String())
		})
	}
}

func createTestSimulator(t *testing.T, reserve0, reserve1, baseFee, wToken0, wToken1 *uint256.Int) *PoolSimulator {
	ep := entity.Pool{
		Address:  "0x1234567890123456789012345678901234567890",
		Exchange: "someswap-v2",
		Type:     "someswap-v2",
		Reserves: []string{reserve0.String(), reserve1.String()},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18},
			{Address: "0xtoken1", Decimals: 6},
		},
		StaticExtra: fmt.Sprintf(`{"baseFee":%s,"wToken0":%s,"wToken1":%s}`,
			baseFee.String(), wToken0.String(), wToken1.String()),
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// TestDynamicFeeFromExtra verifies that dynBps from Extra is added to baseFee
func TestDynamicFeeFromExtra(t *testing.T) {
	reserve := mustFromDecimal("100000000000")
	baseFee := uint256.NewInt(190000)        // 0.019%
	dynBps := uint256.NewInt(150000)         // 0.015% dynamic fee
	wToken1 := mustFromDecimal("1000000000") // 100%

	// Create pool without dynamic fee
	epNoDyn := entity.Pool{
		Address:  "0x1234567890123456789012345678901234567890",
		Exchange: "someswap-v2",
		Type:     "someswap-v2",
		Reserves: []string{reserve.String(), reserve.String()},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18},
			{Address: "0xtoken1", Decimals: 18},
		},
		StaticExtra: fmt.Sprintf(`{"baseFee":%s,"wToken0":0,"wToken1":%s}`,
			baseFee.String(), wToken1.String()),
		Extra: "", // No dynamic fee
	}

	// Create pool with dynamic fee from Extra
	epWithDyn := entity.Pool{
		Address:  "0x1234567890123456789012345678901234567890",
		Exchange: "someswap-v2",
		Type:     "someswap-v2",
		Reserves: []string{reserve.String(), reserve.String()},
		Tokens: []*entity.PoolToken{
			{Address: "0xtoken0", Decimals: 18},
			{Address: "0xtoken1", Decimals: 18},
		},
		StaticExtra: fmt.Sprintf(`{"baseFee":%s,"wToken0":0,"wToken1":%s}`,
			baseFee.String(), wToken1.String()),
		Extra: fmt.Sprintf(`{"dynBps":%s}`, dynBps.String()), // With dynamic fee
	}

	simNoDyn, err := NewPoolSimulator(epNoDyn)
	require.NoError(t, err)

	simWithDyn, err := NewPoolSimulator(epWithDyn)
	require.NoError(t, err)

	amountIn := uint256.NewInt(1_000_000_000)

	// Swap token1 -> token0 (uses wToken1 = 100%)
	resultNoDyn, err := simNoDyn.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: amountIn.ToBig()},
		TokenOut:      "0xtoken0",
	})
	require.NoError(t, err)

	resultWithDyn, err := simWithDyn.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtoken1", Amount: amountIn.ToBig()},
		TokenOut:      "0xtoken0",
	})
	require.NoError(t, err)

	// With dynamic fee, output should be less (more fee taken)
	assert.Greater(t, resultNoDyn.TokenAmountOut.Amount.Cmp(resultWithDyn.TokenAmountOut.Amount), 0)
}

func mustFromDecimal(s string) *uint256.Int {
	clean := ""
	for _, c := range s {
		if c != '_' {
			clean += string(c)
		}
	}
	v, err := uint256.FromDecimal(clean)
	if err != nil {
		panic(err)
	}
	return v
}
