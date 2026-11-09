package maverickv2

import (
	"fmt"
	"maps"
	"math/big"
	"os"
	"sync"
	"testing"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func BenchmarkPoolSimulator_CalcAmountOut(b *testing.B) {
	// Read pool_data.json
	data, err := os.ReadFile("./data/pool_data.json")
	require.NoError(b, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(b, err)

	poolSim, _ := NewPoolSimulator(poolEntity)

	testutil.TestCalcAmountIn(b, poolSim)
}

func TestSimpleSwaps_USDC_USDT(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data

	// Read pool_data.json
	data, err := os.ReadFile("./data/pool_data.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Perform a swap to get swap info
	amountIn := big.NewInt(10000000) // 10 USDC
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // USDC
			Amount: amountIn,
		},
		TokenOut: poolEntity.Tokens[1].Address, // USDT
		Limit:    nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Extract swap info
	swapInfo, ok := result.SwapInfo.(maverickSwapInfo)
	require.True(t, ok)

	// Store state before UpdateBalance
	beforeUpdateReserveA := new(big.Int).Set(poolSim.Info.Reserves[0])
	beforeUpdateReserveB := new(big.Int).Set(poolSim.Info.Reserves[1])

	// Test UpdateBalance
	poolSim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // USDC
			Amount: amountIn,
		},
		TokenAmountOut: pool.TokenAmount{
			Token:  poolEntity.Tokens[1].Address, // USDT
			Amount: result.TokenAmountOut.Amount,
		},
		SwapInfo: swapInfo,
	})

	// Verify state updates
	t.Run("Active Tick Updated", func(t *testing.T) {
		require.Equal(t, swapInfo.activeTick, poolSim.state.ActiveTick, "Active tick should be updated from swap info")
	})

	t.Run("Bins Updated", func(t *testing.T) {
		require.Equal(t, len(swapInfo.bins), len(poolSim.state.Bins), "Bins count should match swap info")
		// Verify specific bins are updated
		for binId, expectedBin := range swapInfo.bins {
			actualBin, exists := poolSim.state.Bins[binId]
			require.True(t, exists, "Bin %d should exist", binId)
			require.Equal(t, expectedBin.TotalSupply.String(), actualBin.TotalSupply.String(), "Bin %d total supply should match", binId)
			require.Equal(t, expectedBin.TickBalance.String(), actualBin.TickBalance.String(), "Bin %d tick balance should match", binId)
		}
	})

	t.Run("Ticks Updated", func(t *testing.T) {
		require.Equal(t, len(swapInfo.ticks), len(poolSim.state.Ticks), "Ticks count should match swap info")
		// Verify specific ticks are updated
		for tickId, expectedTick := range swapInfo.ticks {
			actualTick, exists := poolSim.state.Ticks[tickId]
			require.True(t, exists, "Tick %d should exist", tickId)
			require.Equal(t, expectedTick.ReserveA.String(), actualTick.ReserveA.String(), "Tick %d reserve A should match", tickId)
			require.Equal(t, expectedTick.ReserveB.String(), actualTick.ReserveB.String(), "Tick %d reserve B should match", tickId)
		}
	})

	t.Run("Pool Reserves Updated", func(t *testing.T) {
		expectedReserveA := new(big.Int).Add(beforeUpdateReserveA, amountIn)
		expectedReserveB := new(big.Int).Sub(beforeUpdateReserveB, result.TokenAmountOut.Amount)

		require.Equal(t, expectedReserveA.String(), poolSim.Pool.Info.Reserves[0].String(), "Reserve A should be updated correctly")
		require.Equal(t, expectedReserveB.String(), poolSim.Pool.Info.Reserves[1].String(), "Reserve B should be updated correctly")
	})

	// Test error handling
	t.Run("Invalid SwapInfo Type", func(t *testing.T) {
		// This should not panic but should log a warning
		poolSim.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn: pool.TokenAmount{
				Token:  poolEntity.Tokens[0].Address,
				Amount: big.NewInt(1000),
			},
			TokenAmountOut: pool.TokenAmount{
				Token:  poolEntity.Tokens[1].Address,
				Amount: big.NewInt(1000),
			},
			SwapInfo: "invalid_swap_info", // Wrong type
		})
		// Should not crash - just verify simulator is still functional
		require.NotNil(t, poolSim.state)
	})

	t.Run("CalcAmountIn", func(t *testing.T) {
		testutil.TestCalcAmountIn(t, poolSim, 4) // so slow...
	})
}

func TestSimpleSwaps_MAV_WETH(t *testing.T) {
	t.Parallel()

	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Real USDC/USDT pool data with complete bins data

	// Read pool_data.json
	data, err := os.ReadFile("./data/mavweth.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	cloned := poolSim.CloneState()

	// Perform a swap to get swap info
	amountIn := big.NewInt(1_000_000_000_000_000_000) // 1 MAV
	tokenAmtIn := pool.TokenAmount{
		Token:  poolEntity.Tokens[0].Address, // MAV
		Amount: amountIn,
	}
	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      poolEntity.Tokens[1].Address, // WETH
		Limit:         nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert the expected amount
	expectedAmount := big.NewInt(22034312672685)
	require.Equal(t, expectedAmount.String(), result.TokenAmountOut.Amount.String(), "Swap amount should match expected value")

	cloned.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  tokenAmtIn,
		TokenAmountOut: *result.TokenAmountOut,
		Fee:            *result.Fee,
		SwapInfo:       result.SwapInfo,
	})

	result, err = poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      poolEntity.Tokens[1].Address, // WETH
		Limit:         nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert the expected amount of cloned pool stays the same
	require.Equal(t, expectedAmount.String(), result.TokenAmountOut.Amount.String(), "Swap amount should match expected value")

	result, err = cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: tokenAmtIn,
		TokenOut:      poolEntity.Tokens[1].Address, // WETH
		Limit:         nil,
	})
	require.NoError(t, err)
	require.NotNil(t, result)

	// Assert the expected amount after UpdateBalance
	require.Equal(t, "22034204484076", result.TokenAmountOut.Amount.String(), "Swap amount should match expected value")

	t.Run("CalcAmountIn", func(t *testing.T) {
		testutil.TestCalcAmountIn(t, poolSim, 4) // so slow...
	})
}

func TestDebugSwap_MAV_WETH(t *testing.T) {
	// Enable debug logging
	err := logger.SetLogLevel("debug")
	require.NoError(t, err)

	// Read pool_data.json
	data, err := os.ReadFile("./data/mavweth.json")
	require.NoError(t, err)

	// Unmarshal to entity.Pool
	var poolEntity entity.Pool
	err = json.Unmarshal(data, &poolEntity)
	require.NoError(t, err)

	poolSim, err := NewPoolSimulator(poolEntity)
	require.NoError(t, err)

	// Print pool information
	fmt.Printf("Pool Address: %s\n", poolEntity.Address)
	fmt.Printf("Token A (MAV): %s, decimals: %d\n", poolEntity.Tokens[0].Address, poolEntity.Tokens[0].Decimals)
	fmt.Printf("Token B (WETH): %s, decimals: %d\n", poolEntity.Tokens[1].Address, poolEntity.Tokens[1].Decimals)
	fmt.Printf("Reserve A (MAV): %s\n", poolEntity.Reserves[0])
	fmt.Printf("Reserve B (WETH): %s\n", poolEntity.Reserves[1])
	fmt.Printf("Fee A In: %d\n", poolSim.state.FeeAIn)
	fmt.Printf("Fee B In: %d\n", poolSim.state.FeeBIn)
	fmt.Printf("Protocol Fee Ratio: %d\n", poolSim.state.ProtocolFeeRatio)
	fmt.Printf("Active Tick: %d\n", poolSim.state.ActiveTick)
	// Debug tick information around active tick
	fmt.Printf("\nTick information around active tick %d:\n", poolSim.state.ActiveTick)
	for tick := poolSim.state.ActiveTick - 2; tick <= poolSim.state.ActiveTick+2; tick++ {
		if tickData, ok := poolSim.state.Ticks[tick]; ok {
			fmt.Printf("Tick %d: ReserveA=%s, ReserveB=%s, TotalSupply=%s\n",
				tick, tickData.ReserveA.String(), tickData.ReserveB.String(), tickData.TotalSupply.String())
		} else {
			fmt.Printf("Tick %d: No data\n", tick)
		}
	}
	// Debug bin information
	fmt.Printf("\nBin information:\n")
	binCount := 0
	for binId, bin := range poolSim.state.Bins {
		if binCount < 10 { // Show first 10 bins
			fmt.Printf("Bin %d: tick=%d, totalSupply=%s, tickBalance=%s\n",
				binId, bin.Tick, bin.TotalSupply.String(), bin.TickBalance.String())
		}
		binCount++
	}
	fmt.Printf("Total bins: %d\n", binCount)

	// Test scaling functions
	fmt.Printf("\nTesting scaling functions:\n")
	scaleA := getScale(poolEntity.Tokens[0].Decimals)
	scaleB := getScale(poolEntity.Tokens[1].Decimals)
	fmt.Printf("Scale A (MAV, 18 decimals): %s\n", scaleA.String())
	fmt.Printf("Scale B (WETH, 18 decimals): %s\n", scaleB.String())

	// Perform a swap to get swap info
	amountIn, ok := new(big.Int).SetString("20000000000000000000", 10) // 20 MAV
	require.True(t, ok)
	fmt.Printf("\nSwapping 20 MAV (%s) for WETH\n", amountIn.String())

	result, err := poolSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  poolEntity.Tokens[0].Address, // MAV
			Amount: amountIn,
		},
		TokenOut: poolEntity.Tokens[1].Address, // WETH
		Limit:    nil,
	})

	if err != nil {
		fmt.Printf("Error during calculation: %v\n", err)
		require.NoError(t, err)
	}

	require.NotNil(t, result)

	fmt.Printf("Result amount: %s WETH\n", result.TokenAmountOut.Amount.String())
	fmt.Printf("Expected amount: 440665698723087 WETH\n")
	fmt.Printf("Current amount: %s WETH\n", result.TokenAmountOut.Amount.String())
	fmt.Printf("Gas estimation: %d\n", result.Gas)

	// Check if the result matches expected value
	expectedAmount := big.NewInt(440665698723087)
	if result.TokenAmountOut.Amount.Cmp(expectedAmount) != 0 {
		fmt.Printf("❌ MISMATCH: Expected %s, got %s\n", expectedAmount.String(), result.TokenAmountOut.Amount.String())
		fmt.Printf("Difference: %s\n", new(big.Int).Sub(result.TokenAmountOut.Amount, expectedAmount).String())

		// Calculate percentage difference
		diff := new(big.Int).Sub(result.TokenAmountOut.Amount, expectedAmount)
		percentage := new(big.Int).Mul(diff, big.NewInt(10000))
		percentage.Div(percentage, expectedAmount)
		fmt.Printf("Percentage difference: %s.%02d%%\n",
			new(big.Int).Div(percentage, big.NewInt(100)).String(),
			new(big.Int).Mod(percentage, big.NewInt(100)).Int64())
	} else {
		fmt.Printf("✅ MATCH: Result matches expected value\n")
	}
}

const (
	cloneTokenA = "0x00000000000000000000000000000000000000a1"
	cloneTokenB = "0x00000000000000000000000000000000000000b2"
)

// newClonePool is a one-bin pool: enough for CloneState and a small swap.
func newClonePool(t *testing.T) pool.IPoolSimulator {
	t.Helper()
	extra := `{"feeAIn":100000000000000,"feeBIn":100000000000000,"protocolFeeRatio":0,"activeTick":0,"lastTwaD8":0,` +
		`"bins":{"1":{"tick":0,"totalSupply":"1000000000000000000000","tickBalance":"1000000000000000000000"}},` +
		`"ticks":{"0":{"ReserveA":"1000000000000000000000","ReserveB":"1000000000000000000000",` +
		`"TotalSupply":"1000000000000000000000","BinIdsByTick":{"0":1}}}}`
	p, err := NewPoolSimulator(entity.Pool{Address: "0xmav", Exchange: "maverick-v2", Type: "maverick-v2",
		Reserves: entity.PoolReserves{"1000000000000000000000", "1000000000000000000000"}, Tokens: cloneTestTokens(),
		Extra: extra, StaticExtra: `{"tickSpacing":10}`})
	require.NoError(t, err)
	return p
}

func cloneTestTokens() []*entity.PoolToken {
	return []*entity.PoolToken{{Address: cloneTokenA, Decimals: 18, Swappable: true},
		{Address: cloneTokenB, Decimals: 18, Swappable: true}}
}

func cloneReserveStrings(rs []*big.Int) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.String()
	}
	return out
}

// cloneSwap quotes 1e18 of token A for token B on p and applies the swap to p.
func cloneSwap(t *testing.T, p pool.IPoolSimulator) {
	tokenIn := pool.TokenAmount{Token: cloneTokenA, Amount: bignumber.TenPowInt(18)}
	res, err := p.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: tokenIn, TokenOut: cloneTokenB})
	if err != nil {
		t.Error(err)
		return
	}
	p.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: tokenIn, TokenAmountOut: *res.TokenAmountOut,
		Fee: *res.Fee, SwapInfo: res.SwapInfo})
}

// A swap applied to a clone moves the clone's reserves (GetReserves) and leaves the original's as they were.
func TestCloneStateSwapLeavesTheOriginalReserves(t *testing.T) {
	orig := newClonePool(t)
	before := cloneReserveStrings(orig.GetReserves())

	clone := orig.CloneState()
	cloneSwap(t, clone)

	require.NotEqual(t, before, cloneReserveStrings(clone.GetReserves()), "the clone's swap did not move its reserves")
	require.Equal(t, before, cloneReserveStrings(orig.GetReserves()), "a swap on a clone moved the original's reserves")
}

// Swaps applied to clones of one pool on two goroutines, as two requests of a router do, share no writes:
// under -race this test reports none.
func TestCloneStateSwapsOnClonesConcurrently(t *testing.T) {
	orig := newClonePool(t)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				cloneSwap(t, orig.CloneState())
			}
		}()
	}
	wg.Wait()
}

// The ezETH/WETH pool 0x707e…f171 holds about 0.0526 WETH, all within a few ticks of its active tick. Production
// routes put 6.5-9.8 ezETH into it. The pool consumes about 0.0556 ezETH and the executed swap (tickLimit =
// int32.max) then walks empty ticks until it runs out of gas.
func loadEzEthWeth(t *testing.T) (*PoolSimulator, entity.Pool) {
	t.Helper()
	data, err := os.ReadFile("./data/ezeth_weth_707e.json")
	require.NoError(t, err)
	var ep entity.Pool
	require.NoError(t, json.Unmarshal(data, &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim, ep
}

func quoteOut(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string, amountIn *big.Int) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: amountIn},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	return res
}

// fillable is the part of amountIn the pool's liquidity can take, from the swap the quote runs.
func fillable(t *testing.T, sim *PoolSimulator, zeroForOne bool, amountIn *big.Int, decimals uint8) *big.Int {
	t.Helper()
	scaled := scaleFromAmount(uint256.MustFromBig(amountIn), decimals)
	_, unfilled, _, _, err := swap(sim.state.Clone(true), scaled, zeroForOne, false)
	require.NoError(t, err)
	if unfilled == nil {
		return amountIn
	}
	return new(big.Int).Sub(amountIn, ScaleToAmount(unfilled.Clone(), decimals).ToBig())
}

// Past the liquidity the quote can reach, the swap is refused: the executor's tickLimit is int32.max, so the
// unused input would walk empty ticks until the transaction runs out of gas, and the router does not act on a
// remaining amount. The amount the pool can take still quotes as a full fill.
func TestCalcAmountOut_PastCapacityIsRefused(t *testing.T) {
	t.Parallel()
	sim, ep := loadEzEthWeth(t)
	ezEth, weth := ep.Tokens[0].Address, ep.Tokens[1].Address

	for _, tc := range []struct {
		name            string
		tokenIn, tokOut string
		amountIn        string
	}{
		{"ezETH in, 6.55 (production's chunk)", ezEth, weth, "6550000000000000000"},
		{"ezETH in, 9.83", ezEth, weth, "9825761805897544000"},
		{"WETH in, 1.0", weth, ezEth, "1000000000000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			amountIn := bignumber.NewBig10(tc.amountIn)
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tc.tokenIn, Amount: amountIn},
				TokenOut:      tc.tokOut,
			})
			require.ErrorIs(t, err, ErrInsufficientLiquidity)

			in := sim.GetTokenIndex(tc.tokenIn)
			consumed := fillable(t, sim, in == 0, amountIn, sim.decimals[in])
			require.Positive(t, consumed.Sign())
			require.Negative(t, consumed.Cmp(amountIn))
			res := quoteOut(t, sim, tc.tokenIn, tc.tokOut, consumed)
			require.Positive(t, res.TokenAmountOut.Amount.Sign())
			require.Nil(t, res.RemainingTokenAmountIn)
		})
	}
}

func TestCalcAmountOut_WithinCapacityUnchanged(t *testing.T) {
	t.Parallel()
	sim, ep := loadEzEthWeth(t)
	ezEth, weth := ep.Tokens[0].Address, ep.Tokens[1].Address

	res := quoteOut(t, sim, ezEth, weth, big.NewInt(1e16)) // 0.01 ezETH, well within capacity
	require.Nil(t, res.RemainingTokenAmountIn)
	require.Positive(t, res.TokenAmountOut.Amount.Sign())
}

func TestCalcAmountIn_PastCapacityIsRefused(t *testing.T) {
	t.Parallel()
	sim, ep := loadEzEthWeth(t)
	ezEth, weth := ep.Tokens[0].Address, ep.Tokens[1].Address

	want := bignumber.NewBig10("1000000000000000000") // 1 WETH out; the pool holds about 0.0526
	_, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: weth, Amount: want},
		TokenIn:        ezEth,
	})
	require.ErrorIs(t, err, ErrInsufficientLiquidity)
}

// The quote counts the ticks without liquidity it walks over to reach a tick it swaps in (about 2,540 gas each
// on-chain), but not the walk past the last liquidity, which the executable swap of the consumed amount never
// makes.
func TestSwap_CountsEmptyTicksWalkedOnlyUpToLiquidity(t *testing.T) {
	t.Parallel()
	sim, _ := loadEzEthWeth(t)
	amount := bignumber.NewBig10("6550000000000000000")

	full := sim.state.Clone(true)
	_, unfilled, _, empty, err := swap(full, scaleFromAmount(uint256FromBig(t, amount), 18), true, false)
	require.NoError(t, err)
	require.NotNil(t, unfilled)
	require.Zero(t, empty, "the walk past the last liquidity is not part of the executed swap")

	gapped := sim.state.Clone(true)
	_, ok := gapped.Ticks[-10]
	require.True(t, ok)
	gappedTicks := maps.Clone(gapped.Ticks)
	delete(gappedTicks, -10) // tick -10 now has no liquidity, between liquid ticks -11 and -9
	gapped.Ticks = gappedTicks
	_, unfilled, _, empty, err = swap(gapped, scaleFromAmount(uint256FromBig(t, amount), 18), true, false)
	require.NoError(t, err)
	require.NotNil(t, unfilled)
	require.Equal(t, uint32(1), empty)
	require.Equal(t, int64(GasSwap+GasCrossBin*3+GasEmptyTick*2), swapGas(3, 2))
}

func uint256FromBig(t *testing.T, b *big.Int) *uint256.Int {
	t.Helper()
	u, overflow := uint256.FromBig(b)
	require.False(t, overflow)
	return u
}
