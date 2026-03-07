package obric

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func buildPool(t *testing.T, decimalsX, decimalsY uint8, multYBase, reserveX, reserveY, currentXK, preK string,
	feeMillionth, priceMaxAge, priceUpdateTime uint64, isLocked, enable bool,
) *PoolSimulator {
	t.Helper()

	staticExtra := StaticExtra{
		MultYBase: multYBase,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)

	extra := Extra{
		ReserveX:        reserveX,
		ReserveY:        reserveY,
		CurrentXK:       currentXK,
		PreK:            preK,
		FeeMillionth:    feeMillionth,
		PriceMaxAge:     priceMaxAge,
		PriceUpdateTime: priceUpdateTime,
		IsLocked:        isLocked,
		Enable:          enable,
	}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:  "0xpool",
		Exchange: "obric",
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: "0xtokenX", Decimals: decimalsX, Swappable: true},
			{Address: "0xtokenY", Decimals: decimalsY, Swappable: true},
		},
		Reserves:    entity.PoolReserves{reserveX, reserveY},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)
	return sim
}

func referenceQuote(decimalsX, decimalsY uint8, multYBase, currentXK, preK *big.Int,
	feeMillionth uint64, isXtoY bool, inputAmt *big.Int,
) *big.Int {
	decDiff := int(decimalsY) - int(decimalsX)

	var multFactor *big.Int
	if decDiff < 0 {
		multFactor = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-decDiff)), nil)
	} else {
		multFactor = new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decDiff)), nil)
	}

	var K *big.Int
	if decimalsX > decimalsY {
		K = new(big.Int).Div(preK, multFactor)
		K.Div(K, multYBase)
	} else {
		K = new(big.Int).Mul(preK, multFactor)
		K.Div(K, multYBase)
	}

	currentYK := new(big.Int).Div(K, currentXK)

	var currentLK, currentRK *big.Int
	if isXtoY {
		currentLK = new(big.Int).Set(currentXK)
		currentRK = currentYK
	} else {
		currentLK = currentYK
		currentRK = new(big.Int).Set(currentXK)
	}

	newLK := new(big.Int).Add(currentLK, inputAmt)
	newRK := new(big.Int).Div(K, newLK)

	outputBeforeFee := new(big.Int).Sub(currentRK, newRK)

	fee := new(big.Int).Mul(outputBeforeFee, big.NewInt(int64(feeMillionth)))
	fee.Div(fee, big.NewInt(1_000_000))

	output := new(big.Int).Sub(outputBeforeFee, fee)
	return output
}

func TestCalcAmountOut_XtoY(t *testing.T) {
	// 18-decimal token X, 6-decimal token Y (e.g. ETH -> USDC)
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6,
		"1",                                 // multYBase
		"100000000000000000000",             // reserveX = 100e18
		"200000000000",                      // reserveY = 200000e6
		"50000000000000000000",              // currentXK = 50e18
		"200000000000000000000000000000000", // preK (large enough)
		3000,                                // feeMillionth = 0.3%
		600,                                 // priceMaxAge = 600s
		now,                                 // priceUpdateTime = now
		false,                               // not locked
		true,                                // enabled
	)

	inputAmount := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 1e18

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtokenX",
			Amount: inputAmount,
		},
		TokenOut: "0xtokenY",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "output should be positive")
	assert.Equal(t, "0xtokenY", result.TokenAmountOut.Token)

	// Verify against reference using simulator's internal values
	cxk := sim.currentXK.ToBig()
	pk := sim.preK.ToBig()
	myb := sim.multYBase.ToBig()
	expected := referenceQuote(18, 6, myb, cxk, pk, 3000, true, inputAmount)
	assert.Equal(t, expected.String(), result.TokenAmountOut.Amount.String())
}

func TestCalcAmountOut_YtoX(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6,
		"1",
		"100000000000000000000",
		"200000000000",
		"50000000000000000000",
		"200000000000000000000000000000000",
		3000,
		600,
		now,
		false,
		true,
	)

	inputAmount := big.NewInt(1000000) // 1 USDC (6 decimals)

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtokenY",
			Amount: inputAmount,
		},
		TokenOut: "0xtokenX",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "output should be positive")
	assert.Equal(t, "0xtokenX", result.TokenAmountOut.Token)

	cxk := sim.currentXK.ToBig()
	pk := sim.preK.ToBig()
	myb := sim.multYBase.ToBig()
	expected := referenceQuote(18, 6, myb, cxk, pk, 3000, false, inputAmount)
	assert.Equal(t, expected.String(), result.TokenAmountOut.Amount.String())
}

func TestCalcAmountOut_SameDecimals(t *testing.T) {
	now := uint64(time.Now().Unix())

	// Both 18-decimal tokens
	sim := buildPool(t,
		18, 18,
		"1",
		"1000000000000000000000", // 1000e18
		"1000000000000000000000", // 1000e18
		"500000000000000000000",  // 500e18
		"250000000000000000000000000000000000000", // preK
		5000,
		600,
		now,
		false,
		true,
	)

	inputAmount := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 10e18

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xtokenX",
			Amount: inputAmount,
		},
		TokenOut: "0xtokenY",
	})
	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0)
}

func TestCalcAmountOut_PoolLocked(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, now,
		true, // locked
		true,
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtokenY",
	})
	assert.ErrorIs(t, err, ErrPoolLocked)
}

func TestCalcAmountOut_PoolDisabled(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, now,
		false,
		false, // disabled
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtokenY",
	})
	assert.ErrorIs(t, err, ErrPoolDisabled)
}

func TestCalcAmountOut_PriceStale(t *testing.T) {
	// priceUpdateTime is old enough that now + 20 > priceUpdateTime + priceMaxAge
	staleTime := uint64(time.Now().Unix()) - 1000

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, staleTime,
		false, true,
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtokenY",
	})
	assert.ErrorIs(t, err, ErrPriceStale)
}

func TestCalcAmountOut_ZeroCurrentXK(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"0", // currentXK = 0
		"200000000000000000000000000000000",
		3000, 600, now,
		false, true,
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtokenY",
	})
	assert.ErrorIs(t, err, ErrZeroCurrentXK)
}

func TestCalcAmountOut_InvalidToken(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, now,
		false, true,
	)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xinvalid", Amount: big.NewInt(1000000)},
		TokenOut:      "0xtokenY",
	})
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestCalcAmountOut_FeeDeducted(t *testing.T) {
	now := uint64(time.Now().Unix())

	// Same pool with 0 fee vs 3000 (0.3%) fee
	simNoFee := buildPool(t,
		18, 18, "1",
		"1000000000000000000000", "1000000000000000000000",
		"500000000000000000000",
		"250000000000000000000000000000000000000",
		0, 600, now, false, true,
	)

	simWithFee := buildPool(t,
		18, 18, "1",
		"1000000000000000000000", "1000000000000000000000",
		"500000000000000000000",
		"250000000000000000000000000000000000000",
		3000, 600, now, false, true,
	)

	inputAmount := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	resultNoFee, err := simNoFee.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: inputAmount},
		TokenOut:      "0xtokenY",
	})
	require.NoError(t, err)

	resultWithFee, err := simWithFee.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: inputAmount},
		TokenOut:      "0xtokenY",
	})
	require.NoError(t, err)

	// Output with fee should be less than without fee
	assert.True(t, resultWithFee.TokenAmountOut.Amount.Cmp(resultNoFee.TokenAmountOut.Amount) < 0)
	// Fee should be non-zero
	assert.True(t, resultWithFee.Fee.Amount.Sign() > 0)
}

func TestUpdateBalance(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, now, false, true,
	)

	inputAmount := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: inputAmount},
		TokenOut:      "0xtokenY",
	})
	require.NoError(t, err)

	oldReserveX := new(big.Int).Set(sim.Info.Reserves[0])
	oldReserveY := new(big.Int).Set(sim.Info.Reserves[1])

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: "0xtokenX", Amount: inputAmount},
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
	})

	// Reserve X should increase by inputAmount
	expectedReserveX := new(big.Int).Add(oldReserveX, inputAmount)
	assert.Equal(t, expectedReserveX.String(), sim.Info.Reserves[0].String())

	// Reserve Y should decrease by output amount
	expectedReserveY := new(big.Int).Sub(oldReserveY, result.TokenAmountOut.Amount)
	assert.Equal(t, expectedReserveY.String(), sim.Info.Reserves[1].String())
}

func TestCloneState(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 6, "1",
		"100000000000000000000", "200000000000",
		"50000000000000000000", "200000000000000000000000000000000",
		3000, 600, now, false, true,
	)

	cloned := sim.CloneState().(*PoolSimulator)

	// Modify cloned reserves
	cloned.Info.Reserves[0] = big.NewInt(999)

	// Original should be unchanged
	assert.NotEqual(t, sim.Info.Reserves[0].String(), cloned.Info.Reserves[0].String())
}

func TestCalculateK_DecimalsXGreaterThanY(t *testing.T) {
	// When decimalsX > decimalsY, K = preK / multFactor / multYBase
	sim := &PoolSimulator{
		decimalsX: 18,
		decimalsY: 6,
		multYBase: uint256.NewInt(1),
		preK:      new(uint256.Int).SetUint64(1000000000000000000), // 1e18
	}

	K := sim.calculateK()
	// multFactor = 10^(18-6) = 10^12
	// K = 1e18 / 1e12 / 1 = 1e6
	expected := uint256.NewInt(1000000)
	assert.Equal(t, expected.String(), K.String())
}

func TestCalculateK_DecimalsYGreaterThanX(t *testing.T) {
	// When decimalsY > decimalsX, K = preK * multFactor / multYBase
	sim := &PoolSimulator{
		decimalsX: 6,
		decimalsY: 18,
		multYBase: uint256.NewInt(1),
		preK:      uint256.NewInt(1000000), // 1e6
	}

	K := sim.calculateK()
	// multFactor = 10^(18-6) = 10^12
	// K = 1e6 * 1e12 / 1 = 1e18
	expected := new(uint256.Int).SetUint64(1000000000000000000) // 1e18
	assert.Equal(t, expected.String(), K.String())
}

func TestCalcAmountIn_XtoY(t *testing.T) {
	now := uint64(time.Now().Unix())

	// Balanced pool: currentXK=500e18, K=currentXK^2=250e39 so currentYK=500e18
	sim := buildPool(t,
		18, 18, "1",
		"1000000000000000000000", "1000000000000000000000",
		"500000000000000000000",
		"250000000000000000000000000000000000000000", // 250e39
		3000, 600, now, false, true,
	)

	inputAmount := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 1e18

	resultOut, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: inputAmount},
		TokenOut:      "0xtokenY",
	})
	require.NoError(t, err)

	// Round-trip: CalcAmountIn with amountOut should return amountIn close to original
	resultIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xtokenY", Amount: resultOut.TokenAmountOut.Amount},
		TokenIn:        "0xtokenX",
	})
	require.NoError(t, err)
	assert.True(t, resultIn.TokenAmountIn.Amount.Sign() > 0)

	// With floor division (matching SDK), amountIn may be slightly less than original
	// The difference should be small (within a few units)
	diff := new(big.Int).Abs(new(big.Int).Sub(resultIn.TokenAmountIn.Amount, inputAmount))
	assert.True(t, diff.Cmp(big.NewInt(2)) <= 0,
		"difference %s should be <= 2", diff.String())
}

func TestCalcAmountIn_YtoX(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 18, "1",
		"1000000000000000000000", "1000000000000000000000",
		"500000000000000000000",
		"250000000000000000000000000000000000000000", // 250e39
		3000, 600, now, false, true,
	)

	inputAmount := new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 1e18

	resultOut, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenY", Amount: inputAmount},
		TokenOut:      "0xtokenX",
	})
	require.NoError(t, err)

	resultIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xtokenX", Amount: resultOut.TokenAmountOut.Amount},
		TokenIn:        "0xtokenY",
	})
	require.NoError(t, err)
	assert.True(t, resultIn.TokenAmountIn.Amount.Sign() > 0)

	diff := new(big.Int).Abs(new(big.Int).Sub(resultIn.TokenAmountIn.Amount, inputAmount))
	assert.True(t, diff.Cmp(big.NewInt(2)) <= 0,
		"difference %s should be <= 2", diff.String())
}

func TestCalcAmountIn_Consistency(t *testing.T) {
	now := uint64(time.Now().Unix())

	sim := buildPool(t,
		18, 18, "1",
		"1000000000000000000000", "1000000000000000000000",
		"500000000000000000000",
		"250000000000000000000000000000000000000000", // 250e39
		5000, 600, now, false, true,
	)

	// CalcAmountIn then verify with CalcAmountOut
	desiredOut := new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)) // 5e18

	resultIn, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: "0xtokenY", Amount: desiredOut},
		TokenIn:        "0xtokenX",
	})
	require.NoError(t, err)

	// Use the computed amountIn in CalcAmountOut - output should be close to desired
	resultOut, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xtokenX", Amount: resultIn.TokenAmountIn.Amount},
		TokenOut:      "0xtokenY",
	})
	require.NoError(t, err)

	// With floor division (matching SDK), actual output may be slightly less than desired
	diff := new(big.Int).Abs(new(big.Int).Sub(resultOut.TokenAmountOut.Amount, desiredOut))
	// Should be within 0.01% of desired output
	tolerance := new(big.Int).Div(desiredOut, big.NewInt(10000))
	assert.True(t, diff.Cmp(tolerance) <= 0,
		"difference %s should be <= %s (0.01%%)", diff.String(), tolerance.String())
}

func TestCalcAmountIn_ErrorCases(t *testing.T) {
	now := uint64(time.Now().Unix())

	t.Run("invalid token", func(t *testing.T) {
		sim := buildPool(t, 18, 6, "1",
			"100000000000000000000", "200000000000",
			"50000000000000000000", "200000000000000000000000000000000",
			3000, 600, now, false, true)

		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: "0xinvalid", Amount: big.NewInt(1000)},
			TokenIn:        "0xtokenX",
		})
		assert.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("pool locked", func(t *testing.T) {
		sim := buildPool(t, 18, 6, "1",
			"100000000000000000000", "200000000000",
			"50000000000000000000", "200000000000000000000000000000000",
			3000, 600, now, true, true)

		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: "0xtokenY", Amount: big.NewInt(1000)},
			TokenIn:        "0xtokenX",
		})
		assert.ErrorIs(t, err, ErrPoolLocked)
	})

	t.Run("insufficient liquidity", func(t *testing.T) {
		sim := buildPool(t, 18, 6, "1",
			"100000000000000000000", "200000000000",
			"50000000000000000000", "200000000000000000000000000000000",
			3000, 600, now, false, true)

		// Request more than reserve
		_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: "0xtokenY", Amount: big.NewInt(999999999999)},
			TokenIn:        "0xtokenX",
		})
		assert.ErrorIs(t, err, ErrInsufficientLiquidity)
	})
}
