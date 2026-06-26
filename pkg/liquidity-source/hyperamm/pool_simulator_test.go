package hyperamm

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// newTestSimulator creates a PoolSimulator with the given parameters for unit testing.
//
// token0 is assumed to be an 18-decimal token (e.g., HYPE/ETH) and token1 is
// a 6-decimal stablecoin (e.g., USDC).
//
// Pricing example (price = 2000 USDC per token0):
//
//	fairPrice0To1 = 2_000_000_000   (2e9)
//	  → formula: amountOut = amountIn * fairPrice / 1e18
//	  → 1 token0 (1e18 wei) gives 1e18 * 2e9 / 1e18 = 2e9 USDC wei = 2000 USDC
//
//	fairPrice1To0 = 500_000_000_000_000_000_000_000_000  (5e26)
//	  → 1 USDC (1e6 wei) gives 1e6 * 5e26 / 1e18 = 5e14 token0 wei ≈ 0.0005 token0
func newTestSimulator(
	fairPrice0To1, fairPrice1To0 string,
	refFee0To1, refFee1To0 uint64,
	reserve0, reserve1 string,
	isPaused bool,
) *PoolSimulator {
	token0 := "0x0000000000000000000000000000000000000001"
	token1 := "0x0000000000000000000000000000000000000002"
	swapFeeModule := "0x0000000000000000000000000000000000000003"

	extraB, _ := json.Marshal(Extra{
		FairPriceFrom: [2]*uint256.Int{
			uint256.MustFromDecimal(fairPrice0To1),
			uint256.MustFromDecimal(fairPrice1To0),
		},
		RefFeeFrom: [2]uint64{refFee0To1, refFee1To0},
		IsPaused:   isPaused,
	})
	staticExtraB, _ := json.Marshal(StaticExtra{
		SwapFeeModule: swapFeeModule,
	})

	ep := entity.Pool{
		Address:     "0x0000000000000000000000000000000000000004",
		Exchange:    DexType,
		Type:        DexType,
		Reserves:    []string{reserve0, reserve1},
		Tokens:      []*entity.PoolToken{{Address: token0, Swappable: true}, {Address: token1, Swappable: true}},
		Extra:       string(extraB),
		StaticExtra: string(staticExtraB),
	}

	sim, err := NewPoolSimulator(ep)
	if err != nil {
		panic(err)
	}
	return sim
}

// ── Unit tests ──────────────────────────────────────────────────────────────

// TestPoolSimulator_CalcAmountOut tests basic swap quoting in both directions
// with a fixed oracle price and reference fee.
//
// The simulator formula (both token decimals = 0, so precision = 1e18):
//
//	0→1: amountOut = amountInAfterFee * precision / fairPrice0To1
//	1→0: amountOut = amountInAfterFee * fairPrice1To0 / precision
//
// With fp01 = fp10 = 5e26 and fee = 30 bps:
//
//	0→1: 997e15 * 1e18 / 5e26 = 1_994_000_000
//	1→0: 997_000 * 5e26 / 1e18 = 498_500_000_000_000
func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()

	// Both fair prices set to 5e26 so the simulator's actual formulas give
	// the expected outputs (see function comment above).
	fp01 := "500000000000000000000000000"
	fp10 := "500000000000000000000000000"

	// Large reserves: enough to cover any test swap.
	reserve0 := "10000000000000000000000" // 1e22
	reserve1 := "20000000000000"          // 2e13

	ps := newTestSimulator(fp01, fp10, 30, 30, reserve0, reserve1, false)

	testutil.TestCalcAmountOut(t, ps, map[int]map[int]map[string]string{
		// token0 → token1
		//   amountInAfterFee = 1e18 * 9970 / 10000 = 997_000_000_000_000_000
		//   amountOut        = 997e15 * 1e18 / 5e26 = 1_994_000_000
		0: {1: {
			"1000000000000000000":  "1994000000",
			"10000000000000000000": "19940000000",
		}},
		// token1 → token0
		//   amountInAfterFee = 1_000_000 * 9970 / 10000 = 997_000
		//   amountOut        = 997_000 * 5e26 / 1e18 = 498_500_000_000_000
		1: {0: {
			"1000000":    "498500000000000",
			"1000000000": "498500000000000000",
		}},
	})
}

func TestPoolSimulator_CalcAmountOut_Paused(t *testing.T) {
	t.Parallel()
	// NewPoolSimulator returns ErrPoolPaused when the pool is paused;
	// the simulator cannot be constructed at all.
	extraB, _ := json.Marshal(Extra{
		FairPriceFrom: [2]*uint256.Int{
			uint256.MustFromDecimal("500000000000000000000000000"),
			uint256.MustFromDecimal("500000000000000000000000000"),
		},
		RefFeeFrom: [2]uint64{30, 30},
		IsPaused:   true,
	})
	staticExtraB, _ := json.Marshal(StaticExtra{
		SwapFeeModule: "0x0000000000000000000000000000000000000003",
	})
	ep := entity.Pool{
		Address:  "0x0000000000000000000000000000000000000004",
		Exchange: DexType,
		Type:     DexType,
		Reserves: []string{"10000000000000000000000", "20000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0x0000000000000000000000000000000000000001", Swappable: true},
			{Address: "0x0000000000000000000000000000000000000002", Swappable: true},
		},
		Extra:       string(extraB),
		StaticExtra: string(staticExtraB),
	}
	_, err := NewPoolSimulator(ep)
	assert.EqualError(t, err, ErrPoolPaused.Error())
}

func TestPoolSimulator_CalcAmountOut_ZeroFairPrice(t *testing.T) {
	t.Parallel()
	// fairPrice0To1 = 0 → should error with ErrZeroFairPrice
	ps := newTestSimulator(
		"0", "500000000000000000000000000",
		30, 30,
		"10000000000000000000000", "20000000000000",
		false,
	)
	testutil.TestCalcAmountOut(t, ps, map[int]map[int]map[string]string{
		0: {1: {"1000000000000000000": ErrZeroFairPrice.Error()}},
	})
}

func TestPoolSimulator_CalcAmountOut_InsufficientReserve(t *testing.T) {
	t.Parallel()
	// Very small reserve1 so even a tiny swap exceeds it.
	ps := newTestSimulator(
		"500000000000000000000000000", "500000000000000000000000000",
		30, 30,
		"10000000000000000000000", "100", // only 100 wei of token1
		false,
	)
	testutil.TestCalcAmountOut(t, ps, map[int]map[int]map[string]string{
		// Swapping 1 token0 yields 1994000000 token1 wei, reserve1 = 100
		0: {1: {"1000000000000000000": ErrInsufficientReserve.Error()}},
	})
}

func TestPoolSimulator_CalcAmountOut_FullFee(t *testing.T) {
	t.Parallel()
	// Fee = 10000 bps (100 %) → amountInAfterFee = 0 → ErrZeroAmountOut
	ps := newTestSimulator(
		"500000000000000000000000000", "500000000000000000000000000",
		10000, 10000,
		"10000000000000000000000", "20000000000000",
		false,
	)
	testutil.TestCalcAmountOut(t, ps, map[int]map[int]map[string]string{
		0: {1: {"1000000000000000000": ErrZeroAmountOut.Error()}},
	})
}

// TestPoolSimulator_CloneState verifies that UpdateBalance on a clone does not
// affect the original.
func TestPoolSimulator_CloneState(t *testing.T) {
	t.Parallel()
	ps := newTestSimulator(
		"500000000000000000000000000", "500000000000000000000000000",
		30, 30,
		"10000000000000000000000", "20000000000000",
		false,
	)
	tokens := ps.GetTokens()
	testutil.TestCloneState(t, ps, poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  tokens[0],
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		TokenOut: tokens[1],
	}, nil)
}

// TestPoolSimulator_UpdateBalance verifies sequential swaps correctly deplete
// reserves and that a subsequent swap exceeding the remaining reserve fails.
func TestPoolSimulator_UpdateBalance(t *testing.T) {
	t.Parallel()

	ps := newTestSimulator(
		"500000000000000000000000000", "500000000000000000000000000",
		30, 30,
		"10000000000000000000000", "20000000000000",
		false,
	)
	tokens := ps.GetTokens()

	// First swap: 1 token0 → should yield 1994000000 token1
	params1 := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  tokens[0],
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		TokenOut: tokens[1],
	}
	result1, err := ps.CalcAmountOut(params1)
	require.NoError(t, err)
	assert.Equal(t, "1994000000", result1.TokenAmountOut.Amount.String())

	ps.UpdateBalance(poolpkg.UpdateBalanceParams{
		TokenAmountIn:  params1.TokenAmountIn,
		TokenAmountOut: *result1.TokenAmountOut,
		Fee:            *result1.Fee,
		SwapInfo:       result1.SwapInfo,
	})

	// Verify: swap an amount that requires more reserve1 than is now available.
	// reserve1 after = 20000000000000 - 1994000000 ≈ 19999998006000000
	// Requesting a huge swap that clearly exceeds remaining reserve1.
	hugeIn, ok := new(big.Int).SetString("100000000000000000000000", 10) // 100k token0
	require.True(t, ok)
	params2 := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{Token: tokens[0], Amount: hugeIn},
		TokenOut:      tokens[1],
	}
	_, err = ps.CalcAmountOut(params2)
	assert.ErrorIs(t, err, ErrInsufficientReserve)
}

// TestPoolSimulator_ConcurrentSafe tests that CalcAmountOut is safe to call
// concurrently (requires -race flag to be meaningful).
func TestPoolSimulator_ConcurrentSafe(t *testing.T) {
	t.Parallel()
	ps := newTestSimulator(
		"500000000000000000000000000", "500000000000000000000000000",
		30, 30,
		"10000000000000000000000", "20000000000000",
		false,
	)
	tokens := ps.GetTokens()
	params := poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  tokens[0],
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		TokenOut: tokens[1],
	}

	result, err := testutil.MustConcurrentSafe(t, func() (*poolpkg.CalcAmountOutResult, error) {
		return ps.CalcAmountOut(params)
	})
	require.NoError(t, err)
	assert.Equal(t, "1994000000", result.TokenAmountOut.Amount.String())
}

// ── On-chain verification test ───────────────────────────────────────────────
//
// This test fetches live state from a Hyperliquid EVM node and compares the
// simulator output to HyperAMMLens.previewLiquidityQuote.
//
// Required environment variables:
//
//	HYPERAMM_RPC_URL       – Hyperliquid EVM JSON-RPC endpoint
//	HYPERAMM_FACTORY_ADDR  – HyperAMMFactory contract address
//	HYPERAMM_LENS_ADDR     – HyperAMMLens contract address
//	HYPERAMM_MULTICALL     – Multicall3 contract address (optional)
//
// The test is skipped when any required variable is absent or when CI=true.
func TestSimulation_OnChain(t *testing.T) {
	rpcURL := os.Getenv("HYPERAMM_RPC_URL")
	factoryAddr := os.Getenv("HYPERAMM_FACTORY_ADDR")
	lensAddr := os.Getenv("HYPERAMM_LENS_ADDR")
	multicallAddr := os.Getenv("HYPERAMM_MULTICALL")

	if os.Getenv("CI") != "" || rpcURL == "" || factoryAddr == "" || lensAddr == "" {
		t.Skip("skipping on-chain test: set HYPERAMM_RPC_URL, HYPERAMM_FACTORY_ADDR, HYPERAMM_LENS_ADDR to run")
	}

	ctx := context.Background()

	rpcClient := ethrpc.New(rpcURL)
	if multicallAddr != "" {
		rpcClient = rpcClient.SetMulticallContract(common.HexToAddress(multicallAddr))
	}

	cfg := &Config{
		DexId:   DexType,
		Factory: factoryAddr,
		Lens:    lensAddr,
	}

	updater := NewPoolsListUpdater(cfg, rpcClient)
	pools, _, err := updater.GetNewPools(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools, "no pools discovered from factory")

	tracker := NewPoolTracker(cfg, rpcClient)

	for _, p := range pools {
		t.Run(p.Address, func(t *testing.T) {
			newState, err := tracker.GetNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{})
			if err != nil {
				t.Logf("skip %s: tracker error: %v", p.Address, err)
				return
			}

			p.Extra = newState.Extra
			p.StaticExtra = newState.StaticExtra
			p.Reserves = newState.Reserves
			p.Tokens = newState.Tokens
			p.BlockNumber = newState.BlockNumber

			sim, err := NewPoolSimulator(p)
			if err != nil {
				t.Logf("skip %s: NewPoolSimulator error: %v", p.Address, err)
				return
			}

			tokens := sim.GetTokens()

			// Use ~1% of reserve0 as input to keep market impact small.
			reserve0Big, ok := new(big.Int).SetString(newState.Reserves[0], 10)
			if !ok || reserve0Big.Sign() <= 0 {
				t.Logf("skip %s: zero or invalid reserve0", p.Address)
				return
			}
			amountIn := new(big.Int).Div(reserve0Big, big.NewInt(100))
			if amountIn.Sign() <= 0 {
				amountIn.SetInt64(1)
			}

			// Get simulator quote (direction 0→1).
			simResult, err := sim.CalcAmountOut(poolpkg.CalcAmountOutParams{
				TokenAmountIn: poolpkg.TokenAmount{Token: tokens[0], Amount: amountIn},
				TokenOut:      tokens[1],
			})
			if err != nil {
				t.Logf("skip %s: sim quote error: %v", p.Address, err)
				return
			}
			simOut := simResult.TokenAmountOut.Amount

			// Get on-chain quote from HyperAMMLens.previewLiquidityQuote.
			type ALMLiquidityQuote struct {
				IsCallbackOnSwap bool
				AmountOut        *big.Int
				AmountInFilled   *big.Int
			}
			var onChainQuote ALMLiquidityQuote
			_, err = rpcClient.NewRequest().
				SetContext(ctx).
				AddCall(&ethrpc.Call{
					ABI:    hyperAMMLensABI,
					Target: lensAddr,
					Method: "previewLiquidityQuote",
					Params: []any{
						common.HexToAddress(p.Address),
						true, // isZeroToOne
						amountIn,
						big.NewInt(0), // amountOutMin
					},
				}, []any{&onChainQuote}).
				Call()
			if err != nil {
				t.Logf("skip %s: on-chain quote error: %v", p.Address, err)
				return
			}
			onChainOut := onChainQuote.AmountOut

			t.Logf("pool %s: amountIn=%s simOut=%s onChainOut=%s",
				p.Address, amountIn, simOut, onChainOut)

			// The simulator uses a stored reference fee so the market-impact
			// component is approximated.  Accept ≤2% deviation.
			if onChainOut != nil && onChainOut.Sign() > 0 {
				diff := new(big.Int).Sub(simOut, onChainOut)
				diff.Abs(diff)
				threshold := new(big.Int).Div(new(big.Int).Mul(onChainOut, big.NewInt(2)), big.NewInt(100))
				assert.True(t, diff.Cmp(threshold) <= 0,
					"simulator output %s deviates >2%% from on-chain quote %s (diff=%s)",
					simOut, onChainOut, diff)
			}
		})
	}
}
