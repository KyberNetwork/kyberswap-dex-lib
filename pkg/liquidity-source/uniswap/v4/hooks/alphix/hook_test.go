package alphix

import (
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
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var multicall3 = common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")

func TestHookRegistration(t *testing.T) {
	t.Parallel()

	// Verify all 2 hook addresses are registered
	for _, addr := range HookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, ok, "hook should be registered for %s", addr.Hex())
		assert.Equal(t, valueobject.ExchangeUniswapV4Alphix, hook.GetExchange())
	}

	// Verify an unknown address is not registered as Alphix
	unknownAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	hook, ok := uniswapv4.GetHook(unknownAddr, &uniswapv4.HookParam{})
	assert.False(t, ok, "unknown address should not be registered")
	_ = hook
}

func TestHookFactory_WithExtra(t *testing.T) {
	t.Parallel()

	h := Hook{SwapFee: 3000}
	extraBytes, _ := json.Marshal(h)

	hook, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})
	require.True(t, ok)

	alphixHook, ok := hook.(*Hook)
	require.True(t, ok)
	assert.Equal(t, uniswapv4.FeeAmount(3000), alphixHook.SwapFee)
}

func TestBeforeSwap_ReturnsDynamicFee(t *testing.T) {
	t.Parallel()

	h := Hook{SwapFee: 5000}
	extraBytes, _ := json.Marshal(h)

	hook, _ := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
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
			CalcOut:         true,
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
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		ExtraTickU256: uniswapv3.ExtraTickU256{
			SqrtPriceX96: sqrtPriceAtTick0,
		},
		SwapFee:          3000, // 30 bps
		TickLower:        -8,   // ±8 ticks JIT range
		TickUpper:        8,
		Amount0Available: uint256.NewInt(1_000_000_000_000), // 1M USDC (6 decimals)
		Amount1Available: uint256.NewInt(1_000_000_000_000), // 1M USDT (6 decimals)
		PoolManagerBalances: [2]*uint256.Int{
			uint256.NewInt(1_000_000_000),
			uint256.NewInt(1_000_000_000),
		},
	}

	// Swap 1000 USDC (exactIn, zeroForOne)
	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
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
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		ExtraTickU256: uniswapv3.ExtraTickU256{
			SqrtPriceX96: sqrtPrice,
		},
		SwapFee:          5000,
		TickLower:        -8,
		TickUpper:        8,
		Amount0Available: uint256.NewInt(0),
		Amount1Available: uint256.NewInt(0),
		PoolManagerBalances: [2]*uint256.Int{
			uint256.NewInt(1_000_000_000),
			uint256.NewInt(1_000_000_000),
		},
	}

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
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
		Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		ExtraTickU256: uniswapv3.ExtraTickU256{
			SqrtPriceX96: uint256.NewInt(1000),
		},
		SwapFee:          3000,
		TickLower:        -8,
		TickUpper:        8,
		Amount0Available: uint256.NewInt(500),
		Amount1Available: uint256.NewInt(600),
		PoolManagerBalances: [2]*uint256.Int{
			uint256.NewInt(1_000_000_000),
			uint256.NewInt(1_000_000_000),
		},
	}

	cloned := original.CloneState().(*Hook)

	// Mutate clone and verify original is unaffected
	cloned.Amount0Available = uint256.NewInt(999)
	cloned.SqrtPriceX96 = uint256.NewInt(0)

	assert.Equal(t, uint64(500), original.Amount0Available.Uint64())
	assert.Equal(t, uint64(1000), original.SqrtPriceX96.Uint64())
}

func TestParseHookAddresses(t *testing.T) {
	t.Parallel()

	// JIT hooks
	assert.Equal(t, common.HexToAddress("0x0e4b892Df7C5Bcf5010FAF4AA106074e555660C0"), HookAddresses[0])
	assert.Equal(t, common.HexToAddress("0x5e645C3D580976Ca9e3fe77525D954E73a0Ce0C0"), HookAddresses[1])
	// LVR fee hook
	assert.Equal(t, common.HexToAddress("0x7cBbfF9C4fcd74B221C535F4fB4B1Db04F1B9044"), LvrFeeHookAddresses[0])
}

// --- LvrFeeHook tests ---

func TestLvrFeeHook_Registration(t *testing.T) {
	t.Parallel()

	for _, addr := range LvrFeeHookAddresses {
		hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{})
		assert.True(t, ok, "hook should be registered for %s", addr.Hex())
		assert.Equal(t, valueobject.ExchangeUniswapV4Alphix, hook.GetExchange())
	}
}

func TestLvrFeeHook_Factory_WithExtra(t *testing.T) {
	t.Parallel()

	extra := LvrFeeExtra{SwapFee: 499, HookFee: 5000}
	extraBytes, _ := json.Marshal(extra)

	hook, ok := uniswapv4.GetHook(LvrFeeHookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})
	require.True(t, ok)

	lvrHook, ok := hook.(*LvrFeeHook)
	require.True(t, ok)
	assert.Equal(t, uniswapv4.FeeAmount(499), lvrHook.SwapFee)
	assert.Equal(t, int64(5000), lvrHook.HookFee)
}

func TestLvrFeeHook_BeforeSwap(t *testing.T) {
	t.Parallel()

	extra := LvrFeeExtra{SwapFee: 499}
	extraBytes, _ := json.Marshal(extra)

	hook, _ := uniswapv4.GetHook(LvrFeeHookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})

	result, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut:         true,
		ZeroForOne:      true,
		AmountSpecified: big.NewInt(1_000_000),
	})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(499), result.SwapFee)
	assert.Equal(t, int64(0), result.DeltaSpecified.Int64())
	assert.Equal(t, int64(0), result.DeltaUnspecified.Int64())
}

func TestLvrFeeHook_AfterSwap_NoHookFee(t *testing.T) {
	t.Parallel()

	hook, _ := uniswapv4.GetHook(LvrFeeHookAddresses[0], &uniswapv4.HookParam{})

	result, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000),
		},
		AmountIn:  big.NewInt(1_000_000),
		AmountOut: big.NewInt(999_500),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.HookFee.Int64())
}

func TestLvrFeeHook_AfterSwap_WithHookFee(t *testing.T) {
	t.Parallel()

	extra := LvrFeeExtra{SwapFee: 499, HookFee: 5000}
	extraBytes, _ := json.Marshal(extra)

	hook, _ := uniswapv4.GetHook(LvrFeeHookAddresses[0], &uniswapv4.HookParam{
		HookExtra: extraBytes,
	})

	result, err := hook.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{
			CalcOut:         true,
			ZeroForOne:      true,
			AmountSpecified: big.NewInt(1_000_000_000),
		},
		AmountIn:  big.NewInt(1_000_000_000),
		AmountOut: big.NewInt(999_500_000),
	})
	require.NoError(t, err)
	// 999_500_000 * 5000 / 1_000_000 = 4_997_500
	assert.Equal(t, int64(4_997_500), result.HookFee.Int64())
}

func TestLvrFeeHook_CloneState(t *testing.T) {
	t.Parallel()

	original := &LvrFeeHook{
		Hook:    &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Alphix},
		SwapFee: 499,
		HookFee: 5000,
	}

	cloned := original.CloneState().(*LvrFeeHook)
	cloned.SwapFee = 1000
	cloned.HookFee = 0

	assert.Equal(t, uniswapv4.FeeAmount(499), original.SwapFee)
	assert.Equal(t, int64(5000), original.HookFee)
}

// --- Live RPC tests (skipped in CI) ---

func TestTrack_BaseUSDSUSDC(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 8453},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0x820c137fa70c8691f0e44dc420a5e53c168921dc"}, // USDS
				{Address: "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"}, // USDC
			},
			Extra: `{"sqrtPriceX96":79228143988102516226390}`,
		},
	}
	hook, _ := uniswapv4.GetHook(HookAddresses[0], param)
	_, err := hook.GetReserves(t.Context(), param)
	require.NoError(t, err)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var h Hook
	require.NoError(t, json.Unmarshal(extraStr, &h))
	t.Logf("Base USDS/USDC: fee=%d ticks=[%d,%d] amount0=%s amount1=%s",
		h.SwapFee, h.TickLower, h.TickUpper, h.Amount0Available, h.Amount1Available)

	assert.True(t, h.SwapFee > 0, "fee should be > 0")
}

func TestTrack_ArbitrumUSDCUSDT(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 42161},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"}, // USDC
				{Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"}, // USDT
			},
			Extra: `{"sqrtPriceX96":79221058094279577424188345191}`,
		},
	}
	hook, _ := uniswapv4.GetHook(HookAddresses[1], param)
	_, err := hook.GetReserves(t.Context(), param)
	require.NoError(t, err)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var h Hook
	require.NoError(t, json.Unmarshal(extraStr, &h))
	t.Logf("Arb USDC/USDT: fee=%d ticks=[%d,%d] amount0=%s amount1=%s",
		h.SwapFee, h.TickLower, h.TickUpper, h.Amount0Available, h.Amount1Available)

	assert.True(t, h.SwapFee > 0, "fee should be > 0")
	assert.Equal(t, -8, h.TickLower)
	assert.Equal(t, 8, h.TickUpper)
}

func TestGetReserves_ArbitrumUSDCUSDT(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://arb1.arbitrum.io/rpc").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 42161},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Tokens: []*entity.PoolToken{
				{Address: "0xaf88d065e77c8cC2239327C5EDb3A432268e5831"},
				{Address: "0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"},
			},
			Extra: `{"sqrtPriceX96":79221058094279577424188345191}`,
		},
	}
	hook, _ := uniswapv4.GetHook(HookAddresses[1], param)
	reserves, err := hook.GetReserves(t.Context(), param)
	require.NoError(t, err)
	require.Len(t, reserves, 2)

	t.Logf("Arb USDC/USDT reserves: [%s, %s]", reserves[0], reserves[1])
	assert.NotEqual(t, "0", reserves[0], "USDC reserve should be > 0")
	assert.NotEqual(t, "0", reserves[1], "USDT reserve should be > 0")
}

func TestLvrFeeHook_Track_BaseETHUSDC(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 8453},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Address: "0xebb666a5c6449b83536950b975d74deb32aca1537a501b58161a896816b04da6",
		},
	}
	hook, _ := uniswapv4.GetHook(LvrFeeHookAddresses[0], param)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra LvrFeeExtra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Base ETH/USDC (LVRFee): swapFee=%d hookFee=%d", extra.SwapFee, extra.HookFee)
}

func TestLvrFeeHook_Track_BaseETHcbBTC(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping RPC test in CI")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").SetMulticallContract(multicall3)
	param := &uniswapv4.HookParam{
		Cfg:       &uniswapv4.Config{ChainID: 8453},
		RpcClient: rpcClient,
		Pool: &entity.Pool{
			Address: "0x3860784278e9e481ffd0888430ab2af8f2bb1180069f31cde9e1066728bbe73b",
		},
	}
	hook, _ := uniswapv4.GetHook(LvrFeeHookAddresses[0], param)
	extraStr, err := hook.Track(t.Context(), param)
	require.NoError(t, err)

	var extra LvrFeeExtra
	require.NoError(t, json.Unmarshal(extraStr, &extra))
	t.Logf("Base ETH/cbBTC (LVRFee): swapFee=%d hookFee=%d", extra.SwapFee, extra.HookFee)
}
