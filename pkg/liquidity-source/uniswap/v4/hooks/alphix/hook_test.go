package alphix

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
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func TestHookRegistration(t *testing.T) {
	t.Parallel()

	// Verify all 3 hook addresses are registered
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, ok, "hook should be registered for %s", addr.Hex())
		assert.Equal(t, string(valueobject.ExchangeUniswapV4Alphix), hook.GetExchange())
	}

	// Verify an unknown address is not registered as Alphix
	unknownAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	hook, ok := uniswapv4.GetHook(unknownAddr, &uniswapv4.HookParam{})
	assert.False(t, ok, "unknown address should not be registered")
	_ = hook
}

func TestHookFactory_WithExtra(t *testing.T) {
	t.Parallel()

	extra := AlphixExtra{Fee: 3000}
	extraBytes, _ := json.Marshal(extra)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: string(extraBytes),
	})
	require.True(t, ok)

	alphixHook, ok := hook.(*Hook)
	require.True(t, ok)
	assert.Equal(t, uniswapv4.FeeAmount(3000), alphixHook.swapFee)
}

func TestBeforeSwap_ReturnsDynamicFee(t *testing.T) {
	t.Parallel()

	extra := AlphixExtra{Fee: 5000}
	extraBytes, _ := json.Marshal(extra)

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: string(extraBytes),
	})

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		ExactIn:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000), // 1 USDC
	})
	require.NoError(t, err)

	// Alphix returns the dynamic fee but does NOT take a delta
	assert.Equal(t, uniswapv4.FeeAmount(5000), result.SwapFee)
	assert.Equal(t, int64(0), result.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), result.DeltaUnspecified.Int64())
}

func TestAfterSwap_Noop(t *testing.T) {
	t.Parallel()

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{})

	result, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			ExactIn:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(1_000_000),
		AmountOut: big.NewInt(999_500),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.HookFee.Int64())
}

func TestBeforeSwap_JitSimulation(t *testing.T) {
	t.Parallel()

	// Set up a hook with JIT state: fee, tick range, yield source amounts, and sqrtPrice.
	// This simulates a USDC/USDT-like stablecoin pool at price ~1.0.
	// sqrtPriceX96 at tick 0 = 2^96 = 79228162514264337593543950336
	sqrtPriceAtTick0 := new(uint256.Int).Lsh(uint256.NewInt(1), 96)

	hook := &Hook{
		Hook:             &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook:             HookAddresses[2],
		swapFee:          3000,          // 30 bps
		tickLower:        -8,            // ±8 ticks JIT range
		tickUpper:        8,
		amount0Available: uint256.NewInt(1_000_000_000_000), // 1M USDC (6 decimals)
		amount1Available: uint256.NewInt(1_000_000_000_000), // 1M USDT (6 decimals)
		sqrtPriceX96:     sqrtPriceAtTick0,
	}

	// Swap 1000 USDC (exactIn, zeroForOne)
	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		ExactIn:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000_000), // 1000 USDC
	})
	require.NoError(t, err)

	// JIT should absorb some of the swap
	assert.Equal(t, uniswapv4.FeeAmount(3000), result.SwapFee)
	assert.True(t, result.DeltaSpecified.Sign() > 0, "JIT should consume some input")
	assert.True(t, result.DeltaUnspecified.Sign() < 0, "JIT should produce output (negative)")

	t.Logf("JIT deltaSpecified: %s, deltaUnspecified: %s",
		result.DeltaSpecified.String(), result.DeltaUnspecified.String())
}

func TestBeforeSwap_NoJitWhenNoReserves(t *testing.T) {
	t.Parallel()

	// Hook with tick range but zero reserves — should return fee only, no delta
	sqrtPrice := new(uint256.Int).Lsh(uint256.NewInt(1), 96)
	hook := &Hook{
		Hook:             &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook:             HookAddresses[0],
		swapFee:          5000,
		tickLower:        -8,
		tickUpper:        8,
		amount0Available: uint256.NewInt(0),
		amount1Available: uint256.NewInt(0),
		sqrtPriceX96:     sqrtPrice,
	}

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		ExactIn:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(5000), result.SwapFee)
	assert.Equal(t, int64(0), result.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), result.DeltaUnspecified.Int64())
}

func TestCloneState_DeepCopy(t *testing.T) {
	t.Parallel()

	original := &Hook{
		Hook:             &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook:             HookAddresses[0],
		swapFee:          3000,
		tickLower:        -8,
		tickUpper:        8,
		amount0Available: uint256.NewInt(500),
		amount1Available: uint256.NewInt(600),
		sqrtPriceX96:     uint256.NewInt(1000),
	}

	cloned := original.CloneState().(*Hook)

	// Mutate clone and verify original is unaffected
	cloned.amount0Available.SetUint64(999)
	cloned.sqrtPriceX96.SetUint64(0)

	assert.Equal(t, uint64(500), original.amount0Available.Uint64())
	assert.Equal(t, uint64(1000), original.sqrtPriceX96.Uint64())
}

func TestParseHookAddresses(t *testing.T) {
	t.Parallel()

	// Base hooks
	assert.Equal(t, common.HexToAddress("0x831CfDf7c0E194f5369f204b3DD2481B843d60c0"), HookAddresses[0])
	assert.Equal(t, common.HexToAddress("0x0e4b892Df7C5Bcf5010FAF4AA106074e555660C0"), HookAddresses[1])
	// Arbitrum hook
	assert.Equal(t, common.HexToAddress("0x5e645C3D580976Ca9e3fe77525D954E73a0Ce0C0"), HookAddresses[2])
}

// --- Live RPC tests (skipped in CI) ---

func TestTrack_BaseETHUSDC(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook: HookAddresses[0], // Base ETH/USDC
	}

	extraStr, err := hook.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0x0000000000000000000000000000000000000000"}, // ETH
				{Address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"}, // USDC
			},
		},
	})
	require.NoError(t, err)

	var extra AlphixExtra
	require.NoError(t, json.Unmarshal([]byte(extraStr), &extra))
	t.Logf("Base ETH/USDC: fee=%d ticks=[%d,%d] amount0=%s amount1=%s",
		extra.Fee, extra.TickLower, extra.TickUpper, extra.Amount0Available, extra.Amount1Available)

	assert.True(t, extra.Fee > 0, "fee should be > 0")
	assert.True(t, extra.TickLower < extra.TickUpper, "tick range should be valid")
}

func TestTrack_BaseUSDSUSDC(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook: HookAddresses[1], // Base USDS/USDC
	}

	extraStr, err := hook.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0x820c137fa70c8691f0e44dc420a5e53c168921dc"}, // USDS
				{Address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"}, // USDC
			},
		},
	})
	require.NoError(t, err)

	var extra AlphixExtra
	require.NoError(t, json.Unmarshal([]byte(extraStr), &extra))
	t.Logf("Base USDS/USDC: fee=%d ticks=[%d,%d] amount0=%s amount1=%s",
		extra.Fee, extra.TickLower, extra.TickUpper, extra.Amount0Available, extra.Amount1Available)

	assert.True(t, extra.Fee > 0, "fee should be > 0")
}

func TestTrack_ArbitrumUSDCUSDT(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").SetMulticallContract(multicall3)
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook: HookAddresses[2], // Arbitrum USDC/USDT
	}

	extraStr, err := hook.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"}, // USDC
				{Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"}, // USDT
			},
		},
	})
	require.NoError(t, err)

	var extra AlphixExtra
	require.NoError(t, json.Unmarshal([]byte(extraStr), &extra))
	t.Logf("Arb USDC/USDT: fee=%d ticks=[%d,%d] amount0=%s amount1=%s",
		extra.Fee, extra.TickLower, extra.TickUpper, extra.Amount0Available, extra.Amount1Available)

	assert.True(t, extra.Fee > 0, "fee should be > 0")
	assert.Equal(t, -8, extra.TickLower)
	assert.Equal(t, 8, extra.TickUpper)
}

func TestGetReserves_ArbitrumUSDCUSDT(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").SetMulticallContract(multicall3)
	hook := &Hook{
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		hook: HookAddresses[2],
	}

	reserves, err := hook.GetReserves(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"},
				{Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, reserves, 2)

	t.Logf("Arb USDC/USDT reserves: [%s, %s]", reserves[0], reserves[1])
	assert.NotEqual(t, "0", reserves[0], "USDC reserve should be > 0")
	assert.NotEqual(t, "0", reserves[1], "USDT reserve should be > 0")
}
