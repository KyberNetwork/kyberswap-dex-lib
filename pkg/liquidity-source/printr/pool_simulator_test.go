package printr

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	testBasePair   = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" // WETH
	testToken      = "0x1234567890abcdef1234567890abcdef12345678"
	testPrintrAddr = "0xb77726291b125515d0a7affeea2b04f2ff243172"
)

func makeTestPool(t *testing.T, reserve *uint256.Int, completionThreshold *uint256.Int, tradingFee uint16, paused bool) *PoolSimulator {
	t.Helper()

	staticExtra := StaticExtra{
		PrintrAddr:     testPrintrAddr,
		Token:          testToken,
		BasePair:       testBasePair,
		TotalCurves:    1,
		MaxTokenSupply: "1000000000000000000000000000", // 1e27
		VirtualReserve: "1000000000000000000",          // 1e18
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	extra := Extra{
		Reserve:             reserve,
		CompletionThreshold: completionThreshold,
		TradingFee:          tradingFee,
		Paused:              paused,
	}
	extraBytes, _ := json.Marshal(extra)

	ep := entity.Pool{
		Address:     testToken,
		Exchange:    "printr",
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testBasePair}, {Address: testToken}},
		Reserves:    []string{reserve.ToBig().String(), "0"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func TestPoolSimulator_CalcAmountOut_Buy(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000") // 5e26
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Spend 0.01 ETH to buy tokens
	amountIn := mustFromDecimal("10000000000000000") // 0.01e18

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should receive tokens")
	assert.Equal(t, testToken, result.TokenAmountOut.Token)
	assert.True(t, result.Fee.Amount.Sign() > 0, "should have fee")

	// Verify swap info
	swapInfo := result.SwapInfo.(*SwapInfo)
	assert.True(t, swapInfo.IsBuy)
}

func TestPoolSimulator_CalcAmountOut_Sell(t *testing.T) {
	// Pool with some reserve from prior buys
	reserve := mustFromDecimal("500000000000000000") // 0.5 ETH
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Sell 1 token
	amountIn := mustFromDecimal("1000000000000000000") // 1e18

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: amountIn.ToBig()},
		TokenOut:      testBasePair,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should receive basePair")
	assert.Equal(t, testBasePair, result.TokenAmountOut.Token)
	assert.True(t, result.Fee.Amount.Sign() > 0, "should have fee")

	swapInfo := result.SwapInfo.(*SwapInfo)
	assert.False(t, swapInfo.IsBuy)
}

func TestPoolSimulator_CalcAmountOut_Paused(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, true) // paused

	amountIn := mustFromDecimal("10000000000000000")

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})

	assert.ErrorIs(t, err, ErrContractPaused)
}

func TestPoolSimulator_CalcAmountOut_Graduated(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := uint256.NewInt(0) // graduated
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	amountIn := mustFromDecimal("10000000000000000")

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})

	assert.ErrorIs(t, err, ErrTokenGraduated)
}

func TestPoolSimulator_CalcAmountOut_InvalidToken(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: "0xinvalid", Amount: big.NewInt(1)},
		TokenOut:      testToken,
	})

	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestPoolSimulator_CalcAmountOut_ZeroAmount(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: big.NewInt(0)},
		TokenOut:      testToken,
	})

	assert.ErrorIs(t, err, ErrZeroAmount)
}

func TestPoolSimulator_CalcAmountIn_Buy(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Want exactly 1e18 tokens
	amountOut := mustFromDecimal("1000000000000000000")

	result, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testToken, Amount: amountOut.ToBig()},
		TokenIn:        testBasePair,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountIn.Amount.Sign() > 0, "should need some basePair")
	assert.Equal(t, testBasePair, result.TokenAmountIn.Token)
}

func TestPoolSimulator_CalcAmountIn_Sell(t *testing.T) {
	// Pool with reserve
	reserve := mustFromDecimal("500000000000000000")
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Want exactly 0.001 ETH out
	amountOut := mustFromDecimal("1000000000000000") // 0.001e18

	result, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBasePair, Amount: amountOut.ToBig()},
		TokenIn:        testToken,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountIn.Amount.Sign() > 0, "should need some tokens")
	assert.Equal(t, testToken, result.TokenAmountIn.Token)
}

func TestPoolSimulator_UpdateBalance_Buy(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	reserveBefore := new(uint256.Int).Set(sim.reserve)

	// Simulate a buy
	amountIn := mustFromDecimal("10000000000000000")
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	// Reserve should increase after buy
	assert.True(t, sim.reserve.Gt(reserveBefore),
		"reserve should increase after buy: %s > %s", sim.reserve.String(), reserveBefore.String())
}

func TestPoolSimulator_UpdateBalance_Sell(t *testing.T) {
	reserve := mustFromDecimal("500000000000000000")
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	reserveBefore := new(uint256.Int).Set(sim.reserve)

	// Simulate a sell
	amountIn := mustFromDecimal("1000000000000000000")
	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: amountIn.ToBig()},
		TokenOut:      testBasePair,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken, Amount: amountIn.ToBig()},
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	// Reserve should decrease after sell
	assert.True(t, sim.reserve.Lt(reserveBefore),
		"reserve should decrease after sell: %s < %s", sim.reserve.String(), reserveBefore.String())
}

func TestPoolSimulator_CloneState(t *testing.T) {
	reserve := mustFromDecimal("500000000000000000")
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	cloned := sim.CloneState().(*PoolSimulator)

	// Modify original
	sim.reserve.AddUint64(sim.reserve, 12345)

	// Cloned should be independent
	assert.False(t, cloned.reserve.Eq(sim.reserve),
		"cloned reserve should be independent from original")
}

func TestPoolSimulator_ConsecutiveSwaps(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Do 3 consecutive buys and verify price increases
	buyAmount := mustFromDecimal("100000000000000000") // 0.1 ETH
	var prevTokensOut *big.Int

	for i := 0; i < 3; i++ {
		result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: buyAmount.ToBig()},
			TokenOut:      testToken,
		})
		require.NoError(t, err)

		if prevTokensOut != nil {
			// Each subsequent buy should yield fewer tokens (price increases)
			assert.True(t, result.TokenAmountOut.Amount.Cmp(prevTokensOut) < 0,
				"buy %d should yield fewer tokens than buy %d: %s < %s",
				i+1, i, result.TokenAmountOut.Amount.String(), prevTokensOut.String())
		}
		prevTokensOut = result.TokenAmountOut.Amount

		sim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: testBasePair, Amount: buyAmount.ToBig()},
			TokenAmountOut: *result.TokenAmountOut,
			Fee:            *result.Fee,
			SwapInfo:       result.SwapInfo,
		})
	}
}

func TestPoolSimulator_GetApprovalAddress_Buy(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Buying: basePair is input, no approval needed on Printr
	addr := sim.GetApprovalAddress(testBasePair, testToken)
	assert.Equal(t, "", addr, "buy direction should not require Printr approval")
}

func TestPoolSimulator_GetApprovalAddress_Sell(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Selling: token is input, must approve Printr
	addr := sim.GetApprovalAddress(testToken, testBasePair)
	assert.Equal(t, testPrintrAddr, addr, "sell direction should require Printr approval")
}

func TestPoolSimulator_CalcAmountOut_Buy_RemainingIn(t *testing.T) {
	reserve := uint256.NewInt(0)
	// Very small completion threshold so buy gets capped
	completionThreshold := mustFromDecimal("1000000000000000000") // 1e18 (1 token)
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Spend a large amount - most should be returned as remaining
	amountIn := mustFromDecimal("10000000000000000000") // 10 ETH

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should receive some tokens")

	// With a small completion threshold, should have remaining input
	if result.RemainingTokenAmountIn != nil {
		assert.True(t, result.RemainingTokenAmountIn.Amount.Sign() > 0,
			"should have remaining basePair after capped buy")
		assert.Equal(t, testBasePair, result.RemainingTokenAmountIn.Token)
	}
}

func TestPoolSimulator_CalcAmountOut_Sell_RemainingIn(t *testing.T) {
	// Pool with a small reserve → small issued supply
	reserve := mustFromDecimal("1000000000000000") // 0.001 ETH
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Try to sell way more tokens than issued supply
	amountIn := mustFromDecimal("999000000000000000000000000") // nearly 1e27

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: amountIn.ToBig()},
		TokenOut:      testBasePair,
	})

	require.NoError(t, err)
	assert.True(t, result.TokenAmountOut.Amount.Sign() > 0, "should receive some basePair")

	// Should have remaining tokens (capped by issued supply)
	if result.RemainingTokenAmountIn != nil {
		assert.True(t, result.RemainingTokenAmountIn.Amount.Sign() > 0,
			"should have remaining tokens after capped sell")
		assert.Equal(t, testToken, result.RemainingTokenAmountIn.Token)
	}
}

func TestPoolSimulator_CalcAmountIn_Sell_ZeroReserve(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Want some basePair out by selling tokens, but reserve is 0 (no issued supply)
	amountOut := mustFromDecimal("1000000000000000") // 0.001 ETH

	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: testBasePair, Amount: amountOut.ToBig()},
		TokenIn:        testToken,
	})

	assert.Error(t, err, "should fail with zero reserve (no issued supply)")
	assert.ErrorIs(t, err, ErrInsufficientReserves)
}

func TestPoolSimulator_UpdateBalance_BuySellConsistency(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Buy tokens
	buyAmount := mustFromDecimal("1000000000000000000") // 1 ETH
	buyResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: buyAmount.ToBig()},
		TokenOut:      testToken,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testBasePair, Amount: buyAmount.ToBig()},
		TokenAmountOut: *buyResult.TokenAmountOut,
		Fee:            *buyResult.Fee,
		SwapInfo:       buyResult.SwapInfo,
	})

	reserveAfterBuy := new(uint256.Int).Set(sim.reserve)
	assert.True(t, reserveAfterBuy.Sign() > 0, "reserve should be positive after buy")

	// Now sell all the tokens we bought back
	tokensReceived := buyResult.TokenAmountOut.Amount
	sellResult, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testToken, Amount: tokensReceived},
		TokenOut:      testBasePair,
	})
	require.NoError(t, err)

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: testToken, Amount: tokensReceived},
		TokenAmountOut: *sellResult.TokenAmountOut,
		Fee:            *sellResult.Fee,
		SwapInfo:       sellResult.SwapInfo,
	})

	// Reserve should be positive (fees captured) but less than after buy
	assert.True(t, sim.reserve.Sign() >= 0,
		"reserve should be non-negative after sell-back: %s", sim.reserve.String())
	assert.True(t, sim.reserve.Lt(reserveAfterBuy),
		"reserve should decrease after selling back: %s < %s",
		sim.reserve.String(), reserveAfterBuy.String())
}

func makeTestPoolWithCurves(t *testing.T, totalCurves uint16, reserve *uint256.Int, completionThreshold *uint256.Int, tradingFee uint16) *PoolSimulator {
	t.Helper()

	staticExtra := StaticExtra{
		PrintrAddr:     testPrintrAddr,
		Token:          testToken,
		BasePair:       testBasePair,
		TotalCurves:    totalCurves,
		MaxTokenSupply: "1000000000000000000000000000", // 1e27
		VirtualReserve: "1000000000000000000",          // 1e18
	}
	staticExtraBytes, _ := json.Marshal(staticExtra)

	extra := Extra{
		Reserve:             reserve,
		CompletionThreshold: completionThreshold,
		TradingFee:          tradingFee,
		Paused:              false,
	}
	extraBytes, _ := json.Marshal(extra)

	ep := entity.Pool{
		Address:     testToken,
		Exchange:    "printr",
		Type:        DexType,
		Tokens:      []*entity.PoolToken{{Address: testBasePair}, {Address: testToken}},
		Reserves:    []string{reserve.ToBig().String(), "0"},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func TestPoolSimulator_MultipleCurves(t *testing.T) {
	completionThreshold := mustFromDecimal("500000000000000000000000000")

	sim1 := makeTestPoolWithCurves(t, 1, uint256.NewInt(0), completionThreshold, 100)
	sim3 := makeTestPoolWithCurves(t, 3, uint256.NewInt(0), completionThreshold, 100)

	amountIn := mustFromDecimal("100000000000000000") // 0.1 ETH

	result1, err := sim1.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})
	require.NoError(t, err)

	result3, err := sim3.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})
	require.NoError(t, err)

	// With 3 curves (smaller pool per curve), same ETH should yield fewer tokens
	assert.True(t, result1.TokenAmountOut.Amount.Cmp(result3.TokenAmountOut.Amount) > 0,
		"1-curve pool should yield more tokens (%s) than 3-curve pool (%s)",
		result1.TokenAmountOut.Amount.String(), result3.TokenAmountOut.Amount.String())
}

func TestPoolSimulator_GetMetaInfo(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	// Buy direction: no approval needed
	meta := sim.GetMetaInfo(testBasePair, testToken).(MetaInfo)
	assert.Equal(t, "", meta.ApprovalAddress, "buy should have no approval address")

	// Sell direction: approval needed
	meta = sim.GetMetaInfo(testToken, testBasePair).(MetaInfo)
	assert.Equal(t, testPrintrAddr, meta.ApprovalAddress, "sell should have Printr approval address")
}

func TestPoolSimulator_CalcAmountOut_SwapInfoReserveDelta(t *testing.T) {
	reserve := uint256.NewInt(0)
	completionThreshold := mustFromDecimal("500000000000000000000000000")
	sim := makeTestPool(t, reserve, completionThreshold, 100, false)

	amountIn := mustFromDecimal("100000000000000000") // 0.1 ETH

	result, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: testBasePair, Amount: amountIn.ToBig()},
		TokenOut:      testToken,
	})
	require.NoError(t, err)

	swapInfo := result.SwapInfo.(*SwapInfo)
	assert.True(t, swapInfo.IsBuy)
	assert.True(t, swapInfo.reserveDelta.Sign() > 0,
		"reserveDelta should be positive for buy: %s", swapInfo.reserveDelta.String())
	// reserveDelta should be <= amountIn (it's cost minus fee)
	assert.True(t, !swapInfo.reserveDelta.Gt(uint256.MustFromBig(amountIn.ToBig())),
		"reserveDelta %s should not exceed amountIn %s",
		swapInfo.reserveDelta.String(), amountIn.String())
}
